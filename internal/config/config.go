package config

import (
	"fmt"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

// Global is the process-wide typed configuration. Set by Load/Reload.
var Global *Config

type Config struct {
	App      AppSection
	HTTP     HTTPSection
	Session  SessionSection
	Database DatabaseSection
	Cache    CacheSection
	Log      LogSection
	Upload   UploadSection
	Mail     MailSection
	LDAP     LDAPSection
	Export   ExportSection
	CDN      CDNSection
	OAuth    OAuthSection
	DingTalk DingTalkSection
	I18n     I18nSection
	MCP      MCPSection

	// Root / process-level (also present as unstructured keys for Beego).
	BaseURL         string
	Timezone        string
	AutoLoadDelay   int
	HTTPAddr        string
	HTTPPort        int
	RunMode         string
	AppName         string
}

type AppSection struct {
	Key            string
	HighlightStyle string
	Cover          string
	Avatar         string
	TokenSize      int
	BaiduMapKey    string
}

type HTTPSection struct {
	CopyRequestBody bool
	EnableXSRF      bool
	EnableHTTPS     bool
	HTTPSAddr       string
	HTTPSPort       int
	HTTPSCertFile   string
	HTTPSKeyFile    string
}

type SessionSection struct {
	On               bool
	Name             string
	ServerKey        string
	Provider         string
	ProviderConfig   string
	GCMaxLifetime    int64
}

type DatabaseSection struct {
	Adapter  string
	Host     string
	Port     string
	Database string
	Username string
	Password string
	Prefix   string
}

type CacheSection struct {
	Enable           bool
	Provider         string
	MemoryInterval   int
	FilePath         string
	FileSuffix       string
	FileDirLevel     string
	FileExpiry       string
	MemcacheHost     string
	RedisHost        string
	RedisDB          int
	RedisPassword    string
	RedisPrefix      string
}

type LogSection struct {
	Path      string
	MaxLines  int
	MaxSize   int
	Daily     bool
	MaxDays   int
	Level     string
	IsAsync   bool
	Format    string // file format: json (default) | console; stderr always console-style
}

type UploadSection struct {
	FileExt     string
	FileSize    string
	MaxSize     string
	MaxMemory   string
}

type MailSection struct {
	Enable       bool
	MailNumber   int
	SMTPUserName string
	SMTPHost     string
	SMTPPassword string
	SMTPPort     int
	FormUserName string
	MailExpired  int
	Secure       string
}

type LDAPSection struct {
	Enable    bool
	Host      string
	Port      int
	Attribute string
	Base      string
	User      string
	Password  string
	UserRole  int
	Filter    string
}

type ExportSection struct {
	Enable         bool
	ProcessNum     int
	LimitNum       int
	QueueLimitNum  int
	OutputPath     string
}

type CDNSection struct {
	URL string
	JS  string
	CSS string
	Img string
}

type OAuthSection struct {
	HTTPLoginURL    string
	HTTPLoginSecret string
}

type DingTalkSection struct {
	CorpID    string
	AppKey    string
	AppSecret string
	TmpReader string
	QRKey     string
	QRSecret  string
}

type I18nSection struct {
	DefaultLang string
}

type MCPSection struct {
	Enable        bool
	Listen        string
	StdioMember   string
	TokenRequired bool
	RateLimit     int
}

// Load reads the already-loaded (or loadable) AppConfig into a typed Config,
// assigns Global, and applies Beego BConfig backfill for section-scoped keys.
func Load(path string) (*Config, error) {
	if path != "" {
		if err := web.LoadAppConfig("ini", path); err != nil {
			return nil, fmt.Errorf("load app config: %w", err)
		}
	}
	c := readFromAppConfig()
	Global = c
	ApplyToBeego(c)
	return c, nil
}

// Reload re-reads ConfigurationFile into Global.
func Reload() error {
	_, err := Load(ConfigurationFile)
	return err
}

func readFromAppConfig() *Config {
	return &Config{
		AppName:       rootString("appname", "doc"),
		HTTPAddr:      rootString("httpaddr", ""),
		HTTPPort:      rootInt("httpport", 8181),
		RunMode:       rootString("runmode", "dev"),
		BaseURL:       rootString("baseurl", ""),
		Timezone:      rootString("timezone", "Asia/Shanghai"),
		AutoLoadDelay: rootInt("config_auto_delay", 0),

		App: AppSection{
			Key:            secString("app", "app_key", "doc"),
			HighlightStyle: secString("app", "highlight_style", "github"),
			Cover:          secString("app", "cover", "/static/images/book.jpg"),
			Avatar:         secString("app", "avatar", "/static/images/headimgurl.jpg"),
			TokenSize:      secInt("app", "token_size", 12),
			BaiduMapKey:    secString("app", "baidumapkey", ""),
		},
		HTTP: HTTPSection{
			CopyRequestBody: secBool("http", "copyrequestbody", true),
			EnableXSRF:      secBool("http", "enablexsrf", false),
			EnableHTTPS:     secBool("http", "EnableHTTPS", false),
			HTTPSAddr:       secString("http", "HTTPSAddr", ""),
			HTTPSPort:       secInt("http", "HTTPSPort", 10443),
			HTTPSCertFile:   secString("http", "HTTPSCertFile", "conf/ssl.crt"),
			HTTPSKeyFile:    secString("http", "HTTPSKeyFile", "conf/ssl.key"),
		},
		Session: SessionSection{
			On:             secBool("session", "sessionon", true),
			Name:           secString("session", "sessionname", "doc_id"),
			ServerKey:      secString("session", "beegoserversessionkey", ""),
			Provider:       secString("session", "sessionprovider", "file"),
			ProviderConfig: secString("session", "sessionproviderconfig", "./runtime/session"),
			GCMaxLifetime:  int64(secInt("session", "sessiongcmaxlifetime", 3600)),
		},
		Database: DatabaseSection{
			Adapter:  secString("database", "db_adapter", "mysql"),
			Host:     secString("database", "db_host", "127.0.0.1"),
			Port:     secString("database", "db_port", "3306"),
			Database: secString("database", "db_database", "doc"),
			Username: secString("database", "db_username", "root"),
			Password: secString("database", "db_password", ""),
			Prefix:   secString("database", "db_prefix", "md_"),
		},
		Cache: CacheSection{
			Enable:         secBool("cache", "cache", false),
			Provider:       secString("cache", "cache_provider", "file"),
			MemoryInterval: secInt("cache", "cache_memory_interval", 120),
			FilePath:       secString("cache", "cache_file_path", "./runtime/cache/"),
			FileSuffix:     secString("cache", "cache_file_suffix", ".bin"),
			FileDirLevel:   secString("cache", "cache_file_dir_level", "2"),
			FileExpiry:     secString("cache", "cache_file_expiry", "3600"),
			MemcacheHost:   secString("cache", "cache_memcache_host", "127.0.0.1:11211"),
			RedisHost:      secString("cache", "cache_redis_host", "127.0.0.1:6379"),
			RedisDB:        secInt("cache", "cache_redis_db", 0),
			RedisPassword:  secString("cache", "cache_redis_password", ""),
			RedisPrefix:    secString("cache", "cache_redis_prefix", "doc::cache"),
		},
		Log: LogSection{
			Path:     secString("log", "log_path", "./runtime/logs"),
			MaxLines: secInt("log", "log_maxlines", 1000000),
			MaxSize:  secInt("log", "log_maxsize", 0),
			Daily:    secBool("log", "log_daily", true),
			MaxDays:  secInt("log", "log_maxdays", 7),
			Level:    secString("log", "log_level", "Info"),
			IsAsync:  secBool("log", "log_is_async", true),
			Format:   secString("log", "log_format", "json"),
		},
		Upload: UploadSection{
			FileExt:   secString("upload", "upload_file_ext", "png|jpg|jpeg|gif|txt|doc|docx|pdf"),
			FileSize:  secString("upload", "upload_file_size", "0"),
			MaxSize:   secString("upload", "upload_max_size", "1GB"),
			MaxMemory: secString("upload", "upload_max_memory", "64MB"),
		},
		Mail: MailSection{
			Enable:       secBool("mail", "enable_mail", false),
			MailNumber:   secInt("mail", "mail_number", 5),
			SMTPUserName: secString("mail", "smtp_user_name", ""),
			SMTPHost:     secString("mail", "smtp_host", ""),
			SMTPPassword: secString("mail", "smtp_password", ""),
			SMTPPort:     secInt("mail", "smtp_port", 25),
			FormUserName: secString("mail", "form_user_name", ""),
			MailExpired:  secInt("mail", "mail_expired", 30),
			Secure:       secString("mail", "secure", "NONE"),
		},
		LDAP: LDAPSection{
			Enable:    secBool("ldap", "ldap_enable", false),
			Host:      secString("ldap", "ldap_host", ""),
			Port:      secInt("ldap", "ldap_port", 3268),
			Attribute: secString("ldap", "ldap_attribute", "sAMAccountName"),
			Base:      secString("ldap", "ldap_base", ""),
			User:      secString("ldap", "ldap_user", ""),
			Password:  secString("ldap", "ldap_password", ""),
			UserRole:  secInt("ldap", "ldap_user_role", 2),
			Filter:    secString("ldap", "ldap_filter", "objectClass=posixAccount"),
		},
		Export: ExportSection{
			Enable:        secBool("export", "enable_export", false),
			ProcessNum:    secInt("export", "export_process_num", 1),
			LimitNum:      secInt("export", "export_limit_num", 5),
			QueueLimitNum: secInt("export", "export_queue_limit_num", 100),
			OutputPath:    secString("export", "export_output_path", "./runtime/cache"),
		},
		CDN: CDNSection{
			URL: secString("cdn", "cdn", ""),
			JS:  secString("cdn", "cdnjs", ""),
			CSS: secString("cdn", "cdncss", ""),
			Img: secString("cdn", "cdnimg", ""),
		},
		OAuth: OAuthSection{
			HTTPLoginURL:    secString("oauth", "http_login_url", ""),
			HTTPLoginSecret: secString("oauth", "http_login_secret", ""),
		},
		DingTalk: DingTalkSection{
			CorpID:    secString("dingtalk", "dingtalk_corpid", ""),
			AppKey:    secString("dingtalk", "dingtalk_app_key", ""),
			AppSecret: secString("dingtalk", "dingtalk_app_secret", ""),
			TmpReader: secString("dingtalk", "dingtalk_tmp_reader", ""),
			QRKey:     secString("dingtalk", "dingtalk_qr_key", ""),
			QRSecret:  secString("dingtalk", "dingtalk_qr_secret", ""),
		},
		I18n: I18nSection{
			DefaultLang: secString("i18n", "default_lang", "zh-cn"),
		},
		MCP: MCPSection{
			Enable:        secBool("mcp", "mcp_enable", false),
			Listen:        secString("mcp", "mcp_listen", "127.0.0.1:8280"),
			StdioMember:   secString("mcp", "mcp_stdio_member", "admin"),
			TokenRequired: secBool("mcp", "mcp_token_required", true),
			RateLimit:     secInt("mcp", "mcp_rate_limit", 60),
		},
	}
}

// ApplyToBeego writes section-scoped settings into Beego BConfig so framework
// features (session / HTTPS / XSRF) keep working after keys left the root section.
func ApplyToBeego(c *Config) {
	if c == nil {
		return
	}
	web.BConfig.AppName = c.AppName
	web.BConfig.RunMode = c.RunMode
	web.BConfig.CopyRequestBody = c.HTTP.CopyRequestBody
	web.BConfig.Listen.HTTPAddr = c.HTTPAddr
	web.BConfig.Listen.HTTPPort = c.HTTPPort
	web.BConfig.Listen.EnableHTTPS = c.HTTP.EnableHTTPS
	web.BConfig.Listen.HTTPSAddr = c.HTTP.HTTPSAddr
	web.BConfig.Listen.HTTPSPort = c.HTTP.HTTPSPort
	web.BConfig.Listen.HTTPSCertFile = c.HTTP.HTTPSCertFile
	web.BConfig.Listen.HTTPSKeyFile = c.HTTP.HTTPSKeyFile

	web.BConfig.WebConfig.EnableXSRF = c.HTTP.EnableXSRF
	web.BConfig.WebConfig.Session.SessionOn = c.Session.On
	web.BConfig.WebConfig.Session.SessionName = c.Session.Name
	web.BConfig.WebConfig.Session.SessionProvider = c.Session.Provider
	web.BConfig.WebConfig.Session.SessionProviderConfig = c.Session.ProviderConfig
	web.BConfig.WebConfig.Session.SessionGCMaxLifetime = c.Session.GCMaxLifetime
	if c.Session.ServerKey != "" {
		web.BConfig.WebConfig.Session.SessionName = c.Session.Name
		// Beego stores cookie signing separately; keep name from section.
	}

	web.BConfig.MaxUploadSize = GetUploadMaxSize()
	web.BConfig.MaxMemory = GetUploadMaxMemory()
}

func rootString(key, def string) string {
	return web.AppConfig.DefaultString(key, def)
}

func rootInt(key string, def int) int {
	return web.AppConfig.DefaultInt(key, def)
}

func secString(section, key, def string) string {
	return web.AppConfig.DefaultString(section+"::"+key, def)
}

func secInt(section, key string, def int) int {
	return web.AppConfig.DefaultInt(section+"::"+key, def)
}

func secBool(section, key string, def bool) bool {
	return web.AppConfig.DefaultBool(section+"::"+key, def)
}

// MustGlobal returns Global or a zero-value read from AppConfig (after best-effort Load).
func MustGlobal() *Config {
	if Global != nil {
		return Global
	}
	c := readFromAppConfig()
	Global = c
	return c
}

// PortString returns HTTP port as string (legacy callers).
func PortString() string {
	return strconv.Itoa(MustGlobal().HTTPPort)
}
