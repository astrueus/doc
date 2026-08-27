# 实施文档骨架

写文档时对照代码现状。本仓任务文风格可参考 `docs/round-5/round-5-t9-repo-service.md`、`docs/round-5/round-5-t5-mcp-p1.md`（结构：现状 / 目标 / 不做 / 验收），但**新功能实施文档**落在 `docs/{预定版本}-{短名}/`，不要写进 `docs/round-5/`。

过长再拆文件，不要为拆而拆。没有后续待办可以不建 `06`。

## 文件清单

| 文件 | 内容 |
|------|------|
| `README.md` | 范围、进度表、文档索引、一句话目标、明确不做 |
| `01-需求与口径.md` | 需求拆解、已确认口径、未决、边界 |
| `02-现状分析.md` | 现有链路、关键代码位置（带路径） |
| `03-改造方案.md` | 方案；涉及 HTTP/MCP 契约的须已确认 |
| `04-文件清单与代码要点.md` | 动作 / 路径 / 说明 / 进度 + 可粘贴 Go |
| `05-验收.md` | Web 页面 / MCP 工具 / `make test` 或 `just test` |
| `06-后续待办.md` | 本期不做、后续项；没有则省略 |

## README 进度表示例

```markdown
| 项 | 状态 | 说明 |
|----|------|------|
| 口径确认 | 已确认 | 见 01 |
| DocumentRepo 扩面 | 待落实 | 见 04 |
| 验收 | 未开始 | 见 05 |
```

状态用：`已确认` / `已完成` / `待落实` / `用户已手动` / `不做` / `未开始`。

## 04 文件表示例

```markdown
| 动作 | 路径 | 说明 | 进度 |
|------|------|------|------|
| 改 | `internal/mcp/tools_read.go` | 插入点：`handleGetDocument` 改为走 DocumentRepo | 待落实 |
| 不改 | `internal/router/document.go` | URL 不变 | — |
```

每个「改」项后面跟完整示例代码（可含上下文若干行），不要只写要点。路径用本仓分层：`internal/controller`、`internal/router`、`internal/repository`、`internal/mcp`、`internal/dto/mcpdto` 等。
