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

// Migrate applies every pending SQL migration embedded from migrations/,
// including baseline reference data (cities, concepts, UVT, tax rules).
// Reuses gorm's own connection pool instead of opening a second one.
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

// OpenAndMigrate opens the connection and, if cfg.RunMigrations is true,
// applies pending migrations before returning it. Used by `cifrato serve`;
// `cifrato migrate` calls Open + Migrate directly, ignoring the toggle.
func OpenAndMigrate(cfg Config) (*gorm.DB, error) {
	db, err := Open(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.RunMigrations {
		if err := Migrate(db); err != nil {
			return nil, err
		}
	}
	return db, nil
}
