package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"
	"goravel/database/migrations"
)

func Migrations() []schema.Migration {
	return []schema.Migration{
		&migrations.M20260916171145CreateChoferTable{},
		&migrations.M20260916171339CreatePublicadorTable{},
		&migrations.M20260916171747CreateEmpresasTable{},
		&migrations.M20260916171836CreateCargasTable{},
		&migrations.M20260916171933CreateFacturasTable{},
		&migrations.M20260916172015CreateCargasHistorialTable{},
		&migrations.M20260916204352CreateUsersTable{},
		&migrations.M20260917043858CreateDireccionesTable{},
		&migrations.M20261003000001FixDireccionCoordinatePrecision{},
		&migrations.M20261003000002AddDriverProfilesNetworksAndRatings{},
		&migrations.M20261005000001AssociateInvoicesWithRoleProfiles{},
		&migrations.M20261005000002BackfillInvoiceRoleProfiles{},
		&migrations.M20261006000001CreatePendingRegistrationsTable{},
		&migrations.M20261009000001CreateLoadInterests{},
		&migrations.M20261009000002CreateNotificationOutbox{},
		&migrations.M20261009000003AddVerificationResend{},
		&migrations.M20261009000004CreateCompanyMembershipRequests{},
		&migrations.M20261009000005AddIntegrityConstraints{},
		&migrations.M20261009000006AddInvoiceTemplates{},
		&migrations.M20261009000007CreateCompanyChatMessages{},
		&migrations.M20261009000008CreateUserNotifications{},
		&migrations.M20261010000001AddNotificationActor{},
		&migrations.M20261010000002AddEmpresaProfilePhoto{},
	}
}
