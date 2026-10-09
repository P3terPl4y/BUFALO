package migrations

import "goravel/app/facades"

type M20261009000006AddInvoiceTemplates struct{}

func (*M20261009000006AddInvoiceTemplates) Signature() string {
	return "20261009000006_add_invoice_templates"
}

func (*M20261009000006AddInvoiceTemplates) Up() error {
	_, err := facades.Orm().Query().Exec(`
CREATE TABLE IF NOT EXISTS factura_plantillas (
	id bigserial PRIMARY KEY,
	publicador_id bigint NOT NULL REFERENCES publicadors(id) ON DELETE CASCADE,
	nombre varchar(48) NOT NULL,
	preset varchar(20) NOT NULL DEFAULT '',
	formato varchar(12) NOT NULL DEFAULT 'a4' CHECK (formato IN ('a4','letter','receipt')),
	color varchar(7) NOT NULL DEFAULT '#253746' CHECK (color ~ '^#[0-9a-fA-F]{6}$'),
	bloques_json jsonb NOT NULL CHECK (jsonb_typeof(bloques_json) = 'array' AND jsonb_array_length(bloques_json) <= 16),
	predeterminada boolean NOT NULL DEFAULT false,
	created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT uq_factura_plantilla_nombre UNIQUE (publicador_id, nombre)
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_factura_plantilla_predeterminada ON factura_plantillas(publicador_id) WHERE predeterminada;
ALTER TABLE facturas ADD COLUMN IF NOT EXISTS plantilla_id bigint REFERENCES factura_plantillas(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_facturas_plantilla_id ON facturas(plantilla_id);
`)
	return err
}

func (*M20261009000006AddInvoiceTemplates) Down() error {
	_, err := facades.Orm().Query().Exec(`
DROP INDEX IF EXISTS idx_facturas_plantilla_id;
ALTER TABLE facturas DROP COLUMN IF EXISTS plantilla_id;
DROP TABLE IF EXISTS factura_plantillas;
`)
	return err
}
