package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init 初始化日志
func Init(level string, format string) error {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zapLevel,
	)

	log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return nil
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	if log != nil {
		log.Debug(msg, fields...)
	}
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	if log != nil {
		log.Info(msg, fields...)
	}
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	if log != nil {
		log.Warn(msg, fields...)
	}
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	if log != nil {
		log.Error(msg, fields...)
	}
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	if log != nil {
		log.Fatal(msg, fields...)
	}
}

// With 创建带字段的日志器
func With(fields ...zap.Field) *zap.Logger {
	if log != nil {
		return log.With(fields...)
	}
	return zap.NewNop()
}

// Sync 同步日志
func Sync() error {
	if log != nil {
		return log.Sync()
	}
	return nil
}

// 便捷字段函数
func String(key, val string) zap.Field {
	return zap.String(key, val)
}

func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

func Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

func Float64(key string, val float64) zap.Field {
	return zap.Float64(key, val)
}

func Bool(key string, val bool) zap.Field {
	return zap.Bool(key, val)
}

func Any(key string, val interface{}) zap.Field {
	return zap.Any(key, val)
}

func ErrorField(err error) zap.Field {
	return zap.Error(err)
}
