package logger

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// Ключ для хранения логгера в контексте
type key string

const LoggerKey key = "logger"

var globalLogger zerolog.Logger

func init() {
	// Включаем трассировку ошибок (показывает стек вызовов)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Настройка глобального логгера
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}
	globalLogger = zerolog.New(output).
		With().
		Timestamp().
		Logger()

	// Устанавливаем уровень по умолчанию
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func Get() *zerolog.Logger {
	return &globalLogger
}

// SetLevel устанавливает уровень логирования
func SetLevel(level string) {
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		globalLogger.Warn().Str("level", level).Msg("Invalid log level, using 'info'")
		l = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(l)
}

// WithContext возвращает логгер из контекста или глобальный
func WithContext(ctx context.Context) zerolog.Logger {
	if ctx == nil {
		return globalLogger
	}
	if logger, ok := ctx.Value(LoggerKey).(zerolog.Logger); ok {
		return logger
	}
	return globalLogger
}

// WithCtx возвращает новый контекст с логгером
func WithCtx(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// Debug logs a message at debug level
func Debug(msg string, fields ...map[string]interface{}) {
	logEntry := globalLogger.Debug()
	for _, f := range fields {
		for k, v := range f {
			logEntry.Interface(k, v)
		}
	}
	logEntry.Msg(msg)
}

// Info logs a message at info level
func Info(msg string, fields ...map[string]interface{}) {
	logEntry := globalLogger.Info()
	for _, f := range fields {
		for k, v := range f {
			logEntry.Interface(k, v)
		}
	}
	logEntry.Msg(msg)
}

// Warn logs a message at warn level
func Warn(msg string, fields ...map[string]interface{}) {
	logEntry := globalLogger.Warn()
	for _, f := range fields {
		for k, v := range f {
			logEntry.Interface(k, v)
		}
	}
	logEntry.Msg(msg)
}

// Error logs a message at error level
func Error(msg string, err error, fields ...map[string]interface{}) {
	logEntry := globalLogger.Error().Err(err)
	for _, f := range fields {
		for k, v := range f {
			logEntry.Interface(k, v)
		}
	}
	logEntry.Msg(msg)
}

// Fatal logs a message and calls os.Exit(1)
func Fatal(msg string, err error, fields ...map[string]interface{}) {
	logEntry := globalLogger.Fatal().Err(err)
	for _, f := range fields {
		for k, v := range f {
			logEntry.Interface(k, v)
		}
	}
	logEntry.Msg(msg)
}
