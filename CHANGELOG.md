# Changelog

## Unreleased / Round 5

其余任务（T6 Vite、T12、T14、T16 等）见 [round-5-execution-plan.md](docs/round-5/round-5-execution-plan.md)。已交付项见 **2.3.0** / **2.3.1**。

### Changed

- T8：`BookResult` / `DocumentSearchResult` / `AttachmentResult` / `CommentResult` / `DocumentHistorySimpleResult` / `ConvertBookResult` / `BlogResult` 迁入 `internal/dto/`；查询进 Repo。`SelectMemberResult` 暂留 model（选择器查询仍在 Team/Itemsets）
- T9：MCP 读路径（get/tree/search/list_books/authz/token）与 Web 文档 `Content` 读走 Repository；导出改为 `repository.ConvertBook` / `ExportBookMarkdown`。本轮未建 `internal/service/`

## 2.3.1 — 2026-08-19

无 Breaking。应用行为与 2.3.0 相同；本版补部署脚本与文档，供 Spug 按 Tag 下 Release 包（含常规发布保留历史版本）。

### Added

- Spug 自定义发布前置脚本 `deployments/spug/spug_pre.sh`
- Spug 常规发布：`spug_strip.sh`（代码迁出后）/ `spug_unpack.sh`（应用发布前）；说明见 [deploy-spug-standard.md](docs/deploy-spug/deploy-spug-standard.md)

### Fixed

- `spug_run.sh`：权威配置固定为 REPO 的 `app.conf`；纠正 uploads 套娃软链

### Changed

- Gitea Release 标题改为小写 `doc`，与包名一致

## 2.3.0 — 2026-08-18

> 从 `v2.2.1` 升级到本版请先读下面两条 Breaking，再部署 Release 附件。  
> ⚠️ **Breaking（环境变量硬切）**：`MINDOC_*` **不再兼容**，一律改为 `DOC_*`（监听为 `DOC_ADDR` / `DOC_PORT`；邮件过期为 `DOC_MAIL_EXPIRED`；部署同步为 `DOC_SYNC`）。完整对照见 [docs/round-5/round-5-env-mindoc-to-doc.md](docs/round-5/round-5-env-mindoc-to-doc.md)。  
> ⚠️ **Breaking（默认标识）**：`app_key` 默认 `doc`、`sessionname` 默认 `doc_id`、`cache_redis_prefix` 默认 `doc::cache`。升级须**清空 session 存储**，并视为旧「记住我」cookie / 旧 Redis 前缀缓存失效。

### Added

- MCP T5：`create_book` / `update_book`（元数据最小集，无删书、无封面）
- MCP：`create_document` 支持 `if_exists=update`、可选 `auto_release`（默认不发布，不读项目自动发布开关）
- MCP：`get_document` 按 Unicode 字符截断；`search_document` 返回 `book_identify` / `doc_identify`

### Changed

- 配置 example、docker-compose、sync 脚本、README：环境变量前缀切至 `DOC_*`
- `internal/config` 默认 `app_key` / `sessionname` / Redis 前缀与 example 对齐为 `doc*`
- T15：根目录 `scripts/` 全迁 `deployments/scripts/`；根目录用 `Makefile` / `justfile` 作快捷入口
- T7：新增 `deployments/scripts/test.sh` / `test.ps1`、覆盖率基线、`.gitea/workflows/test.yml` 白名单硬闸
- T2：ORM/分层评估结论入库（本轮维持 beego/orm + 扩 Repo；Round 6+ 既定 ① data → ② biz）

## Unreleased / Round 4

> ⚠️ **Breaking（T6 Session）**：升级后需**清空 session 存储**（file/redis 等 SessionProvider 目录或键），并视为旧「记住我」cookie 失效；用户需重新登录。  
> Session 现仅存 `member_id`；cookie remember 改为 msgpack（`pkg/gob` 实现已换，包名保留）。

### Changed

- 日志：`uber-go/zap` + Lumberjack，经 beego/logs shim 转发；MCP stdio 仍禁 stdout（仅文件 / 可选 stderr）
- 日志体验：默认 `log_level=Info`（旧默认 `Alert` 会被 zap 滤掉 Info）；stderr 固定 beego 风格可读格式（保留颜色），文件默认 json 且剥离 ANSI；shim 使用 `[file:line]`
- Session：不再 gob 序列化整份 `Member`
- 修复：`FilterUser` / 登录页仍按 `model.Member` 断言 Session，T6 只存 `member_id` 后编辑等操作会误跳登录；统一用 `auth.MemberIDFromSession`
- 前端 P1：移除 IE shim / 条件注释；常用静态资源 `cdnjs` 补 `"version"`
- 测试（T10）：补 `pkg/*`、`internal/errs`/`auth`/`logging`/`i18n` 单测；见 `docs/round-1-4/round-4-coverage.md`
- i18n（T5）：移除 `beego/i18n`，新增 `internal/i18n`（保留 `conf/lang/*.ini` 与 `Tr`/`IsExist`/`SetMessage` API）
- Repository（T2）：新增 `internal/repository`（Document/Book/Member）；MCP 写工具乐观锁改走 `DocumentRepo`

## Unreleased / Round 2

> ⚠️ **Breaking**：从旧版本升级到 Round 2 必须**清 session + 清 `runtime/cache/`（及旧 `cache/`）**，用户需重新登录。  
> 步骤见 [docs/round-1-4/upgrade-round-2.md](docs/round-1-4/upgrade-round-2.md)。

### Changed

- 项目布局改为 `cmd/` + `internal/` + `pkg/` + `web/` + `conf/` + `deployments/`
- `conf/app.conf` 按 `[section]` 分组；增加强类型 `config.Config` 与可选 `.env`
- 配置目录由短暂使用的 `configs/` **改回 `conf/`**，对齐 Beego 默认路径，消除启动期 `open conf/app.conf` 噪音
- 路由按域拆分；文档编辑 / 对比 / 模板列表从 `/api/*` 迁到 `/book/*`
- 中间件去重并显式 `middleware.Register()`
- 根目录 `converter/` / `graphics/` / `mail/` 收入 `internal/` 或 `pkg/`；运行时缓存统一 `runtime/cache/`

### Added

- 预留 `internal/mcp/`、`internal/dto/mcpdto/`（Round 3 实现）
