package main

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	// 1. МАГИЯ ПУТЕЙ: Переходим в корень auth-service
	// Сохраняем оригинальную директорию, чтобы вернуть ее после теста
	originalWD, _ := os.Getwd()
	err := os.Chdir("../../") 
	require.NoError(t, err, "не удалось сменить рабочую директорию на корень сервиса")
	defer os.Chdir(originalWD) // Возвращаем как было при выходе из теста

	// 2. Настройка окружения
	// Теперь мы в корне auth-service. Viper легко найдет "config/config.yml"
	os.Setenv("APP_PORT", "8041")
	os.Setenv("APP_MODE", "test")
	os.Setenv("DB_HOST", "localhost")
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", "password123") // Замени на пароль локальной БД, если нужно
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)

	// 3. Запуск приложения
	go func() {
		// Вызываем твой оригинальный run(ctx). 
		// Он будет искать "config/config.yml" и найдет его!
		errChan <- run(ctx)
	}()

	// 4. Ожидание старта (Polling)
	success := false
	for i := 0; i < 10; i++ {
		resp, err := http.Get("http://localhost:8041/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			success = true
			break
		}
		
		// Проверяем, не упал ли run() с другой ошибкой
		select {
		case err := <-errChan:
			t.Fatalf("Приложение упало при старте: %v", err)
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}

	require.True(t, success, "Сервис не поднялся на порту 8041 за 5 секунд")

	// 5. Тестируем Health Check
	t.Run("Health Check", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8041/health")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	// 6. Тестируем Graceful Shutdown
	t.Run("Graceful Shutdown", func(t *testing.T) {
		process, _ := os.FindProcess(os.Getpid())
		_ = process.Signal(syscall.SIGINT)

		select {
		case err := <-errChan:
			assert.NoError(t, err, "Ошибка при остановке сервера")
		case <-time.After(5 * time.Second):
			t.Fatal("Сервис не завершился по сигналу SIGINT")
		}
	})
}