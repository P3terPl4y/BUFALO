package redteam

import (
	"context"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
	"sync"
	"sync/atomic"
	"testing"
)

func TestInterestOutboxCrashRecoveryAndConcurrentClaims(t *testing.T) {
	tests.ResetDB(t)
	fx := seedFixture(t, 6)
	var driver principal
	for _, p := range fx.people {
		if p.role == "chofer" {
			driver = p
		}
	}
	// Queue without sending models a process crash between commit and SMTP.
	if err := services.SendLoadInterest(driver.user.ID, fx.raceLoadID, "Persistent comment", nil); err != nil {
		t.Fatal(err)
	}
	var ids []struct{ ID uint }
	if err := facades.DB().Select(&ids, "SELECT id FROM notification_outbox WHERE sent_at IS NULL"); err != nil || len(ids) != 1 {
		t.Fatalf("missing queued message: %v", err)
	}
	if err := facades.DB().Statement("UPDATE notification_outbox SET lease_token='abandoned',lease_until=CURRENT_TIMESTAMP - interval '1 second'"); err != nil {
		t.Fatal(err)
	}
	var sent atomic.Int32
	sender := func(*models.User, *models.User, *models.Carga, string) error { sent.Add(1); return nil }
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := services.DeliverInterest(ids[0].ID, sender); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if sent.Load() != 1 {
		t.Fatalf("concurrent deliveries: %d", sent.Load())
	}
	if err := services.ProcessInterestOutbox(context.Background(), sender); err != nil {
		t.Fatal(err)
	}
	if sent.Load() != 1 {
		t.Fatal("already sent notification replayed")
	}
}
func TestCompanyMembershipRequiresAdministratorApproval(t *testing.T) {
	tests.ResetDB(t)
	fx := seedFixture(t, 6)
	var driver, admin principal
	for _, p := range fx.people {
		if p.role == "chofer" {
			driver = p
		}
		if p.role == "admin" {
			admin = p
		}
	}
	if driver.user == nil || admin.user == nil {
		t.Fatal("missing fixture principals")
	}
	if _, err := facades.Orm().Query().Model(&models.Chofer{}).Where("user_id = ?", driver.user.ID).Update("empresa_id", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := facades.Orm().Query().Model(&models.User{}).Where("id = ?", driver.user.ID).Update("empresa_id", nil); err != nil {
		t.Fatal(err)
	}
	if err := services.RequestCompanyMembership(driver.user.ID, fx.carrierID); err != nil {
		t.Fatal(err)
	}
	if err := services.RequestCompanyMembership(driver.user.ID, fx.carrierID); err == nil {
		t.Fatal("duplicate request allowed")
	}
	var row services.CompanyMembershipRequest
	if err := facades.Orm().Query().First(&row); err != nil {
		t.Fatal(err)
	}
	if err := services.DecideCompanyMembership(driver.user.ID, row.ID, true); err == nil {
		t.Fatal("driver approved own membership")
	}
	if err := services.DecideCompanyMembership(admin.user.ID, row.ID, true); err != nil {
		t.Fatal(err)
	}
	var profile models.Chofer
	if err := facades.Orm().Query().Where("user_id = ?", driver.user.ID).First(&profile); err != nil {
		t.Fatal(err)
	}
	if profile.EmpresaID != fx.carrierID {
		t.Fatal("membership not applied")
	}
	if err := services.DecideCompanyMembership(admin.user.ID, row.ID, true); err == nil {
		t.Fatal("decision replay allowed")
	}
}
