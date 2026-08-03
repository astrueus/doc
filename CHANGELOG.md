# Changelog

## Unreleased / Round 4

> ⚠️ **Breaking（T6 Session）**：升级后需**清空 session 存储**（file/redis 等 SessionProvider 目录或键），并视为旧「记住我」cookie 失效；用户需重新登录。  
> Session 现仅存 `member_id`；cookie remember 改为 msgpack（`pkg/gob` 实现已换，包名保留）。

### Changed

- 日志：`uber-go/zap` + Lumberjack，经 beego/logs shim 转发；MCP stdio 仍禁 stdout（仅文件 / 可选 stderr）
- 日志体验：默认 `log_level=Info`（旧默认 `Alert` 会被 zap 滤掉 Info）；stderr 固定 beego 风格可读格式（保留颜色），文件默认 json 且剥离 ANSI；shim 使用 `[file:line]`
- Session：不再 gob 序列化整份 `Member`
- 修复：`FilterUser` / 登录页仍按 `model.Member` 断言 Session，T6 只存 `member_id` 后编辑等操作会误跳登录；统一用 `auth.MemberIDFromSession`
- 前端 P1：移除 IE shim / 条件注释；常用静态资源 `cdnjs` 补 `"version"`

## Unreleased / Round 2

> ⚠️ **Breaking**：从旧版本升级到 Round 2 必须**清 session + 清 `runtime/cache/`（及旧 `cache/`）**，用户需重新登录。  
> 步骤见 [docs/upgrade-round-2.md](docs/upgrade-round-2.md)。

### Changed

- 项目布局改为 `cmd/` + `internal/` + `pkg/` + `web/` + `conf/` + `deployments/`
- `conf/app.conf` 按 `[section]` 分组；增加强类型 `config.Config` 与可选 `.env`
- 配置目录由短暂使用的 `configs/` **改回 `conf/`**，对齐 Beego 默认路径，消除启动期 `open conf/app.conf` 噪音
- 路由按域拆分；文档编辑 / 对比 / 模板列表从 `/api/*` 迁到 `/book/*`
- 中间件去重并显式 `middleware.Register()`
- 根目录 `converter/` / `graphics/` / `mail/` 收入 `internal/` 或 `pkg/`；运行时缓存统一 `runtime/cache/`

### Added

- 预留 `internal/mcp/`、`internal/dto/mcpdto/`（Round 3 实现）
