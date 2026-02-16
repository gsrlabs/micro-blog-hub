package db

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	"log"
	"os"
	"testing"

	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.uber.org/zap"

	"github.com/joho/godotenv"

	"github.com/stretchr/testify/assert"
)

// getTestConfig загружает и возвращает конфигурацию для тестирования.
func getTestConfig() *config.Config {
	envPaths := []string{
		"../../../.env",
		"../../.env",
		"../.env",
		".env",
	}

	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			log.Printf("INFO: loaded env from %s", p)
			break
		}
	}

	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		panic("DB_PASSWORD is not set for tests")
	}

	configPaths := []string{
		"../../config/config.yml",
		"../config/config.yml",
		"config/config.yml",
	}

	var cfg *config.Config
	var err error

	for _, p := range configPaths {
		cfg, err = config.Load(p)
		if err == nil {
			log.Printf("INFO: loaded config from %s", p)
			break
		}
	}

	if err != nil {
		panic("failed to load config.yml for tests")
	}

	cfg.Database.Password = dbPass

	// Если в структуре Config есть блок Test, используем его, иначе задаем дефолты
	if cfg.Test.DBHost != "" {
		cfg.Database.Host = cfg.Test.DBHost
	} else {
		cfg.Database.Host = "localhost"
	}

	if cfg.Test.MigrationsPath != "" {
		cfg.Migrations.Path = cfg.Test.MigrationsPath
	} else {
		cfg.Migrations.Path = "../../migrations"
	}

	return cfg
}

// TestDatabaseConnectionAndMigrations проверяет успешное подключение к БД
// и то, что утилита миграций (Goose) инициализировала свою служебную таблицу.
func TestDatabaseConnectionAndMigrations(t *testing.T) {
	cfg := getTestConfig()
	ctx := context.Background()
	logger := zap.NewNop()

	database, err := Connect(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Pool.Close()

	assert.NoError(t, err)
	assert.NotNil(t, database)

	// Проверяем, что служебная таблица goose существует
	var exists bool
	err = database.Pool.QueryRow(
		ctx,
		`SELECT EXISTS (
            SELECT 1 FROM information_schema.tables WHERE table_name = 'goose_db_version'
        )`,
	).Scan(&exists)

	assert.NoError(t, err)
	assert.True(t, exists, "goose_db_version table should exist")
}

// TestUsersTableExists подтверждает, что таблица 'users'
// была корректно создана в БД после запуска миграций.
func TestUsersTableExists(t *testing.T) {
	cfg := getTestConfig()
	ctx := context.Background()
	logger := zap.NewNop()

	database, err := Connect(ctx, cfg, logger)
	assert.NoError(t, err)
	defer database.Pool.Close()

	var exists bool
	err = database.Pool.QueryRow(
		ctx,
		`
        SELECT EXISTS (
            SELECT 1
            FROM information_schema.tables
            WHERE table_name = 'users'
        )
        `,
	).Scan(&exists)

	assert.NoError(t, err)
	assert.True(t, exists, "users table should exist")
}

// TestUsersIndexesExist проверяет наличие критически важных индексов.
// PostgreSQL автоматически создает индексы для PRIMARY KEY и UNIQUE ограничений.
func TestUsersIndexesExist(t *testing.T) {
	cfg := getTestConfig()
	ctx := context.Background()
	logger := zap.NewNop()

	database, err := Connect(ctx, cfg, logger)
	assert.NoError(t, err)
	defer database.Pool.Close()

	// Имена индексов по умолчанию в PostgreSQL для твоей таблицы
	indexes := []string{
		"users_pkey",         // Индекс для id (PRIMARY KEY)
		"users_username_key", // Индекс уникальности для username
		"users_email_key",    // Индекс уникальности для email
	}

	for _, idx := range indexes {
		var exists bool
		err := database.Pool.QueryRow(
			ctx,
			`
            SELECT EXISTS (
                SELECT 1
                FROM pg_indexes
                WHERE indexname = $1
            )
            `,
			idx,
		).Scan(&exists)

		assert.NoError(t, err)
		assert.True(t, exists, "index %s should exist", idx)
	}
}

func TestConnect_NewPoolError(t *testing.T) {
	original := newPoolWithConfig
	defer func() { newPoolWithConfig = original }()

	newPoolWithConfig = func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
		return nil, errors.New("pool error")
	}

	cfg := getTestConfig()
	logger := zap.NewNop()
	db, err := Connect(context.Background(), cfg, logger)

	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestConnect_NoAutoMigrations(t *testing.T) {
	cfg := getTestConfig()
	cfg.Migrations.Auto = false
	logger := zap.NewNop()
	db, err := Connect(context.Background(), cfg, logger)

	assert.NoError(t, err)
	assert.NotNil(t, db)
}

func TestRunMigrations_Success(t *testing.T) {
	cfg := getTestConfig()
	logger := zap.NewNop()

	if _, err := os.Stat(cfg.Migrations.Path); os.IsNotExist(err) {
		t.Skipf("skip migration test, path %s does not exist", cfg.Migrations.Path)
	}

	err := runMigrations(
		fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.Name,
			cfg.Database.SSLMode,
		),
		cfg.Migrations.Path,
		cfg.App.Mode,
		logger)

	assert.NoError(t, err)
}

func TestRunMigrations_OpenError(t *testing.T) {
	originalSqlOpen := sqlOpen
	defer func() { sqlOpen = originalSqlOpen }()
	sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
		return nil, errors.New("open error")
	}

	logger := zap.NewNop()
	err := runMigrations("dsn", "some/path", "debug", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open error")
}

func TestRunMigrations_GooseDialectError(t *testing.T) {
	cfg := getTestConfig()
	logger := zap.NewNop()

	// Если миграций нет, тест пропускаем
	if _, err := os.Stat(cfg.Migrations.Path); os.IsNotExist(err) {
		t.Skipf("skip migration test, path %s does not exist", cfg.Migrations.Path)
	}

	// Для этого теста можно использовать некорректный путь
	// чтобы проверить, что runMigrations вернёт ошибку
	err := runMigrations(
		fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.Name,
			cfg.Database.SSLMode,
		),
		"invalid/path", // deliberately wrong
		cfg.App.Mode,
		logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory does not exist") // или часть текста, которую реально выдаёт goose
}

func TestRunMigrations_UpError(t *testing.T) {
	cfg := getTestConfig()
	logger := zap.NewNop()

	// Используем фиктивный путь к миграциям, чтобы вызвать ошибку goose.Up
	invalidPath := "invalid/migrations/path"

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	err := runMigrations(dsn, invalidPath, cfg.App.Mode, logger)
	assert.Error(t, err)
	// goose.Up возвращает ошибку с текстом про "directory does not exist"
	assert.Contains(t, err.Error(), "directory does not exist")
}
