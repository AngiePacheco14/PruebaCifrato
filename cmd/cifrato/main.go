package main

import (
	"log"
	"os"

	"cifrato/internal/adapters/driven/postgres"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: cifrato <migrate|seed>")
	}

	db, err := postgres.Open(postgres.ConfigFromEnv())
	if err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}

	switch os.Args[1] {
	case "migrate":
		if err := postgres.Migrate(db); err != nil {
			log.Fatalf("migrating: %v", err)
		}
		log.Println("migration completed")
	case "seed":
		if err := postgres.Seed(db); err != nil {
			log.Fatalf("seeding: %v", err)
		}
		if err := postgres.SeedTaxRules(db); err != nil {
			log.Fatalf("seeding tax rules: %v", err)
		}
		log.Println("seed completed")
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}
