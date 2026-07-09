.PHONY: help db-up db-down db-reset db-logs db-shell

help:
	@echo db-up     - levanta Postgres en localhost:5432
	@echo db-down   - apaga Postgres, conserva los datos
	@echo db-reset  - apaga Y BORRA los datos, vuelve a levantar desde cero
	@echo db-logs   - logs de Postgres en vivo
	@echo db-shell  - abre psql dentro del contenedor

db-up:
	docker compose up -d

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d

db-logs:
	docker compose logs -f postgres

db-shell:
	docker compose exec postgres psql -U cifrato -d cifrato
