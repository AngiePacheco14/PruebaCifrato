package postgres

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies every pending SQL migration embedded from migrations/ —
// schema plus the baseline reference data (cities, concepts, UVT, tax
// rules) needed to run withholding calculations end-to-end. It reuses
// gorm's own connection pool instead of opening a second one.
//
// SQL migrations (not gorm.AutoMigrate) are the source of truth here so the
// schema — and the reference data it ships with — is versioned and
// reviewable like any other change, with a real rollback path (.down.sql),
// not just an idempotent "make it look like the struct" pass.
func Migrate(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("postgres: getting underlying *sql.DB: %w", err)
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	if err != nil {
		return fmt.Errorf("postgres: creating migrate driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("postgres: loading embedded migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("postgres: initializing migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: applying migrations: %w", err)
	}
	return nil
}
