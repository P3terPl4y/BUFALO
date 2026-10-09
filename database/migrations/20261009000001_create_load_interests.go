package migrations

import "goravel/app/facades"

type M20261009000001CreateLoadInterests struct{}

func (*M20261009000001CreateLoadInterests) Signature() string {
	return "20261009000001_create_load_interests"
}
func (*M20261009000001CreateLoadInterests) Up() error {
	_, err := facades.Orm().Query().Exec(`CREATE TABLE IF NOT EXISTS load_interests (
 id bigserial PRIMARY KEY, carga_id bigint NOT NULL, chofer_id bigint NOT NULL,
 created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(carga_id, chofer_id)); CREATE INDEX IF NOT EXISTS idx_load_interests_driver_time ON load_interests(chofer_id,created_at)`)
	return err
}
func (*M20261009000001CreateLoadInterests) Down() error {
	_, err := facades.Orm().Query().Exec("DROP TABLE IF EXISTS load_interests")
	return err
}
