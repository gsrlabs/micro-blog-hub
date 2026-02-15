package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Load(t *testing.T) {
	// Создаем временную директорию для тестового конфига
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// 1. Подготавливаем тестовый YAML
	content := `
app:
  port: "8080"
  mode: "debug"
database:
  host: "localhost"
  port: 5432
  user: "user"
  name: "db"
jwt:
  secret: "supersecret"
  expiration_hours: 24
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	t.Run("Load from file success", func(t *testing.T) {
		cfg, err := Load(configPath)
		assert.NoError(t, err)
		assert.Equal(t, "8080", cfg.App.Port)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, "supersecret", cfg.JWT.Secret)
	})

	t.Run("Override with Environment Variables", func(t *testing.T) {
		// Устанавливаем переменную окружения, которая должна перекрыть файл
		expectedPort := "9090"
		os.Setenv("APP_PORT", expectedPort)
		defer os.Unsetenv("APP_PORT") // Чистим за собой

		cfg, err := Load(configPath)
		assert.NoError(t, err)
		assert.Equal(t, expectedPort, cfg.App.Port)
	})

	t.Run("File not found error", func(t *testing.T) {
		_, err := Load("non_existent.yml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("Validation success", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Host:     "localhost",
				Password: "pass",
			},
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("Missing password error", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Host: "localhost",
				// Password empty
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Equal(t, "DB_PASSWORD is required", err.Error())
	})

	t.Run("Missing host error", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Password: "pass",
				// Host empty
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Equal(t, "DB_HOST is required", err.Error())
	})
}