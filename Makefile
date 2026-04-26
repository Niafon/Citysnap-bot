.PHONY: run build test lint migrate-up migrate-down migrate-create docker-up docker-down

# ── Dev ──────────────────────────────────────────
run:
	go run ./cmd/bot/

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/citysnap-bot ./cmd/bot/

test:
	go test ./... -v -cover -race

lint:
	golangci-lint run ./...

# ── Docker ───────────────────────────────────────
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# ── Migrations ───────────────────────────────────
# Требует: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
MIGRATE=migrate -path migrations -database "$(DATABASE_URL)"

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

migrate-force:
	$(MIGRATE) force $(V)

migrate-create:
	migrate create -ext sql -dir migrations -seq $(NAME)

# ── Quick start ──────────────────────────────────
setup: docker-up
	@echo "Waiting for Postgres..."
	@sleep 3
	DATABASE_URL="postgres://app:secret@localhost:5432/citysnap?sslmode=disable" $(MAKE) migrate-up
	@echo "Ready! Run: make run"
