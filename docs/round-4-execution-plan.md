# Round 4 · 执行文档（模型 / 日志 / i18n / 前端现代化）

> 本文是 [refactor-roadmap.md §五 Round 4](./refactor-roadmap.md#🏅-round-4模型--日志--前端现代化3~4-周按需推进) 的**可执行分解**。
> 目标：把 Round 1~3 打好的目录/配置/MCP 基础之上，做**质量类**改造 — 大 Model 拆分、Repository 抽象、`md_` 硬编码修复、日志换 zap、i18n 换 go-i18n/v2、前端 Vite 构建。
> **按需推进：** 本轮所有子项**相互独立**，可以按团队精力挑做，不做也不阻塞其他轮次。

---

## 一、范围与不做清单

### 本轮做

| 序号 | 任务 | 工作量 | 阻塞其他吗 |
|---|---|---|---|
| T1 | `BookModel.go` (34KB) 按功能拆解 | 1 天 | 否 |
| T2 | Repository 抽象（Book / Document / Member 三优先） | 3~5 天 | 否（渐进） |
| T3 | `md_` 表前缀硬编码修复 | 0.5 天 | 否 |
| T4 | 日志换 `uber-go/zap` + Lumberjack 轮转 | 2~3 天 | 否（保留 beego/logs 兼容 shim） |
| T5 | `beego/i18n` → `nicksnyder/go-i18n/v2` | 2~3 天 | 否 |
| T6 | Session/gob 序列化换 msgpack（**根治** Round 2 风险 13） | 1~2 天 | 否 |
| T7 | 缓存方案 B 评估 · 上 `gocache/v3` 或 singleflight + tag | 2~3 天 | 否 |
| T8 | 前端 P1：静态资源版本号 + 删 IE 兼容 shim | 1~2 天 | 否 |
| T9 | 前端 P2：Vite 构建 + vendor 集中 + 抽离内联 JS | 1~2 周 | 否 |
| T10 | 补测试：`pkg/` + `internal/service/`（Repository 抽出后立即补） | 持续 | 否 |
| T11 | （可选）根据 Round 3 MCP 反馈决定是否上倒排/向量检索 | 独立立项 | 否 |
| T12 | （可选）**评估** ORM 迁移到 gorm/ent/sqlc，本轮**只出评估报告不实施** | 3 天 | 否 |

**总工期估计：** 3~4 周（按团队人力）；除 T9 外，其他都可以在同一 sprint 内并行推进。

### 本轮**不做**

- ❌ **不实施 ORM 迁移**（gorm/ent/sqlc）：太重，建议 T12 出评估报告后独立立项，可能是 Round 5
- ❌ **不做 SPA 完整重写**（`web-ui/` 用 Vue 3 + TS）— 定为 P4 长期目标，本轮到 P2 为止
- ❌ **不做 Bootstrap 5 / Tailwind 迁移** — P3 定为独立立项
- ❌ **不实施 CORS/CSP/HSTS 全局安全头**（可挂到 Round 4 尾巴，如果时间足够就做，不阻塞）

---

## 二、前置条件（Round 3 已完成）

- ✅ 目录结构已是 `cmd/` + `internal/`（Round 2）
- ✅ 强类型 `config.Config` 可用（Round 2）
- ✅ MCP 在线，有真实使用数据（Round 3）
- ✅ `internal/errs/` 稳定运行 ≥ 1 个月
- ✅ 团队已熟悉 `internal/dto/`、`internal/model/`、`internal/controller/` 布局

---

## 三、T1 · `BookModel.go` 拆解（1 天）

### 现状

- `internal/model/BookModel.go`（原 `models/BookModel.go`，34KB）— 单文件 900+ 行
- 混合职责：CRUD / 权限 / 树形结构 / 上传附件路径处理 / 图片路径规范化 / 权限迁移

### 拆分方案

```
internal/model/book/
├─ model.go        # type Book struct + Insert / Update / Delete
├─ query.go        # 查询：FindByIdentify / FindByBookId / FindPage / SelectPage / RelateSelectByBook
├─ tree.go         # 章节树相关：GenerateNodes / GetBookById 中的目录构建
├─ permission.go   # 权限检查：HasBook / IsPermit / TransferOwner / PrivatelyOwned
└─ upload.go       # 附件/图片路径处理（原 line 790-863 的图片规范化）
```

**注意：** 拆分后 `package book`；Book 结构体不动字段名，只是**方法散到不同文件**。所有 caller 的调用 `model.NewBook().Find(id)` 变成 `book.New().Find(id)` — 但为了减少 blast radius，**建议**：

- 方案 A（保守）：拆成多文件但保持 `package model`（改动最小，只是同 package 内文件切分）
- 方案 B（激进）：拆成 subpackage `internal/model/book/`（更好组织，但触碰全仓 caller）

**推荐方案 A**：Round 4 做 A，未来（Round 5+）再评估 B。

### 拆分步骤

1. `git mv internal/model/BookModel.go internal/model/book_model.go`（保持同 package）
2. 新建 `internal/model/book_query.go`、`book_tree.go`、`book_permission.go`、`book_upload.go`
3. 用 IDE **移动方法**（IntelliJ/GoLand 的 "Move to file"），每次移动 3~5 个方法，`go build` 一次
4. `go test ./internal/model/...`（如果 T10 已补测试）

### 验收

- `go build ./...` 通过
- `wc -l internal/model/book_*.go` 每个文件 ≤ 250 行
- 所有 caller 无需修改（同 package）

---

## 四、T2 · Repository 抽象（3~5 天）

### 目标

在 `internal/model/` 之上加一层 `internal/repository/`，把**事务 / 缓存 / 查询封装**统一收敛。

```
internal/repository/
├─ book_repo.go
├─ document_repo.go
├─ member_repo.go
└─ tx.go           # UnitOfWork（可选）

internal/service/
├─ book_service.go       # 业务逻辑：CreateBookWithDefaultRole 等
├─ document_service.go   # 复用：MCP 写工具 + 现有 controller 都调这里
└─ ...
```

### Repository 接口示例

```go
// internal/repository/document_repo.go
package repository

import (
    "context"
    "git.itopcms.com/jackliu/doc/internal/model"
)

type DocumentRepo interface {
    Find(ctx context.Context, id int) (*model.Document, error)
    Save(ctx context.Context, d *model.Document) error
    UpdateWithVersion(ctx context.Context, id int, expectVersion int64, patch map[string]any) (int64, error)
    Delete(ctx context.Context, id int) error
}

type documentRepo struct{ orm orm.Ormer }
func NewDocumentRepo(o orm.Ormer) DocumentRepo { return &documentRepo{orm: o} }
```

### 渐进迁移策略

- 本轮**不批量改** controller 的 caller，只**新增** Repository 层
- **MCP 写工具**（Round 3 直接调 `model.NewDocument()`）— 本轮改成调 `repository.NewDocumentRepo(orm.NewOrm()).UpdateWithVersion(...)`；Round 3 的乐观锁逻辑从 `internal/mcp/tools_write.go` 挪到 repo
- 新写的代码**只准**通过 Repository 访问 DB
- 老 controller 保持不变，未来（Round 5+）逐步迁

### 事务封装

```go
// internal/repository/tx.go
type UnitOfWork interface {
    Run(ctx context.Context, fn func(ctx context.Context) error) error
}

// 在 ctx 里塞 orm.Ormer，Repository 从 ctx 取；未在 ctx 里的走全局 default orm
func txOrm(ctx context.Context) orm.Ormer { ... }
```

### 验收

- `internal/repository/` + `internal/service/` 存在
- MCP 写工具走 Repository 层，测试通过
- 至少给 `DocumentRepo` 补 3 个单元测试（用 sqlite 内存库）
- 老 controller 一行未改（渐进原则）

---

## 五、T3 · `md_` 硬编码修复（0.5 天）

### 定位

```powershell
rg 'md_[a-z_]+' --type go
```

**主要位置：**

- `internal/model/Member.go:77` — `o.Raw("select * from md_members where ...")`
- 其他 raw SQL（grep 后逐个 review）
- `internal/model/BookResult.go` / `internal/dto/book_result.go` 里的 raw join 可能也有

### 修复

- 用 `config.GetDatabasePrefix()`（Round 2 T4 已经在 `internal/config/`）
- 或用 model 的 `TableNameWithPrefix()` 方法（现有模式）
- SQL 里的表名用 `fmt.Sprintf("SELECT * FROM %s WHERE ...", (&Member{}).TableNameWithPrefix())`

### 验收

- `rg 'from md_' --type go -i` 无残留（除了单元测试的 fixture SQL）
- 换 DB 前缀（`db_prefix = "abc_"`）后跑一遍冒烟仍能正常

---

## 六、T4 · 日志换 `uber-go/zap`（2~3 天）

> 决策：[§八 决策 2026-07-29](./refactor-roadmap.md#八决策记录decision-log) — `uber-go/zap` 胜出 `log/slog`。

### 依赖

```powershell
go get go.uber.org/zap@latest
go get gopkg.in/natefinch/lumberjack.v2@latest
```

（`go.uber.org/zap` 依赖 `go.uber.org/atomic`，已在 indirect。）

### 新增 `internal/logging/`

```
internal/logging/
├─ logger.go          # NewLogger(cfg config.LogConfig) *zap.Logger
├─ context.go         # LoggerFromCtx / WithLogger 助手
└─ shim_beego.go      # 兼容层：接管 beego/logs 输出到 zap
```

### 初始化

```go
// internal/logging/logger.go
func NewLogger(cfg config.LogConfig) (*zap.Logger, error) {
    encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())  // 生产 JSON
    if cfg.Format == "console" {
        encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())  // 开发人类可读
    }

    fileWriter := zapcore.AddSync(&lumberjack.Logger{
        Filename:   cfg.Path,       // config.Global.Log.Path
        MaxSize:    100,            // MB
        MaxBackups: 30,
        MaxAge:     30,             // days
        Compress:   true,
    })

    core := zapcore.NewTee(
        zapcore.NewCore(encoder, fileWriter,          parseLevel(cfg.Level)),
        zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
    )
    return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}
```

### 兼容 `beego/logs`

`internal/logging/shim_beego.go`：注册一个自定义 `beego/logs` adapter，把 `logs.Info/Error/Debug` 全部转发到 zap。存量 caller 一行不改，仍能享受 zap 的输出格式与轮转。

```go
// 类似做法可参考：https://github.com/beego/beego/issues/xxxx
type zapAdapter struct { logger *zap.Logger }
func (a *zapAdapter) WriteMsg(when time.Time, msg string, level int) error { ... }
func (a *zapAdapter) Init(config string) error { return nil }
// ... 实现 logs.Logger 接口
```

`internal/app/bootstrap.go` 初始化时：

```go
zap := logging.NewLogger(config.Global.Log)
zap.ReplaceGlobals(zap)   // zap 全局
logs.Register("zap", func() logs.Logger { return &zapAdapter{logger: zap} })
logs.SetLogger("zap", "")
```

### 渐进迁移策略

- **本 PR** 只做适配层 + 全局初始化；老 `logs.Error("xxx", err)` 一行不改
- **后续 PR**（可拆多个小 PR）：把关键路径的 log 改成结构化字段：
  ```go
  // 老：
  logs.Error("delete document failed", err)
  // 新：
  zap.L().Error("delete document failed",
      zap.Int("document_id", id),
      zap.Int("member_id", m.MemberId),
      zap.Error(err))
  ```

### 结构化字段约定

- `document_id`, `book_id`, `member_id`, `token_id` — 整型，命名 snake_case
- `error` — `zap.Error(err)`（自动 stacktrace）
- `duration_ms` — 时长
- `trace_id` — 请求追踪（Round 5 引入 OTel 时用）

### 验收

- 日志文件按 lumberjack 策略轮转
- JSON 模式下能被 Filebeat / Fluentd 直接采集
- 老 `logs.Error(...)` 输出到 zap 格式
- `grep -R 'logs\.' internal/` 数量可控，后续 PR 逐步减少

---

## 七、T5 · i18n 换 `nicksnyder/go-i18n/v2`（2~3 天）

### 现状

- `github.com/beego/i18n v0.0.0-20161101132742-e9308947f407` — **11 年前**的老包，`ini` 格式
- `configs/lang/zh-cn.ini` / `configs/lang/en-us.ini`
- 30+ 处 `i18n.Tr(lang, "key")` 调用（templates + Go 代码）

### 目标包

```powershell
go get github.com/nicksnyder/go-i18n/v2@latest
```

### 迁移步骤

1. **保留 `.ini` 语言文件**（迁移成本最低），用 go-i18n 的 `LoadMessageFile` 读取
2. 新增 `internal/i18n/`：
   ```go
   // internal/i18n/i18n.go
   var bundle *i18n.Bundle

   func Init(dir string) error {
       bundle = i18n.NewBundle(language.SimplifiedChinese)
       bundle.RegisterUnmarshalFunc("ini", func(data []byte, v any) error {
           // 自己实现 ini→map[string]string，或转 JSON/TOML
       })
       for _, lang := range []string{"zh-cn", "en-us"} {
           bundle.LoadMessageFile(filepath.Join(dir, lang+".ini"))
       }
       return nil
   }

   func Tr(lang, key string, args ...any) string {
       loc := i18n.NewLocalizer(bundle, lang)
       msg, _ := loc.Localize(&i18n.LocalizeConfig{MessageID: key, TemplateData: args})
       return msg
   }
   ```
3. **建议**：借这次迁移把 `.ini` 转成 `.toml`（go-i18n 原生支持，支持复数、支持模板参数）
4. **模板函数**：把 `beego/i18n` 注册的模板函数替换为自己的 `Tr`：
   ```go
   web.AddFuncMap("i18n", i18n.Tr)
   ```
   模板里 `{{i18n .Lang "hello"}}` 保持不变

### 兼容层

```go
// internal/i18n/shim.go —— 保留旧 API 一段时间
func SetMessage(lang, file string) error { return bundle.LoadMessageFile(file) }
func IsExist(lang string) bool           { return has(lang) }
```

`internal/controller/BaseController.SetLang` 里的 `i18n.IsExist(lang)` 无需改。

### 验收

- 中英切换生效
- 后台管理页 / 前端登录页 / 提示语全对
- `beego/i18n` 依赖可从 `go.mod` 移除
- Round 2 T3 里 `configs/app.conf` 的 `[i18n]` section 仍读到

---

## 八、T6 · Session/gob 换 msgpack（1~2 天 · **可选但强烈建议**）

### 动机

Round 2 风险 13：`gob` 编码类型名 = **"包路径.类型名"**；每次目录/包路径变都会崩。msgpack 只按字段 tag 序列化，与包路径解耦。

### 依赖

```powershell
go get github.com/vmihailenco/msgpack/v5@latest
```

### 改造点

1. `internal/cache/cache.go`（Round 1 shim）：内部 `encoding/gob` → `msgpack`
2. `internal/cache/beego_adapter.go`：同上
3. Beego session 序列化：Beego session 用 gob 序列化 struct，可以：
   - **方案 A**：注册自定义 session serializer（beego v2 支持）
   - **方案 B**：`SetMember` 只塞 `member_id`，`Prepare` 里按 id 查库（重回 Round 1 T6 的 options 缓存思路）
4. 删掉 `internal/app/bootstrap.go`（原 `commands/command.go:113-115`）的 `gob.Register(models.Blog{})` 等硬编码

### 上线策略

- 与 Round 4 T4（日志）一起发一个大版本
- CHANGELOG 明说"需清 session + cache/"（与 Round 2 T9 同套流程）
- 之后**再也不用担心**目录变动导致缓存崩

### 验收

- `rg 'gob\.Register' internal/` 无残留
- 清 session + cache 后启动一切正常
- 目录再挪一次（比如未来 controller 拆域）不会出现反序列化错误

---

## 九、T7 · 缓存方案 B 评估（2~3 天）

> 决策点：Round 1 T5 已抽好 `Cache` 接口，本轮决定"要不要上 gocache/v3 / 分层缓存"。

### 评估维度

| 维度 | 现状（Round 1 T5 后） | 上 gocache/v3 |
|---|---|---|
| 分层（memory + redis） | 手写实现 | 内置 Chain |
| singleflight 防击穿 | 手写 `internal/cache/loader.go` | 内置 Loader |
| metrics | 无 | 内置 Prometheus adapter |
| 缓存 tag | 需自己实现 | 内置 |
| 学习成本 | 0 | 1~2 天 |
| 依赖引入 | 0 | `github.com/eko/gocache/v3` + 各 adapter |

### 决策标准

- 如果 MCP 上线后 DB 压力**没有**明显上升 → 保持轻量方案 A（Round 1）
- 如果需要 metrics 或分层缓存 → 上 gocache/v3

### 若上 gocache/v3

新增 `internal/cache/chained.go`：

```go
memory := cache.New[[]byte](memoryStore)
redis  := cache.New[[]byte](redisStore)
chain  := cache.NewChain[[]byte](memory, redis)

loader := cache.NewLoadable[[]byte](chain, func(ctx, key) ([]byte, error) {
    return loadFromDB(key)
})
```

### 验收

- 出评估报告到 `docs/round-4-cache-evaluation.md`
- 若决定实施：现有 caller 通过 `Cache` 接口享受到分层能力
- Prometheus `/metrics` 里能看到 cache_hit_total / cache_miss_total（若配了 metrics adapter）

---

## 十、T8~T9 · 前端现代化

### T8 · P1（1~2 天）

- **静态资源版本号**：现有 `cdnjs "..." "version"` 机制（`views/widgets/scripts.tpl`）铺开到所有引用
- **删 IE 兼容 shim**：
  - `web/static/respond.min.js` → 删
  - `web/static/html5shiv.min.js` → 删
  - 模板里 `<!--[if lt IE 9]>` 条件注释 → 删

**验收：** Chrome / Firefox / Edge 现代版本行为不变；升级第三方库时 URL 带版本号可避免浏览器缓存。

### T9 · P2（1~2 周）

> 引入 Vite 前端构建。**注意：** 本轮**不做** Bootstrap 升级、不做 Vue 3 迁移；只搭构建流水线 + vendor 集中管理 + 抽离内联 JS。

#### 目录结构

```
web-ui/                       # 【新增】Vite 构建源码
├─ package.json
├─ vite.config.ts
├─ src/
│  ├─ entries/                # 每个页面一个入口，与后端模板对应
│  │  ├─ document-edit.ts
│  │  ├─ book-view.ts
│  │  └─ ...
│  ├─ vendor/                 # 第三方库（jQuery / editor.md / mermaid 等的 import）
│  └─ styles/
└─ tsconfig.json
```

#### 构建产物

```
web/static/dist/
├─ manifest.json              # Vite 生成的 hash 映射
├─ assets/
│  ├─ document-edit.a1b2c3.js
│  └─ document-edit.d4e5f6.css
```

#### 后端模板集成

`internal/controller/BaseController.go` 新增模板函数 `vite_asset(entryName)`：

- 读取 `web/static/dist/manifest.json`
- 返回带 hash 的 JS/CSS 路径
- 模板里 `<script src="{{vite_asset "document-edit"}}"></script>`

#### 开发模式

- `npm run dev` 起 Vite dev server（HMR）
- `internal/app/bootstrap.go` dev 环境下把 `/dist/*` 反代到 Vite dev server
- 生产 `npm run build` 出静态文件，beego 直接 serve

#### 抽离内联 JS

`web/views/*.tpl` 里的 `<script>...</script>` 抽到 `web-ui/src/entries/*.ts`。**渐进做**：每次 PR 抽 1~2 个模板。

#### 验收

- `npm run build` 生成 `web/static/dist/`
- 生产环境访问带 hash 的 JS/CSS 200
- dev 模式 HMR 生效
- 老模板里剩余的 `<script>` 内联 JS 数量 ≤ 5（其他抽走）

---

## 十一、T10 · 补测试（持续）

### 优先级

1. **T2 Repository 抽出后立即补** — Repository 层是**新代码**，天然可测（用 sqlite 内存库）
2. **`pkg/` 通用工具** — 无业务依赖，最好测（`pkg/cryptil` / `pkg/pagination` / `pkg/filetil`）
3. **`internal/errs/` + `internal/config/`** — 逻辑简单
4. **不测**：`internal/controller/`（beego 强绑 request/response，测起来重）

### 依赖

```powershell
go get github.com/stretchr/testify@latest
```

### 目标覆盖率

- Repository：≥ 60% 行覆盖
- `pkg/*`：≥ 40%
- 全项目：≥ 15%（起步）

### CI 集成

- 加 `scripts/test.sh`：`go test -race -cover ./...`
- GitHub Actions（如果用）加测试 job

### 验收

- `go test ./...` 无失败
- coverage report 存到 `docs/round-4-coverage.md`

---

## 十二、T11 · 倒排/向量检索（可选，独立评估）

### 触发条件

Round 3 上线后 2~4 周，收集：

- MCP `search_document` 的召回率（是否 AI 抱怨"搜不到"）
- Web UI 搜索用户投诉
- 数据集大小（文档总数、总字节数）

### 备选

| 方案 | 部署 | 成本 | 场景 |
|---|---|---|---|
| bleve（进程内 Go 库） | 无 | 低 | 中小规模，与主进程绑定 |
| meilisearch | 独立服务（几十 MB Rust） | 中 | 中等规模，中文分词好 |
| qdrant（向量） | 独立服务 | 高 | 语义搜索，需要向量化预处理 |
| Elasticsearch | 重服务 | 高 | 超大规模，通用性最好 |

### 若决定实施

- Round 3 T1 的 `searchProvider` 抽象天然可扩展 → 加 `bleveProvider` / `meilisearchProvider`
- `internal/mcp/search_provider.go` 加实现
- `configs/app.conf` `[search]` section 加 provider 配置

---

## 十三、T12 · ORM 迁移评估报告（3 天 · 不实施）

### 输出

`docs/round-4-orm-migration-evaluation.md`，内容：

1. 三候选对比：`gorm` / `ent` / `sqlc`
2. 迁移工作量估算（按每个 model 迁移 ≈ 0.5~1 天）
3. 破坏面：raw SQL / `md_` 前缀 / `orm.QueryTable().Filter()` 链
4. 收益：类型安全 / 关联查询 / 测试友好
5. 推荐结论（含"暂不迁移"选项）

### 决策会

Round 4 结束前团队开会拍板：
- **A 方案**：不迁移，Repository 层已经吸收大部分问题 → 结束
- **B 方案**：Round 5 立项迁移，选定 ORM
- **C 方案**：只在**新代码**用新 ORM（如 sqlc），存量保持 beego/orm

---

## 十四、PR 拆分

| # | PR | 内容 | 大小 | 依赖 |
|---|---|---|---|---|
| 1 | `refactor(round4): split BookModel by responsibility` | T1 | 中 | 无 |
| 2 | `feat(round4): introduce Repository layer (Document/Book/Member)` | T2 | 中大 | 无 |
| 3 | `fix(round4): remove hardcoded md_ prefix in raw SQL` | T3 | 小 | 无 |
| 4 | `feat(round4): zap logger with lumberjack (beego/logs shim)` | T4 | 中 | 无 |
| 5 | `feat(round4): migrate to nicksnyder/go-i18n/v2` | T5 | 中 | 无 |
| 6 | `refactor(round4): replace gob with msgpack for cache/session` | T6 | 中 | 无（上线**需清缓存+session**） |
| 7 | `docs(round4): cache evaluation (gocache/v3 or stay)` | T7 | 小 | 无 |
| 8 | `chore(round4): frontend P1 – versioning + drop IE shims` | T8 | 小 | 无 |
| 9 | `feat(round4): Vite build pipeline + manifest integration` | T9 | 大 | 无 |
| 10 | `test(round4): repository + pkg tests` | T10 | 中 | PR-2 |
| 11 | `docs(round4): search backend evaluation report` | T11 | 小 | Round 3 数据 |
| 12 | `docs(round4): ORM migration evaluation report` | T12 | 小 | 无 |

**合入顺序建议：** PR-1 → PR-3 → PR-4 → PR-6 → PR-2 → PR-10 → PR-8 → PR-5 → PR-7 → PR-9 → PR-11 → PR-12。
- PR-1/3/4/6 是"内部收拾"，先落
- PR-2/10 一起做（Repo 就补测试）
- PR-8/5/9 前端与 i18n 交叉，PR-9 最重放尾
- 报告类 PR-7/11/12 与团队讨论同步产出

---

## 十五、上线检查清单

### PR-6（gob→msgpack）单独强调

- [ ] 现网 session 存储清空（同 Round 2 T9 SOP）
- [ ] `cache/` 目录清空
- [ ] CHANGELOG 显眼位置写"⚠️ Breaking：需清 session + cache/，用户需重新登录"

### 常规

- [ ] 冒烟测试全通（登录 / 上传 / 导出 / 后台）
- [ ] 日志：JSON 格式正确、按天/大小轮转
- [ ] i18n：中英切换正常，所有 label 全对
- [ ] MCP 依赖的 Repository 层无回归（Round 3 的 10 个工具全测）
- [ ] 前端：Chrome / Firefox / Edge 主流版本正常

---

## 十六、追踪表

| # | 任务 | PR | Commit | 状态 |
|---|---|---|---|---|
| T1 | `BookModel` 拆解 | | | |
| T2 | Repository 抽象 | | | |
| T3 | `md_` 硬编码修复 | | | |
| T4 | zap + Lumberjack | | | |
| T5 | go-i18n/v2 | | | |
| T6 | gob→msgpack | | | |
| T7 | 缓存评估报告 | | | |
| T8 | 前端 P1 | | | |
| T9 | Vite 构建 (P2) | | | |
| T10 | 补测试 | | | |
| T11 | 搜索后端评估 | | | |
| T12 | ORM 迁移评估报告 | | | |

---

## 十七、Round 4 完成后的项目状态

- 目录形态：`cmd/` + `internal/` + `pkg/` + `web/` + `configs/` + `deployments/`（Round 2 定型）
- 配置：强类型 `config.Config`，`[section]` 分组，支持 `.env`
- 缓存：抽象接口 + msgpack 序列化 + 可选分层
- 日志：zap 结构化 + Lumberjack 轮转
- i18n：go-i18n/v2 + toml/ini 双支持
- Model：大文件拆解 + Repository 层 + 部分测试
- MCP：稳定运行，可挂搜索后端
- 前端：Vite 构建，vendor 集中，为 Round 5+ 的 Bootstrap/Vue 升级铺路

### Round 5+ 候选（本轮结束时再讨论）

- ORM 迁移（gorm / ent / sqlc，选 1）
- 前端 P3：Bootstrap 3 → 5 或 Tailwind
- 前端 P4：`web-ui/` Vue 3 + TS SPA
- 倒排 / 向量检索独立服务
- OTel 全链路追踪
- CORS / CSP / HSTS 全局安全头 + secure middleware
- 单元/集成测试覆盖率 ≥ 40%

---

## 十八、参考

- [refactor-roadmap.md §2.4](./refactor-roadmap.md#24-目标四缓存--模型组件升级) — 缓存/模型详述
- [refactor-roadmap.md §三 支线](./refactor-roadmap.md#三支线其他技术债) — 日志/i18n/测试/main.go/gopool/requests
- [refactor-roadmap.md §四](./refactor-roadmap.md#四支线前端与静态资源) — 前端分阶段方案
- [uber-go/zap](https://github.com/uber-go/zap) + [natefinch/lumberjack](https://github.com/natefinch/lumberjack)
- [nicksnyder/go-i18n](https://github.com/nicksnyder/go-i18n)
- [vmihailenco/msgpack](https://github.com/vmihailenco/msgpack)
- [eko/gocache](https://github.com/eko/gocache)
- [testify](https://github.com/stretchr/testify)
- [Vite](https://vitejs.dev/)
