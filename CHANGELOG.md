# Changelog

## Unreleased / Round 2

> ⚠️ **Breaking**：从旧版本升级到 Round 2 必须**清 session + 清 `runtime/cache/`（及旧 `cache/`）**，用户需重新登录。  
> 步骤见 [docs/upgrade-round-2.md](docs/upgrade-round-2.md)。

### Changed

- 项目布局改为 `cmd/` + `internal/` + `pkg/` + `web/` + `deployments/`
- `configs/app.conf` 按 `[section]` 分组；增加强类型 `config.Config` 与可选 `.env`
- 路由按域拆分；文档编辑 / 对比 / 模板列表从 `/api/*` 迁到 `/book/*`
- 中间件去重并显式 `middleware.Register()`
- 根目录 `converter/` / `graphics/` / `mail/` 收入 `internal/` 或 `pkg/`；运行时缓存统一 `runtime/cache/`

### Added

- 预留 `internal/mcp/`、`internal/dto/mcpdto/`（Round 3 实现）
