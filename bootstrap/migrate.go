package bootstrap

import (
	"context"
	"github.com/goravel/framework/database/migration"
	"goravel/app/dbresilience"
	"goravel/app/facades"
	"time"
)

func RunMigrations(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	pool, err := facades.Orm().DB()
	if err != nil {
		return err
	}
	return dbresilience.WithMigrationLock(ctx, pool, func() error {
		return migration.NewMigrator(facades.Artisan(), facades.Schema(), facades.Config().GetString("database.migrations.table", "migrations")).Run()
	})
}
