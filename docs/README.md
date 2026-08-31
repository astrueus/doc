# Doc 项目文档

本目录存放与项目维护、上游同步相关的文档。

> **整体进度快照（2026-08-31，主干 `master`）**  
> Round 1–2 ✅ · Round 3 MCP MVP ✅ · §十七 P0 ✅ · Round 4 代码主线 ✅ · **Round 5 🔶**（批次 A/B 已合入，T12-a/b/c 缓存已落地；搜索 FULLTEXT ⏸）。  
> 明细见 [refactor-roadmap.md §七](./refactor-roadmap.md#七迭代进度追踪) 与 [round-5/round-5-execution-plan.md](./round-5/round-5-execution-plan.md)。

## 入口（本目录）

| 文档 | 说明 |
|------|------|
| [refactor-roadmap.md](./refactor-roadmap.md) | **整体优化与迭代路线图（总纲）** |
| [mcp-integration.md](./mcp-integration.md) | MCP 接入指南（Claude / Cursor / HTTP） |
| [upstream-mindoc-checklist.md](./upstream-mindoc-checklist.md) | 上游 MinDoc 提交跟进 |

## 目录

| 目录 | 内容 |
|------|------|
| [round-1-4/](./round-1-4/) | Round 1–4 执行计划、升级说明、前后端拆分清单 |
| [issue-log/](./issue-log/) | 问题收集（序号递增，待排期） |
| [round-5/](./round-5/) | Round 5 执行计划、评估与任务细化 |
| [router/](./router/) | 路由分类参考与拆分计划 |
| [deploy-spug/](./deploy-spug/) | Spug 部署 |
| [release/](./release/) | 本地 / Actions 发版；[小团队协作草案](./release/git-workflow.md) · [切换执行（A～C 已完成，D/E 待做）](./release/git-workflow-cutover.md) · [大团队参考](./release/git-workflow-large-team.md) |

> **阅读顺序建议**：先看 [refactor-roadmap.md](./refactor-roadmap.md) → 已完成轮次看 [round-1-4/](./round-1-4/) → 进行中看 [round-5/](./round-5/) → 接入 MCP 看 [mcp-integration.md](./mcp-integration.md)。AI 协作约定见仓库根目录 [AGENTS.md](../AGENTS.md)（含项目执行标准与 `.cursor/rules/` 索引）。

## 多轮进度一览

| 轮次 | 主题 | 状态 | 备注 |
|------|------|------|------|
| Round 1 | cobra / cache / errs / 低风险债 | ✅ | 见 [round-1-4](./round-1-4/) |
| Round 2 | 目录一步到位 + 强类型配置 + 路由/中间件 | ✅ | 升级说明 [upgrade-round-2.md](./round-1-4/upgrade-round-2.md) |
| Round 3 | MCP 10 工具 + HTTP Bearer | ✅ MVP | 搜索 FULLTEXT → R5 后 ⏸ |
| Round 4 | 模型 / zap / i18n / 前端 P1 / Repo / 测试 | ✅ 主线 | 遗留移交 Round 5 |
| Round 5 | 工程化 / Vite / 对象存储 / kratos 向评估 | 🔶 | 见 [round-5](./round-5/)；T12-a/b/c 已落地；T3/T4/T13 ⏸ |

**Round 4 已合入（`v2.2.1`）要点：** BookModel 拆分、`md_` 修复、zap、Session 只存 id、前端 P1、`internal/i18n`、Repository 初版、pkg 单测、MCP P0。  
**Round 5 承接：** Vite、缓存/ORM（kratos 向）报告、MCP P1、测试 CI、对象存储、`scripts`↔`deployments`、分层债；**不含**旧 FULLTEXT 实施与拆 bootstrap。

## 上游关系

- 上游项目：[mindoc-org/mindoc](https://github.com/mindoc-org/mindoc)
- 本仓库：`git.itopcms.com/astrueus/doc`
- 可执行文件：`doc`（上游为 `mindoc`）
- 模块路径：`git.itopcms.com/astrueus/doc`（上游为 `github.com/mindoc-org/mindoc`）

## 同步建议

1. 添加 upstream remote：`git remote add upstream https://github.com/mindoc-org/mindoc.git`
2. 按功能 cherry-pick，避免整库 merge
3. 每次移植后改 import 路径与 CLI 文案（`mindoc` → `doc`）
4. 优先从「阶段 0 基础设施」和「阶段 1 搜索」开始
