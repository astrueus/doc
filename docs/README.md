# Doc 项目文档

本目录存放与项目维护、上游同步相关的文档。

> **整体进度快照（2026-08-03，分支 `v2.2.1`）**  
> Round 1–2 ✅ · Round 3 MCP MVP ✅ · §十七 P0 ✅ · Round 4 代码主线 ✅ · **Round 5 ⏳**（工程化 / Vite / 对象存储 / kratos 向评估；搜索 FULLTEXT ⏸）。  
> 明细见 [refactor-roadmap.md §七](./refactor-roadmap.md#七迭代进度追踪) 与 [round-5-execution-plan.md](./round-5-execution-plan.md)。

## 文档索引

| 文档 | 说明 | 进度（概要） |
|------|------|-------------|
| [refactor-roadmap.md](./refactor-roadmap.md) | **整体优化与迭代路线图（总纲）** | Round 1–5 进度见文内 §七 |
| [round-1-execution-plan.md](./round-1-execution-plan.md) | Round 1 · 低风险重构 + 前置 | ✅ 已完成 |
| [round-2-execution-plan.md](./round-2-execution-plan.md) | Round 2 · `cmd/`+`internal/` + 强类型 Config | ✅ 已完成 |
| [round-3-execution-plan.md](./round-3-execution-plan.md) | Round 3 · MCP Server（10 工具 + Bearer） | ✅ MVP；搜索→R5；§十七 P0✅ |
| [round-4-execution-plan.md](./round-4-execution-plan.md) | Round 4 · 模型 / 日志 / i18n / 前端 | ✅ 代码主线；遗留→Round 5 |
| [round-4-coverage.md](./round-4-coverage.md) | Round 4 T10 单测覆盖率快照 | ✅ 基线已建 |
| [round-5-execution-plan.md](./round-5-execution-plan.md) | Round 5 · 工程化 / Vite / 对象存储 / 分层评估 | ⏳；搜索 FULLTEXT ⏸ |
| [mcp-integration.md](./mcp-integration.md) | MCP 接入指南（Claude / Cursor / HTTP） | 随 Round 3/T13 更新 |
| [routers-reference.md](./routers-reference.md) | 路由分类参考 | 参考 |
| [router-split-migration-plan.md](./router-split-migration-plan.md) | 路由拆分与 `/api` 治理（Round 2 T6） | ✅ 已落地 |
| [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) | 前后端目录拆分清单（Round 2） | ✅ 目标目录已落地 |
| [upstream-mindoc-checklist.md](./upstream-mindoc-checklist.md) | 上游 mindoc 提交跟进 | 持续 |
| [upgrade-round-2.md](./upgrade-round-2.md) | Round 2 升级说明（清 session/cache） | 运维 |
| [deploy-spug-local.md](./deploy-spug-local.md) / [deploy-spug-actions.md](./deploy-spug-actions.md) | Spug 部署 | 运维 |
| [release-local.md](./release-local.md) / [release-gitea-actions.md](./release-gitea-actions.md) | 发布 | 运维 |

> **阅读顺序建议**：先看 [refactor-roadmap.md](./refactor-roadmap.md) 建立全局视角 → 按轮次打开 `round-N-execution-plan.md` → 接入 MCP 看 [mcp-integration.md](./mcp-integration.md)。AI 协作约定见仓库根目录 [AGENTS.md](../AGENTS.md)。

## 多轮进度一览

| 轮次 | 主题 | 状态 | 备注 |
|------|------|------|------|
| Round 1 | cobra / cache / errs / 低风险债 | ✅ | 见 round-1 §十 |
| Round 2 | 目录一步到位 + 强类型配置 + 路由/中间件 | ✅ | 见 round-2 §十四；B1/B2 决策跳过 |
| Round 3 | MCP 10 工具 + HTTP Bearer | ✅ MVP | 搜索 FULLTEXT → R5 后 ⏸ 等重定义；P0 经 R4 T13 ✅ |
| Round 4 | 模型 / zap / i18n / 前端 P1 / Repo / 测试 | ✅ 主线 | 遗留移交 Round 5 |
| Round 5 | 工程化 / Vite / 对象存储 / kratos 向评估 | ⏳ | 见 [round-5](./round-5-execution-plan.md)；T3/T4/T13 ⏸ |

**Round 4 已合入（`v2.2.1`）要点：** BookModel 拆分、`md_` 修复、zap、Session 只存 id、前端 P1、`internal/i18n`、Repository 初版、pkg 单测、MCP P0。  
**Round 5 承接：** Vite、缓存/ORM（kratos 向）报告、MCP P1、测试 CI、对象存储、`scripts`↔`deployments`、分层债；**不含**旧 FULLTEXT 实施与拆 bootstrap。

## 上游关系

- 上游项目：[mindoc-org/mindoc](https://github.com/mindoc-org/mindoc)
- 本仓库：`git.itopcms.com/jackliu/doc`
- 可执行文件：`doc`（上游为 `mindoc`）
- 模块路径：`git.itopcms.com/jackliu/doc`（上游为 `github.com/mindoc-org/mindoc`）

## 同步建议

1. 添加 upstream remote：`git remote add upstream https://github.com/mindoc-org/mindoc.git`
2. 按功能 cherry-pick，避免整库 merge
3. 每次移植后改 import 路径与 CLI 文案（`mindoc` → `doc`）
4. 优先从「阶段 0 基础设施」和「阶段 1 搜索」开始
