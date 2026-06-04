package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global application logger.
var Log *zap.SugaredLogger

// Init initializes the global logger.
func Init(level, filePath string) {
	var cfg zap.Config

	level = strings.ToLower(level)
	atomicLevel := zap.NewAtomicLevel()
	switch level {
	case "debug":
		atomicLevel.SetLevel(zapcore.DebugLevel)
	case "warn":
		atomicLevel.SetLevel(zapcore.WarnLevel)
	case "error":
		atomicLevel.SetLevel(zapcore.ErrorLevel)
	default:
		atomicLevel.SetLevel(zapcore.InfoLevel)
	}

	if filePath != "" {
		// File logging
		cfg = zap.NewProductionConfig()
		cfg.OutputPaths = []string{filePath}
		cfg.ErrorOutputPaths = []string{filePath}
	} else {
		// Console logging with color
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	cfg.Level = atomicLevel
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l, err := cfg.Build()
	if err != nil {
		os.Stderr.WriteString("logger init error: " + err.Error() + "\n")
		Log = zap.NewExample().Sugar()
		return
	}

	Log = l.Sugar()
}

// Sync flushes any buffered log entries.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// Debug logs at debug level.
func Debug(msg string, args ...interface{}) {
	if Log != nil {
		Log.Debugw(msg, args...)
	}
}

// Info logs at info level.
func Info(msg string, args ...interface{}) {
	if Log != nil {
		Log.Infow(msg, args...)
	}
}

// Warn logs at warn level.
func Warn(msg string, args ...interface{}) {
	if Log != nil {
		Log.Warnw(msg, args...)
	}
}

// Error logs at error level.
func Error(msg string, args ...interface{}) {
	if Log != nil {
		Log.Errorw(msg, args...)
	}
}

// Fatal logs at fatal level and exits.
func Fatal(msg string, args ...interface{}) {
	if Log != nil {
		Log.Fatalw(msg, args...)
	}
	os.Exit(1)
}