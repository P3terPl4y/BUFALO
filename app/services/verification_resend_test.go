package services_test

import (
	"errors"
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestVerificationResendCapabilityQuotaAndOriginalToken(t *testing.T) {
	tests.ResetDB(t)
	svc := services.NewEmailVerificationService()
	token, err := svc.Start("resend@test.invalid", `{"role":"chofer"}`)
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := svc.IssueResendTicket(token)
	if err != nil {
		t.Fatal(err)
	}
	if ticket == token {
		t.Fatal("activation token disclosed")
	}
	var calls atomic.Int32
	send := func(email, got string) error {
		if email != "resend@test.invalid" || got != token {
			t.Error("delivery changed")
		}
		calls.Add(1)
		return nil
	}
	if err := svc.Resend(token, send); err == nil {
		t.Fatal("confirmation token accepted as resend capability")
	}
	if err := svc.Resend(ticket, send); err == nil {
		t.Fatal("cooldown bypassed")
	}
	for attempt := 0; attempt < 5; attempt++ {
		if err := facades.DB().Statement("UPDATE pending_registrations SET last_sent_at=CURRENT_TIMESTAMP - interval '2 minutes'"); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); _ = svc.Resend(ticket, send) }()
		}
		wg.Wait()
		if calls.Load() != int32(attempt+1) {
			t.Fatal("concurrent resend quota bypass")
		}
	}
	if err := facades.DB().Statement("UPDATE pending_registrations SET last_sent_at=CURRENT_TIMESTAMP - interval '2 minutes'"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Resend(ticket, send); err == nil {
		t.Fatal("total quota bypassed")
	}
	// Confirmation still consumes the original token; callbacks cannot be called
	// with the independent browser resend capability.
	if err := svc.Confirm(ticket, nil); !errors.Is(err, services.ErrInvalidVerificationToken) {
		t.Fatal("resend ticket activates account")
	}
	var pending models.PendingRegistration
	if err := facades.Orm().Query().Where("email = ?", "resend@test.invalid").First(&pending); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pending.Payload, "chofer") || pending.TokenCiphertext == nil || strings.Contains(*pending.TokenCiphertext, token) {
		t.Fatal("plaintext secrets persisted")
	}
	if err := svc.Confirm(token, func(_ orm.Query, payload string) error {
		if !strings.Contains(payload, "chofer") {
			t.Fatal("payload changed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Resend(ticket, send); err == nil {
		t.Fatal("consumed registration resent")
	}

}
