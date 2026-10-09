package migrations

import "goravel/app/facades"

type M20261009000007CreateCompanyChatMessages struct{}

func (*M20261009000007CreateCompanyChatMessages) Signature() string {
	return "20261009000007_create_company_chat_messages"
}

func (*M20261009000007CreateCompanyChatMessages) Up() error {
	_, err := facades.Orm().Query().Exec(`
CREATE TABLE IF NOT EXISTS empresa_chat_mensajes (
 id bigserial PRIMARY KEY,
 empresa_id bigint NOT NULL REFERENCES empresas(id) ON DELETE CASCADE,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 mensaje text NOT NULL CHECK (char_length(btrim(mensaje)) BETWEEN 1 AND 1000),
 moderado_por bigint REFERENCES users(id) ON DELETE SET NULL,
 moderado_en timestamptz,
 created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 CHECK ((moderado_en IS NULL AND moderado_por IS NULL) OR moderado_en IS NOT NULL)
);
CREATE INDEX IF NOT EXISTS idx_empresa_chat_messages_company_id
 ON empresa_chat_mensajes (empresa_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_empresa_chat_messages_user_id
 ON empresa_chat_mensajes (user_id, id DESC);
`)
	return err
}

func (*M20261009000007CreateCompanyChatMessages) Down() error {
	_, err := facades.Orm().Query().Exec("DROP TABLE IF EXISTS empresa_chat_mensajes")
	return err
}
