# Doc 项目文档

本目录存放与项目维护、上游同步相关的文档。

## 文档索引

| 文档 | 说明 |
|------|------|
| [refactor-roadmap.md](./refactor-roadmap.md) | **整体优化与迭代路线图（总纲）**：MCP、目录结构、配置模块、缓存/模型升级、四轮迭代计划 |
| [mcp-integration.md](./mcp-integration.md) | **MCP 接入指南**：Claude Desktop / Cursor / HTTP、10 工具速查；后续体验项见 Round 3 §十七 |
| [round-3-execution-plan.md](./round-3-execution-plan.md) | Round 3 执行计划（含 §十七 MCP 实测后续规划） |
| [round-4-execution-plan.md](./round-4-execution-plan.md) | Round 4 执行计划（含可选 T13 MCP 体验增强） |
| [routers-reference.md](./routers-reference.md) | 路由分类参考（页面渲染 vs 纯接口） |
| [router-split-migration-plan.md](./router-split-migration-plan.md) | 路由按职责拆分与 `/api` 前缀治理计划 |
| [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) | 同仓内前后端目录拆分迁移执行清单 |
| [upstream-mindoc-checklist.md](./upstream-mindoc-checklist.md) | 上游 [mindoc-org/mindoc](https://github.com/mindoc-org/mindoc) 提交跟进清单 |

> **阅读顺序建议**：先看 [refactor-roadmap.md](./refactor-roadmap.md) 建立全局视角，再按需深入到 `router-split-migration-plan.md`（路由）/`frontend-backend-split-migration-plan.md`（目录）/`upstream-mindoc-checklist.md`（上游同步）等执行细则。

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
