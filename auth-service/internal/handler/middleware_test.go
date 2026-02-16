package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// Вспомогательная функция для генерации токена в тестах
func generateTestToken(userID uuid.UUID, username string, secret string, expired bool) string {
	expiration := time.Now().Add(time.Hour)
	if expired {
		expiration = time.Now().Add(-time.Hour)
	}

	claims := &model.UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tString, _ := token.SignedString([]byte(secret))
	return tString
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	h := &AuthHandler{
		cfg: &config.Config{JWT: config.JWTConfig{Secret: secret}},
	}

	userID := uuid.New()
	username := "testuser"

	t.Run("No Token - 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.Use(h.AuthMiddleware)
		r.GET("/test", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authorization required")
	})

	t.Run("Valid Token in Header - 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		token := generateTestToken(userID, username, secret, false)

		r.Use(h.AuthMiddleware)
		r.GET("/test", func(ctx *gin.Context) {
			// Проверяем, что userID попал в контекст
			id, _ := ctx.Get("userID")
			assert.Equal(t, userID, id)
			ctx.Status(http.StatusOK)
		})

		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Expired Token - 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		token := generateTestToken(userID, username, secret, true)

		r.Use(h.AuthMiddleware)
		r.GET("/test", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid token")
	})
}

func TestAuthMiddleware_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	h := &AuthHandler{cfg: &config.Config{JWT: config.JWTConfig{Secret: secret}}}
	userID := uuid.New()
	username := "tester"

	t.Run("Invalid Auth Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(h.AuthMiddleware)
		r.GET("/test", func(c *gin.Context) {})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "BadHeader")
		c.Request = req

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid auth header")
	})

	t.Run("Invalid Token Signature", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(h.AuthMiddleware)
		r.GET("/test", func(c *gin.Context) {})

		token := generateTestToken(userID, username, "wrong-secret", false)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		c.Request = req

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid token")
	})
}

func TestZapLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Создаем наблюдатель за логами
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	t.Run("Log Success Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(ZapLogger(logger))
		r.GET("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong")
		})

		req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(w, req)

		// Проверяем, что логгер что-то записал
		assert.Equal(t, 1, recorded.Len())
		logEntry := recorded.All()[0]

		assert.Equal(t, "request processed", logEntry.Message)
		// Проверяем поля (status, method)
		assert.Equal(t, int64(200), logEntry.ContextMap()["status"])
		assert.Equal(t, "GET", logEntry.ContextMap()["method"])
	})

	t.Run("Log Client Error", func(t *testing.T) {
		recorded.TakeAll() // Очищаем старые логи
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(ZapLogger(logger))
		r.GET("/404", func(c *gin.Context) {
			c.Status(http.StatusNotFound)
		})

		req, _ := http.NewRequest(http.MethodGet, "/404", nil)
		r.ServeHTTP(w, req)

		logEntry := recorded.All()[0]
		assert.Equal(t, zap.WarnLevel, logEntry.Level)
		assert.Equal(t, "client error", logEntry.Message)
	})
}
