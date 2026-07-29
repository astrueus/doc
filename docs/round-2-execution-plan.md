# Round 2 · 执行文档（`cmd/` + `internal/` 一步到位 + 强类型 Config）

> 本文是 [refactor-roadmap.md §五 Round 2](./refactor-roadmap.md#🥈-round-2目录结构调整一步到位激进--配置强类型2~4-周) 的**可执行分解**。
> 目标：**一次性**把项目搬到 `cmd/` + `internal/` 最终形态（不再走 `server/`+`web/`+`deploy/` 过渡）；同时把 `configs/app.conf` 做 `[section]` 分组 + 上强类型 `config.Config`。为 Round 3 MCP 的 `internal/mcp/` 提供最终目录与配置基础。
> **相关文档：** [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md)（附录 A/B 的硬编码定位表**仍适用**，只是目标目录换到本文的 `web/` + `internal/`）；[router-split-migration-plan.md](./router-split-migration-plan.md)（`internal/router/` 拆分）。

---



## 一、范围与不做清单



### 本轮做


| 序号  | 任务                                                                | 影响面                                                                   | 上线感知                          |
| --- | ----------------------------------------------------------------- | --------------------------------------------------------------------- | ----------------------------- |
| T1  | 目录搬迁 · `cmd/doc/` + `internal/**` + `web/` + `deployments/`（PR-1） | 几乎每个 Go 文件（import 改写）                                                 | 需清 session + 清 gob 缓存（见 §七）   |
| T2  | 路径硬编码 + 部署脚本（PR-2）                                                | `commands/`、`controllers/`、`models/BookResult.go`、Docker/spug/systemd | 部署脚本必须同步                      |
| T3  | `configs/app.conf` 内部 `[section]` 分组                              | `configs/app.conf` + `.example`                                       | 无（beego `section::key` 兼容）    |
| T4  | 强类型 `config.Config` struct + `Load()`                             | 新增 `internal/config/config.go`；30+ 调用位改写                              | 无                             |
| T5  | `.env` 支持（`joho/godotenv`）                                        | 新增依赖；`cmd/doc/main.go` 启动时 load                                       | 无（可选）                         |
| T6  | `internal/router/` 按域拆分                                           | 原 `routers/router.go` → 5~6 个域文件                                      | 无                             |
| T7  | `internal/middleware/` 合并                                         | 原 `middleware/filter.go` + `routers/filter.go`                        | 无                             |
| T8  | 预留 `internal/mcp/` + `internal/dto/mcpdto/` 空目录                   | 空 `.gitkeep` + `doc.go`                                               | 无                             |
| T9  | Session/gob 缓存清理写入部署 note                                         | `README.md` + CHANGELOG                                               | 现网升级需**清 session + 清 cache/** |




### 本轮**不做**（明确排除）

- ❌ **不拆 Controller/Model 内部大文件**：`DocumentController.go` (37KB) / `BookModel.go` (34KB) **只搬目录、不拆域**，减小 blast radius。域拆分放 Round 4。
- ❌ **不换 ORM/迁移器**：`beego/orm` 保留；Repository/Service 层不落地（Round 4）。
- ❌ **不换日志**：`beego/logs` 保留（Round 4 换 zap）。
- ❌ **不上 MCP**：只**预留** `internal/mcp/` 空目录，代码 Round 3 写。
- ❌ **不拆** `configs/` **为多文件**：只做单文件 `[section]` 分组（见 [refactor-roadmap.md §八 决策 2026-07-29](./refactor-roadmap.md#八决策记录decision-log)）。

---



## 二、前置条件

- ✅ Round 1 全部完成，尤其：
  - `configs/` 目录已独立
  - `spf13/cobra` 已引入（本轮的 `cmd/doc/main.go` 直接用）
  - `internal/errs/` 已存在
  - `cache.Cache` 接口已存在
- 分支：`refactor/round-2-layout`（长分支，PR-1、PR-2 顺序合入）
- 建议**冻结** `main` 分支的 controllers/ models/ routers/ 变更 1~2 天，避免与目录搬迁产生冲突

---



## 三、目标目录形态（最终版）

```
doc/
├─ cmd/
│  └─ doc/
│     └─ main.go                              # 只做 cobra 入口，body ≤ 20 行
├─ internal/                                  # Go 私有包
│  ├─ app/                                    # 装配层（原 commands/ 的初始化部分）
│  │  ├─ app.go                               # ResolveCommand（当前 commands.RegisterCommand 的核心）
│  │  ├─ bootstrap.go                         # 初始化顺序：config → db → cache → i18n → orm
│  │  └─ web.go                               # 启动 beego web
│  ├─ cli/                                    # cobra 子命令（原 commands/root.go/web.go/install.go/version.go 移入）
│  ├─ config/                                  # 【关键】强类型 Config，原 conf/enumerate.go+mail.go 的 Go 部分
│  │  ├─ config.go                             # Config struct + Load/Reload
│  │  ├─ working_dir.go                        # 原 ResolveWorkingDirectory
│  │  └─ enum.go                               # 原 enumerate.go 里的常量与 URL 工具
│  ├─ controller/                              # ★ Round 2 只搬进来，**不按域再拆子目录**（Round 4 做）
│  │  └─ (所有原 controllers/*.go 平搬)
│  ├─ model/                                   # ★ 同上，只搬不拆
│  │  └─ (所有原 models/*.go 平搬)
│  ├─ dto/                                     # 原 *Result.go 挪进来（同一 PR 内 rename import）
│  │  ├─ book_result.go / member_result.go / document_search_result.go / ...
│  │  └─ mcpdto/                               # 【空目录 · Round 3 写入】
│  ├─ middleware/                              # 合并 middleware/filter.go + routers/filter.go
│  ├─ router/                                  # 按域拆分（见 T6）
│  │  ├─ router.go                             # 汇总入口
│  │  ├─ account.go / manager.go / document.go / book.go / blog.go / api.go
│  ├─ cache/                                   # 原 cache/（内部升级放 Round 4）
│  ├─ converter/                               # 原根目录 converter/（电子书导出）
│  ├─ errs/                                    # Round 1 已建好，import 路径不变（本轮 rename import）
│  └─ mcp/                                     # 【空目录 · Round 3 写入】
├─ pkg/                                        # 可对外复用工具
│  ├─ cryptil / filetil / pagination / requests / gob / krand / password / template_fun
│  ├─ graphics/                                # 原根目录 graphics/（图片裁剪/缩放）
│  ├─ mail/                                    # 原根目录 mail/（SMTP；模板仍在 web/views）
│  └─ (原 utils/ 里通用部分)
├─ configs/                                    # Round 1 已建好；本轮新增 [section] 分组
│  ├─ app.conf / app.conf.example / app.conf.dev.example / app.conf.prod.example
│  └─ lang/  (zh-cn.ini / en-us.ini)
├─ web/                                        # 前端资源
│  ├─ static/                                  # ← 从根 static/ 迁入
│  │  ├─ vendor/ (24 个第三方库)
│  │  ├─ css/ js/ images/ fonts/ editors/
│  └─ views/                                   # ← 从根 views/ 迁入
├─ deployments/                                # ← Docker / spug / systemd 集中
│  ├─ Dockerfile / docker-compose.yml
│  ├─ start.sh / sync_host.sh
│  ├─ systemd/doc.service
│  └─ spug/  (spug 脚本)
├─ scripts/                                    # build.sh / build.bat（保留）
├─ docs/                                       # 保留
├─ runtime/  uploads/                           # 运行时数据（文件缓存/导出在 runtime/cache/）
├─ go.mod / go.sum / README.md / LICENSE.md
└─ favicon.ico / simsun.ttc                    # 迁到 web/static/ 或保留根（评估中，见 T2）
```

**关键原则：**

1. `internal/controller/` 与 `internal/model/` **本轮只平搬**，`Round 4` 才按域拆子目录（`document/`、`book/` 等）。**不允许**在本轮同时拆域，否则 diff 无法 review。
2. `pkg/` 只放**没有业务耦合**的工具；如 `utils/gopool/` 被 model 用了但没耦合业务，也放 `pkg/`；`utils/dingtalk/` 是业务集成，放 `internal/thirdparty/dingtalk/`。
3. `internal/` 天然阻止外部 module 误引，符合 Go 官方 layout 建议。
4. 模块路径**保持** `git.itopcms.com/jackliu/doc`，只改子包 import。

---



## 四、原目录 → 新目录映射总表


| 原位置                                                                                                              | 新位置                                                                                                          | 备注                                                                        |
| ---------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------- |
| `main.go`                                                                                                        | `cmd/doc/main.go`                                                                                            | 本轮同时精简为 cobra 入口                                                          |
| `commands/root.go` / `web.go` / `install.go` / `version.go` / `mcp_stub.go`                                      | `internal/cli/`                                                                                              | Round 1 的 cobra stub 迁入                                                   |
| `commands/command.go`                                                                                            | 拆到 `internal/app/bootstrap.go` + `internal/app/web.go`                                                       | 也是本轮最大最脏的一个文件                                                             |
| `commands/daemon/`                                                                                               | `internal/cli/daemon/`（或保留 `cmd/doc/service.go`）                                                             | 与 `service install/remove` 兼容                                             |
| `commands/migrate/`                                                                                              | `internal/migrate/`                                                                                          | Round 4 或换 golang-migrate 时再改                                             |
| `commands/install.go`（`i18n.SetMessage` 硬编码 `conf/lang/`）                                                        | 改到 `internal/cli/install.go` + 路径改 `configs/lang/`                                                           | Round 1 T1 已改路径，本轮再改 import                                               |
| `conf/enumerate.go`                                                                                              | 拆到 `internal/config/config.go` + `internal/config/working_dir.go` + `internal/config/enum.go`                | Go 源码搬                                                                    |
| `conf/mail.go`                                                                                                   | `internal/config/mail.go`                                                                                    | 同上                                                                        |
| `conf/app.conf` / `.example` / `lang/`                                                                           | 已在 Round 1 T1 迁到 `configs/`，本轮**不动**                                                                         | —                                                                         |
| `controllers/*.go`                                                                                               | `internal/controller/*.go`                                                                                   | 只搬不拆域                                                                     |
| `models/*.go`                                                                                                    | `internal/model/*.go`                                                                                        | 只搬不拆域                                                                     |
| `models/*Result.go` (Book/Member/Attachment/Comment/DocumentSearch/ConvertBook/Blog)                             | `internal/dto/*_result.go`                                                                                   | 顺便 rename 为 snake_case                                                    |
| `middleware/filter.go` + `routers/filter.go`                                                                     | `internal/middleware/` 合并                                                                                    | 见 T7                                                                      |
| `routers/router.go` (148 行)                                                                                      | `internal/router/router.go` + `account.go` / `manager.go` / `book.go` / `document.go` / `blog.go` / `api.go` | 见 T6 与 [router-split-migration-plan.md](./router-split-migration-plan.md) |
| `cache/cache.go` / `cache_null.go` / `iface.go` / `beego_adapter.go` / `null.go`                                 | `internal/cache/`                                                                                            | Round 1 T5 新增文件一起迁                                                        |
| `utils/cryptil/` / `filetil/` / `pagination/` / `requests/` / `gopool/` / `sqltil/` / `ziptil/` / `wkhtmltopdf/` | `pkg/` 下同名                                                                                                   | 通用工具                                                                      |
| `utils/dingtalk/`                                                                                                | `internal/thirdparty/dingtalk/`                                                                              | 业务集成                                                                      |
| `utils/gob.go` / `krand.go` / `password.go` / `template_fun.go` / `url.go` / `html.go` / `ldap.go`               | `pkg/` 或 `internal/`（分文件评估）                                                                                  | 详见 T1 步骤 4                                                                |
| `static/`                                                                                                        | `web/static/`                                                                                                | URL `/static/*` 不变                                                        |
| `views/`                                                                                                         | `web/views/`                                                                                                 | URL 不变，需改 `ViewsPath`                                                     |
| `Dockerfile` / `docker-compose.yml` / `start.sh` / `sync_host.sh`                                                | `deployments/`                                                                                               | PR-2                                                                      |
| `favicon.ico`                                                                                                    | `web/static/favicon.ico`                                                                                     | beego 配置 favicon 路径                                                       |
| `simsun.ttc`                                                                                                     | `web/static/fonts/simsun.ttc`                                                                                | 若 `gocaptcha.SetFontPath` 引用                                              |
| `cache/` (根目录运行时)                                                                                                | **删除**（已并入 `runtime/cache/`）                                                                                 | 配置默认 `./runtime/cache/`；代码在 `internal/cache/`                             |
| `converter/`                                                                                                     | `internal/converter/`                                                                                        | 电子书导出；偏业务，放 `internal/`                                                  |
| `graphics/`                                                                                                      | `pkg/graphics/`                                                                                              | 通用图片裁剪/缩放（非「验证码素材」目录）                                                     |
| `mail/`                                                                                                          | `pkg/mail/`                                                                                                  | SMTP 客户端；邮件 HTML 模板仍在 `web/views/`                                       |
| `runtime/` / `uploads/`                                                                                          | 保留在根                                                                                                         | 运行时数据；文件缓存/导出落盘走 `runtime/cache/`                                             |


---



## 五、PR-1 · 目录搬迁 + import 改写（3~5 天）

> **原则：只搬目录 + 改 import，一行业务逻辑都不改。** 逻辑改动全部推到 PR-2 或后续 Round。



### T1 步骤

**Step 1：分支准备**

```powershell
git checkout -b refactor/round-2-layout-pr1 main
```

**Step 2：批量** `git mv`（一次性搬完，避免中途别人 rebase）

推荐用 PowerShell 脚本（可放到 `scripts/round2_move.ps1`）：

```powershell
# scripts/round2_move.ps1  —— 仅示意，需在 PR-1 前 review
New-Item -ItemType Directory -Force -Path cmd/doc, internal/app, internal/cli, internal/config, `
    internal/controller, internal/model, internal/dto, internal/dto/mcpdto, `
    internal/middleware, internal/router, internal/cache, internal/mcp, `
    internal/thirdparty/dingtalk, `
    pkg/cryptil, pkg/filetil, pkg/pagination, pkg/requests, pkg/gopool, pkg/sqltil, pkg/ziptil, pkg/wkhtmltopdf, `
    web/static, web/views, deployments

git mv main.go cmd/doc/main.go
git mv commands/root.go internal/cli/root.go
git mv commands/web.go internal/cli/web.go
git mv commands/install.go internal/cli/install.go
git mv commands/version.go internal/cli/version.go
git mv commands/mcp_stub.go internal/cli/mcp_stub.go
git mv commands/command.go internal/app/bootstrap.go   # ★ 大文件，先整体搬，PR-1 后续再拆
git mv commands/update.go internal/cli/update.go
git mv commands/daemon internal/cli/daemon
git mv commands/migrate internal/migrate

git mv conf/enumerate.go internal/config/enumerate_legacy.go   # PR-1 阶段先原样搬，T4 再重构
git mv conf/mail.go internal/config/mail.go

git mv controllers/*.go internal/controller/
git mv models/*.go internal/model/
# *Result.go 单独挪 dto/（如需 rename，本步骤保留原名，rename 放 PR-1 末尾统一改）
git mv internal/model/BookResult.go internal/dto/book_result.go
git mv internal/model/MemberResult.go internal/dto/member_result.go
git mv internal/model/AttachmentResult.go internal/dto/attachment_result.go
git mv internal/model/BlogResult.go internal/dto/blog_result.go
git mv internal/model/ConvertBookResult.go internal/dto/convert_book_result.go
git mv internal/model/DocumentSearchResult.go internal/dto/document_search_result.go
git mv internal/model/comment_result.go internal/dto/comment_result.go

git mv middleware/filter.go internal/middleware/legacy_filter.go
git mv routers/filter.go internal/middleware/router_filter.go
git mv routers/router.go internal/router/router.go             # 按域拆分放 T6

git mv cache/cache.go internal/cache/cache.go
git mv cache/cache_null.go internal/cache/cache_null.go
git mv cache/iface.go internal/cache/iface.go
git mv cache/beego_adapter.go internal/cache/beego_adapter.go
git mv cache/null.go internal/cache/null.go

git mv utils/cryptil/*   pkg/cryptil/
git mv utils/filetil/*   pkg/filetil/
git mv utils/pagination/* pkg/pagination/
git mv utils/requests/*  pkg/requests/
git mv utils/gopool/*    pkg/gopool/
git mv utils/sqltil/*    pkg/sqltil/
git mv utils/ziptil/*    pkg/ziptil/
git mv utils/wkhtmltopdf/* pkg/wkhtmltopdf/
git mv utils/dingtalk/*  internal/thirdparty/dingtalk/

# 单文件 utils：分文件评估
git mv utils/gob.go       pkg/gob/gob.go
git mv utils/krand.go     pkg/krand/krand.go
git mv utils/password.go  pkg/password/password.go
git mv utils/template_fun.go pkg/template_fun/template_fun.go
git mv utils/url.go       pkg/url/url.go
git mv utils/html.go      pkg/htmlutil/html.go
git mv utils/ldap.go      internal/thirdparty/ldap/ldap.go   # 业务耦合

# 前端资源
git mv static web/static
git mv views  web/views
git mv favicon.ico web/static/favicon.ico
git mv simsun.ttc  web/static/fonts/simsun.ttc

# 根目录遗留 Go 包（与 utils 同类，统一收编）
git mv converter internal/converter
git mv graphics  pkg/graphics
git mv mail      pkg/mail

# 部署
git mv Dockerfile deployments/Dockerfile
git mv docker-compose.yml deployments/docker-compose.yml
git mv start.sh deployments/start.sh
git mv sync_host.sh deployments/sync_host.sh
```

> ⚠️ **上面脚本仅供参考，实际执行时**：
>
> - **不要**用 `Move-Item`，必须 `git mv`（保留 rename 历史）
> - 每步先 `git status` 确认，再 commit
> - PowerShell 通配符行为 ≠ bash，`git mv controllers/*.go internal/controller/` 需先 `foreach` 展开

**Step 3：全仓 import 改写**

用 `goimports -r` 或 `gofmt -r` 批量替换（在 PR-1 一次搞定，避免多次触碰所有文件）：

```powershell
$replacements = @(
    @{ From = 'git.itopcms.com/jackliu/doc/conf';        To = 'git.itopcms.com/jackliu/doc/internal/config' },
    @{ From = 'git.itopcms.com/jackliu/doc/controllers'; To = 'git.itopcms.com/jackliu/doc/internal/controller' },
    @{ From = 'git.itopcms.com/jackliu/doc/models';      To = 'git.itopcms.com/jackliu/doc/internal/model' },
    @{ From = 'git.itopcms.com/jackliu/doc/routers';     To = 'git.itopcms.com/jackliu/doc/internal/router' },
    @{ From = 'git.itopcms.com/jackliu/doc/middleware';  To = 'git.itopcms.com/jackliu/doc/internal/middleware' },
    @{ From = 'git.itopcms.com/jackliu/doc/cache';       To = 'git.itopcms.com/jackliu/doc/internal/cache' },
    @{ From = 'git.itopcms.com/jackliu/doc/commands';    To = 'git.itopcms.com/jackliu/doc/internal/cli' },
    @{ From = 'git.itopcms.com/jackliu/doc/converter';  To = 'git.itopcms.com/jackliu/doc/internal/converter' },
    @{ From = 'git.itopcms.com/jackliu/doc/graphics';   To = 'git.itopcms.com/jackliu/doc/pkg/graphics' },
    @{ From = 'git.itopcms.com/jackliu/doc/mail';       To = 'git.itopcms.com/jackliu/doc/pkg/mail' },
    @{ From = 'git.itopcms.com/jackliu/doc/utils/cryptil';    To = 'git.itopcms.com/jackliu/doc/pkg/cryptil' },
    @{ From = 'git.itopcms.com/jackliu/doc/utils/filetil';    To = 'git.itopcms.com/jackliu/doc/pkg/filetil' },
    # ... 其余 utils 子包同上
    @{ From = 'git.itopcms.com/jackliu/doc/utils/dingtalk';   To = 'git.itopcms.com/jackliu/doc/internal/thirdparty/dingtalk' }
)
foreach ($r in $replacements) {
    Get-ChildItem -Recurse -Include *.go | ForEach-Object {
        (Get-Content $_.FullName -Raw) -replace [regex]::Escape($r.From), $r.To | Set-Content -NoNewline $_.FullName
    }
}
gofmt -w .
```

> ⚠️ `utils/gob`、`utils/krand` 等单文件是**同名包**，import 路径变化后可能与 `pkg/xxx` 的包名冲突，需要逐个 review 包名。

**Step 4：Result → DTO 的包名调整**

原 `models/BookResult.go` 里的 `package models`，搬到 `internal/dto/` 后要改 `package dto`。同时：

- 所有 `models.BookResult` 引用 → `dto.BookResult`
- 由于 `*Result` 挪出，`internal/model/` 里剩下的类型对 `dto` 有依赖时可能出现**循环 import**：
  - 常见路径：`model.Book.ToResult() dto.BookResult` → `dto` 依赖 `model` 类型 → 若 `model` 又调 `dto` 就循环
  - **本轮策略**：如出现循环，`ToResult()` 方法**暂留** `model` **包**（作为 `model` 到 `dto` 的转换），本轮不搬；或引入 `internal/mapper/` 单独放转换函数
- 建议在 PR-1 内**灰度做**：先只搬 5 个 Result，跑一次 build，没问题再搬剩下的

**Step 5：**`go build` **快照迭代**

```powershell
go build ./...           # 至少要能编译，编译不过就不能提交
go vet ./...
```

- 每消灭 20~30 个编译错误 commit 一次，方便 `git bisect`
- **别** 用 `--force-with-lease` 之类强推；PR-1 分支只有一个 reviewer 也不 push force

**Step 6：**`go.mod` **清理**

```powershell
go mod tidy
```

如果 `pkg/` 里的通用工具想给未来的其他 module 用，可以先只在本 module 内引用（Go 允许）。

### PR-1 验收

- [x] `go build ./...` 通过
- [x] `go vet ./...` 无告警
- [x] `./doc web`（或 `./doc`）能启动、首页 200
- [ ] `./doc install` 能走通
- [ ] Docker 镜像构建能通过（**PR-2 前**，Docker 里的路径仍是旧的，可能启动失败 → PR-2 才修）
- [x] 无循环 import；`go list ./...` 全通过



### PR-1 回滚

单 revert 该 merge commit 即可。**不要**在 main 上直接 commit 修补，一律走 PR。

---



## 六、PR-2 · 路径硬编码 + 部署脚本（2~3 天）

> 目标：把 PR-1 遗留的 `WorkingDirectory` 拼路径、`ViewsPath` / `StaticDir` 指向、Docker/spug/systemd 部署脚本全部对齐新目录。



### T2 硬编码修正清单


| #   | 文件                                                                   | 原代码（Round 1 后仍是老路径）                                    | 改到                                                    |
| --- | -------------------------------------------------------------------- | ------------------------------------------------------ | ----------------------------------------------------- |
| 1   | `internal/app/bootstrap.go`（原 `commands/command.go:311`）             | 验证码字体 `WorkingDir("static", "fonts")`                  | `WorkingDir("web", "static", "fonts")`                |
| 2   | 同上（原 `:332`）                                                         | `StaticDir["/static"] = ..., "static"`                 | `..., "web", "static"`                                |
| 3   | 同上（原 `:334`）                                                         | `WorkingDir("views")`                                  | `WorkingDir("web", "views")`                          |
| 4   | 同上（原 `:337, 342`）                                                    | 二次验证码字体路径                                              | 同上                                                    |
| 5   | 同上（原 `:345-347`）                                                     | `ViewsPath = WorkingDir("views")`                      | `WorkingDir("web", "views")`                          |
| 6   | 同上（原 `:355`）                                                         | `gocaptcha.SetFontPath(WorkingDir("static", "fonts"))` | `WorkingDir("web", "static", "fonts")`                |
| 7   | `internal/dto/book_result.go`（原 `models/BookResult.go:455~472`）      | `filetil.CopyFile(WorkingDir, "static", ...)` × 6 处    | `WorkingDir, "web", "static", ...`                    |
| 8   | `internal/config/enumerate_legacy.go`（原 `conf/enumerate.go:75, 391`） | `"./conf/app.conf"`                                    | `"./configs/app.conf"`（Round 1 T1 已改，此处 double check） |
| 9   | `internal/cli/install.go`（原 `commands/install.go:106`）               | `i18n.SetMessage(lang, "conf/lang/"+lang+".ini")`      | `"configs/lang/"+lang+".ini"`（Round 1 T1 已改）          |
| 10  | `internal/app/bootstrap.go`（原 `commands/command.go:280`）             | 同上 i18n                                                | 同上                                                    |
| 11  | `internal/controller/BaseController.go:79`                           | `ioutil.ReadFile(WebConfig.ViewsPath, ...)`            | 无需改（ViewsPath 已在 #5 里指向新路径）                           |


`WorkingDirectory` 拼路径的位置（保留 `WorkingDirectory` 本身，只是它指向的目录里改成 `web/`）：

- `internal/controller/SettingController.go:119, 141, 151, 163` — 头像上传路径，保持 `uploads/` 不动（用户 URL 不变）
- `internal/controller/BookController.go:367, 377, 498` — 同上
- `internal/controller/DocumentController.go:436, 461, 473, 572, 617` — 附件路径，保持 `uploads/` 不动
- `internal/controller/BlogController.go:501, 518, 528, 611, 657` — 同上
- `internal/controller/ManagerController.go:627, 656, 678` — 同上
- `internal/model/DocumentModel.go:285, 286` — 删除 book 缓存目录，保持 `uploads/` 不动
- `internal/model/BookModel.go:516, 790, 795, 805, 808, 859, 863` — 上传路径处理，保持 `uploads/` 不动

> **结论：** `uploads/` / `runtime/` **保留在根**，URL `/uploads/`* 不变。根目录 Go 包 `converter/` / `graphics/` / `mail/` 已分别迁入 `internal/converter/`、`pkg/graphics/`、`pkg/mail/`。只有 `static/` 和 `views/` 变成 `web/static/` 和 `web/views/` 需要改资源路径。



### T2 部署脚本

`deployments/Dockerfile`**：**

```diff
- COPY conf   /app/conf
- COPY static /app/static
- COPY views  /app/views
- COPY main   /app/doc
+ COPY configs      /app/configs
+ COPY web/static   /app/web/static
+ COPY web/views    /app/web/views
+ COPY doc          /app/doc

WORKDIR /app
- CMD ["./doc"]
+ CMD ["./doc", "web"]
```

（同时把二进制构建阶段的 `RUN go build -o doc main.go` 改成 `./cmd/doc`。）

`deployments/docker-compose.yml`**：**

```diff
volumes:
- - ./conf:/app/conf
- - ./static:/app/static
- - ./views:/app/views
+ - ./configs:/app/configs
+ - ./web/static:/app/web/static
+ - ./web/views:/app/web/views
- ./uploads:/app/uploads
- ./runtime:/app/runtime
```

`deployments/start.sh`**、**`deployments/sync_host.sh`**：** 所有 `./static`、`./views`、`./conf` 全改；参考现有内容逐行 review。

`deployments/systemd/doc.service`**：** 若 `WorkingDirectory=` 指向的目录必须包含 `configs/`、`web/`、`runtime/`；`ExecStart=` 用 `/opt/doc/doc web` 或 `/opt/doc/doc`。

**spug 部署脚本** ([docs/deploy-spug-*.md](./deploy-spug-local.md))：升级步骤加入"新目录结构 pre-check"章节。

### T2 兼容 pre-check

在 `internal/cli/root.go` 里加个启动前自检：

```go
// internal/cli/root.go —— 检测老部署残留
func preflightCheck() {
    if _, err := os.Stat("./conf/app.conf"); err == nil {
        fmt.Println("[warn] detected legacy ./conf/app.conf; please migrate to ./configs/app.conf (Round 2 layout)")
    }
    if _, err := os.Stat("./static"); err == nil {
        fmt.Println("[warn] detected legacy ./static; please migrate to ./web/static")
    }
}
```

只 warn 不 abort，方便运维发现问题。

### PR-2 验收

- [ ] Docker 镜像构建 + `docker compose up` 能起（`docker exec -it doc ls /app` 看到 `configs/ web/ uploads/`）
- [ ] 首页 CSS/JS 200（浏览器 DevTools Network 面板确认 `/static/css/main.css` 未 404）
- [ ] 验证码图片正常
- [ ] i18n 中英切换生效
- [ ] 导出 book 到 zip / pdf 能成功（`BookResult.go` 里的 `filetil.CopyFile` 全部走新路径）
- [ ] spug 部署到 staging 走一遍

---



## 七、Session / gob 缓存清理（**上线必做**）

> 参考 [refactor-roadmap.md §六 风险 13](./refactor-roadmap.md#六关键风险清单)。
> `gob.Register` 编码类型名 = **"包路径.类型名"**；Round 2 后 `models.Member` 变成 `internal/model.Member`，旧数据反序列化必崩。



### 现网升级步骤（模板 for CHANGELOG）

1. **停服**（预留 5 分钟停机窗口）
2. 备份 `conf/`（已在 Round 1 迁走可跳过）、`cache/`、`runtime/session/`
3. **清 session**：
  - `sessionprovider=file` → `rm -rf runtime/session/*`
  - `sessionprovider=redis` → `redis-cli -h <host> FLUSHDB`（**谨慎**：如果 Redis 有其他业务共用，只删对应 prefix 的 key）
  - `sessionprovider=mysql` → `TRUNCATE TABLE mindoc_session`
4. **清 gob 缓存**：
  - `cache_provider=file` → `rm -rf cache/*`
  - `cache_provider=redis` → 同上按 prefix 清
  - `cache_provider=memory` → 重启后自然清
5. 部署新版本（Round 2 的 `deployments/`）
6. 启动，检查首页
7. 通知所有用户**重新登录**



### T9 · 部署 note 写在哪

- `README.md` 加"Round 2 升级须知"章节
- `docs/round-2-execution-plan.md`（本文）§七 作为权威文档
- `CHANGELOG.md` 版本节里显眼位置写"⚠️ Breaking：需清 session + 清 cache/，用户需重新登录"



### 加固手段（可选）

- `internal/controller/BaseController.SetMember` 加 version 字段：session 反序列化失败时**降级 clear session**，用户自动跳登录页而不是 500
  ```go
  type SessionMember struct {
      Version int      `json:"v"`  // 每次结构变更 +1
      Member  Member
  }
  ```
- 更根本的解决：**Round 4** 换 session serializer 为 msgpack/json，从此不再依赖包路径

---



## 八、T3~T5 · 配置分组 + 强类型 Config



### T3 · `configs/app.conf` `[section]` 分组（0.5~1 天）

**目标：** 253 行平铺 → 按 [refactor-roadmap.md §2.3 Step 2](./refactor-roadmap.md#23-目标三配置模块优化) 骨架分 12 个 section。

**做法：**

1. 直接编辑 `configs/app.conf`：加 section 头（如 `[database]`），把原对应键搬到 section 下
2. 同步改 `configs/app.conf.example`
3. **所有调用方**：`web.AppConfig.DefaultString("db_host", ...)` → `web.AppConfig.DefaultString("database::db_host", ...)`
  - 30+ 处，主要集中在 `internal/config/enumerate_legacy.go`（原 `conf/enumerate.go`）
  - **本轮只在**从 `AppConfig` 直接读的地方改；T4 上强类型 Config 后统一收敛，AppConfig 的手动调用会消失

**验收：** 启动后所有配置项行为不变（登录、上传、mail、export 都试一遍）。

**风险：** 漏改一个 `AppConfig.DefaultString("db_host", ...)` 就会读到默认值 → DB 连不上。**必须**用 `Grep AppConfig.DefaultString --type go` 逐一 review。

### T4 · 强类型 `config.Config` struct（1.5~2 天）

**新增** `internal/config/config.go`**：**

```go
// internal/config/config.go
package config

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
    DingTalk DingTalkConfig
    I18n     I18nConfig
    MCP      MCPConfig    // ← Round 3 使用；本轮占位空 struct 也可，但字段先声明齐
}

type MCPConfig struct {
    Enable        bool
    Listen        string
    StdioMember   string
    TokenRequired bool
    RateLimit     int
}

var Global *Config

func Load(path string) (*Config, error) {
    // 1. beego AppConfig.LoadAppConfig("ini", path)
    // 2. 逐 section 读进 Config 各字段
    // 3. 用 reflect + tag 或手写 unmarshal
    // 4. return &Config{...}, nil
}

func Reload() error { ... }  // fsnotify 集成放 Round 4
```

**调用方改写策略（分批）：**

- **Batch 1（本 PR）**：`internal/app/bootstrap.go` 里所有 `AppConfig.DefaultString("db_host", ...)` → `config.Global.Database.Host`
- **Batch 2（本 PR）**：`internal/config/enumerate_legacy.go` 里 30+ Getter 全部走 `config.Global.XXX`
- **Batch 3（本 PR 或独立 PR）**：controllers/models 里散落的 `web.AppConfig.DefaultString(...)`，全局 grep 后收敛

**兼容层：** 保留原 `conf.GetXXX()` Getter 函数（Round 1 后已在 `internal/config/enumerate_legacy.go`）作为 shim，内部改成读 `Global.XXX`。这样其他文件不用改。

**验收：** 全部 shim + 强类型 struct 上线后行为不变；`web.AppConfig.DefaultString(...)` 只剩 T3 里 beego 内部要用的（如 session provider 配置）。

### T5 · `.env` 支持（0.5 天，可选）

```powershell
go get github.com/joho/godotenv
```

`cmd/doc/main.go` 启动时：

```go
func main() {
    _ = godotenv.Load(".env", ".env.local")   // 忽略错误：文件不存在也 ok
    if err := cli.Execute(); err != nil { os.Exit(1) }
}
```

`.gitignore` 加 `.env` 与 `.env.local`。

**验收：** 本地放 `DOC_DB_HOST=127.0.0.1` 到 `.env`，启动能读到（配合 `configs/app.conf` 里 `${DOC_DB_HOST||...}` 语法）。

---



## 九、T6 · `internal/router/` 按域拆分（1 天）

按 [router-split-migration-plan.md](./router-split-migration-plan.md) 与 [routers-reference.md](./routers-reference.md) 拆分：

```
internal/router/
├─ router.go        # func Init() { registerAccount(); registerBook(); ...; registerAPI() }
├─ account.go       # /login /logout /register /find_password /captcha ...
├─ manager.go       # /manager/*
├─ book.go          # /book/* /books/*
├─ document.go      # /docs/* /docs/history/* /docs/attach/*
├─ blog.go          # /blog/*
└─ api.go           # /api/*（如现有 API 路由 + Round 3 的 /mcp/* 也可挂这里）
```

`cmd/doc/main.go` 从 `_ "internal/router"` 触发 init 注册，或改成显式 `router.Init()` 调用（推荐显式，避免 init 副作用）。

---



## 十、T7 · `internal/middleware/` 合并（0.5 天）

`middleware/filter.go` (1KB) 和 `routers/filter.go` (2.8KB) 里都是 beego filter；合并后按职责分：

```
internal/middleware/
├─ auth.go         # 登录校验（原 routers/filter.go 主体）
├─ csrf.go         # CSRF（若有）
├─ logger.go       # 访问日志
├─ recover.go      # panic 恢复
└─ ratelimit.go    # 【占位 · Round 3 MCP 用】
```

`internal/app/bootstrap.go` 里显式注册顺序：`Recover → Logger → CSRF → Auth`。

---



## 十一、T8 · 预留 `internal/mcp/` + `internal/dto/mcpdto/`（10 分钟）

```powershell
New-Item -ItemType Directory -Force -Path internal/mcp, internal/dto/mcpdto
Set-Content internal/mcp/doc.go "// Package mcp is reserved for the MCP server (Round 3).`npackage mcp`n"
Set-Content internal/dto/mcpdto/doc.go "// Package mcpdto holds the MCP tool In/Out DTOs (Round 3).`npackage mcpdto`n"
```

Round 3 直接 `internal/mcp/server.go` 落地，零迁移。

---



## 十二、PR 拆分总表


| #   | PR                                                        | 内容    | 大小                     | 依赖   |
| --- | --------------------------------------------------------- | ----- | ---------------------- | ---- |
| 1   | `refactor(round2): PR-1 layout move + import rewrite`     | T1 全部 | **超大**（touch 每个 Go 文件） | 无    |
| 2   | `refactor(round2): PR-2 fix hardcoded paths + deployment` | T2 全部 | 中                      | PR-1 |
| 3   | `refactor(round2): app.conf [section] grouping`           | T3    | 中                      | PR-1 |
| 4   | `feat(round2): typed config.Config with Load()`           | T4    | 中大                     | PR-3 |
| 5   | `feat(round2): .env support`                              | T5    | 小                      | PR-4 |
| 6   | `refactor(round2): split routers by domain`               | T6    | 中                      | PR-1 |
| 7   | `refactor(round2): merge middleware`                      | T7    | 小                      | PR-1 |
| 8   | `chore(round2): reserve internal/mcp and mcpdto`          | T8    | 极小                     | PR-1 |
| 9   | `docs(round2): upgrade notes for session/cache purge`     | T9    | 小                      | 任意   |


**合入顺序建议：** PR-1 → PR-2 → PR-6 → PR-7 → PR-8 → PR-3 → PR-4 → PR-5 → PR-9。
关键点：PR-3（`[section]`）必须**先于** PR-4（强类型 Config），因为 T4 的 unmarshal 依赖 section 结构。

**PR-1 review 建议：** 请 reviewer 只看 `go build ./...` 和文件树 diff，不看内容 diff（内容全是 rename + import 改写，逐行看无意义）。

---



## 十三、上线检查清单



### 部署前

- [ ] 现网数据库有备份（防最坏情况）
- [ ] Redis / session 存储可清（评估影响面）
- [ ] `cache/` 目录可清
- [ ] 用户已知悉"需要重新登录"



### 部署脚本 pre-check

- [ ] Dockerfile 里的 `COPY` 路径全对
- [ ] docker-compose volume 挂载全对
- [ ] systemd `WorkingDirectory` 指向新结构
- [ ] spug 部署脚本更新
- [ ] `--workDir` 参数（若使用）指向的目录含 `configs/`、`web/`、`runtime/`



### 部署后

- [ ] 首页 200，CSS/JS/字体全 200
- [ ] 登录（旧 session 已清，重新登录）
- [ ] 创建/编辑/删除 book、document
- [ ] 上传附件、上传头像
- [ ] 导出 book (word / pdf)
- [ ] 后台 manager 全走一遍
- [ ] Windows 部署（`doc service install`）能启动
- [ ] Linux systemd 能启动 + `journalctl -u doc` 无错

---



## 十四、追踪表


| #   | 任务                                | PR  | Commit | 状态  |
| --- | --------------------------------- | --- | ------ | --- |
| T1  | PR-1 目录搬迁 + import 改写             |     |        |     |
| T2  | PR-2 硬编码 + 部署脚本                   |     |        |     |
| T3  | `configs/app.conf` `[section]` 分组 |     |        |     |
| T4  | 强类型 `config.Config` + Load()      |     |        |     |
| T5  | `.env` 支持                         |     |        |     |
| T6  | `internal/router/` 拆分             |     |        |     |
| T7  | `internal/middleware/` 合并         |     |        |     |
| T8  | 预留 `internal/mcp/` + `mcpdto/`    |     |        |     |
| T9  | 部署 note（清 session/cache）          |     |        |     |


---



## 十五、Round 3 前置产物核对

- ✅ `internal/mcp/` 空目录存在
- ✅ `internal/dto/mcpdto/` 空目录存在
- ✅ `internal/config/config.go` 已有 `MCPConfig` 字段
- ✅ `configs/app.conf` 有 `[mcp]` section（本 PR 一起加，Round 3 用现成的）
- ✅ `internal/router/api.go`（或 `router.go`）可挂 `/mcp/*` handler
- ✅ `internal/middleware/ratelimit.go` 占位存在
- ✅ `cache.Cache` 接口（Round 1）可用作 MCP HTTP token 缓存
- ✅ `internal/errs/` 可用作 MCP 工具错误返回（`VERSION_CONFLICT` / `CONFIRM_REQUIRED` 等错误码）

以上任一未满足，Round 3 起手会踩坑，回补。

---



## 十六、参考

- [refactor-roadmap.md §2.2](./refactor-roadmap.md#22-目标二前后端目录结构调整规范化) — 目标目录结构决策
- [refactor-roadmap.md §六 关键风险](./refactor-roadmap.md#六关键风险清单) — 尤其 12/13/14 三条 Round 2 专属风险
- [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) — 附录 A/B 硬编码定位（**目标路径换成本文的** `web/`）
- [router-split-migration-plan.md](./router-split-migration-plan.md) — router 域拆分
- [routers-reference.md](./routers-reference.md) — 现有路由分类参考
- [Go Project Layout Standards](https://github.com/golang-standards/project-layout)

