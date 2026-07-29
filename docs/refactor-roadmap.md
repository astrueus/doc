# Doc 项目整体优化与迭代路线图

> 本文是 `doc` 项目**整体重构与优化**的总纲文档，聚焦四大目标：
>
> 1. 接入 MCP，方便 AI 使用文档
> 2. 前后端目录结构规范化
> 3. 配置模块优化（Go 代码与配置文件解耦、分组）
> 4. 缓存 / 模型等基础组件升级
>
> 同时汇总"顺带发现的技术债"与"前端现代化"两条支线，给出四轮可独立上线的迭代计划。
>
> **相关文档：**
>
> - **执行文档（每轮独立可落地）：**
>   - [round-1-execution-plan.md](./round-1-execution-plan.md) — Round 1 · 低风险重构 + 后续前置
>   - [round-2-execution-plan.md](./round-2-execution-plan.md) — Round 2 · `cmd/`+`internal/` 一步到位 + 强类型 Config
>   - [round-3-execution-plan.md](./round-3-execution-plan.md) — Round 3 · MCP Server（10 工具 + Bearer + 搜索）
>   - [round-4-execution-plan.md](./round-4-execution-plan.md) — Round 4 · 模型 / 日志 / i18n / 前端 Vite
> - **参考文档：**
>   - [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) — 前后端目录拆分（Round 2 目标目录已更新，附录 A/B 硬编码定位仍适用）
>   - [router-split-migration-plan.md](./router-split-migration-plan.md) — 路由按职责拆分与 `/api` 前缀治理（Round 2 T6 使用）
>   - [routers-reference.md](./routers-reference.md) — 现有路由分类参考
>   - [upstream-mindoc-checklist.md](./upstream-mindoc-checklist.md) — 上游 MinDoc 提交跟进清单
>
> **文档生成依据：** 当前仓库代码基线（2026-07），Go 1.25 + Beego v2。

---



## 一、现状速览



### 1.1 技术栈


| 层      | 组件                                                      | 状态              |
| ------ | ------------------------------------------------------- | --------------- |
| 语言     | Go 1.25.0                                               | 新               |
| Web 框架 | `beego/v2 v2.3.10`                                      | 社区更新缓慢          |
| ORM    | `beego/orm`                                             | 与框架同命运          |
| i18n   | `beego/i18n v0.0.0-20161101132742-e9308947f407`         | **11 年前**的老包    |
| 缓存     | `beego/cache` + 自封 `cache/`（gob 序列化）                    | 需重构             |
| 模板     | `html/template`（beego 内置）                               | 稳定              |
| CLI 服务 | `kardianos/service`                                     | 稳定              |
| 数据库驱动  | `go-sql-driver/mysql` + `mattn/go-sqlite3`              | 稳定，无 PostgreSQL |
| 前端     | Bootstrap 3.2 + jQuery + editor.md + wangEditor + Vue 2 | 全部偏旧            |




### 1.2 结构问题速览


| 问题                         | 现状                                                                                     | 位置                  |
| -------------------------- | -------------------------------------------------------------------------------------- | ------------------- |
| **配置与代码混目录**               | `conf/` 同时放 `enumerate.go`/`mail.go`（Go 源码）和 `app.conf`/`app.conf.example`/`lang/`（配置） | `conf/`             |
| **配置文件无分组**                | `app.conf` 253 行平铺，session/DB/upload/mail/export/ldap/cdn/cache/log/dingtalk/i18n 全混一起 | `conf/app.conf`     |
| **路由未分组**                  | `web.Router()` 148 行一把梭                                                                | `routers/router.go` |
| **Controller 巨型文件**        | Document 37KB / Book 27KB / Manager 30KB / Blog 20KB                                   | `controllers/`      |
| **Model 巨型文件**             | Book 34KB / BookResult 23KB / Member 17KB                                              | `models/`           |
| **Base.go 空基类**            | 全文只有 `type Model struct {}`                                                            | `models/Base.go`    |
| **无 MCP**                  | 无 AI 接入能力                                                                              | —                   |
| **无 Repository/Service 层** | Controller 直接调 Model，事务/缓存/测试都难做                                                       | —                   |
| **前端资源平铺**                 | 24 个第三方库 + 自写 js/css 全平铺在 `static/` 根下                                                 | `static/`           |




### 1.3 关键代码债


| 债务                                                      | 影响             | 涉及文件数                              |
| ------------------------------------------------------- | -------------- | ---------------------------------- |
| `ioutil.ReadFile` / `ioutil.WriteFile` 等 deprecated API | Go 1.16+ 已废弃   | 12                                 |
| `interface{}` 遍布（未用 `any`）                              | Go 1.18+ 应改    | 20+                                |
| `md_` 表前缀硬编码（未走 `GetDatabasePrefix()`）                  | 换前缀会崩          | 多处 raw SQL                         |
| `gob.Register` 硬编码类型                                    | 加新缓存类型要改初始化    | `commands/command.go:113-115`      |
| `cache.Get` 用 `context.TODO()` 全局共享                     | 无法传超时/取消/trace | `cache/cache.go:16`                |
| `BaseController.Prepare` 每次请求全表读 options                | DB 压力          | `controllers/BaseController.go:68` |
| `smtp_host="${...}""` 结尾多一个引号                           | 配置解析 bug       | `conf/app.conf:106,110`            |
| `orm.DefaultRowsLimit = -1` 全局关分页                       | 潜在性能坑          | `commands/command.go:39`           |
| `main.go` 手写 `os.Args[1] == "service"`                  | 无子命令框架         | `main.go:21-29`                    |


---



## 二、四大目标详细方案



### 2.1 目标一：MCP Server（AI 接入）

> 与 [upstream-mindoc-checklist.md §4.2](./upstream-mindoc-checklist.md) 呼应，落地本地最小方案。
> **SDK 选型**：直接采用官方 `github.com/modelcontextprotocol/go-sdk`（v1.x），避免未来从社区 SDK 迁移的双份成本。详见 §八 决策日志。



#### 现状

- 无 MCP 相关代码，`go.mod` 无 MCP SDK 依赖。
- 搜索只有 `models/DocumentSearchResult.go` 的 SQL `LIKE`，无倒排索引/向量搜索。
- 权限走 Session Cookie，MCP 无天然身份接入通道。
- `MemberToken` **表是邮箱验证码用途**（含 `Email` / `SendTime` / `ValidTime` / 发送次数限制），**不能用于 MCP Bearer Token**。需新增 `member_api_tokens` 表（见下）。



#### 方案（分两期）


| 期                               | 内容                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | 工作量   |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----- |
| **MVP · stdio + 读写工具**          | 新增 `mcp/` 包 + `commands/mcp.go` 子命令（`doc mcp`）。基于 **官方** `modelcontextprotocol/go-sdk` **v1.x**，stdio 模式，暴露 **10 个工具**： **读（4 个）** `search_document` / `get_document` / `list_books` / `list_document_tree` **写（6 个）** `create_document` / `update_document_content`（带乐观锁）/ `append_document_content` / `update_document_meta` / `release_document` / `delete_document`（强制 `confirm: true`） 内部直接调 `models` 层，权限走 `BookResult.FindByIdentify(identify, memberId)`；stdio 免鉴权，用 `mcp_stdio_member` 指定身份 | 2~3 天 |
| **Streamable HTTP + API Token** | 新增 `mcp/http.go`（`http.Handler`，Beego 挂 `/mcp/`*）+ Bearer 鉴权 middleware；**新增** `models/MemberApiToken.go` 表（`token_hash` / `scopes` / `expires_at` / `last_used_at`）+ 后台"生成/撤销 API Token"页面；`configs/app.conf` 新增 `[mcp]` section（`mcp_enable` / `mcp_listen` / `mcp_stdio_member` / `mcp_token_required` / `mcp_rate_limit`）                                                                                                                                                                        | 2~3 天 |




#### 关键设计点

1. **单二进制多入口**：`doc`（web 服务）、`doc mcp`（stdio）、`doc mcp --http`（Streamable HTTP）。共享 `commands.ResolveCommand` 加载 conf/DB/cache/models，只跳过 `web.Run()`。
2. **工具输入用 struct +** `jsonschema` **tag**（官方 SDK 泛型 handler 会自动推 schema），MCP 工具的 In/Out struct 天然是 DTO，可提前落到 `internal/dto/mcpdto/`，为 Round 4 的 Repository/Service 分层铺路。
3. **鉴权分离**：stdio 免鉴权（本地进程即身份，`mcp_stdio_member` 指定 Member）；HTTP 强制 Bearer Token（`Authorization: Bearer <token>` → `sha256` → 查 `member_api_tokens` → 注入 `context` 中的 Member）。
4. **权限统一走现有 BookRole**：`≥ BookEditor(2)` 才有写权限；`delete_document` 额外要求 `confirm: true` 参数。
5. **乐观锁**：`update_document_content` 必须带 `expect_version`（对应 `Document.Version` 时间戳），版本不匹配返回 `VERSION_CONFLICT`，让 AI 自行 `get_document` 后重试。
6. **只写 Markdown**：写工具只更新 `Document.Markdown`，`Content`/`Release` HTML 由现有 `ReleaseContent()` 流程生成（`release_document` 工具或 `auto_release=true` 触发）。
7. `mcp/tools_*.go` **抽象层**：`tools_read.go` / `tools_write.go` 只做"参数校验 + 权限判断 + 调 model + 组装结果"，业务逻辑仍在 `models`。
8. **为将来上倒排索引铺路**：`search_document` 内部走 `searchProvider` 接口，`sql_like` 是默认实现，未来切 `fulltext`/`bleve`/`qdrant` 只改一处。
9. **限流**：HTTP 模式下用 `golang.org/x/time/rate` 写一个 `mcp.Middleware`，通过 `AddReceivingMiddleware` 挂上；写工具与删除工具分别做次数限制（防 AI 批量误操作）。
10. **工具输出统一走 markdown / 结构化字段**，不返回 HTML，AI 侧最好用。



#### 交付物

- `mcp/server.go`、`mcp/http.go`、`mcp/authz.go`、`mcp/errors.go`
- `mcp/tools_read.go`、`mcp/tools_write.go`、`mcp/convert.go`
- `commands/mcp.go`（新增子命令）
- `models/MemberApiToken.go`（新表，**不复用** `MemberToken`）
- `controllers/MemberApiTokenController.go` + `views/member/api_tokens.tpl`（Token 管理页）
- `internal/dto/mcpdto/`（工具 In/Out struct，为 Round 4 铺路）
- `go.mod` 加 `github.com/modelcontextprotocol/go-sdk`（锁定 v1.x 稳定版）
- `configs/app.conf.example` 增加 `[mcp]` section
- `docs/mcp-integration.md`（Claude Desktop / Cursor stdio + HTTP 接入示例）

---



### 2.2 目标二：前后端目录结构调整（规范化）

> **决策更新（2026-07-29）**：本轮**一步到位** `cmd/` + `internal/` 激进方案，不再走 `server/` + `web/` + `deploy/` 过渡形态。
> 理由：过渡方案完成后仍要迁到 `internal/`，等于**搬两次**；MCP 从 Round 3 开始就直接落在 `internal/mcp/` 最终位置。
> [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) 中关于 `ViewsPath` / `StaticDir` / 字体路径 / `conf/lang/` i18n 硬编码 / Docker & spug 脚本改动的**执行细节仍适用**，只是目标目录换成本节的最终结构。



#### 现状问题

```
d:\jcwork\doc\
├─ controllers/    ← 15 个巨型文件（Document 37KB / Book 27KB / Manager 30KB / Blog 20KB）
├─ models/         ← 33 个文件，Model+Result+Service 混一层
├─ routers/        ← 148 行全平铺
├─ views/          ← 按业务分了子目录，还行
├─ static/         ← 24 个第三方库 + 自写 css/js 全平铺
├─ cache/          ← 只有 cache.go + cache_null.go
├─ commands/       ← 混着 CLI 注册 + DB/缓存/日志初始化（562 行）
├─ conf/           ← ⚠️ Go 源码 + 配置文件混在一起
├─ middleware/  + routers/filter.go  ← 中间件两处分裂
├─ Dockerfile / start.sh / sync_host.sh  ← 部署脚本散落根目录
```



#### 目标目录结构（一步到位）

```
doc/
├─ cmd/
│  └─ doc/
│     └─ main.go                 # 只做 flag 解析 + cobra 子命令派发
├─ internal/                     # Go 私有包，防止外部误引
│  ├─ app/                       # 装配层（原 commands/ 的初始化部分）
│  ├─ config/                    # 【新增】强类型 Config struct + Load()（原 conf/enumerate.go/mail.go 的 Go 部分）
│  ├─ controller/                # 按域拆子目录：document/ book/ manager/ blog/ member/ ...
│  ├─ service/                   # 【新增】业务逻辑下沉（Round 4 逐步落地）
│  ├─ repository/                # 【新增】ORM 查询集中（Round 4 逐步落地）
│  ├─ model/                     # 按域拆子目录：book/ document/ member/ ...
│  ├─ dto/                       # 【新增】原 *Result.go 挪进来；含 mcpdto/（Round 3 使用）
│  ├─ middleware/                # 合并原 middleware/ + routers/filter.go
│  ├─ router/                    # 按域拆：api.go / manager.go / document.go / blog.go / router.go(汇总)
│  ├─ cache/                     # 升级（见 §2.4）
│  ├─ converter/                 # 电子书导出（原根目录 converter/）
│  ├─ errs/                      # 【新增】BizError + 错误码
│  └─ mcp/                       # 【新增】MCP server（Round 3 落地）
├─ pkg/                          # 可对外复用工具（原 utils/* 中通用部分）
│  ├─ graphics/                  # 图片裁剪/缩放（原根目录 graphics/）
│  └─ mail/                      # SMTP（原根目录 mail/；模板在 web/views）
├─ configs/                      # 【新增】只放配置文件（不含 .go）
│  ├─ app.conf / app.conf.example
│  # 本轮不拆多文件；`app.conf` 内部走 [section] 分组，见 §2.3
│  └─ lang/                      # zh-cn.ini / en-us.ini
├─ web/                          # 前端资源
│  ├─ static/
│  │  ├─ vendor/                 # 24 个第三方库集中管理
│  │  ├─ css/  js/  images/  fonts/  editors/
│  └─ views/
├─ deployments/                  # 【新增】Docker / spug / systemd 等
│  ├─ Dockerfile
│  ├─ docker-compose.yml
│  ├─ start.sh / sync_host.sh
│  └─ scripts/
├─ scripts/                      # 构建脚本 build.sh / build.bat
├─ docs/                         # 项目文档（保留）
├─ runtime/  uploads/            # 运行时数据（文件缓存/导出在 runtime/cache/）
├─ go.mod  go.sum
└─ README.md  LICENSE.md
```



#### 关键改动点

1. `cmd/` **+** `internal/` 是 Go 生态事实标准（`golang-standards/project-layout`），`internal/` 天然防止外部 module 误引。
2. `conf/` **彻底拆解**（激进方案必须一起做，否则「一步到位」不成立）：
  - `conf/enumerate.go` / `conf/mail.go`（Go 源码）→ `internal/config/`（`package config`）
  - `conf/app.conf` / `.example` → `configs/app.conf`
  - `conf/lang/*.ini` → `configs/lang/`
  - 全仓 30+ 处 `import "git.itopcms.com/jackliu/doc/conf"` → `internal/config`
  - 硬编码 `"./conf/app.conf"`、`"conf/lang/"+lang+".ini"` → `"./configs/..."`（含 `commands/install.go:106`、`commands/command.go:280` 等）
3. **Controller 拆分**（可选，建议 Round 4 再做）：`DocumentController.go` 37KB 按方法组拆成 `internal/controller/document/read.go`、`edit.go`、`history.go`、`export.go`。**Round 2 只搬目录、不拆域**，减小 blast radius。
4. `routers` **拆分**：见 [router-split-migration-plan.md](./router-split-migration-plan.md)，落到 `internal/router/`。
5. **模板/静态资源路径**：`commands/command.go:311, 332, 334, 337, 342, 345-347` 的 `ViewsPath` / `StaticDir` / 字体路径全部同步；`BookResult.go` 里的导出资源拷贝路径也要一起改（详见 frontend-backend-split-migration-plan.md 附录 A）。
6. **中间件合并**：原 `middleware/filter.go` + `routers/filter.go` 合到 `internal/middleware/`。
7. `utils/` **拆分**：通用工具（`cryptil`、`filetil`、`pagination`、`requests` 等）→ `pkg/`；耦合业务的（如 `gopool` 若被 model 用）留 `internal/`。
8. **模块路径不变**：`git.itopcms.com/jackliu/doc` 保持，只改子包 import。



#### 迁移执行阶段（Round 2 内部拆两次 PR）


| PR                        | 内容                                                                                     | 工作量   | 可回滚         |
| ------------------------- | -------------------------------------------------------------------------------------- | ----- | ----------- |
| **PR-1：目录搬迁 + import 改写** | 全仓 `git mv` + `gofmt -r` + `goimports`；不改逻辑；启动跑通                                       | 3~5 天 | 是（单 revert） |
| **PR-2：路径硬编码 + 部署脚本修正**   | `ViewsPath` / `StaticDir` / `conf/lang/` / `./conf/app.conf` / Docker / spug 脚本 / 启动验证 | 2~3 天 | 是           |


**Round 2 总工期**：2~~4 周（含测试与联调），比轻量方案的 1~~2 周长约 1~~2 周，但**换来最终形态**，Round 3~~4 无重复搬迁。

#### 迁移风险

- **模板/静态路径硬编码分散**：`commands/command.go`、`controllers/BlogController.go`、`controllers/SettingController.go`、`models/BookResult.go` 都有 `conf.WorkingDirectory` 拼路径，PR-2 要一并处理。
- `i18n.SetMessage(lang, "conf/lang/...")` **硬编码**：`commands/install.go:106`、`commands/command.go:280` 各一处，必改。
- **Session 存了** `models.Member` **结构体**（`controllers/BaseController.go`）：改包路径后旧 session 反序列化会失败，Round 2 上线前需要清 session 或在 `SetMember` 加 version 字段做降级。
- `gob.Register` **硬编码类型**（`commands/command.go:113-115`）：包路径变了 gob 类型名会变，缓存里旧数据反序列化会崩，Round 2 上线前需清缓存。
- **Docker / spug / systemd 脚本**：`Dockerfile`、`start.sh`、`sync_host.sh`、`doc.service`、`spug_run.sh` 都假设从根目录启动、`./conf/app.conf`、`./views/`、`./static/`；PR-2 全部改到新路径。
- `WorkingDirectory` **兼容性**：现有 `--workDir` 参数指向的目录里必须包含 `configs/`、`web/`、`runtime/` 等新目录结构；老部署环境升级时要一起搬。
- **PR 大小控制**：PR-1 会触碰几乎每个 Go 文件（import 改写），review 只能看 diff summary，**必须先在个人分支跑完编译 + 冒烟测试再合**。

---



### 2.3 目标三：配置模块优化



#### 现状问题

1. **代码和配置文件混一层**：`conf/enumerate.go`（401 行）+ `mail.go`（Go）+ `app.conf`（253 行）+ `app.conf.example` + `lang/` 全在 `conf/`。
2. `app.conf` **未做分组**：253 行大平铺，只靠注释分块。
3. **配置访问方式割裂**：全部通过 `web.AppConfig.DefaultString("xxx", ...)`，散落在 30+ 文件里，**没有强类型 struct**。
4. `enumerate.go` **三合一**：常量 + 配置读取器（30+ Getter）+ URL 工具函数挤在同一文件。
5. `.example` **与真实** `.conf` **靠人肉同步**。
6. **多份** `AppConfig` **调用**忽略了 err，且无缓存。
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

**Step 2：单文件内 section 分组**（本轮先做；不拆多文件）

> **决策（2026-07-29）**：本轮**不**拆 `conf.d/` 多文件方案，先把 253 行平铺的 `app.conf` 通过 beego ini 原生的 `[section]` 语法做**同文件分组**。理由：① beego `web.AppConfig` 天然支持 `section::key` 语法，改动量小；② 单文件仍便于运维、diff、Docker 挂载；③ 拆多文件后维护 include 顺序、部署脚本、样例同步都更重，收益不明显；④ Step 3 上强类型 `Config` struct 后，无论单文件还是多文件对调用方都是透明的。

改造后的 `configs/app.conf` 骨架：

```ini
# ---- 根配置（不属于任何 section） ----
appname  = doc
runmode  = ${DOC_RUN_MODE||dev}
httpport = ${DOC_HTTP_PORT||8181}
baseurl  = ${DOC_BASE_URL||}

[session]
sessionprovider = ${DOC_SESSION_PROVIDER||file}
sessionname     = mindoc_id
...

[database]
db_adapter = ${DOC_DB_ADAPTER||mysql}
db_host    = ${DOC_DB_HOST||127.0.0.1}
...

[cache]
cache_provider = ${DOC_CACHE_PROVIDER||file}
...

[mail]
enable_mail = ${DOC_ENABLE_MAIL||false}
smtp_host   = ${DOC_SMTP_HOST||smtp.163.com}
smtp_port   = ${DOC_SMTP_PORT||25}
...

[upload]
upload_file_size = ${DOC_UPLOAD_FILE_SIZE||10M}
...

[log]
log_level = ${DOC_LOG_LEVEL||info}
log_path  = ${DOC_LOG_PATH||./runtime/logs}
...

[ldap]
ldap_enable = ${DOC_LDAP_ENABLE||false}
...

[dingtalk]
dingtalk_enable = ${DOC_DINGTALK_ENABLE||false}
...

[oauth]                          # 【新增】微信/企微/Google（如需）
...

[export]
export_process_num = ${DOC_EXPORT_PROCESS_NUM||1}
...

[cdn]
cdn_url = ${DOC_CDN_URL||}
...

[i18n]
i18n_default_lang = ${DOC_I18N_DEFAULT_LANG||zh-cn}

[mcp]                            # 【新增】MCP 配置
mcp_enable         = ${DOC_MCP_ENABLE||false}
mcp_listen         = ${DOC_MCP_LISTEN||127.0.0.1:8280}
mcp_stdio_member   = ${DOC_MCP_STDIO_MEMBER||admin}
mcp_token_required = ${DOC_MCP_TOKEN_REQUIRED||true}
mcp_rate_limit     = ${DOC_MCP_RATE_LIMIT||60}
```

调用方兼容期用 `web.AppConfig.DefaultString("mcp::mcp_enable", "false")` 之类语法，后续通过 Step 3 强类型 struct 统一收敛。

**未来可选**：若配置项进一步膨胀（比如超过 500 行、跨环境覆盖变复杂），再评估拆到 `configs/conf.d/*.conf` 多文件按字典序 merge。本轮不做。

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

- 引入 `[github.com/eko/gocache/v3](https://github.com/eko/gocache)`，支持 Chain (memory + redis)、Loader、Metrics 一站式
- 或用 `github.com/allegro/bigcache` + `go-redis` 手工组合

**Redis 客户端升级**：`beego/cache/redis` → `redis/go-redis/v9`（indirect 已存在，直接改）。

#### 2.4.2 Model 层

**现状问题**

1. `beego/orm` **维护缓慢**，beego v2 也基本停更。
2. 大文件：`BookModel.go` 34KB、`BookResult.go` 23KB、`Member.go` 17KB。
3. **无 Repository 层**：controller 直接调 `models.NewDocument().Find(id)`，事务/缓存/测试都难做。
4. **手写 SQL 硬编码表前缀**：`models/Member.go:77` 里 `o.Raw("select * from md_members where ...")` 写死了 `md_`。
5. `commands/migrate/` 是自研简易迁移器，功能有限。
6. `ioutil.ReadFile` **/** `ioutil.WriteFile` **deprecated API** 在 12 个文件里还在用。
7. `interface{}` **遍布**（Go 1.18+ 应用 `any`）。
8. `models/Base.go` 全文只有 `type Model struct {}` — 未来空基类，可以直接删或做真正的通用能力。

**建议路线**


| 升级项            | 方案                                                                                                                                  | 工作量                    |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| ORM            | ① 稳妥：留 beego/orm，只封 Repository 层 ② 激进：换 `gorm.io/gorm` 或 `ent`（类型安全，事务/关联更好） ③ 折中：换 `sqlc`（写 SQL 生成 Go，性能最好）                        | ①1 周 / ②2~3 周 / ③1.5 周 |
| Migrations     | 换 `[golang-migrate](https://github.com/golang-migrate/migrate)` 或 `[goose](https://github.com/pressly/goose)`，SQL 文件放 `migrations/` | 2~3 天                  |
| deprecated API | 全局 `ioutil.ReadFile` → `os.ReadFile`；`ioutil.WriteFile` → `os.WriteFile`；`interface{}` → `any`                                      | 半天                     |
| 大 Model 拆分     | `BookModel.go` (34KB) 按功能拆：`book/model.go`、`book/query.go`、`book/tree.go`、`book/permission.go`                                      | 1 天                    |
| DTO/Result 分离  | `BookResult.go` / `MemberResult.go` / `DocumentSearchResult.go` 挪到 `internal/dto/`                                                  | 0.5 天                  |
| 表前缀硬编码修复       | 全局搜 `md_` 定位 raw SQL，改用 `conf.GetDatabasePrefix()`                                                                                  | 0.5 天                  |


---



## 三、支线：其他技术债


| 主题                         | 现状问题                                                                                     | 建议                                                                                                                                     | 优先级 |
| -------------------------- | ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- | --- |
| **日志**                     | `commands/RegisterLogger` 用 `beego/logs`，无结构化，`logs.Error("xx", err)` 拼字符串               | **换** `uber-go/zap`（业界事实标准 · 性能最佳 · 生态完善），结构化字段 + Sugared/非 Sugared 双 API + `zapcore.Core` 便于对接 OpenTelemetry / Sentry / Lumberjack 轮转 | 中   |
| **HTTP 框架**                | Beego v2 社区活跃度不如 echo/gin/fiber                                                          | 短期继续 beego；长期规划评估。**切走成本很大**（模板/session/orm 全绑定 beego），不建议现在动                                                                          | 低   |
| **错误处理**                   | `errors.New` + `fmt.Errorf` 混用，无错误分类；`if err != nil { c.JsonResult(6001, "系统内部错误") }` 遍地 | 引入 `pkg/errors` 或 `cockroachdb/errors`；定义 `BizError { Code; Msg }`；controller 统一 `WriteError(err)`                                     | 高   |
| **中间件**                    | `middleware/filter.go` 和 `routers/filter.go` 两处分裂                                        | 合并到 `middleware/`：`auth.go` / `logger.go` / `recover.go` / `csrf.go` / `ratelimit.go`                                                  | 中   |
| **限流/防刷**                  | 无                                                                                        | `golang.org/x/time/rate` 或 `uber-go/ratelimit`，API 接口层做                                                                                | 中   |
| **API 文档**                 | 无 OpenAPI/Swagger                                                                        | `swaggo/swag` 注释生成，或迁到 `huma`                                                                                                          | 中   |
| **验证码**                    | `lifei6671/gocaptcha` 老库，字体路径硬编码                                                         | 换 `mojocn/base64Captcha`（前端直接 base64）                                                                                                  | 低   |
| **测试**                     | 全项目**未见** `_test.go`                                                                     | 至少给 `pkg/`* 和 `internal/service/*` 补测试，用 `testify`                                                                                     | 高   |
| **i18n**                   | `beego/i18n v0.0.0-20161101132742-e9308947f407` **11 年前**老包                              | 换 `nicksnyder/go-i18n/v2`（toml/json，支持复数）                                                                                              | 中   |
| **gopool**                 | 自研 `utils/gopool/`，简单包装                                                                  | 换 `panjf2000/ants`（业界事实标准）                                                                                                             | 低   |
| **requests**               | 自研 `utils/requests/`                                                                     | 换 `go-resty/resty`                                                                                                                     | 低   |
| `main.go` **优化**           | 手写 `os.Args[1] == "service"` 判断                                                          | 换 `spf13/cobra` + `spf13/viper`（可一起替代配置解析）                                                                                             | 高   |
| **BaseController.Prepare** | 每次请求读 `models.NewOption().All()` 全表                                                      | 加 5 分钟内存缓存或启动时装载，变更时清缓存                                                                                                                | 高   |
| **session**                | `sessionprovider=file`（默认）                                                               | 生产建议 `redis`；serializer 用 msgpack                                                                                                      | 中   |
| **CORS / 安全头**             | 未见 CORS、CSP、HSTS 配置                                                                      | `middleware/secure.go` 统一处理                                                                                                            | 中   |
| **Docker**                 | Dockerfile 6KB，Ubuntu focal 基础镜像                                                         | 多阶段 + `distroless` 或 `alpine`，镜像大小从几百 MB 降到 40MB 内                                                                                     | 中   |


---



## 四、支线：前端与静态资源

**现状**（`static/`）

- Bootstrap 3.2（2014 年，官方已 EOL）
- jQuery 全家桶（jquery / jstree / layer / nprogress / select2 / cropper / webuploader / respond.js / html5shiv）
- ✅ editor.md **已升 v1.7.17**（历史 PR 完成）
- ✅ katex **已修复 404**（历史 PR 完成）
- ✅ mermaid **已升 10.x**（历史 PR 完成）
- Vue.js 2

> ⚠️ **本路线图不再列 P0 前端修复项**。editor.md / katex / mermaid 三项已在历史 PR 中完成，若后续再次生成 roadmap 或做代码扫描，**不要**把它们当作待办重新列入。相关状态见 [upstream-mindoc-checklist.md §2.1](./upstream-mindoc-checklist.md) 的历史记录。

**分阶段方案**


| 阶段         | 内容                                                                               | 工作量       |
| ---------- | -------------------------------------------------------------------------------- | --------- |
| ~~**P0**~~ | ~~修 katex 404、editor.md 升 v1.7.17、mermaid 升 10.x~~ ✅ **已完成（历史 PR）**              | ~~0.5 天~~ |
| **P1**     | 静态资源加版本号（现有 `cdnjs "..." "version"` 机制铺开）；删 `respond.js` / `html5shiv`（不再支持 IE8） | 1~2 天     |
| **P2**     | 引入前端构建工具（Vite），vendor 集中管理；抽离 `views/*.tpl` 里的内联 JS                              | 1~2 周     |
| **P3**     | 逐步替换：Bootstrap 3 → Bootstrap 5 或 Tailwind；jQuery 组件 → Vue 3 组件（增量迁移，不用一次性 SPA 化） | 3~4 周     |
| **P4**     | 后端只做 API + 模板；`web-ui/` 用 Vue 3 + TypeScript + Vite 做完整 SPA；老模板路由保留兼容期           | 长期        |


---



## 五、实施顺序（四轮迭代）

> **优先级说明（2026-07）**：
>
> - **MCP 保留在 Round 3**。MCP 只硬依赖 Round 1（cobra 子命令 / cache 抽象 / 错误处理基础），软依赖 Round 2 的强类型 config。让 Round 2 先完成一步到位的 `cmd/`+`internal/` 目录搬迁 + 强类型 config，Round 3 的 `mcp/` 包直接写在最终目录 `internal/mcp/` 下，**零重复搬迁**，且 MCP 可直接使用 `config.Global.MCP.XXX`。
> - `BaseController.Prepare` 缓存（§三 高优先级）从 Round 2 提到 Round 1，纯性能优化、风险低、且不阻塞任何后续项。
> - Round 1 加入 **错误处理基础（**`BizError` **+** `JsonError` **helper）**，为 Round 3 的 MCP 工具统一错误返回铺路（也让 Round 2 目录搬迁时顺带把老 controller 的错误返回收敛）。
> - Round 1 新增 `cobra`，正是为了 Round 3 的 `doc mcp` 子命令。



### 🥇 Round 1：低风险重构 + 后续轮次前置准备（1 周）

- [x] **配置 Step 1+5**：`configs/` 目录独立 + 修 `smtp_host` / `smtp_port` 双引号 bug
- [x] `ioutil` **全局替换** `os.ReadFile` **/** `os.WriteFile`（12 文件）
- [x] `interface{}` **→** `any`（Go 1.18+）
- [x] `main.go` **用** `cobra`（为 Round 3 的 `doc mcp` 子命令铺路）
- [x] **缓存方案 A**：`cache.Cache` 抽接口 + `NullCache/MemoryCache/RedisCache/FileCache` 独立文件 + 加 `context` 传递
- [x] `BaseController.Prepare` **加 options 缓存**（§三 高优先级；从 Round 2 提前，纯性能优化不阻塞任何后续）
- [x] **错误处理基础**：`internal/errs/` 定义 `BizError{Code, Msg}` + `controllers/base` 加 `JsonError(err)` helper（§三 高优先级；为 Round 3 MCP 工具错误返回铺路）



**风险：** 低。全部是内部重构，对用户零感知。

### 🥈 Round 2：目录结构调整（一步到位激进）+ 配置强类型（2~4 周）

> 内部收拾轮次。**一步到位** `cmd/` + `internal/` 布局，见 §2.2。为 Round 3 MCP 落地打好最终目录形态与配置基础。

- [ ] **PR-1 目录搬迁 + import 改写**：`cmd/doc/main.go` + `internal/`** + `configs/` + `web/` + `deploy/`（`conf/` 彻底拆解到 `internal/config/` 与 `configs/`）
- [ ] **PR-2 路径硬编码 + 部署脚本**：`ViewsPath` / `StaticDir` / `conf/lang/` / `./conf/app.conf` / Docker / spug / systemd 全部改到新路径
- [ ] **配置 Step 2+3+4**：`configs/app.conf` 内部 `[section]` 分组 + 强类型 `config.Config` struct（`internal/config/`）+ `.env` 支持（含 `[mcp]` section 占位，Round 3 直接使用）
- [ ] `internal/router/` **按域拆分**（对齐 router-split-migration-plan.md）
- [ ] **中间件合并**：`middleware/filter.go` + `routers/filter.go` → `internal/middleware/`
- [ ] 预留 `internal/mcp/` 与 `internal/dto/mcpdto/` 空目录（Round 3 直接写入，无重复搬迁）

**风险：** 中高。触碰几乎每个 Go 文件的 import；Docker/session/gob 缓存都需要同步。建议开专门的 `refactor/layout` 分支，PR-1 与 PR-2 分次合并，每次都通过 `go build` + Docker 构建 + 冒烟测试。详见 §2.2 迁移风险与 §六 风险 12~14。

### 🥉 Round 3：MCP + 搜索基础（2~3 周）

> 用户价值最高的一轮。**MCP 支持读写文档**，AI 助手直接接入。代码直接写在 Round 2 完成的最终目录（`internal/mcp/`），零重复搬迁。

- [ ] **搜索最小方案**（对齐 upstream-mindoc-checklist.md §1.1）：MySQL FULLTEXT / SQLite FTS5 + 标题加权
- [ ] **MCP MVP · stdio**：官方 `modelcontextprotocol/go-sdk` v1.x，10 个工具（4 读 + 6 写）
  - [ ] 读：`search_document` / `get_document` / `list_books` / `list_document_tree`
  - [ ] 写：`create_document` / `update_document_content`（乐观锁）/ `append_document_content` / `update_document_meta` / `release_document` / `delete_document`（`confirm: true`）
- [ ] `models/MemberApiToken.go` 新表 + 后台管理页（**不复用** `MemberToken`）
- [ ] **MCP Streamable HTTP + Bearer Token**：Beego 挂 `/mcp/`*，`golang.org/x/time/rate` 限流
- [ ] `internal/dto/mcpdto/`：工具 In/Out struct（为 Round 4 Repository/Service 分层铺路）
- [ ] `docs/mcp-integration.md`：Claude Desktop / Cursor stdio + HTTP 接入示例

**风险：** 中。MCP 是新增功能，不影响存量；写工具通过现有 `BookRole` + 乐观锁 + `confirm` 参数控制风险。搜索改动限于 `models/DocumentSearchResult.go` + 建索引。

### 🏅 Round 4：模型 / 日志 / 前端现代化（3~4 周，按需推进）

- [ ] **模型层**：`BookModel.go` (34KB) 拆解 + Repository 抽象 + `md_` 硬编码修复
- [ ] **日志换** `uber-go/zap` + 结构化字段（`zap.String` / `zap.Error` 等），保留 `beego/logs` 兼容 shim 作过渡
- [ ] `beego/i18n` **换** `nicksnyder/go-i18n/v2`
- [ ] **前端 P1~P2**：Vite 构建，vendor 集中化
- [ ] （可选）根据 Round 3 MCP 使用反馈，评估是否上倒排索引（bleve / meilisearch）

**风险：** 较高，但可拆多个小 PR。**ORM 迁移建议单独立项**，别混进来。

---



## 六、关键风险清单


| #   | 风险                                          | 触发场景                                                                                                                                           | 对策                                                                                                                                                   |
| --- | ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `beego/orm` 的 `gob` + `md_` 前缀假设深入很多代码      | `models/Member.go:77` 的 raw SQL 写死了 `md_members`；迁移 ORM 或改前缀会崩                                                                                 | 迁移前全局搜 `md_` 定位，统一走 `GetDatabasePrefix()`                                                                                                            |
| 2   | Session 里存了 `models.Member` 结构体             | `BaseController.go:49`；改 Member 字段/加字段要评估旧 session 反序列化兼容性                                                                                     | `SetMember` 里塞 version 字段，异常时降级重登                                                                                                                    |
| 3   | `init()` 副作用重                               | `commands/command.go` + `enumerate.go` 都有 `init()`；重构时可能踩到"包变量在 `init` 里读了还没初始化的配置"                                                            | 重构时理清初始化顺序，最好显式 `Init(cfg *Config)`                                                                                                                  |
| 4   | `config_auto_delay` 热加载                     | `commands/command.go:468-512`；重构配置时要保留或明确宣布废弃                                                                                                  | Round 2 完成时明确表态                                                                                                                                      |
| 5   | `orm.DefaultRowsLimit = -1` 全局关掉了默认分页       | `commands/command.go:39`；重构 model 时不要依赖默认                                                                                                      | 显式在每个 Query 里写 `Limit()`                                                                                                                             |
| 6   | Beego `web.BConfig.WebConfig.ViewsPath` 硬编码 | `commands/command.go:345-347`；目录变动要同步                                                                                                          | Round 2 迁移时统一改                                                                                                                                       |
| 7   | 前端 vendor 无版本管理                             | 升级/回退困难                                                                                                                                        | Round 4 引入 Vite 时用 npm/pnpm 管起来                                                                                                                      |
| 8   | **AI 通过 MCP 批量误删/误覆盖文档**                    | `delete_document` 若无保护，AI 幻觉可导致成片文档消失                                                                                                          | ① 强制 `confirm: true` 参数；② 每分钟删除/写入次数限流（`golang.org/x/time/rate`）；③ 写工具保存前存快照到 `DocumentHistory`；④ Round 3 上线前先在测试项目跑                                 |
| 9   | **AI 与人同时编辑同一文档**                           | AI 覆盖人的未保存改动，或反之                                                                                                                               | `update_document_content` 强制带 `expect_version`（对应 `Document.Version` 时间戳）做乐观锁；版本不匹配返回 `VERSION_CONFLICT`，AI 侧 `get_document` 后重试                     |
| 10  | **MCP API Token 泄露**                        | Token 一旦泄露，AI 侧任何写权限都可能被滥用                                                                                                                     | ① 数据库只存 `sha256(token)`，不存明文；② 支持 `expires_at` 和一键撤销；③ 记录 `last_used_at`，异常访问可审计；④ HTTP 强制 HTTPS（部署要求）                                               |
| 11  | **误将** `MemberToken` **当 API Token 用**      | `MemberToken` 是邮箱验证码用途，含 `Email`/`SendTime`/发送次数限制，用作 API Token 会破坏原有找回密码逻辑                                                                    | 明确新建 `member_api_tokens` 表（见 §2.1）；两张表职责分离                                                                                                           |
| 12  | **Round 2 一步到位迁移触碰几乎每个文件**                  | `cmd/`+`internal/` 激进方案导致全仓 import 改写；PR 巨大，review 只能看 diff summary                                                                            | ① PR-1 只搬目录 + `goimports`，不改任何逻辑；② PR-2 单独处理路径硬编码与部署脚本；③ 每个 PR 合并前跑通 `go build` + Docker 构建 + 冒烟测试；④ 开专门 `refactor/layout` 分支，避免与业务开发冲突              |
| 13  | **gob 缓存与 session 反序列化在包路径变更后失败**           | `commands/command.go:113-115` `gob.Register` 硬编码类型名 = `包路径.类型名`；`controllers/BaseController.go` session 里存了 `models.Member`；Round 2 后旧数据反序列化会崩 | ① Round 2 上线前发布 note 明确要求**清** `cache/` **目录 + 清 session store**；② `SetMember` 加 version 字段做降级自动重登；③ 有条件的话在 Round 1 就把 gob 换 msgpack/json，从根上避免绑定包路径 |
| 14  | **老部署环境** `--workDir` **目录结构不匹配**           | 现有 `--workDir` 指向的目录假设根下有 `conf/`、`views/`、`static/`；Round 2 后应该有 `configs/`、`web/views/`、`web/static/`                                        | ① 部署脚本升级步骤加入"目录结构迁移"检查；② `commands/install.go` 首次启动检测新结构并给出清晰错误提示；③ 更新 `README.md` 与 `docs/mcp-integration.md` 的部署示例                                 |


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
- [ ] `BaseController.Prepare` options 缓存
- [ ] `internal/errs/` + `BizError` + `JsonError` helper



### Round 2（一步到位 cmd/+internal/ + 配置强类型）

- [ ] PR-1 目录搬迁到 `cmd/doc/` + `internal/**` + `configs/` + `web/` + `deploy/`
- [ ] PR-1 `conf/` 拆解：Go → `internal/config/`；配置文件 → `configs/`；`lang/` → `configs/lang/`
- [ ] PR-1 全仓 30+ import 路径改写
- [ ] PR-2 `ViewsPath` / `StaticDir` / 字体路径 / 导出资源拷贝路径修正
- [ ] PR-2 `conf/lang/` i18n 硬编码修正（`commands/install.go`、`command.go`）
- [ ] PR-2 Docker / spug / systemd / `start.sh` / `sync_host.sh` 路径修正
- [ ] `configs/app.conf` 内部 `[section]` 分组（不拆多文件）
- [ ] 强类型 `config.Config` struct（含 `MCPConfig` 段占位）
- [ ] `.env` 支持
- [ ] `internal/router/` 按域拆分
- [ ] `internal/middleware/` 合并原 middleware + routers/filter.go
- [ ] 预留 `internal/mcp/` 与 `internal/dto/mcpdto/` 空目录
- [ ] Session/gob 缓存清理提示写入部署 note



### Round 3（MCP + 搜索）

- [ ] 搜索 FULLTEXT/FTS5 + 标题加权
- [ ] MCP stdio · 4 个读工具（`search_document` / `get_document` / `list_books` / `list_document_tree`）
- [ ] MCP stdio · 6 个写工具（`create_document` / `update_document_content` / `append_document_content` / `update_document_meta` / `release_document` / `delete_document`）
- [ ] `models/MemberApiToken.go` + 后台管理页
- [ ] MCP Streamable HTTP + Bearer Token + 限流
- [ ] `internal/dto/mcpdto/`
- [ ] `docs/mcp-integration.md`



### Round 4

- [ ] `BookModel.go` 拆分
- [ ] Repository 抽象
- [ ] `md_` 硬编码修复
- [ ] `zap` 日志（结构化字段 + Lumberjack 轮转）
- [ ] `nicksnyder/go-i18n/v2`
- [ ] Vite 前端构建

---



## 八、决策记录（Decision Log）


| 日期             | 决策项                                                                 | 决定                                                                        | 原因                                                                                                                                                                                                                                                                            |
| -------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-07-23     | 是否本轮切换 HTTP 框架？                                                     | 否，继续 beego v2                                                             | 模板/session/orm 全绑定 beego，切走成本远超收益                                                                                                                                                                                                                                             |
| 2026-07-23     | ORM 是否本轮迁移 gorm/ent？                                                | 否，先封 Repository 层                                                         | 减小 blast radius，Round 4 再评估                                                                                                                                                                                                                                                   |
| 2026-07-23     | 配置文件分组是否引入 viper？                                                   | 保留 beego `LoadAppConfig` + include 合并                                     | 减少依赖，`${ENV                                                                                                                                                                                                                                                                   |
| 2026-07-23     | MCP 是否本轮做 HTTP 模式？                                                  | 分两步（先 stdio，再 HTTP）                                                       | stdio 无鉴权顾虑，先解决 AI 接入这一刚需                                                                                                                                                                                                                                                     |
| 2026-07-23     | ~~目录结构是否走激进的~~ `cmd/` ~~+~~ `internal/` ~~方案？~~                     | ~~短期走~~ `server/` ~~+~~ `web/` ~~+~~ `deploy/`~~；~~`internal/` ~~作为长期目标~~ | ~~已有 frontend-backend-split-migration-plan.md 详细方案，避免朝令夕改~~                                                                                                                                                                                                                   |
| **2026-07-29** | **目录结构：一步到位 vs 分阶段？**                                               | **一步到位** `cmd/` **+** `internal/` **激进方案**                                | ① 分阶段先轻量再激进等于**搬两次家**；② MCP（Round 3）从第一天就落在 `internal/mcp/` 最终位置；③ `conf/` 一次拆到位（Go → `internal/config/`；配置 → `configs/`），避免长期"半激进"状态；④ Round 2 从 1~~2 周延到 2~~4 周，换来 Round 3~4 零重复搬迁；⑤ frontend-backend-split-migration-plan.md 中的路径修正细节仍适用，但目标目录换成本方案                        |
| 2026-07-23     | **MCP SDK 选型？（mark3labs/mcp-go vs 官方 modelcontextprotocol/go-sdk）** | **直接用官方** `modelcontextprotocol/go-sdk` **v1.x**                          | ① 官方已到 v1（semver 稳定），mcp-go 仍是 v0.x，升级本身就有 API 破坏性风险；② 官方跟 MCP spec 最快（已跟到 2026-07-28）；③ 与 Round 4 的 Repository/DTO 分层天然契合（struct schema = DTO）；④ 避免未来从 mcp-go 迁移的双份成本                                                                                                        |
| 2026-07-23     | **MCP 是否支持写入文档？**                                                   | **是**，MVP 就要 6 个写工具                                                       | 用户核心诉求：AI 助手要能创建/更新文档；通过 `BookRole ≥ Editor` + 乐观锁 `expect_version` + `delete_document` 强制 `confirm: true` 控制风险                                                                                                                                                               |
| 2026-07-23     | **MCP Bearer Token 是否复用** `MemberToken` **表？**                      | **否**，新建 `member_api_tokens` 表                                            | `MemberToken` 是邮箱验证码用途（`Email`/`SendTime`/发送次数限制），职责不同；两表分离避免破坏找回密码逻辑                                                                                                                                                                                                         |
| 2026-07-23     | **四轮优先级：MCP 与目录调整孰先？**                                              | **MCP 保留在 Round 3，目录调整放 Round 2**                                         | ① MCP 只硬依赖 Round 1（cobra / cache / 错误处理），软依赖 Round 2 的强类型 config；② 让 Round 2 先完成一步到位的 `cmd/`+`internal/` 目录搬迁，Round 3 的 `mcp/` 包直接写在 `internal/mcp/` 最终位置，**零重复搬迁**；③ 用户价值仅推迟 2~3 周，换 MCP 代码从第一天就在正确目录下                                                                       |
| **2026-07-29** | **配置文件本轮是否拆多文件？**                                                   | **否**，`configs/app.conf` 内部 `[section]` 分组即可；`conf.d/` 多文件方案作为**未来可选**    | ① beego ini 原生支持 `[section]` + `section::key`，改动量最小；② 单文件便于运维/diff/Docker 挂载；③ Step 3 上强类型 `Config` struct 后，无论单/多文件对调用方透明；④ 现有配置 253 行，未到拆文件收益门槛                                                                                                                             |
| **2026-07-29** | **日志库选型：slog vs zap？**                                              | `uber-go/zap`                                                             | ① 业界事实标准，性能最佳（分配数最少）；② 生态最完善：Lumberjack 轮转 / Sentry / OpenTelemetry 桥接现成；③ Sugared/非 Sugared 双 API 便于从 `beego/logs` 渐进式迁移（先用 Sugared 保留 printf 风格，再逐步换成 zap.Field）；④ `slog` 是标准库但社区桥接仍在补齐，且团队后续接入 APM/告警系统时 zap 生态更省事；⑤ Round 1 已经引入 `cobra` 等新依赖，多一个 `go.uber.org/zap` 边际成本低 |
| **2026-07-29** | **前端 P0 修复项（editor.md / katex / mermaid）是否再做？**                     | **不做**，已在历史 PR 完成                                                         | ① editor.md 已升 v1.7.17；② katex 404 已修；③ mermaid 已升 10.x；④ **本路线图后续任何一轮不再重复列入**，避免下次生成/审阅时又把它们当待办；如未来因回归再次出现，另开新条目                                                                                                                                                             |


---



## 九、附录



### 9.1 相关外部资料

- [Go Project Layout Standards](https://github.com/golang-standards/project-layout)
- [Model Context Protocol 规范](https://modelcontextprotocol.io/)
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) — 官方 Go SDK（本项目采用）
- [MCP Go SDK Quick Start](https://go.sdk.modelcontextprotocol.io/quick_start/)
- [redis/go-redis](https://github.com/redis/go-redis)
- [nicksnyder/go-i18n](https://github.com/nicksnyder/go-i18n)
- [spf13/cobra](https://github.com/spf13/cobra) + [viper](https://github.com/spf13/viper)
- [uber-go/zap](https://github.com/uber-go/zap) — 日志（本项目采用）+ [natefinch/lumberjack](https://github.com/natefinch/lumberjack) 轮转



### 9.2 本文档与其他文档的关系

```text
refactor-roadmap.md（本文，总纲）
│
├─► 【每轮执行文档】具体到文件/行号/命令的可执行分解
│   ├─► round-1-execution-plan.md    （Round 1 · T1~T7 详细步骤 + PR 拆分）
│   ├─► round-2-execution-plan.md    （Round 2 · PR-1/PR-2 + 目录映射总表）
│   ├─► round-3-execution-plan.md    （Round 3 · MCP 10 工具 + Bearer + 搜索）
│   └─► round-4-execution-plan.md    （Round 4 · T1~T12 按需推进）
│
├─► 【参考文档】历史决策与横向清单
│   ├─► frontend-backend-split-migration-plan.md   （目标二 · 硬编码定位表，Round 2 引用）
│   ├─► router-split-migration-plan.md              （目标二 · 路由拆分详情，Round 2 T6 引用）
│   ├─► routers-reference.md                        （现有路由分类参考）
│   └─► upstream-mindoc-checklist.md                （功能特性对齐 · MCP/搜索/前端等）
```

**阅读顺序建议：**

1. 新加入项目：先读本文 §一~五（现状 + 四大目标 + 迭代计划）
2. 开始某一轮：直接跳到对应 `round-N-execution-plan.md`，逐 T 执行
3. 遇到具体文件/硬编码：查 `frontend-backend-split-migration-plan.md` 附录 A/B 或 `routers-reference.md`
4. 决策疑问：查本文 §八 决策日志

