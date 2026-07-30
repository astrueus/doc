package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

// app_key
func GetAppKey() string {
	return MustGlobal().App.Key
}

func GetDatabasePrefix() string {
	return MustGlobal().Database.Prefix
}

// 获取默认头像
func GetDefaultAvatar() string {
	return URLForWithCdnImage(MustGlobal().App.Avatar)
}

// GetDefaultLang returns i18n::default_lang (fallback zh-cn).
func GetDefaultLang() string {
	lang := MustGlobal().I18n.DefaultLang
	if lang == "" {
		return "zh-cn"
	}
	return lang
}

// 获取阅读令牌长度.
func GetTokenSize() int {
	return MustGlobal().App.TokenSize
}

// 获取默认文档封面.
func GetDefaultCover() string {
	return URLForWithCdnImage(MustGlobal().App.Cover)
}

// 获取允许的商城文件的类型.
func GetUploadFileExt() []string {
	ext := MustGlobal().Upload.FileExt

	temp := strings.Split(ext, "|")

	exts := make([]string, len(temp))

	i := 0
	for _, item := range temp {
		if item != "" {
			exts[i] = item
			i++
		}
	}
	return exts
}

// ParseDataSize 解析带单位的体积配置，支持 KB/MB/GB 后缀；无单位时按 KB 计。
// 无法解析时返回 0。
func ParseDataSize(size string) int64 {
	size = strings.TrimSpace(size)
	if size == "" || size == "0" {
		return 0
	}
	upper := strings.ToUpper(size)
	if strings.HasSuffix(upper, "GB") {
		if s, e := strconv.ParseInt(strings.TrimSpace(size[:len(size)-2]), 10, 64); e == nil {
			return s * 1024 * 1024 * 1024
		}
		return 0
	}
	if strings.HasSuffix(upper, "MB") {
		if s, e := strconv.ParseInt(strings.TrimSpace(size[:len(size)-2]), 10, 64); e == nil {
			return s * 1024 * 1024
		}
		return 0
	}
	if strings.HasSuffix(upper, "KB") {
		if s, e := strconv.ParseInt(strings.TrimSpace(size[:len(size)-2]), 10, 64); e == nil {
			return s * 1024
		}
		return 0
	}
	if s, e := strconv.ParseInt(size, 10, 64); e == nil {
		return s * 1024
	}
	return 0
}

// 获取上传文件允许的最大值（业务层单文件限制）
func GetUploadFileSize() int64 {
	return ParseDataSize(MustGlobal().Upload.FileSize)
}

// GetUploadMaxSize 框架层 HTTP 请求体上限，对应 Beego MaxUploadSize；默认 1GB。
func GetUploadMaxSize() int64 {
	if n := ParseDataSize(MustGlobal().Upload.MaxSize); n > 0 {
		return n
	}
	return 1 << 30
}

// GetUploadMaxMemory multipart 解析内存阈值，对应 Beego MaxMemory；默认 64MB。
func GetUploadMaxMemory() int64 {
	if n := ParseDataSize(MustGlobal().Upload.MaxMemory); n > 0 {
		return n
	}
	return 1 << 26
}

// 是否启用导出
func GetEnableExport() bool {
	return MustGlobal().Export.Enable
}

// 同一项目导出线程的并发数
func GetExportProcessNum() int {
	exportProcessNum := MustGlobal().Export.ProcessNum

	if exportProcessNum <= 0 || exportProcessNum > 4 {
		exportProcessNum = 1
	}
	return exportProcessNum
}

// 导出项目队列的并发数量
func GetExportLimitNum() int {
	exportLimitNum := MustGlobal().Export.LimitNum

	if exportLimitNum < 0 {
		exportLimitNum = 1
	}
	return exportLimitNum
}

// 等待导出队列的长度
func GetExportQueueLimitNum() int {
	exportQueueLimitNum := MustGlobal().Export.QueueLimitNum

	if exportQueueLimitNum <= 0 {
		exportQueueLimitNum = 100
	}
	return exportQueueLimitNum
}

// 默认导出项目的缓存目录
func GetExportOutputPath() string {
	out := MustGlobal().Export.OutputPath
	if out == "" {
		out = filepath.Join(WorkingDirectory, "runtime", "cache")
	}
	return filepath.Join(out, "books")
}

// 判断是否是允许商城的文件类型.
func IsAllowUploadFileExt(ext string) bool {

	if strings.HasPrefix(ext, ".") {
		ext = string(ext[1:])
	}
	exts := GetUploadFileExt()

	for _, item := range exts {
		if item == "*" {
			return true
		}
		if strings.EqualFold(item, ext) {
			return true
		}
	}
	return false
}

// 重写生成URL的方法，加上完整的域名
func URLFor(endpoint string, values ...any) string {
	baseUrl := MustGlobal().BaseURL
	pathUrl := web.URLFor(endpoint, values...)

	if baseUrl == "" {
		baseUrl = BaseUrl
	}
	if strings.HasPrefix(pathUrl, "http://") {
		return pathUrl
	}
	if strings.HasPrefix(pathUrl, "/") && strings.HasSuffix(baseUrl, "/") {
		return baseUrl + pathUrl[1:]
	}
	if !strings.HasPrefix(pathUrl, "/") && !strings.HasSuffix(baseUrl, "/") {
		return baseUrl + "/" + pathUrl
	}
	return baseUrl + web.URLFor(endpoint, values...)
}

func URLForNotHost(endpoint string, values ...any) string {
	baseUrl := MustGlobal().BaseURL
	pathUrl := web.URLFor(endpoint, values...)

	if baseUrl == "" {
		baseUrl = "/"
	}
	if strings.HasPrefix(pathUrl, "http://") {
		return pathUrl
	}
	if strings.HasPrefix(pathUrl, "/") && strings.HasSuffix(baseUrl, "/") {
		return baseUrl + pathUrl[1:]
	}
	if !strings.HasPrefix(pathUrl, "/") && !strings.HasSuffix(baseUrl, "/") {
		return baseUrl + "/" + pathUrl
	}
	return baseUrl + web.URLFor(endpoint, values...)
}

func URLForWithCdnImage(p string) string {
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	cdn := MustGlobal().CDN.Img
	//如果没有设置cdn，则使用baseURL拼接
	if cdn == "" {
		baseUrl := MustGlobal().BaseURL
		if baseUrl == "" {
			baseUrl = "/"
		}

		if strings.HasPrefix(p, "/") && strings.HasSuffix(baseUrl, "/") {
			return baseUrl + p[1:]
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(baseUrl, "/") {
			return baseUrl + "/" + p
		}
		return baseUrl + p
	}
	if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
		return cdn + string(p[1:])
	}
	if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
		return cdn + "/" + p
	}
	return cdn + p
}

func URLForWithCdnCss(p string, v ...string) string {
	cdn := MustGlobal().CDN.CSS
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	filePath := WorkingDir(p)

	if f, err := os.Stat(filePath); err == nil && !strings.Contains(p, "?") && len(v) > 0 && v[0] == "version" {
		p = p + fmt.Sprintf("?v=%s", f.ModTime().Format("20060102150405"))
	}
	//如果没有设置cdn，则使用baseURL拼接
	if cdn == "" {
		baseUrl := MustGlobal().BaseURL
		if baseUrl == "" {
			baseUrl = "/"
		}

		if strings.HasPrefix(p, "/") && strings.HasSuffix(baseUrl, "/") {
			return baseUrl + p[1:]
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(baseUrl, "/") {
			return baseUrl + "/" + p
		}
		return baseUrl + p
	}
	if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
		return cdn + string(p[1:])
	}
	if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
		return cdn + "/" + p
	}
	return cdn + p
}

func URLForWithCdnJs(p string, v ...string) string {
	cdn := MustGlobal().CDN.JS
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}

	filePath := WorkingDir(p)

	if f, err := os.Stat(filePath); err == nil && !strings.Contains(p, "?") && len(v) > 0 && v[0] == "version" {
		p = p + fmt.Sprintf("?v=%s", f.ModTime().Format("20060102150405"))
	}

	//如果没有设置cdn，则使用baseURL拼接
	if cdn == "" {
		baseUrl := MustGlobal().BaseURL
		if baseUrl == "" {
			baseUrl = "/"
		}

		if strings.HasPrefix(p, "/") && strings.HasSuffix(baseUrl, "/") {
			return baseUrl + p[1:]
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(baseUrl, "/") {
			return baseUrl + "/" + p
		}
		return baseUrl + p
	}
	if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
		return cdn + string(p[1:])
	}
	if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
		return cdn + "/" + p
	}
	return cdn + p
}
