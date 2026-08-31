# Round 5 · 执行文档（工程化收尾 + 对象存储 + 分层评估）

> 本文是 [refactor-roadmap.md §五 Round 5](../refactor-roadmap.md#🎯-round-5搜索可用化--前端构建--工程化收尾3~5-周按需推进) 的**可执行分解**。  
> **定位：** 承接 Round 3/4 遗留中**仍可推进**的项——补齐评估报告、Vite、测试工程化、分层债；并新增**对象存储**、**`scripts/`→`deployments/`**、**OAuth2 登录重写**。  
> **按需推进：** 大块（Vite / 对象存储）可独立 sprint；评估类与小增强可并行。
>
> **状态（2026-08-31 修订）：** 🔶 **批次 A + T5/T8/T9 已合入**；尾巴已收口（T7 云端 CI 绿）；**T12-a/b/c 已落地**（Token 仍待 T12-d）。搜索 T3/T4 **持续冻结**。明细以 [§十四 追踪表](#十四追踪表) 为准。  
> **当日决策：** 见 [§一附](#一附2026-08-03-决策修订)（搜索 FULLTEXT 暂缓重定义、bootstrap 暂不拆、ORM 评估朝 kratos 靠拢等）。

---

## 一、范围与不做清单

### 本轮做

| 序号 | 任务 | 工作量 | 来源 | 阻塞其他吗 |
|---|---|---|---|---|
| T1 | 缓存**完全重构**评估（L1+L2 / 防击穿穿透雪崩 / Soft-TTL） | 0.5~1 天 | Round 4 T7 + 2026-08-05 决策 | 否（结论驱动 T12） |
| T2 | ORM / 分层评估报告（**朝 kratos 分层靠拢**；模型可生成；本轮只评估不实施） | 2~4 天 | Round 4 T12 + 本轮决策 | 否（实施另立项） |
| T3 | ~~搜索最小方案 FULLTEXT/FTS5~~ | — | Round 3 T1 | ⏸ **暂不实施**（见 §一附） |
| T4 | （可选）倒排 / 向量检索评估 | 独立 | Round 4 T11 | ⏸ **随搜索方案冻结**（见 §一附） |
| T5 | MCP 体验 P1：upsert / get 截断 / search 带 identify + **Book 写最小集**（create/update） | 1~1.5 天 | Round 4 T13 P1 + §17.3 修订 | 否 |
| T6 | 前端 P2：Vite 构建 + vendor 集中 + 抽离内联 JS | 1~2 周 | Round 4 T9 | 否（建议单独 sprint） |
| T7 | 测试工程化：测试脚本 + CI job + 覆盖率门槛（路径见 T15） | 1~2 天 | Round 4 T10 缺口 | 否；与 T15 协调落点 |
| T8 | `*Result` → `internal/dto/`（解耦 model 循环依赖） | 2~4 天 | Round 2 收尾 B1 | 否 |
| T9 | Repository 扩面 +（可选）`internal/service/` | 3~5 天 | Round 4 有意缺口 | 否（渐进；与 T2 方向对齐） |
| T10 | （可选）Controller 按域拆分（优先 `DocumentController`） | — | Round 2 延后 | ⏸ **本轮暂不拆** |
| T11 | （可选）安全头：CORS / CSP / HSTS + secure middleware | 1~2 天 | Round 4+ 候选 | 否 |
| T12 | 缓存**完全重构**实施（Facade + Ristretto + Redis） | 4~7 天 | T1 终裁 | 建议与 T9 协同 |
| T13 | ~~拆 `bootstrap.go`~~ | — | Round 2 B2 | ⏸ **待定，暂时不拆**（见 §一附） |
| **T14** | **对象存储完全重构（S3 API）+ 旧 `uploads/` 全量迁移** | 2~3 周 | **本轮新增** | 否（建议独立 sprint） |
| **T15** | **`scripts/` → `deployments/scripts/`**（方案 A 全迁 + 根 Makefile/justfile） | 0.5~1 天 | **本轮新增** | 影响 T7 脚本落点 |
| **T16** | **OAuth2 登录重写**：统一 Provider；钉钉迁入；新增企业微信 | 3~5 天 | upstream §3.2 / #851 | 否（建议独立切片） |

**总工期估计：** 3~5 周（含 Vite / 对象存储 / OAuth2）；搜索与 bootstrap **不占工期**。

### 本轮**不做** / 冻结（明确排除）

- ⏸ **不实施** Round 3/5 原定的 **MySQL FULLTEXT / SQLite FTS5 + 标题加权 + Provider**：当前技术方案不够完善；**待后续重新定义搜索方案**。在新方案落地前，**相关代码与文档底稿一律不再推进**（含 T3 实施、以及依赖「先上 FULLTEXT」的改造）。过渡期继续 `LIKE`。
- ⏸ **不拆** `internal/app/bootstrap.go`（T13）：**待定，暂时不拆**。
- ❌ **不实施完整 ORM / 框架替换**：本轮只出 **T2 评估报告**；若结论为「按 kratos 分层迁移」，单独立项（Round 6+）。
- ❌ **不做** MCP `delete_book`、Book 封面/成员等重能力 —— 见 [T5 细化 §五](./round-5-t5-mcp-p1.md)；**本轮做** `create_book` / `update_book` 最小集。  
- ❌ **不做前端 P3/P4**：Bootstrap 3→5 / Tailwind、Vue 3 SPA。  
- ❌ **不上 OpenTelemetry** —— 本轮不排。
- ⏸ **不上倒排/向量检索评估实施**（T4）：在搜索总方案重定义之前不启动；有数据也不抢跑 FULLTEXT 旧路线。

---

## 一附、2026-08-03 决策修订

| # | 决策 | 结论 | 影响任务 |
|---|---|---|---|
| 1 | 搜索最小方案（FULLTEXT/FTS5 + Provider） | **暂不实施**；方案不完善，**须重新定义**；重定义前不再处理 | T3 ⏸；T4 ⏸；过渡期 LIKE |
| 2 | 拆 `bootstrap.go` | **待定，暂时不拆** | T13 ⏸ |
| 3 | 文件上传与对象存储 | **本轮增加规划与落地切片**：新上传支持对象存储 + 旧 `uploads/` 迁移 | **T14** |
| 4 | `scripts/` 与 `deployments/` | **方案 A 全迁** + 根 **`Makefile`/`justfile`** 快捷入口 | **T15**（牵动 T7 路径） |
| 5 | ORM 评估方向 | **偏向朝 [go-kratos](https://go-kratos.dev/) 分层靠拢**；期望 **model 可自动化生成**；模型 / 业务逻辑 / 数据访问拆开 | **T2** 重写评估维度 |
| 6 | MCP Book 写工具（2026-08-04） | **本轮做最小集** `create_book` / `update_book`；**不做** `delete_book`（修订 Round 3 §17.3） | **T5** |
| 7 | DocumentController 拆分（2026-08-04） | **本轮暂不拆**；解冻后用子包按域拆，**禁止平铺**多文件 | **T10** ⏸ |
| 8 | OAuth2 登录重写（2026-08-04） | **本轮做**：统一 Provider + 钉钉迁移 + 企微；修订 upstream「仅并行加企微、不重写」 | **T16** |
| 9 | 缓存完全重构（2026-08-05） | **不兼容旧 beego/cache**；L1 Ristretto + L2 Redis + Soft-TTL / 防护套件；见 T1/T12 细化 | **T1** ✅ 结论 · **T12** 实施 |

已记入 [refactor-roadmap.md §八](../refactor-roadmap.md#八决策记录decision-log)。

---

## 二、遗留来源与前置

### 2.1 从哪里来

| 遗留项 | 原轮次位置 | Round 5 承接 |
|---|---|---|
| 搜索仍为 `LIKE`；FULLTEXT 未做 | Round 3 T1 | **T3 ⏸**（冻结，等新方案） |
| MCP P1 | Round 3 §十七 / Round 4 T13 | **T5** |
| 缓存报告 / gocache | Round 4 T7 | **T1** ✅ 完全重构结论 → **T12** 实施 |
| ORM 评估 | Round 4 T12 | **T2**（方向改为 kratos 分层） |
| 倒排/向量 | Round 4 T11 | **T4 ⏸**（随搜索冻结） |
| Vite P2 | Round 4 T9 | **T6** |
| 无测试脚本 / CI 硬闸 | Round 4 T10 | **T7**（+ **T15** 落点） |
| 无 Service；Repo 仅写路径 | Round 4 T2 | **T9** |
| `*Result` 仍在 `model/` | Round 2 B1 | **T8** |
| `bootstrap.go` 未拆 | Round 2 B2 | **T13 ⏸** |
| Controller 域拆分 | Round 2 | **T10**（可选） |
| CORS/CSP/HSTS | Round 4+ | **T11**（可选） |
| 上传仅本地 `uploads/` | 现状 | **T14**（新增） |
| 根目录 `scripts/build.*`（已迁走） | Round 2 后部署已在 `deployments/` | **T15**（已落地） |
| 钉钉登录无统一 OAuth；无企微 | upstream §3.2 | **T16**（新增） |

### 2.2 前置条件（开工前核对）

| 前置 | 状态 | 说明 |
|---|---|---|
| Round 1–4 目录 / MCP / 质量主线 | ✅ | `v2.2.1` |
| 对象存储选型意向（S3 兼容 / 厂商 SDK） | 待确认 | T14 开工前定默认适配器 |
| 团队同意 Vite / OSS 单独 sprint | 待确认 | T6、T14 体量大 |

### 2.3 建议开工顺序

```
批次 A（文档 / 低风险）
  T1 缓存报告 → T2 ORM/分层（kratos 方向）报告 → T15 scripts 落点评估
  → T7 测试 CI（按 T15 结论选路径）

批次 B（MCP + 分层债）
  T5 MCP P1
  T8 *Result→dto → T9 Repo/Service
  （T10 Controller 拆分本轮不开）

批次 C（大块 · 建议独立 sprint）
  T6 Vite P2
  T14 对象存储 + 旧数据迁移
  T16 OAuth2 登录重写（钉钉 + 企微）

批次 D（缓存重构 · 可与 B/C 交错）
  T12 缓存完全重构（建议 T9 读路径就绪后或并行约定边界）

冻结（本轮不开工）
  T3 搜索 FULLTEXT · T4 倒排 · T13 拆 bootstrap
```

**一句话：** 报告与 CI、MCP P1、分层债、Vite、对象存储、OAuth2、**缓存完全重构**可推进；**搜索旧方案、bootstrap、Controller 平铺拆分冻结**；ORM 只评估并朝 kratos 分层对齐。

---

## 三、T1 · 缓存完全重构评估（0.5~1 天）

> 承接 [round-4-execution-plan.md §九](../round-1-4/round-4-execution-plan.md#九t7--缓存方案-b-评估2~3-天)。  
> **细化方案 / 终裁：** [round-5-cache-evaluation.md](./round-5-cache-evaluation.md)（**2026-08-05：完全重构，不兼容旧方案**）

### 输出

已定稿，核心结论：

1. 废弃业务路径上的 `beego/cache`  
2. **L1 Ristretto + L2 Redis** + `GetOrLoad`  
3. 防护：singleflight、负缓存、TTL jitter、Soft-TTL 自动重建、Tag + Pub/Sub  
4. 可选引擎加速：jetcache-go（Facade 包装）  

### 验收

- [x] 报告合入；决策写入 §一附 #9  
- [x] T12 升格为实施项（非「维持 A 则关闭」）  

---

## 四、T2 · ORM / 分层评估报告（朝 kratos 靠拢 · 不实施）

> 承接 Round 4 T12，**评估维度按 2026-08-03 决策修订**。  
> **细化方案：** [round-5-orm-migration-evaluation.md](./round-5-orm-migration-evaluation.md)  
> 输出：`docs/round-5/round-5-orm-migration-evaluation.md`

### 评估目标（本轮共识）

1. **架构方向：** 向 [go-kratos](https://go-kratos.dev/) 的分层习惯靠拢（不必本轮引入完整 kratos 运行时），典型映射示意：
   - `api` / 契约（可选 protobuf）  
   - `internal/biz`（或 `service`）— **业务逻辑**  
   - `internal/data`（或强化现有 `repository`）— **数据访问**  
   - **model / entity** — 尽量 **自动化生成**，少手写 CRUD 胶水  
2. **模型可生成：** 对比 `ent` / `gorm gen` / `sqlc` / protobuf+代码生成 等，选出与「生成 model + 手写 biz」最匹配的路径。  
3. **拆分：** 明确现状 `internal/model`（实体 + 查询 + 业务混杂）如何迁到「生成实体 / 手写领域逻辑 / Repo」三层；与已有 `DocumentRepo` 如何衔接。  
4. **与 Beego 关系：** 可先 **① 仅数据层** kratos 化，但须同时规划 **② 业务层**（非可选项）；Web 仍 Beego；③ 全量换 kratos HTTP 默认不做。

### 内容要求

| 节 | 内容 |
|---|---|
| 现状债 | `beego/orm`、raw SQL、`*Result`、Controller 直调 model |
| 候选对比 | ① 维持 beego/orm + 扩 Repo；② ent/sqlc/gorm gen 生成 model + biz；③ 逐步引入 kratos 布局（biz/data/service）；④（可选）远期完整 kratos |
| 生成链路 | 表结构 / schema 来源、生成命令、与 migrate 的关系 |
| 工作量 | 按域（Document / Book / Member）粗估人天 |
| 破坏面 | 模板绑定、MCP、Session、前缀 `GetDatabasePrefix()` |
| 推荐 | **A 本轮维持渐进 Repo**；Round 6+ **既定**走 **① 数据层 kratos 化 → ② 业务层 kratos 化**（按域，先 Document）；③ 全量 kratos HTTP 默认不做 —— 见 [细化方案 §6.2](./round-5-orm-migration-evaluation.md#62-round-6-分阶段规划既定路径非可选项) |

### 验收

- [x] 报告合入；§八 决策日志写入拍板结论  
- [x] **本轮无** ORM/框架切换实施 PR  

---

## 五、T3 · 搜索最小方案 FULLTEXT / FTS5 — ⏸ 暂不实施

> 承接 Round 3 [§六](../round-1-4/round-3-execution-plan.md) 底稿。  
> **决策（2026-08-03）：技术方案不够完善，暂不实施；须后续重新定义搜索方案。在新方案定义并立项前，本任务及相关跟进（含旧 FULLTEXT 路线的编码）一律不再处理。**  
> **2026-08-31：** 再次确认**持续冻结**，本轮不排引擎选型、不做 spike、不改搜索代码。  
> **过渡：** Web / MCP 继续 `LIKE`。

### 历史底稿（仅存档，勿按此开工）

原步骤摘要（MySQL FULLTEXT / SQLite FTS5 / Provider / `[search]`）仍见 Round 3 §六；**不作为本轮验收项**。

### 解冻条件（须同时满足）

1. 有一份**新的**搜索技术方案（可另文 `docs/round-5/search-redesign.md`），覆盖引擎、索引列、中文分词、迁移/锁表、与 MCP 一致性；  
2. 路线图 / 本追踪表将 T3 改回可排期，并指定执行轮次。

### 验收（本轮）

- [x] 追踪表标 ⏸；不做 FULLTEXT 实施 PR  
- [ ] （解冻后）按新方案另开任务，不沿用本节目「默认验收」

---

## 六、T4 · 倒排 / 向量检索评估 — ⏸ 冻结

> 原触发条件（召回投诉、体量）仍有效，但 **在搜索总方案（T3）重定义之前不启动评估**，避免与旧 FULLTEXT 路线绑死。

### 验收（本轮）

- [x] 追踪表标 ⏸；无评估报告强制产出  

---

## 七、T5 · MCP 体验 P1（1~1.5 天）

> 承接 Round 3 §十七 P1，并**纳入 Book 写最小集**（修订 §17.3）。  
> **细化方案：** [round-5-t5-mcp-p1.md](./round-5-t5-mcp-p1.md)

| 项 | 说明 |
|---|---|
| P1-1 | `create_document` 增加 `if_exists=update`、可选 `auto_release`（默认 false；不读 `book.auto_release`） |
| P1-2 | `get_document` 截断选项 |
| P1-3 | `search_document` 返回 `book_identify` / `doc_identify`（仍可基于 LIKE） |
| P1-4 | **`create_book` / `update_book`**（元数据最小集；**不做** `delete_book`、封面） |

### 验收

- [x] `mcpdto` + [mcp-integration.md](../mcp-integration.md) 更新  
- [x] 权限 / 乐观锁与现网一致；原 10 工具 + P0 不回归（代码与 `internal/mcp` 单测，2026-08-31 核对）  
- [x] Book：可建空项目并 update 元数据；无 delete_book  
- [x] `create_document` 可用 `auto_release` 控制是否立刻发布  

---

## 八、T6 · 前端 P2 Vite（1~2 周）

> 步骤沿用 [round-4 §十 T9](../round-1-4/round-4-execution-plan.md#t9--p21~2-周)。本轮正式排期。  
> **细化方案：** [round-5-t6-vite.md](./round-5-t6-vite.md)

- `web-ui/` + `web/static/dist/` + `vite_asset`  
- 渐进抽离内联 JS；**不做** Bootstrap 升级 / Vue SPA  

### 验收

- [ ] `npm run build`；关键页无回退；内联 script 下降或清单化剩余  

---

## 九、T7 · 测试工程化（1~2 天）

> **细化方案：** [round-5-t7-testing.md](./round-5-t7-testing.md)（落点见 [T15 方案 A](./round-5-scripts-layout.md)）

### 做

1. 测试入口脚本：`go test -race -cover ./...`（可白名单起步）  
   - **路径（T15 = A）：** `deployments/scripts/test.sh`（及 ps1）  
2. CI 加测试 job；至少闸 `pkg/` + `internal/errs|auth|logging|i18n|repository`  
3. 覆盖率快照文档；门槛先保基线不回退  

### 验收

- [x] 一键跑测写入 README / AGENTS  
- [x] 本地白名单绿；CI workflow 已入库；**2026-08-31 确认云端绿**  

---

## 十、T8 · `*Result` → `internal/dto/`（2~4 天）

> **细化方案：** [round-5-t8-result-dto.md](./round-5-t8-result-dto.md)

1. 纯展示结构迁 `internal/dto/`  
2. 含查询逻辑的先抽 Repo，避免循环依赖  
3. 按类型分 PR  

### 验收

- [x] 清单完成或标明暂留原因；编译与相关包测试通过
- [x] 页面 / MCP 调用链已接好（2026-08-31 按代码核对；`SelectMemberResult` 暂留 model）

---

## 十一、T9 · Repository 扩面 + 可选 Service（3~5 天）

> **细化方案：** [round-5-t9-repo-service.md](./round-5-t9-repo-service.md)

1. MCP 读工具也走 Repo  
2. 高频 Web 读路径试点  
3. 可选 `internal/service/`（命名上可向 T2 的 biz 层靠拢，避免与未来 kratos 布局冲突）  

原则：渐进；不借机换 ORM。

### 验收

- [x] MCP 读写经 Repo（或文档化例外）；`go test ./internal/repository ./internal/mcp` 通过
- [x] 本轮以代码与单测收口（2026-08-31）；阅读页 `Read` 仍走 model 缓存，待 T12

---

## 十二、T10 · Controller 域拆分 — ⏸ 本轮暂不拆

> **细化方案：** [round-5-t10-controller-split.md](./round-5-t10-controller-split.md)  
> **决策（2026-08-04）：** Round 5 **暂不拆** `DocumentController`。解冻后采用 **子包按域拆**，**禁止**根目录平铺 `DocumentReadController.go` 一类方案。

### 验收（本轮）

- [x] 追踪表标 ⏸；无拆分 PR  

---

## 十三、T11 · 安全头（可选）

> **细化方案：** [round-5-t11-security-headers.md](./round-5-t11-security-headers.md)

CORS / CSP / HSTS（可关）；建议 T6 后再收紧 CSP。

### 验收

- [ ] conf 可开关；主站与 `/mcp` 冒烟  

---

## 十三附、T12 · 缓存完全重构实施（4~7 天）

> **细化方案：** [round-5-t12-cache-impl.md](./round-5-t12-cache-impl.md)  
> **不再**作为「维持 A 则跳过」的闸门项；上线前仍要压测与指标。  
> **进度（2026-08-31）：** T12-a/b/c 已落地（Document/Blog 走 Aside）；T12-d MCP Token / 压测未做。

### 做

- Facade `Aside[T]` + Ristretto + Redis + Soft-TTL / 负缓存 / jitter / Tag / Pub/Sub  
- Document / Blog / MCP Token 全量切换；旧 key 前缀废弃  

### 验收

- [ ] 见 T12 细化文档验收清单  

---

## 十三乙、T13 · 拆 bootstrap.go — ⏸ 待定，暂时不拆

> Round 2 B2。**决策（2026-08-03）：待定，暂时不拆。** 功能可用；不挡本轮其他项。  
> 若未来要拆：`app.go` + `bootstrap.go` + `web.go`，另开任务并改追踪表状态。

### 验收（本轮）

- [x] 追踪表标 ⏸；无拆分 PR  

---

## 十三丙、T14 · 对象存储完全重构 + 全量迁移（2~3 周）

> **细化方案：** [round-5-t14-object-storage.md](./round-5-t14-object-storage.md)（2026-08-05：不兼容旧写盘；S3 API；全量迁移）

### 本轮目标

1. **完全重构**：`internal/storage.BlobStore`；业务禁止直写本地 `uploads/`  
2. **远程仅 S3 API**：默认引擎 `aws-sdk-go-v2`；覆盖 MinIO / AWS / OSS / COS 等兼容后端  
3. **读路径**：生产默认预签名（或 CDN）；proxy 仅 fallback  
4. **全量迁移**：`doc storage migrate`（inventory → upload → verify → rewrite DB/正文 → cutover）  
5. **兼容性矩阵**：MinIO CI 必过 + 至少一家公有云抽测  

### 建议 PR 拆分

| PR | 内容 |
|---|---|
| T14-a | 接口 + local + s3（aws-sdk-go-v2）+ 配置 + 单测 |
| T14-b | 全部上传/下载/删除点切换 |
| T14-c | 全量 migrate + rewrite + 运维/矩阵文档 |

### 验收

- [ ] 见细化方案 §十  

---

## 十三丁、T15 · `scripts/` 是否迁入 `deployments/`（0.5~1 天）

> **细化方案：** [round-5-scripts-layout.md](./round-5-scripts-layout.md)  
> **决策（2026-08-04）：** ✅ **方案 A · 全迁** — `scripts/*` → `deployments/scripts/` 后**删除根 `scripts/`**；根目录用 **`Makefile` / `justfile`** 做快捷入口。

### 现状

- `scripts/`：`build.sh` / `build.bat` + [scripts/README.md](../../scripts/README.md)  
- `deployments/`：Docker / compose / spug / systemd / `start.sh` 等（Round 2 已集中部署资产）

### 评估问题

1. 构建脚本与部署脚本是否应同属「运维/发布」树？→ **是（A）**  
2. CI、文档、spug 是否已写死 `scripts/` 路径？→ 主要为文档；搬迁时批量改  
3. 测试脚本（T7）落点？→ **`deployments/scripts/test.*`**

### 方案对照（已拍板 A）

| 方案 | 做法 | 结果 |
|---|---|---|
| **A** | **迁入**：`scripts/*` → `deployments/scripts/` 后**删除根 `scripts/`**；根 `Makefile`/`justfile`；改 CI/文档 | ✅ **已选** |
| **B** | **维持**：`scripts/` = 构建+测试；`deployments/` = 运行时部署 | 未选 |
| **C** | **折中**：仅新建测试/发布辅助进 `deployments/scripts/`；`build.*` 暂留根 `scripts/` | 未选 |

### 验收

- [x] 书面结论写入 [round-5-scripts-layout.md](./round-5-scripts-layout.md)  
- [x] 路径搬迁 + 引用更新（README、release 文档、Actions）且 `deployments/scripts/build.sh` / `release.sh` 可跑  
- [x] T7 使用 `deployments/scripts/test.sh`  

---

## 十三戊、T16 · OAuth2 登录重写（3~5 天）

> **细化方案：** [round-5-t16-oauth2.md](./round-5-t16-oauth2.md)  
> 来源：[upstream-mindoc-checklist.md §3.2](../upstream-mindoc-checklist.md#32-oauth2-登录重写)；**本轮正式重写**（非「仅并行加企微」）。

### 目标

1. 统一 OAuth Provider 抽象（授权 / 换票 / 映射 Member）  
2. 现有**钉钉**登录迁入新框架并回归  
3. 新增**企业微信** Provider + 登录页入口  
4. 配置 / 环境变量按 `DOC_*` 硬切  

### 验收

- [ ] 钉钉登录行为不回归  
- [ ] 企微可完成登录建 session（enable 时）  
- [ ] 本地密码 / LDAP 不回归；`state` 校验有效  
- [ ] 细化方案与部署说明入库  

---

## 十四、追踪表

> 规划日：2026-08-03；**修订：2026-08-31 尾巴收口 + T12-a/b**。  
> 批次 A / T5 **无独立 PR**（功能分支快进直推 `master`）。T5 分支 `feat/r5-mcp-p1` 按约定保留。环境变量 `MINDOC_*`→`DOC_*` 不在 T 编号内，代码硬切为 `b7e7317`。

| # | 任务 | PR | Commit | 状态 |
|---|---|---|---|---|
| T1 | 缓存完全重构评估 | — | `0137b26` | ✅ 结论已定 |
| T2 | ORM/分层评估（kratos 方向） | — | `602ddb4` | ✅ 结论已定（本轮不实施） |
| T3 | 搜索 FULLTEXT/FTS5 | — | — | ⏸ **持续冻结**（不排期、不处理） |
| T4 | 倒排/向量评估 | — | — | ⏸ **持续冻结** |
| T5 | MCP P1 + Book 写最小集 | — | `80ee298` | ✅ 已合入；2026-08-31 验收框补勾 |
| T6 | Vite P2 | | | ⏳ |
| T7 | 测试 CI / 覆盖率门槛 | — | `602ddb4` | ✅ 脚本+workflow；**云端 CI 已确认绿** |
| T8 | `*Result` → dto | | | ✅ 已实施（`SelectMemberResult` 暂留 model） |
| T9 | Repo 扩面 + 可选 Service | | | ✅ 已实施（未建 service；阅读页 T12-c 已切） |
| T10 | Controller 域拆分 | — | — | ⏸ **暂不拆**（解冻后勿平铺） |
| T11 | 安全头 middleware | | | ⏳ 可选 |
| T12 | 缓存完全重构实施 | | | 🔶 **T12-a/b/c 已落地**；d ⏳ |
| T13 | 拆 bootstrap.go | — | — | ⏸ **待定，暂不拆** |
| T14 | 对象存储完全重构 + 全量迁移 | | | ⏳ |
| T15 | scripts → deployments **方案 A 全迁** | — | `602ddb4` | ✅ 已搬迁 |
| T16 | OAuth2 登录重写（钉钉 + 企微） | | | ⏳ |

---

## 十五、PR 拆分建议

| # | PR 标题（示例） | 内容 | 大小 | 依赖 |
|---|---|---|---|---|
| 1 | `docs(round5): cache evaluation` | T1 | 小 | 无 |
| 2 | `docs(round5): ORM/kratos-layer evaluation` | T2 | 小 | 无 |
| 3 | — | T3 | — | ⏸ 不开 |
| 4 | — | T4 | — | ⏸ 不开 |
| 5 | `feat(mcp): P1 upsert/truncate/search identify + create/update_book` | T5 | 中 | 无 |
| 6 | `feat(frontend): Vite pipeline + vite_asset` | T6 | 大 | 独立 |
| 7 | `ci: add test script and coverage gate` | T7 | 小 | 建议 T15 后或并行约定路径 |
| 8 | `refactor(dto): move *Result out of model` | T8 | 中 | 无 |
| 9 | `refactor: expand repository (+ optional service)` | T9 | 中 | 与 T8 串行若冲突 |
| 10 | — | T10 | — | ⏸ 不开 |
| 11 | `feat(security): CORS/CSP/HSTS middleware` | T11 | 小 | 建议 T6 后 |
| 12a–d | `feat(cache): aside L1/L2 + migrate callers` | T12 | 大 | T1；建议与 T9 协调 |
| 13 | — | T13 | — | ⏸ 不开 |
| 14a–c | `feat(storage): blobstore s3 + migrate all uploads` | T14 | 大 | 可独立 |
| 15 | `chore: relocate scripts under deployments + root Makefile/justfile` | T15 方案 A | 小 | 无 |
| 16a–b | `feat(auth): oauth provider + dingtalk migrate` / `feat(auth): wework provider` | T16 | 中 | 可独立 |

---

## 十六、上线 / 冒烟清单

- [x] MCP：stdio / HTTP；P1（含 create_book / update_book）— 2026-08-31 按代码与单测核对  
- [ ] 前端：Vite 后关键页 200  
- [x] `go test` / CI 绿（白名单脚本已绿；**云端 Actions 已确认**）  
- [ ] 对象存储：业务无直写盘；MinIO + 预签名；全量 migrate/verify/rewrite  
- [ ] OAuth2：钉钉回归；企微（若启用）登录成功  
- [x] 环境变量命名：硬切为 `DOC_*`（**不**再兼容 `MINDOC_*`；代码与 example 已切；部署连库冒烟仍须勾） — 见 [round-5-env-mindoc-to-doc.md](./round-5-env-mindoc-to-doc.md)  
- [ ] 若动缓存：新前缀生效；压测击穿/穿透；指标可看；可回滚 `mode`（T12-c 已切 Document/Blog；Token 与压测仍待 T12-d）  
- [x] **不做** FULLTEXT 升级步骤（本轮冻结；2026-08-31 再次确认持续不处理）  

---

## 十七、Round 6+ 候选

- 按 T2：**① 数据层 kratos 化（生成 model + `data`）→ ② 业务层（`biz` / service）**，按域闭环（先 Document）  
- **搜索方案重定义**后重启 T3/T4  
- 拆 bootstrap（若仍有必要）  
- 前端 P3/P4、OTel、覆盖率 ≥ 40%  
- Beego 替换为完整 kratos HTTP（仅当出现硬阻塞；默认不做）  

---

## 十八、参考

- [refactor-roadmap.md](../refactor-roadmap.md)  
- [round-3-execution-plan.md](../round-1-4/round-3-execution-plan.md) — 搜索旧底稿、§十七 P1  
- [round-4-execution-plan.md](../round-1-4/round-4-execution-plan.md) — 遗留移交  
- [deployments/scripts/README.md](../../deployments/scripts/README.md) — 与 T15 相关（搬迁后）  
- [go-kratos](https://go-kratos.dev/) · [Vite](https://vitejs.dev/) · [eko/gocache](https://github.com/eko/gocache)  
- S3 API / MinIO 文档（T14）  

### 本轮细化方案索引

| 任务 | 文档 |
|---|---|
| T1 缓存评估 | [round-5-cache-evaluation.md](./round-5-cache-evaluation.md) |
| T2 ORM/分层评估 | [round-5-orm-migration-evaluation.md](./round-5-orm-migration-evaluation.md) |
| T5 MCP P1 | [round-5-t5-mcp-p1.md](./round-5-t5-mcp-p1.md) |
| T6 Vite | [round-5-t6-vite.md](./round-5-t6-vite.md) |
| T7 测试工程化 | [round-5-t7-testing.md](./round-5-t7-testing.md) |
| T8 `*Result`→dto | [round-5-t8-result-dto.md](./round-5-t8-result-dto.md) |
| T9 Repo/Service | [round-5-t9-repo-service.md](./round-5-t9-repo-service.md) |
| T10 Controller 拆分 | [round-5-t10-controller-split.md](./round-5-t10-controller-split.md) |
| T11 安全头 | [round-5-t11-security-headers.md](./round-5-t11-security-headers.md) |
| T12 缓存 B 实施 | [round-5-t12-cache-impl.md](./round-5-t12-cache-impl.md) |
| T14 对象存储 | [round-5-t14-object-storage.md](./round-5-t14-object-storage.md) |
| T15 scripts 落点 | [round-5-scripts-layout.md](./round-5-scripts-layout.md) |
| T16 OAuth2 登录重写 | [round-5-t16-oauth2.md](./round-5-t16-oauth2.md) |
| 环境变量 MINDOC→DOC | [round-5-env-mindoc-to-doc.md](./round-5-env-mindoc-to-doc.md) |

---

## 十九、目录锚点

- [§一 范围](#一范围与不做清单)  
- [§一附 决策修订](#一附2026-08-03-决策修订)  
- [§四 T2 ORM/分层](#四t2--orm--分层评估报告朝-kratos-靠拢--不实施)  
- [§五 T3 搜索冻结](#五t3--搜索最小方案-fulltext--fts5--⏸-暂不实施)  
- [§十三丙 T14 对象存储](#十三丙t14--对象存储上传--旧数据迁移1~2-周)  
- [§十三丁 T15 scripts](#十三丁t15--scripts-是否迁入-deployments0.5~1-天)  
- [§十三戊 T16 OAuth2](#十三戊t16--oauth2-登录重写3~5-天)  
- [§十四 追踪表](#十四追踪表)  
- [§十八 细化方案索引](#本轮细化方案索引)  
