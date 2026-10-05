package migrations

import "goravel/app/facades"

// M20261005000002BackfillInvoiceRoleProfiles runs after the schema transaction
// has committed, so updating invoices does not deadlock against its own DDL.
type M20261005000002BackfillInvoiceRoleProfiles struct{}

func (m *M20261005000002BackfillInvoiceRoleProfiles) Signature() string {
	return "20261005000002_backfill_invoice_role_profiles"
}

func (m *M20261005000002BackfillInvoiceRoleProfiles) Up() error {
	if !facades.Schema().HasTable("facturas") || !facades.Schema().HasTable("cargas") {
		return nil
	}
	if !facades.Schema().HasColumn("facturas", "publicador_id") ||
		!facades.Schema().HasColumn("cargas", "publicador_id") {
		return nil
	}
	return facades.DB().Statement(`
		UPDATE facturas AS f
		SET publicador_id = c.publicador_id,
		    chofer_id = COALESCE(f.chofer_id, c.chofer_id)
		FROM cargas AS c
		WHERE f.carga_id = c.id
		  AND (f.publicador_id IS NULL OR f.chofer_id IS NULL)`)
}

func (m *M20261005000002BackfillInvoiceRoleProfiles) Down() error {
	// Ownership data is intentionally retained on rollback. Deleting these
	// associations would make existing invoices inaccessible to their owners.
	return nil
}
