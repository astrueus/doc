package logging

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var registerOnce sync.Once

// RegisterBeeLoggerAdapter registers the "zap" beego logs adapter once.
func RegisterBeeLoggerAdapter() {
	registerOnce.Do(func() {
		logs.Register("zap", func() logs.Logger {
			return &zapAdapter{logger: zap.L()}
		})
	})
}

// SetAdapterLogger updates the logger used by subsequent adapter instances
// created via SetLogger("zap"). Call after zap.ReplaceGlobals.
func SetAdapterLogger(logger *zap.Logger) {
	if logger == nil {
		return
	}
	adapterLogger.Store(logger)
}

var adapterLogger atomicLogger

type atomicLogger struct {
	mu sync.RWMutex
	l  *zap.Logger
}

func (a *atomicLogger) Store(l *zap.Logger) {
	a.mu.Lock()
	a.l = l
	a.mu.Unlock()
}

func (a *atomicLogger) Load() *zap.Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.l != nil {
		return a.l
	}
	return zap.L()
}

type zapAdapter struct {
	logger    *zap.Logger
	formatter logs.LogFormatter
}

func (a *zapAdapter) Init(config string) error {
	a.logger = adapterLogger.Load()
	return nil
}

func (a *zapAdapter) WriteMsg(lm *logs.LogMsg) error {
	if lm == nil {
		return nil
	}
	logger := a.logger
	if logger == nil {
		logger = adapterLogger.Load()
	}
	msg := lm.Msg
	if len(lm.Args) > 0 {
		msg = fmt.Sprintf(lm.Msg, lm.Args...)
	}
	if lm.Prefix != "" {
		msg = lm.Prefix + " " + msg
	}
	// Keep ANSI for stderr (access-log colors); file core strips via stripANSICore.
	if lm.FilePath != "" {
		msg = fmt.Sprintf("[%s:%d] %s", filepath.Base(lm.FilePath), lm.LineNumber, msg)
	}
	level := mapBeegoLevel(lm.Level)
	logger.Log(level, msg)
	return nil
}

func (a *zapAdapter) Destroy() {}
func (a *zapAdapter) Flush() {
	_ = adapterLogger.Load().Sync()
}
func (a *zapAdapter) SetFormatter(f logs.LogFormatter) {
	a.formatter = f
}

func mapBeegoLevel(level int) zapcore.Level {
	switch level {
	case logs.LevelEmergency, logs.LevelAlert, logs.LevelCritical, logs.LevelError:
		return zapcore.ErrorLevel
	case logs.LevelWarning:
		return zapcore.WarnLevel
	case logs.LevelNotice, logs.LevelInformational:
		return zapcore.InfoLevel
	case logs.LevelDebug:
		return zapcore.DebugLevel
	default:
		return zapcore.InfoLevel
	}
}
