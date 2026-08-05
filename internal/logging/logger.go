package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.itopcms.com/astrueus/doc/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Options 配置 NewLogger。
type Options struct {
	// SuppressConsole 跳过 stderr 控制台 core（MCP stdio 用）。
	SuppressConsole bool
	// LogDir 为 log.log 所在目录（由调用方解析好）。
	LogDir string
}

// NewLogger 构建带 lumberjack 轮转的 zap logger。
// 文件格式跟随 sec.Format（默认 json），写入前剥离 ANSI。
// 启用控制台时输出到 stderr 并保留颜色，绝不写 stdout。
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
	fileEncoder := buildFileEncoder(sec.Format)

	cores := []zapcore.Core{
		stripANSICore{Core: zapcore.NewCore(fileEncoder, fileWriter, level)},
	}
	if !opts.SuppressConsole {
		cores = append(cores, zapcore.NewCore(buildConsoleEncoder(true), zapcore.AddSync(os.Stderr), level))
	}

	// caller 由 beego LogMsg（shim）提供，不用 zap 栈，避免误导为 shim 路径。
	logger := zap.New(zapcore.NewTee(cores...), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger, nil
}

func buildFileEncoder(format string) zapcore.Encoder {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "console" {
		return buildConsoleEncoder(false)
	}
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.CallerKey = "" // shim 已把 [file:line] 写入 msg
	return zapcore.NewJSONEncoder(cfg)
}

func buildConsoleEncoder(color bool) zapcore.Encoder {
	cfg := zapcore.EncoderConfig{
		TimeKey:          "ts",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "", // shim 已把 [file:line] 写入 msg
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeTime:       beegoTimeEncoder,
		EncodeDuration:   zapcore.StringDurationEncoder,
		ConsoleSeparator: " ",
	}
	if color {
		cfg.EncodeLevel = coloredBeegoLevelEncoder
	} else {
		cfg.EncodeLevel = beegoLevelEncoder
	}
	return zapcore.NewConsoleEncoder(cfg)
}

func levelTag(l zapcore.Level) string {
	switch l {
	case zapcore.DebugLevel:
		return "[D]"
	case zapcore.InfoLevel:
		return "[I]"
	case zapcore.WarnLevel:
		return "[W]"
	case zapcore.ErrorLevel:
		return "[E]"
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return "[F]"
	default:
		return "[I]"
	}
}

func beegoLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(levelTag(l))
}

// coloredBeegoLevelEncoder 对齐 beego/core/logs 控制台 brush 色码：
// Emergency 1;37, Alert 1;36, Critical 1;35, Error 1;31, Warning 1;33,
// Notice 1;32, Informational 1;34, Debug 1;44。
func coloredBeegoLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var color string
	switch l {
	case zapcore.DebugLevel:
		color = "\x1b[1;44m" // Debug — background blue
	case zapcore.InfoLevel:
		color = "\x1b[1;34m" // Informational — blue
	case zapcore.WarnLevel:
		color = "\x1b[1;33m" // Warning — yellow
	case zapcore.ErrorLevel:
		color = "\x1b[1;31m" // Error — red
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		color = "\x1b[1;35m" // Critical — magenta
	case zapcore.FatalLevel:
		color = "\x1b[1;37m" // Emergency — white
	default:
		color = "\x1b[1;34m"
	}
	enc.AppendString(color + levelTag(l) + "\x1b[0m")
}

func beegoTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006/01/02 15:04:05.000"))
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
