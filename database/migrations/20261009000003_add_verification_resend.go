package migrations

import "goravel/app/facades"

type M20261009000003AddVerificationResend struct{}

func (*M20261009000003AddVerificationResend) Signature() string {
	return "20261009000003_add_verification_resend"
}
func (*M20261009000003AddVerificationResend) Up() error {
	_, err := facades.Orm().Query().Exec(`ALTER TABLE pending_registrations ADD COLUMN IF NOT EXISTS resend_hash varchar(64), ADD COLUMN IF NOT EXISTS token_ciphertext text, ADD COLUMN IF NOT EXISTS last_sent_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP, ADD COLUMN IF NOT EXISTS resend_count integer NOT NULL DEFAULT 0 CHECK(resend_count BETWEEN 0 AND 5); CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_resend_hash ON pending_registrations(resend_hash) WHERE resend_hash IS NOT NULL`)
	return err
}
func (*M20261009000003AddVerificationResend) Down() error {
	_, err := facades.Orm().Query().Exec(`ALTER TABLE pending_registrations DROP COLUMN resend_hash,DROP COLUMN token_ciphertext,DROP COLUMN last_sent_at,DROP COLUMN resend_count`)
	return err
}
