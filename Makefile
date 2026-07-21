DATABASE_URL ?= postgres://focus:focus@localhost:5432/focus?sslmode=disable
MIGRATIONS := backend/migrations

.PHONY: test migrate migrate-down

test:
	cd backend && go test ./...

# Requires golang-migrate CLI (https://github.com/golang-migrate/migrate). Migrations land in T-02.
migrate:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1
