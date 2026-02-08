BINARY_NAME=auth-service
# Путь к точке входа
MAIN_PATH=auth-service/cmd/app/main.go

# .PHONY указывает, что это не файлы, а команды
.PHONY: all test clean up down logs lint db-shell info

# По умолчанию (если просто написать 'make') выполнится info
all: info

test:
	@echo "Running tests..."
	go test -v -p 1 ./...

# 🐳 Docker: Поднять контейнеры
up:
	@echo "Starting Docker containers..."
	docker compose up -d

# 🛑 Docker: Остановить контейнеры
down:
	@echo "Stopping Docker containers..."
	docker compose down

# 🐳 Docker: Поднять контейнеры (с пересборкой)
rebuild:
	@echo "Build and starting Docker containers..."
	docker compose up --build -d

# 📜 Docker: Посмотреть логи
logs:
	docker compose logs -f

# 🔍 Линтер (проверка кода, если установлен golangci-lint)
lint:
	golangci-lint run

# 🔌 Подключиться к БД (psql) внутри контейнера
db-shell:
	docker compose exec postgres psql -U postgres -d auth_db

info:
	@echo "Введите следующие команды:"
	@echo "make up - Поднять контейнеры"
	@echo "make down - Остановить контейнеры"
	@echo "make rebuild - Поднять контейнеры (с пересборкой)"
	@echo "make logs - Посмотреть логи"
	@echo "make lint - Запустить линтер"
	@echo "make db-shell - Подключиться к БД"
	@echo "make test - Запустить тесты"