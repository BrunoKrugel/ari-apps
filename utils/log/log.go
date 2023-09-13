package log

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger *Logger
)

type Logger struct {
	*zap.Logger
}

func InitGlobalLogger(config zap.Config) error {
	config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	logger, err := config.Build()
	if err != nil {
		return err
	}
	globalLogger = &Logger{Logger: logger}
	return nil
}

func GlobalLogger() *Logger {
	return globalLogger
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := []zap.Field{
		zap.String("traceID", TraceIDFromContext(ctx)),
	}
	return &Logger{Logger: l.With(fields...)}
}

func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value("traceID").(string); ok {
		return traceID
	}
	return ""
}

func InfoWithContext(ctx context.Context, message string, fields ...zap.Field) {
	GlobalLogger().WithContext(ctx).Info(message, fields...)
}

func DebugWithContext(ctx context.Context, message string, fields ...zap.Field) {
	GlobalLogger().WithContext(ctx).Debug(message, fields...)
}

func ErrorWithContext(ctx context.Context, message string, fields ...zap.Field) {
	GlobalLogger().WithContext(ctx).Error(message, fields...)
}
