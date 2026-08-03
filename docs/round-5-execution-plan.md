# Round 5 · 执行文档（搜索可用化 + 前端构建 + 工程化收尾）

> 本文是 [refactor-roadmap.md §五 Round 5](./refactor-roadmap.md#🎯-round-5搜索可用化--前端构建--工程化收尾3~5-周按需推进) 的**可执行分解**。
> **定位：** 承接 Round 3/4 **明确遗留**与「Round 5+ 候选」中可立刻排期的项——把搜索从 `LIKE` 过渡到可用方案、补齐评估报告、落地 Vite 构建、补测试工程化，并消化分层债（`*Result` / Repo 扩面）。  
> **按需推进：** 大块（Vite）与数据驱动项（倒排索引、上 gocache）可独立 sprint；评估类与小增强可并行。
>
> **状态（2026-08-03 规划）：** ⏳ **未开工**。遗留清单来源见 [§二](#二遗留来源与前置)。明细以 [§十四 追踪表](#十四追踪表) 为准。

---

## 一、范围与不做清单

### 本轮做

| 序号 | 任务 | 工作量 | 来源 | 阻塞其他吗 |
|---|---|---|---|---|
| T1 | 缓存方案评估报告（默认「维持 A」；有数据再议 gocache / singleflight） | 0.5~1 天 | Round 4 T7 | 否（T12 实施依赖本报告） |
| T2 | ORM 迁移评估报告（只评估不实施） | 1~3 天 | Round 4 T12 | 否（实施另立项） |
| T3 | 搜索最小方案：MySQL FULLTEXT / SQLite FTS5 + 标题加权 + Provider | 2~4 天 | Round 3 T1 | 否（MCP/Web 搜索共用） |
| T4 | （可选）倒排 / 向量检索评估或 PoC（bleve / meilisearch 等） | 独立 | Round 4 T11 | 否；**数据闸门** |
| T5 | MCP 体验 P1：upsert / get 截断 / search 带 identify | 0.5~1 天 | Round 4 T13 P1 / Round 3 §十七 | 否 |
| T6 | 前端 P2：Vite 构建 + vendor 集中 + 抽离内联 JS | 1~2 周 | Round 4 T9 | 否（建议单独 sprint） |
| T7 | 测试工程化：`scripts/test.sh` + CI job + 覆盖率门槛 | 1~2 天 | Round 4 T10 缺口 | 否 |
| T8 | `*Result` → `internal/dto/`（解耦 model 循环依赖） | 2~4 天 | Round 2 收尾 B1 | 否 |
| T9 | Repository 扩面 +（可选）`internal/service/` | 3~5 天 | Round 4 有意缺口 | 否（渐进） |
| T10 | （可选）Controller 按域拆分（优先 `DocumentController`） | 3~5 天 | Round 2 延后 / 路线图 | 否 |
| T11 | （可选）安全头：CORS / CSP / HSTS + secure middleware | 1~2 天 | Round 4+ 候选 | 否 |
| T12 | （闸门）缓存方案 B **实施**（gocache/v3 或 singleflight + tag） | 2~3 天 | Round 4 T7 实施 | **依赖 T1 结论 + 压力数据** |
| T13 | （低优）拆 `internal/app/bootstrap.go` → `app.go` + `bootstrap.go` + `web.go` | 0.5~1 天 | Round 2 收尾 B2 | 否 |

**总工期估计：** 3~5 周（含 Vite）；若 Vite 单开 sprint，其余约 2~3 周可并行收口。

### 本轮**不做**（明确排除）

- ❌ **不实施完整 ORM 替换**（gorm / ent / sqlc）：本轮只出 **T2 评估报告**；若结论为「迁移」，单独立项（Round 6+），勿与 Vite / 搜索并行硬刚。
- ❌ **不做前端 P3/P4**：Bootstrap 3→5 / Tailwind、Vue 3 + TS 完整 SPA —— 仍为长期目标。
- ❌ **不做 MCP Book 写工具**（`create_book` / `update_book` / `delete_book`）—— 延续 Round 3 §17.3。
- ❌ **不上 OpenTelemetry 全链路** —— 候选保留，本轮不排。
- ❌ **不强制上倒排/向量检索** —— T4 仅在有 2~4 周召回/投诉/体量数据时评估；无数据则维持 T3 的 SQL 方案。

---

## 二、遗留来源与前置

### 2.1 从哪里来

| 遗留项 | 原轮次位置 | Round 5 承接 |
|---|---|---|
| 搜索仍为 `LIKE`；FULLTEXT/FTS5 未做 | Round 3 T1 ⏸ | **T3** |
| MCP P1（upsert / get 截断 / search identify） | Round 3 §十七 / Round 4 T13 | **T5** |
| 缓存「维持 A」报告未写；gocache 未实施 | Round 4 T7 | **T1** → 有条件 **T12** |
| ORM 评估报告未写 | Round 4 T12 | **T2** |
| 倒排/向量检索等数据 | Round 4 T11 | **T4**（闸门） |
| Vite P2 未开始 | Round 4 T9 | **T6** |
| 无 `scripts/test.sh` / CI 硬闸 | Round 4 T10 | **T7** |
| 无 `internal/service/`；Repo 仅 MCP 写路径 | Round 4 T2 有意简化 | **T9** |
| `*Result` 仍在 `model/` | Round 2 B1 ⏭ | **T8** |
| `bootstrap.go` 未拆 | Round 2 B2 ⏭ | **T13**（低优） |
| Controller 大文件未按域拆 | Round 2「本轮不做」 | **T10**（可选） |
| CORS/CSP/HSTS | Round 4+ 候选 | **T11**（可选） |

权威完成度快照：[round-4-execution-plan.md §十六附](./round-4-execution-plan.md#十六附完成度核查2026-08-03)、[docs/README.md](./README.md)。

### 2.2 前置条件（开工前核对）

| 前置 | 状态（规划日） | 说明 |
|---|---|---|
| Round 1–2 目录 / `conf/` / 强类型 Config | ✅ | 定型 |
| Round 3 MCP MVP + §十七 P0 | ✅ | 写路径已走 `DocumentRepo` |
| Round 4 代码主线（T1–T6、T8、T10 pkg、T2/T5/T13 P0） | ✅ | 合入 `v2.2.1` |
| 生产或 staging 有 MCP / 搜索使用反馈 | ⚠️ | **T4 / T12 拍板**需要；T1–T3/T5–T11 不强制 |
| 团队同意 Vite 单独 sprint 或可并行 | 待确认 | T6 体量大，建议排期时显式切分 |

### 2.3 建议开工顺序

```
批次 A（文档 / 低风险，可立刻）
  T1 缓存报告 → T2 ORM 报告 → T7 测试 CI

批次 B（搜索 + MCP，可并行）
  T3 FULLTEXT/FTS5 Provider → T5 MCP P1
  （有数据后再）T4 倒排评估

批次 C（分层债）
  T8 *Result→dto → T9 Repo/Service 扩面 →（可选）T10 Controller 拆分

批次 D（大块前端 · 建议独立 sprint）
  T6 Vite P2

批次 E（闸门 / 低优）
  T12 仅当 T1 结论要上缓存 B
  T11 安全头、T13 bootstrap 拆分 —— 有空再做
```

**一句话：** 先把「欠的报告 + 搜索可用 + CI」收口；Vite 单开；ORM/gocache/倒排靠报告与数据拍板，默认不实施大迁移。

---

## 三、T1 · 缓存方案评估报告（0.5~1 天）

> 承接 [round-4-execution-plan.md §九](./round-4-execution-plan.md#九t7--缓存方案-b-评估2~3-天)。Round 4 **只允许「维持 A」的保守结论**；本轮把报告写完，并定义何时允许实施（→ T12）。

### 输出

`docs/round-5-cache-evaluation.md`（或并入本文附录），至少包含：

1. **现状（方案 A）**：`internal/cache` 接口 + msgpack + 现有实现（file/memory/redis 等）能力与缺口  
2. **候选 B**：`eko/gocache/v3` vs `singleflight` + 本地 tag 失效 —— 成本、依赖、与 Beego/MCP 读写路径的契合度  
3. **观测建议**：需要哪些指标（QPS、DB 慢查询、MCP 写后读一致性）才拍板上 B  
4. **结论三选一**：维持 A / 仅上 singleflight / 上 gocache（附最小落地切片）

### 验收

- [ ] 报告合入 `docs/`，并在路线图决策日志记一笔  
- [ ] 若结论为「维持 A」：关闭 T12 或标为 ⏭  
- [ ] 若结论为「上 B」：写明前置指标与 T12 范围，**仍不在本任务实施**

---

## 四、T2 · ORM 迁移评估报告（1~3 天 · 不实施）

> 承接 [round-4-execution-plan.md §十三](./round-4-execution-plan.md#十三t12--orm-迁移评估报告3-天--不实施)。输出路径可改为：

`docs/round-5-orm-migration-evaluation.md`

### 内容要求（与 Round 4 一致）

1. 三候选对比：`gorm` / `ent` / `sqlc`  
2. 工作量（按 model 粗估）  
3. 破坏面：raw SQL、`GetDatabasePrefix()`、Filter 链、Repository 初版如何衔接  
4. 收益与风险  
5. 推荐：**A 不迁** / **B Round 6+ 全迁** / **C 仅新代码用新 ORM**

### 验收

- [ ] 报告合入；团队拍板写入 [refactor-roadmap.md §八](./refactor-roadmap.md#八决策记录decision-log)  
- [ ] **本轮无** ORM 切换 PR（除非另开 Round 6 立项）

---

## 五、T3 · 搜索最小方案 FULLTEXT / FTS5（2~4 天）

> 承接 Round 3 [§六 搜索](./round-3-execution-plan.md) 底稿与 T1。目标：Web 全局搜索与 MCP `search_document` **同一 Provider**，摆脱纯 `LIKE` 作为唯一路径。

### 步骤（摘要）

1. 抽象 `SearchProvider`（若 Round 3 草稿未落地则本轮新建于 `internal/` 合适包，如 `internal/search/`）  
2. **MySQL**：`FULLTEXT` + ngram（或项目既定 parser）；标题加权  
3. **SQLite**：FTS5 虚表或等价方案；install/upgrade 文档写清  
4. 降级：`sqlLikeProvider` 保留，无索引时自动 fallback  
5. 接入：`DocumentSearchResult` / MCP `search_document` 走 Provider  
6. `conf/app.conf` `[search]`（若尚无）增加开关：`provider=auto|fulltext|like`

### 风险

- 大表 `ALTER TABLE ADD FULLTEXT` 锁表 → **升级手册**要求手工/低峰执行，勿 silent 锁生产  
- 中文分词效果因引擎而异 → 验收以「明显优于 LIKE」为底线，不追求搜索产品级

### 验收

- [ ] MySQL / SQLite 各至少一条自动化或手工冒烟：按标题命中优先  
- [ ] MCP `search_document` 与 Web 搜索同源 Provider  
- [ ] 无索引环境仍可 LIKE 降级，不崩  
- [ ] 更新 [mcp-integration.md](./mcp-integration.md) / upgrade note

---

## 六、T4 · 倒排 / 向量检索评估（可选 · 数据闸门）

> 承接 Round 4 T11。触发条件与 Round 4 相同：

- MCP / Web 搜索召回投诉或「搜不到」反馈  
- 文档体量上来后 LIKE/FULLTEXT 不够用  
- 有人愿意运维独立服务（若选 meilisearch 等）

### 输出

`docs/round-5-search-backend-evaluation.md`：bleve / meilisearch /（可选）向量方案对比 + **做 / 不做**结论。

### 验收

- [ ] 无数据则本任务 **⏭ 跳过**，在追踪表注明原因  
- [ ] 有数据则给出推荐与是否单独立项（不强制本轮实施）

---

## 七、T5 · MCP 体验 P1（0.5~1 天）

> 承接 Round 3 §十七 P1 / Round 4 T13 未做部分。**仍不含** Book 写工具。

| 项 | 说明 |
|---|---|
| P1-1 | `upsert_document` 或 `create_document` 增加 `if_exists=update`（按 `book_id + identify`） |
| P1-2 | `get_document`：`include_release=false` / `markdown_max_chars` 等截断 |
| P1-3 | `search_document` 返回 `book_identify` / `doc_identify` |

### 验收

- [ ] 工具契约写入 `mcpdto` + [mcp-integration.md](./mcp-integration.md)  
- [ ] 乐观锁 / 权限行为与现有写工具一致  
- [ ] 回归：既有 10 工具 + P0 行为不回归

---

## 八、T6 · 前端 P2 Vite（1~2 周）

> 完整步骤沿用 [round-4-execution-plan.md §十 T9](./round-4-execution-plan.md#t9--p21~2-周)，本轮正式排期执行。要点：

- 新增 `web-ui/`（Vite + TS entries）  
- 产物 `web/static/dist/` + `manifest.json`  
- 模板函数 `vite_asset`  
- dev 反代 / 生产静态  
- **渐进**抽离 `web/views` 内联 JS（每 PR 1~2 模板）  
- **不做** Bootstrap 升级、不做 Vue SPA

### 验收

- [ ] `npm run build` 成功；生产资源带 hash 可访问  
- [ ] 关键页面（文档编辑 / 阅读等）无功能回退  
- [ ] 内联 `<script>` 数量明显下降（目标 ≤ 5 或文档化剩余清单）

---

## 九、T7 · 测试工程化（1~2 天）

### 做

1. `scripts/test.sh`（及 Windows 可用的 `scripts/test.ps1` 或文档说明）：`go test -race -cover ./...`（可按包白名单起步）  
2. CI（Gitea Actions / 现有流水线）加测试 job；失败阻断 merge（至少对 `pkg/` + `internal/errs|auth|logging|i18n|repository`）  
3. 更新 [round-4-coverage.md](./round-4-coverage.md) 或新建 `docs/round-5-coverage.md` 快照  
4. 覆盖率门槛：**先保基线不回退**；全仓硬闸可设宽松（如 ≥ 15% 起步），Repository / pkg 维持 Round 4 目标

### 验收

- [ ] 本地一键跑测文档写入 README 或 AGENTS  
- [ ] CI 绿；主分支受保护（若仓库策略允许）

---

## 十、T8 · `*Result` → `internal/dto/`（2~4 天）

> Round 2 B1 曾因 orm/`NewBook()` 循环依赖跳过。本轮原则：

1. **纯展示/组装结构**迁入 `internal/dto/`  
2. **仍含查询逻辑**的，先抽方法到 Repository / 保留 thin wrapper，避免 dto↔model 循环  
3. 一次迁一类（Book / Document / Member…），每类可独立 PR

### 验收

- [ ] 选定清单中的 `*Result` 迁完或文档标明「暂留 + 原因」  
- [ ] `go build ./...`；关键页面（项目首页、文档阅读）冒烟  
- [ ] 决策日志更新 B1 最终态

---

## 十一、T9 · Repository 扩面 + 可选 Service（3~5 天）

### 扩面优先级

1. MCP **读**工具也走 Repository（与写路径对称）  
2. 高频 Web 读路径（文档树、Book 元数据）择一二试点  
3. （可选）新增 `internal/service/`：编排乐观锁、权限、历史快照；Controller/MCP 变薄  

### 原则

- **渐进**：老 Controller 可不一次性改完  
- 有 Service 则测 Service；无则继续测 Repository  
- 不借机做 ORM 替换

### 验收

- [ ] 至少 MCP 读写均经 Repo（或文档说明仍直调的例外）  
- [ ] 新增/扩充测试；覆盖率不回退  

---

## 十二、T10 · Controller 域拆分（可选）

优先：`DocumentController` → `internal/controller/document/{read,edit,history,export}.go`（同 package 或子 package，与 Round 4 BookModel 拆分风格对齐）。

### 验收

- [ ] 路由注册无行为变化  
- [ ] 文件体量下降；无逻辑大改（纯搬迁 + 小整理）

---

## 十三、T11 · 安全头（可选）

- 可配置 CORS（若有跨域 API / MCP HTTP 需求）  
- CSP / HSTS 以**可关闭的默认安全基线**落地，避免一次弄死内联脚本（**与 T6 Vite 顺序协调**：Vite 后再收紧 CSP 更稳）  
- 统一 middleware，挂 `internal/middleware/`

### 验收

- [ ] `conf` 可开关；文档说明生产推荐值  
- [ ] 主站与 `/mcp` 冒烟通过  

---

## 十四、追踪表

> 规划日：2026-08-03。合入后填 PR / Commit / 状态。

| # | 任务 | PR | Commit | 状态 |
|---|---|---|---|---|
| T1 | 缓存评估报告 | | | ⏳ |
| T2 | ORM 评估报告 | | | ⏳ |
| T3 | 搜索 FULLTEXT/FTS5 + Provider | | | ⏳ |
| T4 | 倒排/向量评估 | | | ⏳ 闸门 |
| T5 | MCP P1 | | | ⏳ |
| T6 | Vite P2 | | | ⏳ |
| T7 | 测试 CI / 覆盖率门槛 | | | ⏳ |
| T8 | `*Result` → dto | | | ⏳ |
| T9 | Repo 扩面 + 可选 Service | | | ⏳ |
| T10 | Controller 域拆分 | | | ⏳ 可选 |
| T11 | 安全头 middleware | | | ⏳ 可选 |
| T12 | 缓存 B 实施 | | | ⏳ 闸门 |
| T13 | 拆 bootstrap.go | | | ⏳ 低优 |

---

## 十五、PR 拆分建议

| # | PR 标题（示例） | 内容 | 大小 | 依赖 |
|---|---|---|---|---|
| 1 | `docs(round5): cache evaluation stay-A or scheme-B` | T1 | 小 | 无 |
| 2 | `docs(round5): ORM migration evaluation` | T2 | 小 | 无 |
| 3 | `feat(search): FULLTEXT/FTS5 provider with LIKE fallback` | T3 | 中 | 无 |
| 4 | `docs(round5): search backend evaluation` | T4 | 小 | 数据 |
| 5 | `feat(mcp): P1 upsert / truncate / search identify` | T5 | 小 | 无（与 T3 可并行，注意 dto） |
| 6 | `feat(frontend): Vite pipeline + vite_asset` | T6 | 大 | 建议独立 |
| 7 | `ci: add test script and coverage gate` | T7 | 小 | 无 |
| 8 | `refactor(dto): move *Result out of model` | T8 | 中 | 无 |
| 9 | `refactor: expand repository (+ optional service)` | T9 | 中 | T8 部分交叉时串行 |
| 10 | `refactor(controller): split DocumentController` | T10 | 中 | 可选 |
| 11 | `feat(security): CORS/CSP/HSTS middleware` | T11 | 小 | 建议 T6 后 |
| 12 | `feat(cache): gocache or singleflight` | T12 | 中 | T1 |
| 13 | `refactor(app): split bootstrap.go` | T13 | 小 | 无 |

---

## 十六、上线 / 冒烟清单

- [ ] 搜索：中英文关键词；无索引环境降级  
- [ ] MCP：stdio initialize；HTTP Bearer；P1 工具（若做）  
- [ ] 前端：编辑页 / 阅读页静态资源 200（Vite 后）  
- [ ] `go test` / CI 绿  
- [ ] 若动 session/缓存实现：按既有 SOP 清 session + cache  
- [ ] FULLTEXT 升级步骤写入 upgrade / CHANGELOG  

---

## 十七、Round 6+ 候选（本轮结束再讨论）

- ORM 全量或「新代码新 ORM」实施（取决于 T2）  
- 前端 P3（Bootstrap 5 / Tailwind）、P4（Vue 3 SPA）  
- 倒排/向量检索独立服务落地（取决于 T4）  
- OpenTelemetry  
- 覆盖率冲到 ≥ 40%  
- Beego 大版本或替代框架评估（仅当社区风险不可接受）

---

## 十八、参考

- [refactor-roadmap.md](./refactor-roadmap.md) — 总纲 §五 / §七  
- [round-3-execution-plan.md](./round-3-execution-plan.md) — T1 搜索底稿、§十七 P1  
- [round-4-execution-plan.md](./round-4-execution-plan.md) — T7/T9/T11/T12/T13、§十六附遗留  
- [round-4-coverage.md](./round-4-coverage.md) — 测试基线  
- [mcp-integration.md](./mcp-integration.md) — MCP 接入与体验约定  
- [upgrade-round-2.md](./upgrade-round-2.md) — 升级 SOP 可对照扩展搜索/缓存说明  
- [Vite](https://vitejs.dev/) · [eko/gocache](https://github.com/eko/gocache)

---

## 十九、目录锚点（便于链接）

- [§一 范围](#一范围与不做清单)  
- [§二 遗留来源](#二遗留来源与前置)  
- [§十四 追踪表](#十四追踪表)  
- [§十五 PR 拆分](#十五pr-拆分建议)  
