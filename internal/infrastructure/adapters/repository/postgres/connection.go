package postgres

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	// RunMigrations gates OpenAndMigrate's automatic migration-on-connect —
	// matching bia-electronic-bills' RUN_MIGRATIONS toggle. Defaults to true:
	// Cifrato runs as a single instance for this technical test, so there's
	// no multi-replica race to worry about.
	RunMigrations bool
}

func ConfigFromEnv() Config {
	return Config{
		Host:          getEnv("DB_HOST", "localhost"),
		Port:          getEnv("DB_PORT", "5432"),
		User:          getEnv("DB_USER", "cifrato"),
		Password:      getEnv("DB_PASSWORD", "cifrato"),
		DBName:        getEnv("DB_NAME", "cifrato"),
		SSLMode:       getEnv("DB_SSLMODE", "disable"),
		RunMigrations: getEnvBool("RUN_MIGRATIONS", true),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=America/Bogota",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func Open(cfg Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("postgres: opening connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres: getting underlying *sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}
