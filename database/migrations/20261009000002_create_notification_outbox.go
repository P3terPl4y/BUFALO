package migrations

import "goravel/app/facades"

type M20261009000002CreateNotificationOutbox struct{}

func (*M20261009000002CreateNotificationOutbox) Signature() string {
	return "20261009000002_create_notification_outbox"
}
func (*M20261009000002CreateNotificationOutbox) Up() error {
	_, err := facades.Orm().Query().Exec(`CREATE TABLE IF NOT EXISTS notification_outbox (
 id bigserial PRIMARY KEY, interest_id bigint NOT NULL UNIQUE,
 payload text NOT NULL, attempts integer NOT NULL DEFAULT 0 CHECK(attempts >= 0),
 available_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP, lease_until timestamptz,
 lease_token varchar(64), sent_at timestamptz, created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
 ); CREATE INDEX IF NOT EXISTS idx_notification_outbox_pending ON notification_outbox(available_at,id) WHERE sent_at IS NULL`)
	return err
}
func (*M20261009000002CreateNotificationOutbox) Down() error {
	_, err := facades.Orm().Query().Exec("DROP TABLE notification_outbox")
	return err
}
