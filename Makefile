.PHONY: dev dev-backend dev-frontend up down migrate migrate-down sqlc test build fmt lint tidy

# Spin up Postgres + Keycloak.
up:
	docker compose up -d

down:
	docker compose down

# Run backend and frontend dev servers (in separate shells).
dev-backend:
	cd backend && go run ./cmd/levitate

dev-frontend:
	cd frontend && npm run dev

# Convenience: start infra, then print next steps.
dev: up
	@echo ""
	@echo "Infra is up. In two separate shells, run:"
	@echo "  make dev-backend"
	@echo "  make dev-frontend"

# Database migrations (golang-migrate). Requires `migrate` CLI on PATH.
DB_URL ?= postgres://levitate:levitate@localhost:5432/levitate?sslmode=disable
MIGRATIONS := backend/migrations

migrate:
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" down 1

# sqlc code generation.
sqlc:
	cd backend && sqlc generate

test:
	cd backend && go test ./...

build:
	cd backend && go build -o ./dist/levitate ./cmd/levitate

fmt:
	cd backend && go fmt ./...

tidy:
	cd backend && go mod tidy
