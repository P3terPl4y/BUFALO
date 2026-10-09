package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"goravel/app/facades"
)

type M20261006000001CreatePendingRegistrationsTable struct{}

func (m *M20261006000001CreatePendingRegistrationsTable) Signature() string {
	return "20261006000001_create_pending_registrations_table"
}

func (m *M20261006000001CreatePendingRegistrationsTable) Up() error {
	if facades.Schema().HasTable("pending_registrations") {
		return nil
	}
	return facades.Schema().Create("pending_registrations", func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.String("email", 100)
		table.String("token_hash", 64)
		table.Text("payload")
		table.TimestampTz("expires_at")
		table.TimestampTz("created_at").Nullable()
		table.TimestampTz("updated_at").Nullable()
		table.Unique("email").Name("idx_pending_registrations_email")
		table.Unique("token_hash").Name("idx_pending_registrations_token_hash")
		table.Index("expires_at").Name("idx_pending_registrations_expires_at")
	})
}

func (m *M20261006000001CreatePendingRegistrationsTable) Down() error {
	return facades.Schema().DropIfExists("pending_registrations")
}
