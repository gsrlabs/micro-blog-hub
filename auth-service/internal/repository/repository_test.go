package repository

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/db"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// getTestConfig загружает конфигурацию для тестов.
func getTestConfig() *config.Config {
	// Ищем .env файл, поднимаясь по дереву каталогов вверх
	envPaths := []string{
		"../../../.env", // Корень micro-blog-hub
		"../../.env",    // Корень auth-service
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

	// Настройка для тестов
	if cfg.Test.DBHost != "" {
		cfg.Database.Host = cfg.Test.DBHost
	} else {
		cfg.Database.Host = "localhost" // Подключаемся к проброшенному порту Docker
	}

	// Отключаем автоматические миграции при каждом коннекте, 
	// так как они уже прогнаны в db_test.go
	cfg.Migrations.Auto = false

	return cfg
}

// setupTestDB инициализирует тестовое окружение, подключается к БД,
// создает репозиторий и возвращает функцию для очистки таблицы (TRUNCATE).
func setupTestDB(t *testing.T) (AuthRepository, func()) {
	cfg := getTestConfig()
	ctx := context.Background()
	logger := zap.NewNop()

	database, err := db.Connect(ctx, cfg, logger)
	require.NoError(t, err, "failed to connect to db")


	repo := NewAuthRepository(database.Pool, logger)

	// Функция очистки (вызывается через defer в самом тесте)
	cleanup := func() {
		// Очищаем таблицу users. CASCADE нужен, если появятся связанные таблицы.
		_, err := database.Pool.Exec(ctx, "TRUNCATE users RESTART IDENTITY CASCADE")
		if err != nil {
			log.Printf("failed to truncate table users: %v", err)
		}
		database.Pool.Close()
	}

	return repo, cleanup
}

// TestAuthRepo_Lifecycle проверяет базовый флоу: создание, получение и удаление пользователя.
func TestAuthRepo_Lifecycle(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Подготавливаем тестовые данные
	newUser := &model.User{
		Username: "john_doe",
		Email:    "john@example.com",
		Password: "hashed_password_123", // В БД поле называется password_hash, но в структуре у тебя Password
	}


	var savedID uuid.UUID

	// 1. CREATE
	t.Run("Create", func(t *testing.T) {
		id, err := repo.Create(ctx, newUser)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, id, "ID должен быть сгенерирован")
		savedID = id
	})

	// 2. GET BY ID
	t.Run("GetByID", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, savedID)
		require.NoError(t, err)
		assert.Equal(t, savedID, fetched.ID)
		assert.Equal(t, "john_doe", fetched.Username)
		assert.Equal(t, "john@example.com", fetched.Email)
		assert.Equal(t, "hashed_password_123", fetched.Password)
		assert.False(t, fetched.CreatedAt.IsZero(), "CreatedAt должен быть заполнен базой")
	})

	// 3. GET BY EMAIL
	t.Run("GetByEmail", func(t *testing.T) {
		fetched, err := repo.GetByEmail(ctx, "john@example.com")
		require.NoError(t, err)
		assert.Equal(t, savedID, fetched.ID)
		assert.Equal(t, "john_doe", fetched.Username)
	})

	// 4. DELETE
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, savedID)
		assert.NoError(t, err)

		// Проверяем, что удаление вернуло ошибку NotFound для несуществующего ID
		err = repo.Delete(ctx, savedID)
		assert.ErrorIs(t, err, ErrNotFound)

		// Проверяем, что получить пользователя больше нельзя
		_, err = repo.GetByID(ctx, savedID)
		assert.ErrorIs(t, err, pgx.ErrNoRows) // Твой код возвращает оригинальную ошибку pgx, если не нашел строку
	})
}

// TestAuthRepo_Updates проверяет обновление профиля, email, пароля и обработку дубликатов.
func TestAuthRepo_Updates(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаем двух пользователей для проверки конфликтов уникальности
	user1 := &model.User{Username: "user1", Email: "user1@example.com", Password: "pwd"}
	user2 := &model.User{Username: "user2", Email: "user2@example.com", Password: "pwd"}

	id1, err := repo.Create(ctx, user1)
	require.NoError(t, err)

	_, err = repo.Create(ctx, user2)
	require.NoError(t, err)

	t.Run("UpdateProfile_Success", func(t *testing.T) {
		err := repo.UpdateProfile(ctx, id1, "new_user1_name")
		assert.NoError(t, err)

		fetched, _ := repo.GetByID(ctx, id1)
		assert.Equal(t, "new_user1_name", fetched.Username)
	})

	t.Run("UpdateProfile_Duplicate", func(t *testing.T) {
		// Пытаемся занять имя второго пользователя
		err := repo.UpdateProfile(ctx, id1, "user2")
		assert.ErrorIs(t, err, ErrDuplicateUsername)
	})

	t.Run("UpdateEmail_Success", func(t *testing.T) {
		err := repo.UpdateEmail(ctx, id1, "new@example.com")
		assert.NoError(t, err)

		fetched, _ := repo.GetByID(ctx, id1)
		assert.Equal(t, "new@example.com", fetched.Email)
	})

	t.Run("UpdateEmail_Duplicate", func(t *testing.T) {
		// Пытаемся занять email второго пользователя
		err := repo.UpdateEmail(ctx, id1, "user2@example.com")
		assert.ErrorIs(t, err, ErrDuplicateEmail)
	})

	t.Run("UpdatePassword", func(t *testing.T) {
		err := repo.UpdatePassword(ctx, id1, "new_hashed_pwd")
		assert.NoError(t, err)

		fetched, _ := repo.GetByID(ctx, id1)
		assert.Equal(t, "new_hashed_pwd", fetched.Password)
	})

	t.Run("Updates_NotFound", func(t *testing.T) {
		fakeID := uuid.New()
		errProfile := repo.UpdateProfile(ctx, fakeID, "ghost")
		assert.ErrorIs(t, errProfile, ErrNotFound)

		errEmail := repo.UpdateEmail(ctx, fakeID, "ghost@ghost.com")
		assert.ErrorIs(t, errEmail, ErrNotFound)

		errPwd := repo.UpdatePassword(ctx, fakeID, "ghost_pwd")
		assert.ErrorIs(t, errPwd, ErrNotFound)
	})
}

// TestAuthRepo_GetUsers проверяет выборку списка пользователей с учетом LIMIT, OFFSET и сортировки.
func TestAuthRepo_GetUsers(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаем 3 пользователей. Из-за ORDER BY created_at DESC 
	// последним вставленный будет первым в результате.
	users := []*model.User{
		{Username: "u1", Email: "u1@example.com", Password: "p"},
		{Username: "u2", Email: "u2@example.com", Password: "p"},
		{Username: "u3", Email: "u3@example.com", Password: "p"},
	}

	for _, u := range users {
		_, err := repo.Create(ctx, u)
		require.NoError(t, err)
	}

	t.Run("Limit and Offset", func(t *testing.T) {
		// Берем 2 пользователей, пропуская 0 (должны получить u3 и u2)
		list, err := repo.GetUsers(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, list, 2)
		assert.Equal(t, "u3", list[0].Username) // Проверка сортировки DESC
		assert.Equal(t, "u2", list[1].Username)

		// Берем оставшихся, пропуская первых 2 (должны получить только u1)
		list2, err := repo.GetUsers(ctx, 2, 2)
		require.NoError(t, err)
		assert.Len(t, list2, 1)
		assert.Equal(t, "u1", list2[0].Username)
	})

	t.Run("Empty Result", func(t *testing.T) {
		// Берем с большим отступом, где пользователей уже нет
		list, err := repo.GetUsers(ctx, 10, 100)
		require.NoError(t, err)
		assert.NotNil(t, list, "Слайс должен быть инициализирован, а не nil")
		assert.Len(t, list, 0)
	})
}

