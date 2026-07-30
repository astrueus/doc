package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

var (
	ConfigurationFile = "./conf/app.conf"
	WorkingDirectory  = "./"
	LogFile           = "./runtime/logs"
	BaseUrl           = ""
	AutoLoadDelay     = 0
)

// ResolveWorkingDirectory 解析程序工作目录。
// 优先级：-dir 参数 > 环境变量 DOC_HOME > 可执行文件所在目录 > 当前工作目录。
func ResolveWorkingDirectory(dirFlag string) (string, error) {
	if p := strings.TrimSpace(dirFlag); p != "" {
		return filepath.Abs(p)
	}
	if p := strings.TrimSpace(os.Getenv("DOC_HOME")); p != "" {
		return filepath.Abs(p)
	}
	if exe, err := filepath.Abs(os.Args[0]); err == nil {
		return filepath.Dir(exe), nil
	}
	return filepath.Abs(".")
}

// WorkingDir joins paths under WorkingDirectory.
func WorkingDir(elem ...string) string {
	elems := append([]string{WorkingDirectory}, elem...)
	return filepath.Join(elems...)
}

func init() {
	if p, err := filepath.Abs("./conf/app.conf"); err == nil {
		ConfigurationFile = p
	}
	if p, err := filepath.Abs("./"); err == nil {
		WorkingDirectory = p
	}
	if p, err := filepath.Abs("./runtime/logs"); err == nil {
		LogFile = p
	}
	// 提前加载 + typed Config + BConfig 回填：beego 默认从 ./conf/app.conf 读根键；
	// session 在 [session]，必须 ApplyToBeego，否则路由注册会把 SessionOn=false 固化进路由。
	if _, err := os.Stat(ConfigurationFile); err == nil {
		if _, err := Load(ConfigurationFile); err != nil {
			// 预加载失败仍保底 SessionOn，ResolveCommand 会再 Load
			web.BConfig.WebConfig.Session.SessionOn = true
		}
	} else {
		web.BConfig.WebConfig.Session.SessionOn = true
	}
}
