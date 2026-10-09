package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
	"log"
	"time"
)

type interestDelivery struct {
	Driver    models.User
	Publisher models.User
	Load      models.Carga
	Comment   string
}
type interestOutbox struct {
	ID          uint
	InterestID  uint
	Payload     string
	Attempts    int
	AvailableAt time.Time
	CreatedAt   time.Time
}

func (interestOutbox) TableName() string { return "notification_outbox" }

// Claims are atomic across processes. A crashed worker loses its lease after
// two minutes. SMTP acceptance followed by a crash can still cause a duplicate:
// delivery is at least once, not exactly once.
func DeliverInterest(id uint, sender InterestSender) error {
	if sender == nil {
		return errors.New("notification sender unavailable")
	}
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	token := hex.EncodeToString(random[:])
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var rows []interestOutbox
	err := facades.DB().WithContext(ctx).Select(&rows, `UPDATE notification_outbox SET lease_token=$1, lease_until=CURRENT_TIMESTAMP + interval '2 minutes', attempts=attempts+1 WHERE id=$2 AND sent_at IS NULL AND available_at <= CURRENT_TIMESTAMP AND (lease_until IS NULL OR lease_until < CURRENT_TIMESTAMP) RETURNING id, payload, attempts`, token, id)
	if err != nil || len(rows) == 0 {
		return err
	}
	row := rows[0]
	plaintext, deliveryErr := facades.Crypt().DecryptString(row.Payload)
	var delivery interestDelivery
	if deliveryErr == nil {
		deliveryErr = json.Unmarshal([]byte(plaintext), &delivery)
	}
	if deliveryErr == nil {
		deliveryErr = sender(&delivery.Driver, &delivery.Publisher, &delivery.Load, delivery.Comment)
	}
	finishCtx, finishCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer finishCancel()
	if deliveryErr == nil {
		return facades.DB().WithContext(finishCtx).Statement(`UPDATE notification_outbox SET sent_at=CURRENT_TIMESTAMP,payload='',lease_until=NULL,lease_token=NULL WHERE id=$1 AND lease_token=$2`, id, token)
	}
	// Exponential backoff capped at one hour. Retain undelivered messages instead
	// of silently dropping them after an arbitrary number of attempts.
	seconds := 30 * (1 << min(row.Attempts-1, 7))
	seconds = min(seconds, 3600)
	if err := facades.DB().WithContext(finishCtx).Statement(`UPDATE notification_outbox SET available_at=CURRENT_TIMESTAMP + $1 * interval '1 second',lease_until=NULL,lease_token=NULL WHERE id=$2 AND lease_token=$3`, seconds, id, token); err != nil {
		return err
	}
	return deliveryErr
}

func ProcessInterestOutbox(ctx context.Context, sender InterestSender) error {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var rows []interestOutbox
	if err := facades.DB().WithContext(queryCtx).Select(&rows, `SELECT id FROM notification_outbox WHERE sent_at IS NULL AND available_at <= CURRENT_TIMESTAMP AND (lease_until IS NULL OR lease_until < CURRENT_TIMESTAMP) ORDER BY available_at,id LIMIT 8`); err != nil {
		return err
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := DeliverInterest(row.ID, sender); err != nil {
			// Never log the encrypted payload, message, email, or SMTP error contents.
			log.Printf("notification delivery deferred id=%d", row.ID)
		}
	}
	cleanupCtx, cleanupCancel := context.WithTimeout(ctx, 3*time.Second)
	defer cleanupCancel()
	return facades.DB().WithContext(cleanupCtx).Statement(`DELETE FROM notification_outbox WHERE id IN (SELECT id FROM notification_outbox WHERE sent_at < CURRENT_TIMESTAMP - interval '7 days' ORDER BY sent_at LIMIT 256)`)
}
func RunInterestOutbox(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ProcessInterestOutbox(ctx, NewEmailService().SendLoadInterest); err != nil && ctx.Err() == nil {
				log.Print("notification outbox unavailable")
			}
		}
	}
}

// InterestOutboxStatus exposes counts and age only, never message contents.
func InterestOutboxStatus(parent context.Context) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	var rows []struct {
		Pending       int64
		Retrying      int64
		OldestSeconds float64
	}
	err := facades.DB().WithContext(ctx).Select(&rows, `SELECT count(*) AS pending,count(*) FILTER (WHERE attempts>0) AS retrying,COALESCE(EXTRACT(EPOCH FROM CURRENT_TIMESTAMP-min(created_at)),0)::double precision AS oldest_seconds FROM notification_outbox WHERE sent_at IS NULL`)
	if err != nil {
		return map[string]any{"status": "unavailable"}, err
	}
	if len(rows) == 0 {
		return map[string]any{"status": "unavailable"}, errors.New("outbox status unavailable")
	}
	r := rows[0]
	status := "ok"
	if r.OldestSeconds > 300 {
		status = "delayed"
	}
	return map[string]any{"status": status, "pending": r.Pending, "retrying": r.Retrying, "oldest_seconds": r.OldestSeconds}, nil
}
