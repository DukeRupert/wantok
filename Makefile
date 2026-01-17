.PHONY: run build create-admin sqlc migrate dev docker-build docker-up docker-down docker-create-admin docker-logs

# Run the server (development)
run:
	go run ./cmd/server

# Run with development settings (insecure cookies, bind all interfaces)
dev:
	SECURE_COOKIES=false HOST=0.0.0.0 go run ./cmd/server

# Build binary (pure Go, no CGO)
build:
	CGO_ENABLED=0 go build -o wantok ./cmd/server

# Create an admin user (local)
create-admin:
	go run ./cmd/server --create-admin

# Generate SQLC code after modifying queries
sqlc:
	sqlc generate

# Create a new migration (usage: make migrate name=add_feature)
migrate:
	goose -dir internal/database/migrations create $(name) sql

# --- Docker commands ---

# Build Docker image
docker-build:
	docker build -t wantok:latest .

# Start containers
docker-up:
	docker compose up -d

# Stop containers
docker-down:
	docker compose down

# Create admin user in Docker (container must be running)
docker-create-admin:
	docker compose exec -it wantok /app/wantok --create-admin

# View logs
docker-logs:
	docker compose logs -f wantok
