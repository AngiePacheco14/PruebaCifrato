package main

import (
	"log"
	"net/http"
	"os"

	"cifrato/internal/infrastructure/adapters/repository/postgres"
	"cifrato/internal/infrastructure/dependence"
	"cifrato/internal/infrastructure/rest/router"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: cifrato <migrate|serve>")
	}

	switch os.Args[1] {
	case "migrate":
		db, err := postgres.Open(postgres.ConfigFromEnv())
		if err != nil {
			log.Fatalf("connecting to postgres: %v", err)
		}
		if err := postgres.Migrate(db); err != nil {
			log.Fatalf("migrating: %v", err)
		}
		log.Println("migration completed")
	case "serve":
		serve()
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func serve() {
	container := dependence.NewWire()

	// PORT is what Render (and most PaaS) inject automatically; HTTP_ADDR is
	// this app's own convention and wins if explicitly set, so local dev
	// (.env with HTTP_ADDR=:8080) keeps working unchanged in either context.
	addr := ":8080"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		addr = v
	}
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, router.NewRouter(container)); err != nil {
		log.Fatalf("serving: %v", err)
	}
}
