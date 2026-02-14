# Переменные
BINARY_NAME=auth-service
MAIN_PATH=auth-service/cmd/app/main.go
MIGRATIONS_DIR=auth-service/migrations
DB_DSN="postgres://postgres:password123@localhost:5432/auth_db?sslmode=disable"

.PHONY: all test clean up down rebuild logs lint db-shell migrate-status migrate-new info

all: info

# --- Разработка ---
test:
	@echo "Running tests..."
	go test -v -p 1 ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

clean:
	@echo "Cleaning binaries..."
	rm -f $(BINARY_NAME)
	go clean

# --- Docker ---
up:
	@echo "Starting Docker containers..."
	docker compose up -d

down:
	@echo "Stopping Docker containers..."
	docker compose down

rebuild:
	@echo "Rebuilding and starting..."
	docker compose up --build -d

logs:
	@echo "Showing logs (press Ctrl+C to stop)..."
	docker compose logs -f

# --- База данных ---
db-shell:
	@echo "Connecting to database..."
	docker compose exec postgres psql -U postgres -d auth_db

# Показать статус миграций (нужен установленный goose локально)
migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres $(DB_DSN) status

# Создать новую миграцию: make migrate-new name=add_users_table
migrate-new:
	@if [ -z "$(name)" ]; then echo "Error: 'name' is required. Example: make migrate-new name=init"; exit 1; fi
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

# --- Помощь ---
info:
	@echo "Доступные команды:"
	@echo "  make up           - Поднять проект в Docker"
	@echo "  make down         - Остановить проект"
	@echo "  make rebuild      - Пересобрать и запустить"
	@echo "  make logs         - Логи контейнеров"
	@echo "  make test         - Запустить тесты"
	@echo "  make db-shell     - Зайти в консоль PSQL"
	@echo "  make migrate-new  - Создать миграцию (нужно name=имя)"
	@echo "  make clean        - Удалить временные файлы"