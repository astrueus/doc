package i18n

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Package i18n 替代 github.com/beego/i18n。
// 加载既有 conf/lang/*.ini（section.key），对外提供 Tr / IsExist / SetMessage。

var (
	mu       sync.RWMutex
	messages = map[string]map[string]string{} // lang -> "section.key" -> text
)

// SetMessage 为指定语言加载 beego 风格的 ini 语言包（如 zh-cn）。
func SetMessage(lang, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	m := make(map[string]string)
	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if section != "" {
			key = section + "." + key
		}
		m[key] = val
	}
	if err := sc.Err(); err != nil {
		return err
	}

	mu.Lock()
	messages[lang] = m
	mu.Unlock()
	return nil
}

// IsExist 报告该语言是否已加载。
func IsExist(lang string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := messages[lang]
	return ok
}

// Tr 返回指定语言下 key 的译文。
// 若传入额外参数，按 fmt.Sprintf 格式化（兼容 beego 用法）。
func Tr(lang, key string, args ...any) string {
	mu.RLock()
	m := messages[lang]
	var text string
	var ok bool
	if m != nil {
		text, ok = m[key]
	}
	mu.RUnlock()
	if !ok {
		text = key
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}

// Reset 清空已加载文案（仅测试使用）。
func Reset() {
	mu.Lock()
	messages = map[string]map[string]string{}
	mu.Unlock()
}
