package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"time"

	"goravel/app/facades"
	"goravel/app/models"
)

type emailNotice struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Link    string `json:"link"`
}
type emailNoticeRow struct {
	ID       uint
	UserID   uint
	Payload  string
	Attempts int
}

func (emailNoticeRow) TableName() string { return "notification_email_outbox" }

// CreateUserNotification persists the inbox item and, when requested, its email
// in the caller's business transaction.
func CreateUserNotification(tx interface{ Create(any) error }, userID uint, eventType, title, message, resourceType string, resourceID uint, email bool, actorUserIDs ...uint) error {
	if userID == 0 || len(title) > 160 || len(message) > 1000 {
		return errors.New("invalid notification")
	}
	var rid *uint
	if resourceID > 0 {
		rid = &resourceID
	}
	var actorID *uint
	if len(actorUserIDs) > 0 && actorUserIDs[0] > 0 {
		id := actorUserIDs[0]
		actorID = &id
	}
	n := models.UserNotification{UserID: userID, ActorUserID: actorID, EventType: eventType, Title: title, Message: message, ResourceType: resourceType, ResourceID: rid, CreatedAt: time.Now().UTC()}
	if err := tx.Create(&n); err != nil {
		return err
	}
	if !email {
		return nil
	}
	var user models.User
	if err := facades.Orm().Query().Where("id = ? AND is_active = ?", userID, true).First(&user); err != nil {
		return err
	}
	link := ""
	if resourceType == "load" && resourceID > 0 {
		link = fmt.Sprintf("%s/loads/%d", facades.Config().GetString("http.url"), resourceID)
	}
	body, err := json.Marshal(emailNotice{To: user.Email, Subject: "Notificación | BUFALO: " + title, Title: title, Message: message, Link: link})
	if err != nil {
		return err
	}
	sealed, err := facades.Crypt().EncryptString(string(body))
	if err != nil {
		return err
	}
	return tx.Create(&emailNoticeRow{UserID: userID, Payload: sealed})
}

func deliverNotice(id uint) error {
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	token := hex.EncodeToString(nonce[:])
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var rows []emailNoticeRow
	err := facades.DB().WithContext(ctx).Select(&rows, `UPDATE notification_email_outbox SET lease_token=$1,lease_until=CURRENT_TIMESTAMP+interval '2 minutes',attempts=attempts+1 WHERE id=$2 AND sent_at IS NULL AND available_at<=CURRENT_TIMESTAMP AND (lease_until IS NULL OR lease_until<CURRENT_TIMESTAMP) RETURNING id,payload,attempts`, token, id)
	if err != nil || len(rows) == 0 {
		return err
	}
	row := rows[0]
	plain, err := facades.Crypt().DecryptString(row.Payload)
	var notice emailNotice
	if err == nil {
		err = json.Unmarshal([]byte(plain), &notice)
	}
	if err == nil {
		body := fmt.Sprintf("<p><strong>%s</strong></p><p>%s</p>", html.EscapeString(notice.Title), html.EscapeString(notice.Message))
		if notice.Link != "" {
			body += fmt.Sprintf(`<p><a href="%s">Abrir BUFALO</a></p>`, html.EscapeString(notice.Link))
		}
		err = NewEmailService().Send(facades.Config().GetString("mail.from.address"), notice.To, notice.Subject, body)
	}
	finish, cancelFinish := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFinish()
	if err == nil {
		return facades.DB().WithContext(finish).Statement(`UPDATE notification_email_outbox SET sent_at=CURRENT_TIMESTAMP,payload='',lease_until=NULL,lease_token=NULL WHERE id=$1 AND lease_token=$2`, id, token)
	}
	seconds := min(30*(1<<min(row.Attempts-1, 7)), 3600)
	if updateErr := facades.DB().WithContext(finish).Statement(`UPDATE notification_email_outbox SET available_at=CURRENT_TIMESTAMP+$1*interval '1 second',lease_until=NULL,lease_token=NULL WHERE id=$2 AND lease_token=$3`, seconds, id, token); updateErr != nil {
		return updateErr
	}
	return err
}

func ProcessEmailNoticeOutbox(ctx context.Context) {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var rows []emailNoticeRow
	if err := facades.DB().WithContext(queryCtx).Select(&rows, `SELECT id FROM notification_email_outbox WHERE sent_at IS NULL AND available_at<=CURRENT_TIMESTAMP AND (lease_until IS NULL OR lease_until<CURRENT_TIMESTAMP) ORDER BY available_at,id LIMIT 8`); err != nil {
		return
	}
	for _, r := range rows {
		if ctx.Err() != nil {
			return
		}
		if err := deliverNotice(r.ID); err != nil {
			log.Printf("user notification email deferred id=%d", r.ID)
		}
	}
	cleanup, cancelCleanup := context.WithTimeout(ctx, 3*time.Second)
	defer cancelCleanup()
	_ = facades.DB().WithContext(cleanup).Statement(`DELETE FROM notification_email_outbox WHERE sent_at<CURRENT_TIMESTAMP-interval '7 days' AND id IN (SELECT id FROM notification_email_outbox WHERE sent_at<CURRENT_TIMESTAMP-interval '7 days' ORDER BY sent_at LIMIT 256)`)
}

func ListUserNotifications(userID uint, page int) ([]models.UserNotification, int64, error) {
	page, _ = NormalizePagination(page, 30)
	q := facades.Orm().Query().Model(&models.UserNotification{}).Where("user_id = ?", userID)
	total, err := q.Count()
	if err != nil {
		return nil, 0, err
	}
	var rows []models.UserNotification
	err = facades.Orm().Query().With("Actor").Where("user_id = ?", userID).Order("created_at desc,id desc").Limit(30).Offset((page - 1) * 30).Find(&rows)
	return rows, total, err
}

func UnreadUserNotificationCount(userID uint) (int64, error) {
	return facades.Orm().Query().Model(&models.UserNotification{}).Where("user_id = ? AND read_at IS NULL", userID).Count()
}

func MarkUserNotificationRead(userID, notificationID uint) error {
	_, err := facades.Orm().Query().Model(&models.UserNotification{}).Where("id = ? AND user_id = ?", notificationID, userID).Where("read_at IS NULL").Update("read_at", time.Now().UTC())
	return err
}
