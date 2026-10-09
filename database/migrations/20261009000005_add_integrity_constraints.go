package migrations

import "goravel/app/facades"

type M20261009000005AddIntegrityConstraints struct{}

func (*M20261009000005AddIntegrityConstraints) Signature() string {
	return "20261009000005_add_integrity_constraints"
}

// NOT VALID avoids scanning under the initial ALTER lock. VALIDATE then rejects
// inconsistent legacy rows without rewriting or deleting commercial data.
func (*M20261009000005AddIntegrityConstraints) Up() error {
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL lock_timeout = '3s'"); err != nil {
		return err
	}
	constraints := []struct{ table, name, definition string }{
		{"chofer_calificaciones", "ck_rating_score", "CHECK(puntaje BETWEEN 1 AND 5)"},
		{"chofers", "ck_driver_rating", "CHECK(rating_count >= 0 AND rating_average BETWEEN 0 AND 5)"},
		{"red_choferes", "fk_network_company", "FOREIGN KEY(empresa_id) REFERENCES empresas(id) ON DELETE CASCADE"},
		{"red_choferes", "fk_network_driver", "FOREIGN KEY(chofer_id) REFERENCES chofers(id) ON DELETE CASCADE"},
		{"load_interests", "fk_interest_load", "FOREIGN KEY(carga_id) REFERENCES cargas(id) ON DELETE CASCADE"},
		{"load_interests", "fk_interest_driver", "FOREIGN KEY(chofer_id) REFERENCES chofers(id) ON DELETE CASCADE"},
		{"chofer_calificaciones", "fk_rating_load", "FOREIGN KEY(carga_id) REFERENCES cargas(id)"},
		{"chofer_calificaciones", "fk_rating_driver", "FOREIGN KEY(chofer_id) REFERENCES chofers(id)"},
		{"chofer_calificaciones", "fk_rating_publisher", "FOREIGN KEY(publicador_id) REFERENCES publicadors(id)"},
	}
	for _, c := range constraints {
		if _, err := tx.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = '" + c.table + "'::regclass AND conname = '" + c.name + "') THEN ALTER TABLE " + c.table + " ADD CONSTRAINT " + c.name + " " + c.definition + " NOT VALID; END IF; END $$"); err != nil {
			return err
		}
		if _, err := tx.Exec("ALTER TABLE " + c.table + " VALIDATE CONSTRAINT " + c.name); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (*M20261009000005AddIntegrityConstraints) Down() error {
	_, err := facades.Orm().Query().Exec(`ALTER TABLE chofer_calificaciones DROP CONSTRAINT IF EXISTS ck_rating_score,DROP CONSTRAINT IF EXISTS fk_rating_load,DROP CONSTRAINT IF EXISTS fk_rating_driver,DROP CONSTRAINT IF EXISTS fk_rating_publisher; ALTER TABLE chofers DROP CONSTRAINT IF EXISTS ck_driver_rating; ALTER TABLE red_choferes DROP CONSTRAINT IF EXISTS fk_network_company,DROP CONSTRAINT IF EXISTS fk_network_driver; ALTER TABLE load_interests DROP CONSTRAINT IF EXISTS fk_interest_load,DROP CONSTRAINT IF EXISTS fk_interest_driver`)
	return err
}
