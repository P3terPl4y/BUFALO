package migrations

import "goravel/app/facades"

type M20261010000002AddEmpresaProfilePhoto struct{}

func (*M20261010000002AddEmpresaProfilePhoto) Signature() string {
	return "20261010000002_add_empresa_profile_photo"
}

func (*M20261010000002AddEmpresaProfilePhoto) Up() error {
	_, err := facades.Orm().Query().Exec(`ALTER TABLE empresas ADD COLUMN IF NOT EXISTS profile_photo varchar(255) NULL`)
	return err
}

func (*M20261010000002AddEmpresaProfilePhoto) Down() error {
	_, err := facades.Orm().Query().Exec(`ALTER TABLE empresas DROP COLUMN IF EXISTS profile_photo`)
	return err
}
