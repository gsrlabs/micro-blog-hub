BINARY_NAME=auth-service
# Путь к точке входа
MAIN_PATH=auth-service/cmd/app/main.go

# .PHONY указывает, что это не файлы, а команды
.PHONY: all build run test clean swag docker-up docker-down docker-logs lint

# По умолчанию (если просто написать 'make') выполнится build
all: build

# 🏗 Сборка приложения
build:
	@echo "Building application..."
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

# 🚀 Запуск локально (без Докера)
run:
	@echo "Running application..."
	go run $(MAIN_PATH)

# 🧪 Запуск всех тестов (Unit + Integration)
test:
	@echo "Running tests..."
	go test -v -p 1 ./...


# 🧹 Очистка (удаление бинарников и временных файлов)
clean:
	@echo "Cleaning up..."
	go clean
	rm -rf bin/

# 🐳 Docker: Поднять контейнеры
docker-up:
	@echo "Starting Docker containers..."
	docker compose up -d

# 🐳 Docker: Поднять контейнеры (с пересборкой)
docker-rebuild:
	@echo "Build and starting Docker containers..."
	docker compose up --build -d

# 🛑 Docker: Остановить контейнеры
docker-down:
	@echo "Stopping Docker containers..."
	docker compose down

# 📜 Docker: Посмотреть логи
docker-logs:
	docker compose logs -f

# 🔍 Линтер (проверка кода, если установлен golangci-lint)
lint:
	golangci-lint run

# 🔌 Подключиться к БД (psql) внутри контейнера
db-shell:
	docker compose exec postgres psql -U postgres -d subscriptions