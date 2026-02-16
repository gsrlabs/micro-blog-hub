package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"go.uber.org/zap"
)

// AuthMiddleware проверяет валидность JWT
func (h *AuthHandler) AuthMiddleware(c *gin.Context) {
	tokenString, err := c.Cookie("token")
	if err != nil {
		// Если нет в куках, пробуем достать из заголовка Authorization: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		// Убираем "Bearer "
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth header"})
			return
		}
	}

	// Парсим токен
	token, err := jwt.ParseWithClaims(tokenString, &model.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.cfg.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Если токен валиден, достаем claims
	claims, ok := token.Claims.(*model.UserClaims)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	// ВАЖНО: Кладем UserID в контекст, чтобы следующие хендлеры знали, кто делает запрос
	c.Set("userID", claims.UserID)
	c.Set("username", claims.Username)

	c.Next()
}

// ZapLogger — это middleware, который заменяет стандартный логгер Gin на наш Zap
func ZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Обработка запроса дальше по цепочке
		c.Next()

		// После того как запрос обработан, собираем данные
		latency := time.Since(start)
		status := c.Writer.Status()

		// Формируем структурированный лог
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
		}

		if len(c.Errors) > 0 {
			// Если внутри обработчика случились ошибки
			for _, e := range c.Errors.Errors() {
				logger.Error(e, fields...)
			}
		} else if status >= 500 {
			logger.Error("server error", fields...)
		} else if status >= 400 {
			logger.Warn("client error", fields...)
		} else {
			logger.Info("request processed", fields...)
		}
	}
}
