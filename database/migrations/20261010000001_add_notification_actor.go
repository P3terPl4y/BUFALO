package migrations

import "goravel/app/facades"

type M20261010000001AddNotificationActor struct{}

func (*M20261010000001AddNotificationActor) Signature() string {
	return "20261010000001_add_notification_actor"
}

func (*M20261010000001AddNotificationActor) Up() error {
	_, err := facades.Orm().Query().Exec(`
ALTER TABLE user_notifications
ADD COLUMN IF NOT EXISTS actor_user_id bigint NULL REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_user_notifications_actor ON user_notifications(actor_user_id) WHERE actor_user_id IS NOT NULL;
`)
	return err
}

func (*M20261010000001AddNotificationActor) Down() error {
	_, err := facades.Orm().Query().Exec(`
DROP INDEX IF EXISTS idx_user_notifications_actor;
ALTER TABLE user_notifications DROP COLUMN IF EXISTS actor_user_id;
`)
	return err
}
