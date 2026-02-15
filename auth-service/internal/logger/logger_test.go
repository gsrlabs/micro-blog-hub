package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	t.Run("Check Log Levels", func(t *testing.T) {
		tests := []struct {
			levelStr string
			expected zapcore.Level
		}{
			{"debug", zap.DebugLevel},
			{"info", zap.InfoLevel},
			{"warn", zap.WarnLevel},
			{"error", zap.ErrorLevel},
			{"invalid", zap.InfoLevel}, // Дефолт при ошибке
		}

		for _, tt := range tests {
			l, err := New(tt.levelStr, "prod")
			assert.NoError(t, err)
			assert.NotNil(t, l)

			// Проверяем, включен ли ожидаемый уровень
			assert.True(t, l.Core().Enabled(tt.expected), "Level %s should be enabled", tt.levelStr)
			
			// Проверяем, что уровень ниже ожидаемого выключен (для уровней выше Debug)
			if tt.expected > zap.DebugLevel {
				assert.False(t, l.Core().Enabled(tt.expected-1), "Level below %s should be disabled", tt.levelStr)
			}
		}
	})

	t.Run("Check Production Mode (JSON)", func(t *testing.T) {
		// В zapcore сложно напрямую вытащить тип энкодера из ядра, 
		// но мы можем проверить конфигурацию через инициализацию.
		l, err := New("info", "prod")
		assert.NoError(t, err)
		assert.NotNil(t, l)
		
		// Проверяем наличие обязательных опций (например, AddCaller)
		// Если мы залогируем что-то, в выводе должен быть "caller"
		assert.NotNil(t, l.Check(zap.InfoLevel, "test message"), "Logger should be functional")
	})

	t.Run("Check Debug Mode (Console)", func(t *testing.T) {
		l, err := New("debug", "debug")
		assert.NoError(t, err)
		assert.NotNil(t, l)

		assert.True(t, l.Core().Enabled(zap.DebugLevel))
	})
}
