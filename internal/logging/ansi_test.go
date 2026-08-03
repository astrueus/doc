package logging

import (
	"bytes"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestStripANSI(t *testing.T) {
	in := "| \x1b[42m 200 \x1b[0m| GET"
	got := stripANSI(in)
	want := "|  200 | GET"
	if got != want {
		t.Fatalf("stripANSI: got %q want %q", got, want)
	}
}

func TestStripANSICore_fileClean_consoleKeepsColor(t *testing.T) {
	var fileBuf, consoleBuf bytes.Buffer
	level := zapcore.DebugLevel
	msg := "status \x1b[42m200\x1b[0m"

	fileCore := stripANSICore{Core: zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&fileBuf),
		level,
	)}
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&consoleBuf),
		level,
	)
	logger := zap.New(zapcore.NewTee(fileCore, consoleCore))
	logger.Info(msg)
	_ = logger.Sync()

	if bytes.Contains(fileBuf.Bytes(), []byte("\x1b")) || bytes.Contains(fileBuf.Bytes(), []byte(`\u001b`)) {
		t.Fatalf("file still has ANSI: %s", fileBuf.String())
	}
	if !bytes.Contains(fileBuf.Bytes(), []byte("status 200")) {
		t.Fatalf("file missing stripped msg: %s", fileBuf.String())
	}
	if !bytes.Contains(consoleBuf.Bytes(), []byte("\x1b[42m")) {
		t.Fatalf("console lost ANSI: %s", consoleBuf.String())
	}
}
