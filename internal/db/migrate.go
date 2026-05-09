package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationSQL embed.FS

// RunMigrations applies embedded SQL migrations in order using golang-migrate (Flyway-style revision tracking).
//
// Typical workflow when changing the schema:
//   - Add siblings like 000002_your_change.up.sql and 000002_your_change.down.sql under migrations/
//   - Update ORM structs in models.go to match—migrations remain the source of truth for DDL
//
// You can also use the migrate CLI against the same folder and your Postgres URL; see https://github.com/golang-migrate/migrate
func RunMigrations(database *gorm.DB) error {
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}

	src, err := iofs.New(migrationSQL, "migrations")
	if err != nil {
		return fmt.Errorf("migration embed source: %w", err)
	}

	driver, err := pgxmigrate.WithInstance(sqlDB, &pgxmigrate.Config{
		MigrationsTable: "whatsapp_table_migrations",
	})
	if err != nil {
		return fmt.Errorf("postgres migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
