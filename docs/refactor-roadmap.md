# Doc 项目整体优化与迭代路线图

> 本文是 `doc` 项目**整体重构与优化**的总纲文档，聚焦四大目标：
> 1. 接入 MCP，方便 AI 使用文档
> 2. 前后端目录结构规范化
> 3. 配置模块优化（Go 代码与配置文件解耦、分组）
> 4. 缓存 / 模型等基础组件升级
>
> 同时汇总"顺带发现的技术债"与"前端现代化"两条支线，给出四轮可独立上线的迭代计划。
>
> **相关文档：**
> - [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) — 前后端目录拆分执行清单（同仓内）
> - [router-split-migration-plan.md](./router-split-migration-plan.md) — 路由按职责拆分与 `/api` 前缀治理
> - [routers-reference.md](./routers-reference.md) — 现有路由分类参考
> - [upstream-mindoc-checklist.md](./upstream-mindoc-checklist.md) — 上游 MinDoc 提交跟进清单
>
> **文档生成依据：** 当前仓库代码基线（2026-07），Go 1.25 + Beego v2。

---

## 一、现状速览

### 1.1 技术栈

| 层 | 组件 | 状态 |
|---|---|---|
| 语言 | Go 1.25.0 | 新 |
| Web 框架 | `beego/v2 v2.3.10` | 社区更新缓慢 |
| ORM | `beego/orm` | 与框架同命运 |
| i18n | `beego/i18n v0.0.0-20161101132742-e9308947f407` | **11 年前**的老包 |
| 缓存 | `beego/cache` + 自封 `cache/`（gob 序列化） | 需重构 |
| 模板 | `html/template`（beego 内置） | 稳定 |
| CLI 服务 | `kardianos/service` | 稳定 |
| 数据库驱动 | `go-sql-driver/mysql` + `mattn/go-sqlite3` | 稳定，无 PostgreSQL |
| 前端 | Bootstrap 3.2 + jQuery + editor.md + wangEditor + Vue 2 | 全部偏旧 |

### 1.2 结构问题速览

| 问题 | 现状 | 位置 |
|---|---|---|
| **配置与代码混目录** | `conf/` 同时放 `enumerate.go`/`mail.go`（Go 源码）和 `app.conf`/`app.conf.example`/`lang/`（配置） | `conf/` |
| **配置文件无分组** | `app.conf` 253 行平铺，session/DB/upload/mail/export/ldap/cdn/cache/log/dingtalk/i18n 全混一起 | `conf/app.conf` |
| **路由未分组** | `web.Router()` 148 行一把梭 | `routers/router.go` |
| **Controller 巨型文件** | Document 37KB / Book 27KB / Manager 30KB / Blog 20KB | `controllers/` |
| **Model 巨型文件** | Book 34KB / BookResult 23KB / Member 17KB | `models/` |
| **Base.go 空基类** | 全文只有 `type Model struct {}` | `models/Base.go` |
| **无 MCP** | 无 AI 接入能力 | — |
| **无 Repository/Service 层** | Controller 直接调 Model，事务/缓存/测试都难做 | — |
| **前端资源平铺** | 24 个第三方库 + 自写 js/css 全平铺在 `static/` 根下 | `static/` |

### 1.3 关键代码债

| 债务 | 影响 | 涉及文件数 |
|---|---|---|
| `ioutil.ReadFile` / `ioutil.WriteFile` 等 deprecated API | Go 1.16+ 已废弃 | 12 |
| `interface{}` 遍布（未用 `any`） | Go 1.18+ 应改 | 20+ |
| `md_` 表前缀硬编码（未走 `GetDatabasePrefix()`） | 换前缀会崩 | 多处 raw SQL |
| `gob.Register` 硬编码类型 | 加新缓存类型要改初始化 | `commands/command.go:113-115` |
| `cache.Get` 用 `context.TODO()` 全局共享 | 无法传超时/取消/trace | `cache/cache.go:16` |
| `BaseController.Prepare` 每次请求全表读 options | DB 压力 | `controllers/BaseController.go:68` |
| `smtp_host="${...}""` 结尾多一个引号 | 配置解析 bug | `conf/app.conf:106,110` |
| `orm.DefaultRowsLimit = -1` 全局关分页 | 潜在性能坑 | `commands/command.go:39` |
| `main.go` 手写 `os.Args[1] == "service"` | 无子命令框架 | `main.go:21-29` |

---

## 二、四大目标详细方案

### 2.1 目标一：MCP Server（AI 接入）

> 与 [upstream-mindoc-checklist.md §4.2](./upstream-mindoc-checklist.md) 呼应，落地本地最小方案。

#### 现状

- 无 MCP 相关代码，`go.mod` 无 `mark3labs/mcp-go`。
- 搜索只有 `models/DocumentSearchResult.go` 的 SQL `LIKE`，无倒排索引/向量搜索。
- 权限走 Session Cookie，MCP 无天然身份接入通道。

#### 方案（分两期）

| 期 | 内容 | 工作量 |
|---|---|---|
| **MVP** | 新增 `mcp/` 包 + `commands/mcp.go` 子命令（`doc mcp`）。基于 `github.com/mark3labs/mcp-go` **stdio 模式**，暴露 4 个工具：`search_document(query, book?, limit)`、`get_document(id 或 book+identify)`、`list_books()`、`list_document_tree(book_key)`。内部直接调 `models` 层，权限按"公开项目 + 有 Token 的项目"过滤 | 1~1.5 天 |
| **Streamable HTTP** | 增加 `mcp/http.go`，用 beego 挂 SSE / Streamable HTTP，配 Bearer Token（复用 `MemberToken` 表）；`conf/app.conf` 加 `mcp_enable` / `mcp_listen` / `mcp_token_required` | 1~2 天 |

#### 关键设计点

1. **单二进制多入口**：`doc`（web 服务）、`doc mcp`（stdio）、`doc mcp --http`（Streamable HTTP）。共享 `commands.ResolveCommand` 加载 conf/DB/cache/models，只跳过 `web.Run()`。
2. **工具输出统一走 markdown**（不是 HTML），AI 侧最好用。
3. **`mcp/tools.go` 抽象层**：把"权限过滤 + 摘要生成"集中处理，避免下沉到 controller 的 view 逻辑。
4. **为将来上倒排索引铺路**：`search_document` 内部走 `searchProvider` 接口，`sql_like` 是默认实现，未来切 `fulltext`/`bleve`/`qdrant` 只改一处。

#### 交付物

- `mcp/server.go`、`mcp/tools.go`（≈200 行）
- `commands/mcp.go`（新增子命令）
- `go.mod` 加 `github.com/mark3labs/mcp-go`
- `conf/app.conf.example` 增加 mcp 段
- `docs/mcp-integration.md`（Claude Desktop / Cursor 接入示例）

---

### 2.2 目标二：前后端目录结构调整（规范化）

> 与 [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) 深度配合。已有文档采用"轻量方案"（`server/` + `web/` + `deploy/`，保留 `conf/` 在根），本目标可选**激进方案**作为**长期目标**参考。

#### 现状问题

```
d:\jcwork\doc\
├─ controllers/    ← 15 个巨型文件
├─ models/         ← 30+ 文件，Model+Result+Service 混一层
├─ routers/        ← 148 行全平铺
├─ views/          ← 按业务分了子目录，还行
├─ static/         ← 24 个第三方库 + 自写 css/js 全平铺
├─ cache/          ← 只有 cache.go + cache_null.go
├─ commands/       ← 混着 CLI 注册 + DB/缓存/日志初始化（562 行）
├─ conf/           ← ⚠️ Go 源码 + 配置文件混在一起
```

#### 短期目标（对齐已有 frontend-backend-split-migration-plan）

采用 `server/` + `web/` + `deploy/` 三段式，**`conf/` 保留在根**（因为它既是配置目录又是 Go 包）。执行细节见该文档。

#### 长期目标（可选，激进方案）

```
doc/
├─ cmd/
│  └─ doc/main.go                # 只做 flag 解析 + 子命令派发（cobra）
├─ internal/                     # Go 私有包，防止外部误引
│  ├─ app/                       # 装配层（原 commands 的初始化部分）
│  ├─ controller/                # 按域拆子目录：document/ book/ manager/ ...
│  ├─ service/                   # 【新增】业务逻辑下沉
│  ├─ model/                     # 按域拆子目录：book/ document/ member/ ...
│  ├─ dto/                       # 【新增】原 *Result.go 挪进来
│  ├─ repository/                # 【新增】ORM 查询集中
│  ├─ middleware/
│  ├─ router/                    # 按域拆：api.go / manager.go / document.go / blog.go / router.go(汇总)
│  ├─ cache/                     # 升级（见 2.4）
│  ├─ config/                    # 【新增】Go 侧配置结构体
│  └─ mcp/                       # 【新增】MCP server
├─ pkg/                          # 可复用工具（原 utils/*）
├─ configs/                      # 【新增】只放配置文件（不放 .go）
│  ├─ app.conf / app.conf.example
│  ├─ conf.d/                    # 分组配置
│  └─ lang/
├─ web/                          # 前端资源
│  ├─ static/vendor/  css/  js/  images/  fonts/  editors/
│  └─ views/
├─ scripts/
├─ deployments/                  # 【新增】Dockerfile / docker-compose 等
├─ docs/
├─ runtime/  uploads/
```

#### 关键改动点

1. **`cmd/` + `internal/`** 是 Go 生态事实标准（`golang-standards/project-layout`）。
2. **Controller 拆分**：`DocumentController.go` 37KB 按方法组拆成 `controller/document/read.go`、`edit.go`、`history.go`、`export.go`。
3. **`routers` 拆分**：见 [router-split-migration-plan.md](./router-split-migration-plan.md)。
4. **模板/静态资源路径**：`web.BConfig.WebConfig.ViewsPath`、`StaticDir` 在 `commands/command.go:345-347` 硬编码，随目录变动同步改。

#### 迁移风险

- 模板路径、静态资源路径、上传目录、日志目录、session 目录都在 `conf/app.conf` 里配置，需要一起改。
- 建议**分两次 PR**：① 只搬目录 + 改 import；② 拆大文件。

---

### 2.3 目标三：配置模块优化

#### 现状问题

1. **代码和配置文件混一层**：`conf/enumerate.go`（401 行）+ `mail.go`（Go）+ `app.conf`（253 行）+ `app.conf.example` + `lang/` 全在 `conf/`。
2. **`app.conf` 未做分组**：253 行大平铺，只靠注释分块。
3. **配置访问方式割裂**：全部通过 `web.AppConfig.DefaultString("xxx", ...)`，散落在 30+ 文件里，**没有强类型 struct**。
4. **`enumerate.go` 三合一**：常量 + 配置读取器（30+ Getter）+ URL 工具函数挤在同一文件。
5. **`.example` 与真实 `.conf` 靠人肉同步**。
6. **多份 `AppConfig` 调用**忽略了 err，且无缓存。
7. **bug**：`smtp_host="${MINDOC_SMTP_HOST||smtp.163.com}""` 结尾多引号（`conf/app.conf:106,110`）。

#### 分四步落地

**Step 1：目录分离**（配合目标二）

```
configs/
├─ app.conf                      # 主配置
├─ app.conf.example
├─ app.conf.dev                  # 【新增】开发环境覆盖，git ignored
├─ app.conf.prod.example         # 【新增】生产环境示例
└─ lang/
   ├─ zh-cn.ini
   └─ en-us.ini
```

**Step 2：拆分 `app.conf` 为多个 include**（启动时按字典序 merge）

```
configs/
├─ app.conf                      # 只保留 appname/runmode/httpport/baseurl 等根配置
└─ conf.d/
   ├─ 10-session.conf
   ├─ 20-database.conf           # MySQL / SQLite / (未来 PostgreSQL)
   ├─ 30-cache.conf
   ├─ 40-mail.conf
   ├─ 50-upload.conf
   ├─ 60-log.conf
   ├─ 70-ldap.conf
   ├─ 71-dingtalk.conf
   ├─ 72-oauth.conf              # 【新增】微信/企微/Google（如需）
   ├─ 80-export.conf             # PDF/EPUB/MOBI
   ├─ 90-cdn.conf
   ├─ 91-i18n.conf
   └─ 99-mcp.conf                # 【新增】MCP 配置
```

**Step 3：强类型 config struct**（推荐）

新增 `internal/config/config.go`：

```go
type Config struct {
    App      AppConfig
    HTTP     HTTPConfig
    Session  SessionConfig
    Database DatabaseConfig
    Cache    CacheConfig
    Log      LogConfig
    Upload   UploadConfig
    Mail     MailConfig
    LDAP     LDAPConfig
    Export   ExportConfig
    CDN      CDNConfig
    OAuth    OAuthConfig
    MCP      MCPConfig            // 新增
    I18n     I18nConfig
}

var Global *Config

func Load(path string) (*Config, error) { ... }  // 一次读完，强类型
func Reload() error { ... }                       // 配合 fsnotify
```

调用方从 `web.AppConfig.DefaultString("cache_provider", "file")` 变成 `config.Global.Cache.Provider`。**IDE 补全 + 编译期检查**。

**Step 4：环境变量与敏感字段**

- 保留现有 `${MINDOC_XXX||default}` 语法。
- 兼容 `DOC_XXX` 前缀（新品牌）。
- 支持 `.env` 文件（用 `github.com/joho/godotenv`）。
- 敏感字段（`db_password`/`smtp_password`/`ldap_password`/`dingtalk_app_secret`）支持 `_FILE` 后缀（读文件内容），配合 Docker secrets。

**Step 5：修 bug**

- `conf/app.conf:106` `smtp_host` 结尾多引号
- `conf/app.conf:110` `smtp_port` 结尾多引号
- `enumerate.go` 30+ 个 `web.AppConfig.DefaultString` 每次都 map 查找，改为 `Load()` 时缓存

---

### 2.4 目标四：缓存 / 模型组件升级

#### 2.4.1 缓存层

**现状问题**（`cache/cache.go` 全文 95 行）

1. **全局单例 + 包级变量**，无法在测试里 mock。
2. `nilctx = context.TODO()` 全局共享 context，调用方无法传超时/取消/trace。
3. **手工 gob 序列化**：加类型要改 `commands/command.go:113-115` 的 `gob.Register`。
4. `Get(key, e interface{})` 用 `errors.New("get cache error:" + err.Error())` 丢失原始 err，无法 `errors.Is`。
5. `NullCache` 直接实现 Beego cache 接口，跟不上 Beego v2 未来接口变动。
6. **无分层缓存**（memory + redis 二级）、**无缓存击穿保护**（singleflight）、**无预热**、**无 metrics**。
7. Beego 自带 redis cache 底层是 `redigo`，社区更活跃的是 `redis/go-redis/v9`（**已经在 indirect 依赖里**）。

**方案 A：轻量改造**（低成本，1~1.5 天，推荐先做）

```go
// internal/cache/cache.go
type Cache interface {
    Get(ctx context.Context, key string, dst any) error
    Set(ctx context.Context, key string, val any, ttl time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    IsExist(ctx context.Context, key string) (bool, error)
    Incr(ctx context.Context, key string) (int64, error)
    Clear(ctx context.Context) error
}

// 实现：MemoryCache（本地 LRU）、RedisCache（go-redis v9）、FileCache（保留）、NullCache
```

要点：
- 用 `github.com/vmihailenco/msgpack/v5` 或 `encoding/json` 替代 gob，去掉 `gob.Register` 硬编码
- 加 `singleflight` 防击穿：`cache.GetOrLoad(ctx, key, ttl, func() (any, error) { ... })`
- 支持缓存 tag（`cache.DeleteByTag("book:123")`）

**方案 B：直接换成熟组件**（2~3 天）

- 引入 [`github.com/eko/gocache/v3`](https://github.com/eko/gocache)，支持 Chain (memory + redis)、Loader、Metrics 一站式
- 或用 `github.com/allegro/bigcache` + `go-redis` 手工组合

**Redis 客户端升级**：`beego/cache/redis` → `redis/go-redis/v9`（indirect 已存在，直接改）。

#### 2.4.2 Model 层

**现状问题**

1. **`beego/orm` 维护缓慢**，beego v2 也基本停更。
2. 大文件：`BookModel.go` 34KB、`BookResult.go` 23KB、`Member.go` 17KB。
3. **无 Repository 层**：controller 直接调 `models.NewDocument().Find(id)`，事务/缓存/测试都难做。
4. **手写 SQL 硬编码表前缀**：`models/Member.go:77` 里 `o.Raw("select * from md_members where ...")` 写死了 `md_`。
5. `commands/migrate/` 是自研简易迁移器，功能有限。
6. **`ioutil.ReadFile` / `ioutil.WriteFile` deprecated API** 在 12 个文件里还在用。
7. **`interface{}` 遍布**（Go 1.18+ 应用 `any`）。
8. `models/Base.go` 全文只有 `type Model struct {}` — 未来空基类，可以直接删或做真正的通用能力。

**建议路线**

| 升级项 | 方案 | 工作量 |
|---|---|---|
| ORM | ① 稳妥：留 beego/orm，只封 Repository 层<br>② 激进：换 `gorm.io/gorm` 或 `ent`（类型安全，事务/关联更好）<br>③ 折中：换 `sqlc`（写 SQL 生成 Go，性能最好） | ①1 周 / ②2~3 周 / ③1.5 周 |
| Migrations | 换 [`golang-migrate`](https://github.com/golang-migrate/migrate) 或 [`goose`](https://github.com/pressly/goose)，SQL 文件放 `migrations/` | 2~3 天 |
| deprecated API | 全局 `ioutil.ReadFile` → `os.ReadFile`；`ioutil.WriteFile` → `os.WriteFile`；`interface{}` → `any` | 半天 |
| 大 Model 拆分 | `BookModel.go` (34KB) 按功能拆：`book/model.go`、`book/query.go`、`book/tree.go`、`book/permission.go` | 1 天 |
| DTO/Result 分离 | `BookResult.go` / `MemberResult.go` / `DocumentSearchResult.go` 挪到 `internal/dto/` | 0.5 天 |
| 表前缀硬编码修复 | 全局搜 `md_` 定位 raw SQL，改用 `conf.GetDatabasePrefix()` | 0.5 天 |

---

## 三、支线：其他技术债

| 主题 | 现状问题 | 建议 | 优先级 |
|---|---|---|---|
| **日志** | `commands/RegisterLogger` 用 `beego/logs`，无结构化，`logs.Error("xx", err)` 拼字符串 | 换 `log/slog`（Go 1.21+ 标准库）或 `zap`，与 OpenTelemetry 对接 | 中 |
| **HTTP 框架** | Beego v2 社区活跃度不如 echo/gin/fiber | 短期继续 beego；长期规划评估。**切走成本很大**（模板/session/orm 全绑定 beego），不建议现在动 | 低 |
| **错误处理** | `errors.New` + `fmt.Errorf` 混用，无错误分类；`if err != nil { c.JsonResult(6001, "系统内部错误") }` 遍地 | 引入 `pkg/errors` 或 `cockroachdb/errors`；定义 `BizError { Code; Msg }`；controller 统一 `WriteError(err)` | 高 |
| **中间件** | `middleware/filter.go` 和 `routers/filter.go` 两处分裂 | 合并到 `middleware/`：`auth.go` / `logger.go` / `recover.go` / `csrf.go` / `ratelimit.go` | 中 |
| **限流/防刷** | 无 | `golang.org/x/time/rate` 或 `uber-go/ratelimit`，API 接口层做 | 中 |
| **API 文档** | 无 OpenAPI/Swagger | `swaggo/swag` 注释生成，或迁到 `huma` | 中 |
| **验证码** | `lifei6671/gocaptcha` 老库，字体路径硬编码 | 换 `mojocn/base64Captcha`（前端直接 base64） | 低 |
| **测试** | 全项目**未见** `_test.go` | 至少给 `pkg/*` 和 `internal/service/*` 补测试，用 `testify` | 高 |
| **i18n** | `beego/i18n v0.0.0-20161101132742-e9308947f407` **11 年前**老包 | 换 `nicksnyder/go-i18n/v2`（toml/json，支持复数） | 中 |
| **gopool** | 自研 `utils/gopool/`，简单包装 | 换 `panjf2000/ants`（业界事实标准） | 低 |
| **requests** | 自研 `utils/requests/` | 换 `go-resty/resty` | 低 |
| **`main.go` 优化** | 手写 `os.Args[1] == "service"` 判断 | 换 `spf13/cobra` + `spf13/viper`（可一起替代配置解析） | 高 |
| **BaseController.Prepare** | 每次请求读 `models.NewOption().All()` 全表 | 加 5 分钟内存缓存或启动时装载，变更时清缓存 | 高 |
| **session** | `sessionprovider=file`（默认） | 生产建议 `redis`；serializer 用 msgpack | 中 |
| **CORS / 安全头** | 未见 CORS、CSP、HSTS 配置 | `middleware/secure.go` 统一处理 | 中 |
| **Docker** | Dockerfile 6KB，Ubuntu focal 基础镜像 | 多阶段 + `distroless` 或 `alpine`，镜像大小从几百 MB 降到 40MB 内 | 中 |

---

## 四、支线：前端与静态资源

**现状**（`static/`）

- Bootstrap 3.2（2014 年，官方已 EOL）
- jQuery 全家桶（jquery / jstree / layer / nprogress / select2 / cropper / webuploader / respond.js / html5shiv）
- editor.md v1.5.0（[upstream-mindoc-checklist.md §2.1](./upstream-mindoc-checklist.md) 指出应升到 v1.7.17）
- Vue.js 2
- katex 缺 `.min.` 文件，reader 页 CSS 404

**分阶段方案**

| 阶段 | 内容 | 工作量 |
|---|---|---|
| **P0** | 修 katex 404、editor.md 升 v1.7.17、mermaid 升 10.x（详见 upstream-mindoc-checklist.md §2.1） | 0.5 天 |
| **P1** | 静态资源加版本号（现有 `cdnjs "..." "version"` 机制铺开）；删 `respond.js` / `html5shiv`（不再支持 IE8） | 1~2 天 |
| **P2** | 引入前端构建工具（Vite），vendor 集中管理；抽离 `views/*.tpl` 里的内联 JS | 1~2 周 |
| **P3** | 逐步替换：Bootstrap 3 → Bootstrap 5 或 Tailwind；jQuery 组件 → Vue 3 组件（增量迁移，不用一次性 SPA 化） | 3~4 周 |
| **P4** | 后端只做 API + 模板；`web-ui/` 用 Vue 3 + TypeScript + Vite 做完整 SPA；老模板路由保留兼容期 | 长期 |

---

## 五、实施顺序（四轮迭代）

### 🥇 Round 1：低风险 · 高性价比（1 周）

- [ ] **配置 Step 1+5**：`configs/` 目录独立 + 修 `smtp_host` 双引号 bug
- [ ] **`ioutil` 全局替换 `os.ReadFile` / `os.WriteFile`**
- [ ] **`interface{}` → `any`**（Go 1.18+）
- [ ] **`main.go` 用 `cobra`**
- [ ] **缓存方案 A**：`cache.Cache` 抽接口 + `NullCache/MemoryCache/RedisCache/FileCache` 独立文件 + 加 `context` 传递
- [ ] **前端 P0**：修 katex 404 + editor.md 升级（引用 upstream-mindoc-checklist.md §2.1）

**风险：** 低。全部是内部重构，对用户零感知。

### 🥈 Round 2：目录结构调整（1~2 周）

- [ ] **对齐 frontend-backend-split-migration-plan.md**：搬迁到 `server/` + `web/` + `deploy/`
- [ ] **配置 Step 2+3+4**：`conf.d/` 分组 + 强类型 `config.Config` struct + `.env` 支持
- [ ] **`routers` 按域拆分**（对齐 router-split-migration-plan.md）
- [ ] **`BaseController.Prepare` 加 options 缓存**

**风险：** 中。所有 import 路径、模板路径、脚本路径要同步改。建议开专门的 `refactor/layout` 分支。

### 🥉 Round 3：MCP + 搜索基础（1~2 周）

- [ ] **MCP MVP**：stdio + 4 个基础工具（`search_document` / `get_document` / `list_books` / `list_document_tree`）
- [ ] **搜索最小方案**（对齐 upstream-mindoc-checklist.md §1.1）：MySQL FULLTEXT / SQLite FTS5 + 标题加权
- [ ] **MCP Streamable HTTP**：Bearer Token（复用 `MemberToken`）
- [ ] **`docs/mcp-integration.md`**：Claude Desktop / Cursor 接入示例

**风险：** 中。MCP 是新增功能，不影响存量。搜索改动限于 `models/DocumentSearchResult.go` + 建索引。

### 🏅 Round 4：模型 / 日志 / 前端现代化（3~4 周，按需推进）

- [ ] **模型层**：`BookModel.go` (34KB) 拆解 + Repository 抽象 + `md_` 硬编码修复
- [ ] **日志换 `slog`** + 结构化字段
- [ ] **`beego/i18n` 换 `nicksnyder/go-i18n/v2`**
- [ ] **前端 P1~P2**：Vite 构建，vendor 集中化

**风险：** 较高，但可拆多个小 PR。**ORM 迁移建议单独立项**，别混进来。

---

## 六、关键风险清单

| # | 风险 | 触发场景 | 对策 |
|---|---|---|---|
| 1 | `beego/orm` 的 `gob` + `md_` 前缀假设深入很多代码 | `models/Member.go:77` 的 raw SQL 写死了 `md_members`；迁移 ORM 或改前缀会崩 | 迁移前全局搜 `md_` 定位，统一走 `GetDatabasePrefix()` |
| 2 | Session 里存了 `models.Member` 结构体 | `BaseController.go:49`；改 Member 字段/加字段要评估旧 session 反序列化兼容性 | `SetMember` 里塞 version 字段，异常时降级重登 |
| 3 | `init()` 副作用重 | `commands/command.go` + `enumerate.go` 都有 `init()`；重构时可能踩到"包变量在 `init` 里读了还没初始化的配置" | 重构时理清初始化顺序，最好显式 `Init(cfg *Config)` |
| 4 | `config_auto_delay` 热加载 | `commands/command.go:468-512`；重构配置时要保留或明确宣布废弃 | Round 2 完成时明确表态 |
| 5 | `orm.DefaultRowsLimit = -1` 全局关掉了默认分页 | `commands/command.go:39`；重构 model 时不要依赖默认 | 显式在每个 Query 里写 `Limit()` |
| 6 | Beego `web.BConfig.WebConfig.ViewsPath` 硬编码 | `commands/command.go:345-347`；目录变动要同步 | Round 2 迁移时统一改 |
| 7 | 前端 vendor 无版本管理 | 升级/回退困难 | Round 4 引入 Vite 时用 npm/pnpm 管起来 |

---

## 七、迭代进度追踪

> 完成一项就把 `[ ]` 改成 `[x]`，附上 commit hash 或 PR 链接。

### Round 1
- [ ] `configs/` 目录独立
- [ ] `smtp_host` / `smtp_port` 双引号 bug 修复
- [ ] `ioutil` → `os`（12 文件）
- [ ] `interface{}` → `any`
- [ ] `main.go` 引入 `cobra`
- [ ] `cache.Cache` 接口抽象
- [ ] KaTeX / editor.md 前端修复

### Round 2
- [ ] 目录搬迁到 `server/` + `web/` + `deploy/`
- [ ] `conf.d/` 分组配置
- [ ] 强类型 `config.Config` struct
- [ ] `.env` 支持
- [ ] `routers` 按域拆分
- [ ] `BaseController.Prepare` options 缓存

### Round 3
- [ ] MCP stdio MVP
- [ ] 搜索 FULLTEXT/FTS5 + 标题加权
- [ ] MCP Streamable HTTP + Bearer Token
- [ ] `docs/mcp-integration.md`

### Round 4
- [ ] `BookModel.go` 拆分
- [ ] Repository 抽象
- [ ] `md_` 硬编码修复
- [ ] `slog` 日志
- [ ] `nicksnyder/go-i18n/v2`
- [ ] Vite 前端构建

---

## 八、决策记录（Decision Log）

| 日期 | 决策项 | 决定 | 原因 |
|---|---|---|---|
| 2026-07-23 | 是否本轮切换 HTTP 框架？ | 否，继续 beego v2 | 模板/session/orm 全绑定 beego，切走成本远超收益 |
| 2026-07-23 | ORM 是否本轮迁移 gorm/ent？ | 否，先封 Repository 层 | 减小 blast radius，Round 4 再评估 |
| 2026-07-23 | 配置文件分组是否引入 viper？ | 保留 beego `LoadAppConfig` + include 合并 | 减少依赖，`${ENV||default}` 语法已够用 |
| 2026-07-23 | MCP 是否本轮做 HTTP 模式？ | 分两步（先 stdio，再 HTTP） | stdio 无鉴权顾虑，先解决 AI 接入这一刚需 |
| 2026-07-23 | 目录结构是否走激进的 `cmd/` + `internal/` 方案？ | 短期走 `server/` + `web/` + `deploy/`；`internal/` 作为长期目标 | 已有 frontend-backend-split-migration-plan.md 详细方案，避免朝令夕改 |

---

## 九、附录

### 9.1 相关外部资料

- [Go Project Layout Standards](https://github.com/golang-standards/project-layout)
- [Model Context Protocol 规范](https://modelcontextprotocol.io/)
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- [redis/go-redis](https://github.com/redis/go-redis)
- [nicksnyder/go-i18n](https://github.com/nicksnyder/go-i18n)
- [spf13/cobra](https://github.com/spf13/cobra) + [viper](https://github.com/spf13/viper)
- [uber-go/zap](https://github.com/uber-go/zap) / [log/slog](https://pkg.go.dev/log/slog)

### 9.2 本文档与其他文档的关系

```text
refactor-roadmap.md（本文，总纲）
├─► frontend-backend-split-migration-plan.md   （目标二 · 短期执行细节）
├─► router-split-migration-plan.md              （目标二 · 路由拆分执行细节）
├─► routers-reference.md                        （现有路由分类参考）
└─► upstream-mindoc-checklist.md                （功能特性对齐 · MCP/搜索/前端等）
```
