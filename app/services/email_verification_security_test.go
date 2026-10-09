package services_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"

	"github.com/goravel/framework/contracts/database/orm"
)

func TestPendingRegistrationCannotBeReplacedByAnotherRequest(t *testing.T) {
	tests.ResetDB(t)
	s := services.NewEmailVerificationService()
	token, err := s.Start("owner@test.com", "original payload")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Start(" OWNER@test.com ", "attacker replacement")
	if err == nil {
		t.Fatal("second unauthenticated request replaced the original token")
	}
	err = s.Confirm(token, func(_ orm.Query, payload string) error {
		if payload != "original payload" {
			return errors.New("payload replaced")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("original token must remain valid: %v", err)
	}
}

func TestConcurrentConfirmationConsumesTokenExactlyOnce(t *testing.T) {
	tests.ResetDB(t)
	s := services.NewEmailVerificationService()
	token, err := s.Start("concurrent@test.com", "payload")
	if err != nil {
		t.Fatal(err)
	}
	var callbacks, successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Confirm(token, func(tx orm.Query, _ string) error {
				callbacks.Add(1)
				return tx.Create(&models.User{Name: "Confirmed", Email: "concurrent@test.com", Password: "hash", Role: "chofer", IsActive: true})
			})
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, services.ErrInvalidVerificationToken) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || callbacks.Load() != 1 || tests.CountUsers(t) != 1 {
		t.Fatalf("successes=%d callbacks=%d users=%d", successes.Load(), callbacks.Load(), tests.CountUsers(t))
	}
}

func TestConcurrentPendingRegistrationKeepsOneToken(t *testing.T) {
	tests.ResetDB(t)
	s := services.NewEmailVerificationService()
	var successes atomic.Int32
	var wg sync.WaitGroup
	for n := 0; n < 16; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Start(" ONE@test.com ", "payload")
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, services.ErrRegistrationUnavailable) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	count, err := facades.Orm().Query().Model(&models.PendingRegistration{}).Count()
	if err != nil || count != 1 || successes.Load() != 1 {
		t.Fatalf("success=%d count=%d err=%v", successes.Load(), count, err)
	}
}

func TestConfirmationRechecksExpiryAfterWaitingForRowLock(t *testing.T) {
	tests.ResetDB(t)
	s := services.NewEmailVerificationService()
	token, err := s.Start("lock-expiry@test.com", "payload")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := facades.Orm().Query().Model(&models.PendingRegistration{}).Where("email = ?", "lock-expiry@test.com").Update("expires_at", time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var pending models.PendingRegistration
	if err := tx.Model(&models.PendingRegistration{}).Where("email = ?", "lock-expiry@test.com").LockForUpdate().First(&pending); err != nil {
		t.Fatal(err)
	}
	var called atomic.Bool
	done := make(chan error, 1)
	go func() { done <- s.Confirm(token, func(orm.Query, string) error { called.Store(true); return nil }) }()
	// The token expires while Confirm waits for this transaction's row lock.
	time.Sleep(time.Until(pending.ExpiresAt) + 100*time.Millisecond)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, services.ErrInvalidVerificationToken) || called.Load() {
		t.Fatalf("expired token accepted after lock wait: err=%v callback=%v", err, called.Load())
	}
}

func TestPendingCleanupIsBoundedAndPreservesLiveTokens(t *testing.T) {
	tests.ResetDB(t)
	for n := 0; n < 4; n++ {
		expires := time.Now().Add(-time.Hour)
		if n == 3 {
			expires = time.Now().Add(time.Hour)
		}
		row := models.PendingRegistration{Email: fmt.Sprintf("cleanup-%d@test.com", n), TokenHash: fmt.Sprintf("token-%d", n), Payload: "encrypted", ExpiresAt: expires}
		if err := facades.Orm().Query().Create(&row); err != nil {
			t.Fatal(err)
		}
	}
	if err := services.CleanupPending(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	count, err := facades.Orm().Query().Model(&models.PendingRegistration{}).Count()
	if err != nil || count != 2 {
		t.Fatal(count, err)
	}
	if err := services.CleanupPending(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	count, err = facades.Orm().Query().Model(&models.PendingRegistration{}).Count()
	if err != nil || count != 1 {
		t.Fatal(count, err)
	}
	var live models.PendingRegistration
	if err := facades.Orm().Query().Where("email = ?", "cleanup-3@test.com").First(&live); err != nil {
		t.Fatal("live token removed", err)
	}
}
