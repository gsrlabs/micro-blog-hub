package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"io"

	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/db"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/handler"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/repository"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/service"
)

func getTestConfig() *config.Config {
    // 1. Пытаемся найти .env файл, поднимаясь выше по директориям
    envPaths := []string{
        ".env",           // Если запустили из корня auth-service
        "../.env",        // Если запустили из tests/
        "../../.env",     // На всякий случай
    }

    envLoaded := false
    for _, p := range envPaths {
        if err := godotenv.Load(p); err == nil {
            log.Printf("INFO: loaded env from %s", p)
            envLoaded = true
            break
        }
    }
    
    if !envLoaded {
        log.Println("WARN: .env file not found, relying on system environment variables")
    }

    // 2. Достаем пароль БД (критично для тестов)
    dbPass := os.Getenv("DB_PASSWORD")
    if dbPass == "" {
        // Если пароля нет, тесты упадут на подключении, поэтому паникуем сразу
        panic("DB_PASSWORD is not set. Check your .env file or environment variables")
    }

    // 3. Пытаемся найти config.yml
    configPaths := []string{
        "config/config.yml",
        "../config/config.yml",
        "../../config/config.yml",
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
        panic(fmt.Sprintf("failed to load config.yml: %v", err))
    }

    // 4. Переопределяем параметры для тестового окружения
    cfg.Database.Password = dbPass

    // Если в конфиге прописан хост для тестов (например, "db" для Docker), используем его
    if cfg.Test.DBHost != "" {
        cfg.Database.Host = cfg.Test.DBHost
    } else {
        cfg.Database.Host = "localhost"
    }

    // Путь к миграциям тоже должен быть относительным места запуска
    if cfg.Test.HandlerMigrationsPath != "" {
        cfg.Migrations.Path = cfg.Test.HandlerMigrationsPath
    } else {
        cfg.Migrations.Path = "../migrations" // Из папки tests/ идем на уровень выше
    }

    return cfg
}

func setupTestServer(t *testing.T) (*httptest.Server, func()) {
    cfg := getTestConfig()
    gin.SetMode(gin.TestMode)

    ctx := context.Background()
    logger := zap.NewNop()
    database, err := db.Connect(ctx, cfg, logger)
    require.NoError(t, err)

    // Очистка перед тестом
    _, err = database.Pool.Exec(ctx, "TRUNCATE users RESTART IDENTITY CASCADE")
    require.NoError(t, err)

    
    repo := repository.NewAuthRepository(database.Pool, logger)
    svc := service.NewAuthService(repo, logger, cfg)
    h := handler.NewAuthHandler(svc, logger, cfg)

    r := gin.Default()

    // 1. Публичные маршруты авторизации
    auth := r.Group("/auth")
    {
        auth.POST("/signup", h.SignUp)
        auth.POST("/signin", h.SignIn)
        auth.POST("/logout", h.Logout)
    }

    // 2. Маршруты пользователей (пагинация и поиск)
    // ВАЖНО: Проверь, чтобы эти пути совпадали с теми, что ты вызываешь в тестах!
    r.GET("/users", h.GetUsers)
    r.GET("/users/:id", h.GetByID) 
    r.GET("/users/search", h.GetByEmail)

    // 3. Защищенные маршруты (профиль)
    // Здесь нужен твой middleware. Если его нет, закомментируй .Use()
    protected := r.Group("/user")
    protected.Use(h.AuthMiddleware) 
    {
        protected.GET("/profile", h.GetProfile)
        protected.PUT("/profile", h.ChangeProfile)
        protected.PUT("/email", h.ChangeEmail)
        protected.PUT("/password", h.ChangePassword)
        protected.DELETE("", h.Delete)
    }

    ts := httptest.NewServer(r)

    cleanup := func() {
        ts.Close()
        database.Pool.Close()
    }

    return ts, cleanup
}

func request(t *testing.T, url, method string, payload any, cookies []*http.Cookie) ([]byte, int, []*http.Cookie) {
	var body io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = bytes.NewBuffer(data)
	}

	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, resp.Cookies()
}

func TestAuth_FullLifecycle(t *testing.T) {
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	var authCookies []*http.Cookie

	t.Run("Step 1: SignUp Success", func(t *testing.T) {
		payload := map[string]string{
			"username": "tester",
			"email":    "test@test.com",
			"password": "password123",
		}
		body, status, _ := request(t, ts.URL+"/auth/signup", "POST", payload, nil)
		
		assert.Equal(t, http.StatusCreated, status)
		assert.Contains(t, string(body), "user registered")
	})

	t.Run("Step 2: SignIn Success & Get Cookie", func(t *testing.T) {
		payload := map[string]string{
			"email":    "test@test.com",
			"password": "password123",
		}
		_, status, cookies := request(t, ts.URL+"/auth/signin", "POST", payload, nil)
		
		assert.Equal(t, http.StatusOK, status)
		assert.NotEmpty(t, cookies)
		authCookies = cookies // Сохраняем куку для следующих тестов
	})

	t.Run("Step 3: Get Profile (Authorized)", func(t *testing.T) {
		body, status, _ := request(t, ts.URL+"/user/profile", "GET", nil, authCookies)
		
		assert.Equal(t, http.StatusOK, status)
		var user map[string]any
		json.Unmarshal(body, &user)
		assert.Equal(t, "tester", user["username"])
	})

	t.Run("Step 4: Logout", func(t *testing.T) {
		_, status, cookies := request(t, ts.URL+"/auth/logout", "POST", nil, authCookies)
		assert.Equal(t, http.StatusOK, status)
		// Проверяем, что кука протухла (MaxAge < 0)
		assert.True(t, cookies[0].MaxAge < 0)
	})
}

func TestAuth_GetUsers_Pagination(t *testing.T) {
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	// Создадим пару юзеров напрямую через API
	for i := 1; i <= 3; i++ {
		payload := map[string]string{
			"username": "user" + strconv.Itoa(i),
			"email":    "user" + strconv.Itoa(i) + "@test.com",
			"password": "password123",
		}
		request(t, ts.URL+"/auth/signup", "POST", payload, nil)
	}

	t.Run("List with Limit", func(t *testing.T) {
		body, status, _ := request(t, ts.URL+"/users?limit=2&offset=0", "GET", nil, nil)
		assert.Equal(t, http.StatusOK, status)

		var users []map[string]any
		json.Unmarshal(body, &users)
		assert.Len(t, users, 2)
	})
    
    t.Run("Get By ID Invalid Format", func(t *testing.T) {
		_, status, _ := request(t, ts.URL+"/users/not-a-uuid", "GET", nil, nil)
		assert.Equal(t, http.StatusBadRequest, status)
	})
}