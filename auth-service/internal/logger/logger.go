package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string) (*zap.Logger, error) {

	var zapLevel zapcore.Level

	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel // По умолчанию INFO
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder

	if level == "debug" {
		// Для разработки: цветной вывод в консоль
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		// Для продакшена: JSON
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Ядро логгера
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapLevel)

	logger := zap.New(core, zap.AddCaller())
	return logger, nil
}

