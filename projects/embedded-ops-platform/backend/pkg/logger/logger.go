package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// Init initializes the global logger.
func Init(mode string) {
	var cfg zap.Config

	if mode == "debug" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02T15:04:05.000Z0700"))
		}
	}

	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	var err error
	Log, err = cfg.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		os.Stderr.WriteString("failed to init logger: " + err.Error() + "\n")
		os.Exit(1)
	}
}

// Sync flushes any buffered log entries.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// --- Convenience wrappers for zap.Field creation ---

// String creates a string field.
func String(key, value string) zap.Field { return zap.String(key, value) }

// Int creates an int field.
func Int(key string, value int) zap.Field { return zap.Int(key, value) }

// Duration creates a duration field.
func Duration(key string, value time.Duration) zap.Field { return zap.Duration(key, value) }

// ErrField creates a zap.Field from an error (named to avoid conflict with Error log function).
func ErrField(err error) zap.Field { return zap.Error(err) }

// Bool creates a bool field.
func Bool(key string, value bool) zap.Field { return zap.Bool(key, value) }

// Float64 creates a float64 field.
func Float64(key string, value float64) zap.Field { return zap.Float64(key, value) }

// Any creates a field for any value.
func Any(key string, value interface{}) zap.Field { return zap.Any(key, value) }

// --- Log level functions ---

// Debug logs a debug message.
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

// Info logs an info message.
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

// Warn logs a warning message.
func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

// Error logs an error message.
func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// Fatal logs a fatal message and exits.
func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}