package app

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bytes"
	"encoding/json"
	"net/http"

	"git.itopcms.com/astrueus/doc/internal/cache"
	cfg "git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/logging"
	"git.itopcms.com/astrueus/doc/internal/model"
	"git.itopcms.com/astrueus/doc/pkg/filetil"
	beegoCache "github.com/beego/beego/v2/client/cache"
	_ "github.com/beego/beego/v2/client/cache/memcache"
	_ "github.com/beego/beego/v2/client/cache/redis"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"git.itopcms.com/astrueus/doc/internal/i18n"
	"github.com/fsnotify/fsnotify"
	"github.com/lifei6671/gocaptcha"
	"go.uber.org/zap"
)

// RegisterDataBase 注册数据库
func RegisterDataBase() {
	logs.Info("正在初始化数据库配置.")
	g := cfg.MustGlobal()
	dbadapter := g.Database.Adapter
	orm.DefaultTimeLoc = time.Local
	orm.DefaultRowsLimit = -1

	if strings.EqualFold(dbadapter, "mysql") {
		host := g.Database.Host
		database := g.Database.Database
		username := g.Database.Username
		password := g.Database.Password

		timezone := g.Timezone
		location, err := time.LoadLocation(timezone)
		if err == nil {
			orm.DefaultTimeLoc = location
		} else {
			logs.Error("加载时区配置失败 timezone=%s err=%v", timezone, err)
		}

		port := g.Database.Port

		dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=%s", username, password, host, port, database, url.QueryEscape(timezone))

		if err := orm.RegisterDataBase("default", "mysql", dataSource); err != nil {
			logs.Error("注册默认数据库失败->", err)
			os.Exit(1)
		}

	} else if strings.EqualFold(dbadapter, "sqlite3") {

		database := g.Database.Database
		if strings.HasPrefix(database, "./") {
			database = filepath.Join(cfg.WorkingDirectory, string(database[1:]))
		}
		if p, err := filepath.Abs(database); err == nil {
			database = p
		}

		dbPath := filepath.Dir(database)

		if _, err := os.Stat(dbPath); err != nil && os.IsNotExist(err) {
			_ = os.MkdirAll(dbPath, 0777)
		}

		err := orm.RegisterDataBase("default", "sqlite3", database)

		if err != nil {
			logs.Error("注册默认数据库失败->", err)
		}
	} else {
		logs.Error("不支持的数据库类型.")
		os.Exit(1)
	}
	logs.Info("数据库初始化完成.")
}

// RegisterModel 注册Model
func RegisterModel() {
	orm.RegisterModelWithPrefix(cfg.GetDatabasePrefix(),
		new(model.Member),
		new(model.Book),
		new(model.Relationship),
		new(model.Option),
		new(model.Document),
		new(model.Attachment),
		new(model.Logger),
		new(model.MemberToken),
		new(model.MemberApiToken),
		new(model.DocumentHistory),
		new(model.Migration),
		new(model.Label),
		new(model.Blog),
		new(model.Template),
		new(model.Team),
		new(model.TeamMember),
		new(model.TeamRelationship),
		new(model.Itemsets),
	)
	//migrate.RegisterMigration()
}

// suppressConsoleLogger 跳过 beego console 适配器，避免 MCP stdio 污染 stdout。
var suppressConsoleLogger bool

// SuppressConsoleLogger 关闭控制台日志（须在 doc mcp stdio 的 bootstrap 之前调用）。
func SuppressConsoleLogger() {
	suppressConsoleLogger = true
	_ = logs.GetBeeLogger().DelLogger("console")
}

// RegisterLogger 注册日志（zap + lumberjack；beego/logs 经 shim 转发）
func RegisterLogger(logDir string) {
	g := cfg.MustGlobal()

	if logDir == "" {
		logPath, err := filepath.Abs(g.Log.Path)
		if err == nil {
			logDir = logPath
		} else {
			logDir = cfg.WorkingDir("runtime", "logs")
		}
	}

	logger, err := logging.NewLogger(g.Log, logging.Options{
		SuppressConsole: suppressConsoleLogger,
		LogDir:          logDir,
	})
	if err != nil {
		log.Printf("init zap logger failed: %v", err)
		return
	}
	zap.ReplaceGlobals(logger)
	logging.SetAdapterLogger(logger)
	logging.RegisterBeeLoggerAdapter()

	_ = logs.GetBeeLogger().DelLogger("console")
	_ = logs.GetBeeLogger().DelLogger("file")
	_ = logs.GetBeeLogger().DelLogger("zap")

	logs.SetLogFuncCall(true)
	logs.EnableFuncCallDepth(true)
	if err := logs.SetLogger("zap", ""); err != nil {
		log.Printf("set zap beego adapter failed: %v", err)
	}
	if g.Log.IsAsync {
		logs.Async(1e3)
	}
}

// RegisterCommand is retained for compatibility; CLI entry is cli.Execute().
// Deprecated: use cli.Execute().
func RegisterCommand() {
	// no-op: cobra entry is wired from cmd/doc via cli.Execute()
}

// 注册模板函数
func RegisterFunction() {
	err := web.AddFuncMap("config", model.GetOptionValue)

	if err != nil {
		logs.Error("注册函数 config 出错 ->", err)
		os.Exit(-1)
	}
	err = web.AddFuncMap("cdn", func(p string) string {
		g := cfg.MustGlobal()
		cdn := g.CDN.URL
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			return p
		}
		//如果没有设置cdn，则使用baseURL拼接
		if cdn == "" {
			baseUrl := g.BaseURL

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
	})
	if err != nil {
		logs.Error("注册函数 cdn 出错 ->", err)
		os.Exit(-1)
	}

	err = web.AddFuncMap("cdnjs", cfg.URLForWithCdnJs)
	if err != nil {
		logs.Error("注册函数 cdnjs 出错 ->", err)
		os.Exit(-1)
	}
	err = web.AddFuncMap("cdncss", cfg.URLForWithCdnCss)
	if err != nil {
		logs.Error("注册函数 cdncss 出错 ->", err)
		os.Exit(-1)
	}
	err = web.AddFuncMap("cdnimg", cfg.URLForWithCdnImage)
	if err != nil {
		logs.Error("注册函数 cdnimg 出错 ->", err)
		os.Exit(-1)
	}
	//重写url生成，支持配置域名以及域名前缀
	err = web.AddFuncMap("urlfor", cfg.URLFor)
	if err != nil {
		logs.Error("注册函数 urlfor 出错 ->", err)
		os.Exit(-1)
	}
	err = web.AddFuncMap("date_format", func(t time.Time, format string) string {
		return t.In(time.Local).Format(format)
	})
	if err != nil {
		logs.Error("注册函数 date_format 出错 ->", err)
		os.Exit(-1)
	}

	err = web.AddFuncMap("i18n", i18n.Tr)
	if err != nil {
		logs.Error("注册函数 i18n 出错 ->", err)
		os.Exit(-1)
	}
	langs := strings.Split("en-us|zh-cn", "|")
	for _, lang := range langs {
		msgFile := cfg.WorkingDir("conf", "lang", lang+".ini")
		if err := i18n.SetMessage(lang, msgFile); err != nil {
			logs.Error("Fail to set message file: " + err.Error())
			return
		}
	}
}

// 解析命令
func ResolveCommand(args []string) {
	flagSet := flag.NewFlagSet("Doc command: ", flag.ExitOnError)
	var configFile, workingDir, logFile string
	flagSet.StringVar(&configFile, "config", "", "Doc configuration file.")
	flagSet.StringVar(&workingDir, "dir", "", "Doc working directory (overrides DOC_HOME).")
	flagSet.StringVar(&logFile, "log", "", "Doc log file path.")

	if err := flagSet.Parse(args); err != nil {
		log.Fatal("解析命令失败 ->", err)
	}

	resolvedDir, err := cfg.ResolveWorkingDirectory(workingDir)
	if err != nil {
		log.Fatal("解析工作目录失败 ->", err)
	}
	cfg.WorkingDirectory = resolvedDir

	if configFile != "" {
		cfg.ConfigurationFile = configFile
	} else {
		cfg.ConfigurationFile = cfg.WorkingDir("conf", "app.conf")
		exampleConfig := cfg.WorkingDir("conf", "app.conf.example")
		if !filetil.FileExists(cfg.ConfigurationFile) && filetil.FileExists(exampleConfig) {
			_ = filetil.CopyFile(cfg.ConfigurationFile, exampleConfig)
		}
	}
	if err := gocaptcha.SetFontPath(cfg.WorkingDir("web", "static", "fonts")); err != nil {
		log.Fatal("读取字体文件时出错 -> ", err)
	}

	if _, err := cfg.Load(cfg.ConfigurationFile); err != nil {
		log.Fatal("An error occurred:", err)
	}
	g := cfg.MustGlobal()

	if logFile != "" {
		cfg.LogFile = logFile
	} else {
		logPath, err := filepath.Abs(g.Log.Path)
		if err == nil {
			cfg.LogFile = logPath
		} else {
			cfg.LogFile = cfg.WorkingDir("runtime", "logs")
		}
	}
	// 尽早注册，使 bootstrap 的 Info/Debug 统一格式。
	RegisterLogger(cfg.LogFile)

	logs.Info("工作目录 ->", cfg.WorkingDirectory)
	logs.Info("typed config applied → session_on=%v provider=%s httpport=%d",
		g.Session.On, g.Session.Provider, g.HTTPPort)

	web.BConfig.MaxUploadSize = cfg.GetUploadMaxSize()
	web.BConfig.MaxMemory = cfg.GetUploadMaxMemory()
	logs.Info("上传限制 -> MaxUploadSize=%d MaxMemory=%d business_upload_file_size=%d",
		web.BConfig.MaxUploadSize, web.BConfig.MaxMemory, cfg.GetUploadFileSize())

	cfg.AutoLoadDelay = g.AutoLoadDelay
	uploads := cfg.WorkingDir("uploads")

	_ = os.MkdirAll(uploads, 0666)

	web.BConfig.WebConfig.StaticDir["/static"] = filepath.Join(cfg.WorkingDirectory, "web", "static")
	web.BConfig.WebConfig.StaticDir["/uploads"] = uploads
	web.BConfig.WebConfig.ViewsPath = cfg.WorkingDir("web", "views")
	web.BConfig.WebConfig.Session.SessionCookieSameSite = http.SameSiteDefaultMode

	fonts := cfg.WorkingDir("web", "static", "fonts")

	if !filetil.FileExists(fonts) {
		log.Fatal("Font path not exist.")
	}
	if err := gocaptcha.SetFontPath(filepath.Join(cfg.WorkingDirectory, "web", "static", "fonts")); err != nil {
		log.Fatal("读取字体失败 ->", err)
	}

	RegisterDataBase()
	RegisterCache()
	RegisterModel()

}

// 注册缓存管道
func RegisterCache() {
	g := cfg.MustGlobal()
	if !g.Cache.Enable {
		cache.Init(&cache.NullCache{})
		return
	}
	logs.Info("正常初始化缓存配置.")
	cacheProvider := g.Cache.Provider
	if cacheProvider == "file" {
		cacheFilePath := g.Cache.FilePath
		if strings.HasPrefix(cacheFilePath, "./") {
			cacheFilePath = filepath.Join(cfg.WorkingDirectory, string(cacheFilePath[1:]))
		}
		fileCache := beegoCache.NewFileCache()

		fileConfig := make(map[string]string, 0)

		fileConfig["CachePath"] = cacheFilePath
		fileConfig["DirectoryLevel"] = g.Cache.FileDirLevel
		fileConfig["EmbedExpiry"] = g.Cache.FileExpiry
		fileConfig["FileSuffix"] = g.Cache.FileSuffix

		bc, err := json.Marshal(&fileConfig)
		if err != nil {
			logs.Error("初始化file缓存失败:", err)
			os.Exit(1)
		}

		_ = fileCache.StartAndGC(string(bc))

		cache.Init(fileCache)

	} else if cacheProvider == "memory" {
		cacheInterval := g.Cache.MemoryInterval
		memory := beegoCache.NewMemoryCache()
		beegoCache.DefaultEvery = cacheInterval
		cache.Init(memory)
	} else if cacheProvider == "redis" {
		var redisConfig struct {
			Conn     string `json:"conn"`
			Password string `json:"password"`
			DbNum    string `json:"dbNum"`
			Key      string `json:"key"`
		}
		redisConfig.DbNum = "0"
		redisConfig.Conn = g.Cache.RedisHost
		if key := g.Cache.RedisPrefix; key != "" {
			redisConfig.Key = key
		}
		if pwd := g.Cache.RedisPassword; pwd != "" {
			redisConfig.Password = pwd
		}
		if dbNum := g.Cache.RedisDB; dbNum > 0 {
			redisConfig.DbNum = strconv.Itoa(dbNum)
		}

		bc, err := json.Marshal(&redisConfig)
		if err != nil {
			logs.Error("初始化Redis缓存失败:", err)
			os.Exit(1)
		}
		redisCache, err := beegoCache.NewCache("redis", string(bc))

		if err != nil {
			logs.Error("初始化Redis缓存失败:", err)
			os.Exit(1)
		}

		cache.Init(redisCache)
	} else if cacheProvider == "memcache" {

		var memcacheConfig struct {
			Conn string `json:"conn"`
		}
		memcacheConfig.Conn = g.Cache.MemcacheHost

		bc, err := json.Marshal(&memcacheConfig)
		if err != nil {
			logs.Error("初始化 Memcache 缓存失败 ->", err)
			os.Exit(1)
		}
		memcache, err := beegoCache.NewCache("memcache", string(bc))

		if err != nil {
			logs.Error("初始化 Memcache 缓存失败 ->", err)
			os.Exit(1)
		}

		cache.Init(memcache)

	} else {
		cache.Init(&cache.NullCache{})
		logs.Warn("不支持的缓存管道,缓存将禁用 ->", cacheProvider)
		return
	}
	logs.Info("缓存初始化完成.")
}

// 自动加载配置文件.修改了监听端口号和数据库配置无法自动生效.
func RegisterAutoLoadConfig() {
	if cfg.AutoLoadDelay > 0 {

		watcher, err := fsnotify.NewWatcher()

		if err != nil {
			logs.Error("创建配置文件监控器失败 ->", err)
		}
		go func() {
			for {
				select {
				case ev, ok := <-watcher.Events:
					if !ok {
						return
					}
					//如果是修改了配置文件
					if ev.Op&fsnotify.Write == fsnotify.Write || ev.Op&fsnotify.Create == fsnotify.Create {
						if err := cfg.Reload(); err != nil {
							logs.Error("An error occurred ->", err)
							continue
						}
						RegisterCache()
						RegisterLogger("")
						logs.Info("配置文件已加载 ->", cfg.ConfigurationFile)
					} else if ev.Op&fsnotify.Rename == fsnotify.Rename {
						_ = watcher.Add(cfg.ConfigurationFile)
					}
					logs.Info(ev.String())
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					logs.Error("配置文件监控器错误 ->", err)

				}
			}
		}()

		err = watcher.Add(cfg.ConfigurationFile)

		if err != nil {
			logs.Error("监控配置文件失败 ->", err)
		}
	}
}

// 注册错误处理方法.
func RegisterError() {
	web.ErrorHandler("404", func(writer http.ResponseWriter, request *http.Request) {
		var buf bytes.Buffer

		data := make(map[string]any)
		data["ErrorCode"] = 404
		data["ErrorMessage"] = "页面未找到或已删除"

		if err := web.ExecuteViewPathTemplate(&buf, "errors/error.tpl", web.BConfig.WebConfig.ViewsPath, data); err == nil {
			_, _ = fmt.Fprint(writer, buf.String())
		} else {
			_, _ = fmt.Fprint(writer, data["ErrorMessage"])
		}
	})
	web.ErrorHandler("401", func(writer http.ResponseWriter, request *http.Request) {
		var buf bytes.Buffer

		data := make(map[string]any)
		data["ErrorCode"] = 401
		data["ErrorMessage"] = "请与 Web 服务器的管理员联系，以确认您是否具有访问所请求资源的权限。"

		if err := web.ExecuteViewPathTemplate(&buf, "errors/error.tpl", web.BConfig.WebConfig.ViewsPath, data); err == nil {
			_, _ = fmt.Fprint(writer, buf.String())
		} else {
			_, _ = fmt.Fprint(writer, data["ErrorMessage"])
		}
	})
}

func init() {

	if configPath, err := filepath.Abs(cfg.ConfigurationFile); err == nil {
		cfg.ConfigurationFile = configPath
	}

	// 验证码字符集：去掉易混淆字符，仅大写+数字
	gocaptcha.TextCharacters = []rune("23456789ABCDEFGHJKMNPQRSTUVWXYZ")

	if err := gocaptcha.SetFontPath(cfg.WorkingDir("web", "static", "fonts")); err != nil {
		log.Fatal("读取字体文件失败 ->", err)
	}

	if p, err := filepath.Abs(os.Args[0]); err == nil {
		cfg.WorkingDirectory = filepath.Dir(p)
	}
}
