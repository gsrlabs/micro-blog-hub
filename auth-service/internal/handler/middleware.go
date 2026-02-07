package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)



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