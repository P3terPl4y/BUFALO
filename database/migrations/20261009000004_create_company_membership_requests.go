package migrations

import "goravel/app/facades"

type M20261009000004CreateCompanyMembershipRequests struct{}

func (*M20261009000004CreateCompanyMembershipRequests) Signature() string {
	return "20261009000004_create_company_membership_requests"
}
func (*M20261009000004CreateCompanyMembershipRequests) Up() error {
	_, err := facades.Orm().Query().Exec(`CREATE TABLE IF NOT EXISTS company_membership_requests (
 id bigserial PRIMARY KEY,user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,empresa_id bigint NOT NULL REFERENCES empresas(id),
 status varchar(16) NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected')),
 decided_by bigint REFERENCES users(id),decided_at timestamptz,created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 CHECK ((status='pending' AND decided_by IS NULL AND decided_at IS NULL) OR (status<>'pending' AND decided_by IS NOT NULL AND decided_at IS NOT NULL))
 ); CREATE UNIQUE INDEX IF NOT EXISTS idx_membership_pending_user ON company_membership_requests(user_id) WHERE status='pending'; CREATE INDEX IF NOT EXISTS idx_membership_status ON company_membership_requests(status,id)`)
	return err
}
func (*M20261009000004CreateCompanyMembershipRequests) Down() error {
	_, err := facades.Orm().Query().Exec("DROP TABLE company_membership_requests")
	return err
}
