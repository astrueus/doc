# Round 6 · 执行文档（Vite / 搜索 / 拆分债）

> 本文是 [refactor-roadmap.md](../refactor-roadmap.md) Round 6 的可执行分解。  
> **定位：** 承接 Round 5 **明确不再本轮处理**的项（2026-09-01）。编号沿用 R5（T3/T4/T6/T10/T11/T13），方便对照旧文档。  
> **状态（2026-09-01）：** 📝 已划入，**未开工**。Round 5 剩余落地项仍是 **T14 对象存储**、**T16 联邦登录**。

---

## 一、从 Round 5 划入

| 原 R5 | 任务 | 划入原因 |
|---|---|---|
| T6 | 前端 P2 Vite | 体量大，本轮不排 |
| T3 | 搜索方案（原 FULLTEXT 路线已否） | 持续冻结，等 [search-redesign.md](./search-redesign.md) |
| T4 | 倒排 / 向量（hybrid）评估 | 随 T3，解冻前不开 |
| T10 | Controller 按域拆分 | R5 已 ⏸；解冻后禁止平铺多文件 |
| T11 | 安全头 CORS/CSP/HSTS | 可选；建议 T6 后再收紧 CSP |
| T13 | 拆 `bootstrap.go` | 待定，暂时不拆 |

**本轮仍不算进 Round 6 开工默认集、但可并列的既定路径（来自 R5 T2）：** ① 数据层 kratos 化（生成 model + `data`）→ ② 业务层（`biz`）；③ 全量 kratos HTTP 默认不做。见 [round-5-orm-migration-evaluation.md](../round-5/round-5-orm-migration-evaluation.md)。

### 明确不做（与 R5 一致）

- ❌ 完整 ORM / 框架替换实施（未单独立项前）
- ❌ 前端 P3/P4（Bootstrap5 / Vue SPA）——可作更后候选，不进本表默认开工
- ❌ MCP `delete_book`、OpenTelemetry
- ❌ 按旧 FULLTEXT/FTS5 底稿编码

---

## 二、建议顺序（开工时）

```
T6 Vite P2（独立 sprint）
  → T11 安全头（CSP 跟构建走）
T10 Controller 子包拆分（T9 已完成，可与 T6 错开）
T3 须先定稿 search-redesign 再实施；T4 更后
T13 仅当 bootstrap 成为阻塞时再开
kratos ① data → ② biz：按域，先 Document，与 Vite 无强依赖
```

---

## 三、任务与文档

| # | 任务 | 细化 | 状态 |
|---|---|---|---|
| T6 | Vite P2 | [round-6-t6-vite.md](./round-6-t6-vite.md) | ⏳ 未开工 |
| T3 | 搜索重定义后实施 | [search-redesign.md](./search-redesign.md) | ⏸ 引擎未决 |
| T4 | hybrid / 向量评估 | 同上 §十 | ⏸ 随 T3 |
| T10 | Controller 子包按域拆 | [round-6-t10-controller-split.md](./round-6-t10-controller-split.md) | ⏸ 方向已定、未拆 |
| T11 | CORS / CSP / HSTS | [round-6-t11-security-headers.md](./round-6-t11-security-headers.md) | ⏳ 建议 T6 后 |
| T13 | 拆 bootstrap.go | 下文 §四 | ⏸ 待定 |

Round 5 侧旧路径已改成跳转 stub：`docs/round-5/round-5-t6-vite.md` 等。

---

## 四、T13 · 拆 bootstrap.go（备忘）

Round 2 B2。功能可用，不挡 T6/T14。若拆：`app.go` + `bootstrap.go` + `web.go`，另开任务。未开工前不要为「整洁」而拆。

---

## 五、解冻搜索（T3）

须同时满足：

1. [search-redesign.md](./search-redesign.md) 引擎选型拍板  
2. 本追踪表将 T3 改为可排期  

过渡期 Web / MCP 继续 `LIKE`。

---

## 六、验收（划入本次）

- [x] Round 5 追踪表标明移交；R5 不再把 T6/T3/T4/T10/T11/T13 当本轮待办  
- [ ] 上述任务本身的实施验收：开工后再勾  

---

## 七、参考

- [round-5-execution-plan.md](../round-5/round-5-execution-plan.md)（已完成项与 T14/T16）  
- [round-4-execution-plan.md](../round-1-4/round-4-execution-plan.md) T9 Vite 底稿  
- [round-5-orm-migration-evaluation.md](../round-5/round-5-orm-migration-evaluation.md)
