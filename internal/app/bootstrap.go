package app

import (
	"encoding/gob"
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

	"git.itopcms.com/jackliu/doc/internal/cache"
	cfg "git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/model"
	"git.itopcms.com/jackliu/doc/pkg/filetil"
	beegoCache "github.com/beego/beego/v2/client/cache"
	_ "github.com/beego/beego/v2/client/cache/memcache"
	_ "github.com/beego/beego/v2/client/cache/redis"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/i18n"
	"github.com/fsnotify/fsnotify"
	"github.com/lifei6671/gocaptcha"
)

// RegisterDataBase 注册数据库
func RegisterDataBase() {
	logs.Info("正在初始化数据库配置.")
	dbadapter, _ := web.AppConfig.String("db_adapter")
	orm.DefaultTimeLoc = time.Local
	orm.DefaultRowsLimit = -1

	if strings.EqualFold(dbadapter, "mysql") {
		host, _ := web.AppConfig.String("db_host")
		database, _ := web.AppConfig.String("db_database")
		username, _ := web.AppConfig.String("db_username")
		password, _ := web.AppConfig.String("db_password")

		timezone, _ := web.AppConfig.String("timezone")
		location, err := time.LoadLocation(timezone)
		if err == nil {
			orm.DefaultTimeLoc = location
		} else {
			logs.Error("加载时区配置失败 timezone=%s err=%v", timezone, err)
		}

		port, _ := web.AppConfig.String("db_port")

		dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=%s", username, password, host, port, database, url.QueryEscape(timezone))

		if err := orm.RegisterDataBase("default", "mysql", dataSource); err != nil {
			logs.Error("注册默认数据库失败->", err)
			os.Exit(1)
		}

	} else if strings.EqualFold(dbadapter, "sqlite3") {

		database, _ := web.AppConfig.String("db_database")
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

// RegisterLogger 注册日志
func RegisterLogger(log string) {

	logs.SetLogFuncCall(true)
	_ = logs.SetLogger("console")
	logs.EnableFuncCallDepth(true)

	if web.AppConfig.DefaultBool("log_is_async", true) {
		logs.Async(1e3)
	}
	if log == "" {
		logPath, err := filepath.Abs(web.AppConfig.DefaultString("log_path", cfg.WorkingDir("runtime", "logs")))
		if err == nil {
			log = logPath
		} else {
			log = cfg.WorkingDir("runtime", "logs")
		}
	}

	logPath := filepath.Join(log, "log.log")

	if _, err := os.Stat(log); os.IsNotExist(err) {
		_ = os.MkdirAll(log, 0755)
	}

	logConfig := make(map[string]any, 1)

	logConfig["filename"] = logPath
	logConfig["perm"] = "0755"
	logConfig["rotate"] = true

	if maxLines := web.AppConfig.DefaultInt("log_maxlines", 1000000); maxLines > 0 {
		logConfig["maxLines"] = maxLines
	}
	if maxSize := web.AppConfig.DefaultInt("log_maxsize", 1<<28); maxSize > 0 {
		logConfig["maxsize"] = maxSize
	}
	if !web.AppConfig.DefaultBool("log_daily", true) {
		logConfig["daily"] = false
	}
	if maxDays := web.AppConfig.DefaultInt("log_maxdays", 7); maxDays > 0 {
		logConfig["maxdays"] = maxDays
	}
	if level := web.AppConfig.DefaultString("log_level", "Trace"); level != "" {
		switch level {
		case "Emergency":
			logConfig["level"] = logs.LevelEmergency
		case "Alert":
			logConfig["level"] = logs.LevelAlert
		case "Critical":
			logConfig["level"] = logs.LevelCritical
		case "Error":
			logConfig["level"] = logs.LevelError
		case "Warning":
			logConfig["level"] = logs.LevelWarning
		case "Notice":
			logConfig["level"] = logs.LevelNotice
		case "Informational":
			logConfig["level"] = logs.LevelInformational
		case "Debug":
			logConfig["level"] = logs.LevelDebug
		}
	}
	b, err := json.Marshal(logConfig)
	if err != nil {
		logs.Error("初始化文件日志时出错 ->", err)
		_ = logs.SetLogger("file", `{"filename":"`+logPath+`"}`)
	} else {
		_ = logs.SetLogger(logs.AdapterFile, string(b))
	}

	logs.SetLogFuncCall(true)
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
		cdn := web.AppConfig.DefaultString("cdn", "")
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			return p
		}
		//如果没有设置cdn，则使用baseURL拼接
		if cdn == "" {
			baseUrl := web.AppConfig.DefaultString("baseurl", "")

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
		msgFile := cfg.WorkingDir("configs", "lang", lang+".ini")
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
	logs.Info("工作目录 ->", cfg.WorkingDirectory)

	if configFile != "" {
		cfg.ConfigurationFile = configFile
	} else {
		cfg.ConfigurationFile = cfg.WorkingDir("configs", "app.conf")
		exampleConfig := cfg.WorkingDir("configs", "app.conf.example")
		if !filetil.FileExists(cfg.ConfigurationFile) && filetil.FileExists(exampleConfig) {
			_ = filetil.CopyFile(cfg.ConfigurationFile, exampleConfig)
		}
	}
	if err := gocaptcha.SetFontPath(cfg.WorkingDir("web", "static", "fonts")); err != nil {
		log.Fatal("读取字体文件时出错 -> ", err)
	}

	if err := web.LoadAppConfig("ini", cfg.ConfigurationFile); err != nil {
		log.Fatal("An error occurred:", err)
	}

	web.BConfig.MaxUploadSize = cfg.GetUploadMaxSize()
	web.BConfig.MaxMemory = cfg.GetUploadMaxMemory()
	logs.Info("上传限制 -> MaxUploadSize=%d MaxMemory=%d business_upload_file_size=%d",
		web.BConfig.MaxUploadSize, web.BConfig.MaxMemory, cfg.GetUploadFileSize())

	if logFile != "" {
		cfg.LogFile = logFile
	} else {
		logPath, err := filepath.Abs(web.AppConfig.DefaultString("log_path", cfg.WorkingDir("runtime", "logs")))
		if err == nil {
			cfg.LogFile = logPath
		} else {
			cfg.LogFile = cfg.WorkingDir("runtime", "logs")
		}
	}

	cfg.AutoLoadDelay = web.AppConfig.DefaultInt("config_auto_delay", 0)
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
	RegisterLogger(cfg.LogFile)

}

// 注册缓存管道
func RegisterCache() {
	isOpenCache := web.AppConfig.DefaultBool("cache", false)
	if !isOpenCache {
		cache.Init(&cache.NullCache{})
		return
	}
	logs.Info("正常初始化缓存配置.")
	cacheProvider, _ := web.AppConfig.String("cache_provider")
	if cacheProvider == "file" {
		cacheFilePath := web.AppConfig.DefaultString("cache_file_path", "./runtime/cache/")
		if strings.HasPrefix(cacheFilePath, "./") {
			cacheFilePath = filepath.Join(cfg.WorkingDirectory, string(cacheFilePath[1:]))
		}
		fileCache := beegoCache.NewFileCache()

		fileConfig := make(map[string]string, 0)

		fileConfig["CachePath"] = cacheFilePath
		fileConfig["DirectoryLevel"] = web.AppConfig.DefaultString("cache_file_dir_level", "2")
		fileConfig["EmbedExpiry"] = web.AppConfig.DefaultString("cache_file_expiry", "120")
		fileConfig["FileSuffix"] = web.AppConfig.DefaultString("cache_file_suffix", ".bin")

		bc, err := json.Marshal(&fileConfig)
		if err != nil {
			logs.Error("初始化file缓存失败:", err)
			os.Exit(1)
		}

		_ = fileCache.StartAndGC(string(bc))

		cache.Init(fileCache)

	} else if cacheProvider == "memory" {
		cacheInterval := web.AppConfig.DefaultInt("cache_memory_interval", 60)
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
		redisConfig.Conn = web.AppConfig.DefaultString("cache_redis_host", "")
		if key := web.AppConfig.DefaultString("cache_redis_prefix", ""); key != "" {
			redisConfig.Key = key
		}
		if pwd := web.AppConfig.DefaultString("cache_redis_password", ""); pwd != "" {
			redisConfig.Password = pwd
		}
		if dbNum := web.AppConfig.DefaultInt("cache_redis_db", 0); dbNum > 0 {
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
		memcacheConfig.Conn = web.AppConfig.DefaultString("cache_memcache_host", "")

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
						if err := web.LoadAppConfig("ini", cfg.ConfigurationFile); err != nil {
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
	gob.Register(model.Member{})

	if p, err := filepath.Abs(os.Args[0]); err == nil {
		cfg.WorkingDirectory = filepath.Dir(p)
	}
}
