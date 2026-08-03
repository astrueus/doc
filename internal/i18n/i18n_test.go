package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetMessageTrIsExist(t *testing.T) {
	Reset()
	dir := t.TempDir()
	path := filepath.Join(dir, "zh-cn.ini")
	content := `[common]
home = 首页
creator = 创始人

[message]
param_error = 参数错误 %s
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetMessage("zh-cn", path); err != nil {
		t.Fatal(err)
	}
	if !IsExist("zh-cn") {
		t.Fatal("expected zh-cn to exist")
	}
	if IsExist("en-us") {
		t.Fatal("en-us should not exist yet")
	}
	if got := Tr("zh-cn", "common.home"); got != "首页" {
		t.Fatalf("home: got %q", got)
	}
	if got := Tr("zh-cn", "message.param_error", "x"); got != "参数错误 x" {
		t.Fatalf("sprintf: got %q", got)
	}
	if got := Tr("zh-cn", "missing.key"); got != "missing.key" {
		t.Fatalf("missing: got %q", got)
	}
}
