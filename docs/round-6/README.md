# Round 6

承接 Round 5 **未做且本轮不再排期**的项：Vite、搜索重定义、Controller 拆分、安全头、拆 bootstrap。总表见 [round-6-execution-plan.md](./round-6-execution-plan.md)。

**进度（2026-09-01）：** 📝 已从 Round 5 划入；**尚未开工**。Round 5 仍推进 T14 对象存储、T16 OAuth2。

## 执行计划

| 文档 | 说明 |
|------|------|
| [round-6-execution-plan.md](./round-6-execution-plan.md) | 本轮任务总表 |
| [search-redesign.md](./search-redesign.md) | T3/T4 解冻前提（搜索重定义） |

## 任务细化（编号沿用 Round 5，避免和已合入项撞名）

| 文档 | 任务 |
|------|------|
| [round-6-t6-vite.md](./round-6-t6-vite.md) | T6 Vite P2 |
| [round-6-t10-controller-split.md](./round-6-t10-controller-split.md) | T10 Controller 拆分 |
| [round-6-t11-security-headers.md](./round-6-t11-security-headers.md) | T11 安全头（建议 T6 之后） |

T13 拆 `bootstrap.go` 无独立细化文，见执行计划。kratos 数据层/业务层仍按 Round 5 T2 既定路径，本轮可排、默认不与 Vite 绑死。

总纲：[../refactor-roadmap.md](../refactor-roadmap.md)
