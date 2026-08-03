package logging

import "go.uber.org/zap/zapcore"

// stripANSICore wraps a core and strips ANSI escapes from the message before write.
// Used for the file sink so JSON/plain files stay clean while stderr keeps colors.
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
