package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git.itopcms.com/jackliu/doc/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Options configures NewLogger.
type Options struct {
	// SuppressConsole skips stderr console core (MCP stdio).
	SuppressConsole bool
	// LogDir is the directory for log.log (already resolved by caller).
	LogDir string
}

// NewLogger builds a zap logger with lumberjack file rotation.
// Console output (if enabled) goes to stderr only — never stdout.
func NewLogger(sec config.LogSection, opts Options) (*zap.Logger, error) {
	dir := opts.LogDir
	if dir == "" {
		dir = sec.Path
	}
	if dir == "" {
		dir = "./runtime/logs"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	filename := filepath.Join(dir, "log.log")

	maxSizeMB := 100
	if sec.MaxSize > 0 {
		if sec.MaxSize >= 1024*1024 {
			maxSizeMB = sec.MaxSize / (1024 * 1024)
			if maxSizeMB < 1 {
				maxSizeMB = 1
			}
		} else {
			maxSizeMB = sec.MaxSize
		}
	}
	maxAge := sec.MaxDays
	if maxAge <= 0 {
		maxAge = 30
	}
	maxBackups := 30

	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	})

	level := parseLevel(sec.Level)
	encoder := buildEncoder(sec.Format)

	cores := []zapcore.Core{
		zapcore.NewCore(encoder, fileWriter, level),
	}
	if !opts.SuppressConsole {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), level))
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger, nil
}

func buildEncoder(format string) zapcore.Encoder {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "console" {
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		return zapcore.NewConsoleEncoder(cfg)
	}
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewJSONEncoder(cfg)
}

func parseLevel(level string) zapcore.LevelEnabler {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "emergency", "alert", "critical", "error":
		return zapcore.ErrorLevel
	case "warning", "warn":
		return zapcore.WarnLevel
	case "notice", "informational", "info":
		return zapcore.InfoLevel
	case "debug", "trace":
		return zapcore.DebugLevel
	default:
		return zapcore.InfoLevel
	}
}
