package migrations

import (
	"errors"

	"github.com/goravel/framework/contracts/database/schema"
	"goravel/app/facades"
)

type M20261005000001AssociateInvoicesWithRoleProfiles struct{}

func (m *M20261005000001AssociateInvoicesWithRoleProfiles) Signature() string {
	return "20261005000001_associate_invoices_with_role_profiles"
}

func (m *M20261005000001AssociateInvoicesWithRoleProfiles) Up() error {
	if !facades.Schema().HasTable("facturas") {
		return nil
	}
	if err := facades.Schema().Table("facturas", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("facturas", "emisor_id") {
			table.UnsignedBigInteger("emisor_id").Nullable().Change()
		}
		if facades.Schema().HasColumn("facturas", "receptor_id") {
			table.UnsignedBigInteger("receptor_id").Nullable().Change()
		}
		if !facades.Schema().HasColumn("facturas", "publicador_id") {
			table.UnsignedBigInteger("publicador_id").Nullable()
			table.Index("publicador_id").Name("idx_facturas_publicador_id")
		}
		if !facades.Schema().HasColumn("facturas", "emisor_tipo") {
			table.String("emisor_tipo", 20).Default("publicador")
			table.Index("emisor_tipo").Name("idx_facturas_emisor_tipo")
		}
	}); err != nil {
		return err
	}

	return nil
}

func (m *M20261005000001AssociateInvoicesWithRoleProfiles) Down() error {
	if !facades.Schema().HasTable("facturas") {
		return nil
	}
	var nullInvoices int64
	if err := facades.DB().Select(&nullInvoices, "SELECT COUNT(*) FROM facturas WHERE emisor_id IS NULL OR receptor_id IS NULL"); err != nil {
		return err
	}
	if nullInvoices > 0 {
		return errors.New("cannot roll back invoice profile associations while company IDs are null")
	}
	return facades.Schema().Table("facturas", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("facturas", "emisor_tipo") {
			table.DropColumn("emisor_tipo")
		}
		if facades.Schema().HasColumn("facturas", "publicador_id") {
			table.DropColumn("publicador_id")
		}
	})
}
