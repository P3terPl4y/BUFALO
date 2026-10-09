package main

import (
	"context"
	"goravel/bootstrap"
	"log"
)

func main() {
	bootstrap.Boot()
	if err := bootstrap.RunMigrations(context.Background()); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}
}
