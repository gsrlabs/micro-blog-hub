package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string, mode string) (*zap.Logger, error) {
	// Определяем уровень логирования
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel // По умолчанию INFO
	}

	// Конфигурация энкодера (формат вывода)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // Время в формате 2024-02-07T...

	var encoder zapcore.Encoder
	if mode == "debug" {
		// Для разработки: цветной вывод в консоль
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		// Для продакшена: JSON
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Ядро логгера
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapLevel)

	// Создаем логгер (AddCaller добавляет имя файла и строку, где вызван лог)
	logger := zap.New(core, zap.AddCaller())

	return logger, nil
}
