package logging

import "go.uber.org/zap/zapcore"

// stripANSICore 包装底层 core，写入前剥离消息中的 ANSI 转义。
// 用于文件 sink，保证 JSON/纯文本干净，同时 stderr 仍可保留颜色。
type stripANSICore struct {
	zapcore.Core
}

func (c stripANSICore) With(fields []zapcore.Field) zapcore.Core {
	return stripANSICore{Core: c.Core.With(fields)}
}

func (c stripANSICore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c stripANSICore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	ent.Message = stripANSI(ent.Message)
	return c.Core.Write(ent, fields)
}
