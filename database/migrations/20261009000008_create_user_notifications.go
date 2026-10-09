package migrations

import "goravel/app/facades"

type M20261009000008CreateUserNotifications struct{}

func (*M20261009000008CreateUserNotifications) Signature() string {
	return "20261009000008_create_user_notifications"
}
func (*M20261009000008CreateUserNotifications) Up() error {
	_, err := facades.Orm().Query().Exec(`CREATE TABLE IF NOT EXISTS user_notifications (
 id bigserial PRIMARY KEY,user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 event_type varchar(64) NOT NULL,title varchar(160) NOT NULL,message varchar(1000) NOT NULL,
 resource_type varchar(32) NOT NULL DEFAULT '',resource_id bigint,read_at timestamptz,created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
 ); CREATE INDEX IF NOT EXISTS idx_user_notifications_inbox ON user_notifications(user_id,created_at DESC,id DESC);
 CREATE TABLE IF NOT EXISTS notification_email_outbox (
 id bigserial PRIMARY KEY,user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 payload text NOT NULL,attempts integer NOT NULL DEFAULT 0 CHECK(attempts>=0),available_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 lease_until timestamptz,lease_token varchar(64),sent_at timestamptz,created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
 ); CREATE INDEX IF NOT EXISTS idx_notification_email_pending ON notification_email_outbox(available_at,id) WHERE sent_at IS NULL`)
	return err
}
func (*M20261009000008CreateUserNotifications) Down() error {
	_, err := facades.Orm().Query().Exec("DROP TABLE IF EXISTS notification_email_outbox; DROP TABLE IF EXISTS user_notifications")
	return err
}
