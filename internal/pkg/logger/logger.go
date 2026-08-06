package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/mkoteknik/ospanel/internal/pkg/config"
)

// Logger zap logger wrapper
type Logger struct {
	*zap.SugaredLogger
	zap *zap.Logger
}

// New yeni bir logger oluşturur
func New(cfg config.LogConfig) (*Logger, error) {
	level := parseLevel(cfg.Level)

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var core zapcore.Core

	switch cfg.Output {
	case "file":
		file, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(file),
			level,
		)
	default:
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
	}

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
		zap:           zapLogger,
	}, nil
}

// Sync logger buffer'ını temizler
func (l *Logger) Sync() {
	if l.zap != nil {
		l.zap.Sync()
	}
}

// Debugw debug seviyesinde loglama
func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Debugw(msg, keysAndValues...)
}

// Infow info seviyesinde loglama
func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Infow(msg, keysAndValues...)
}

// Warnw warning seviyesinde loglama
func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Warnw(msg, keysAndValues...)
}

// Errorw error seviyesinde loglama
func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Errorw(msg, keysAndValues...)
}

// Fatalw fatal seviyesinde loglama ve programı sonlandırır
func (l *Logger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Fatalw(msg, keysAndValues...)
}

// parseLevel string'den zapcore.Level çevirir
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
