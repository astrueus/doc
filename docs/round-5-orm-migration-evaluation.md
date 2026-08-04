# Round 5 · T2 · ORM / 分层评估报告（朝 kratos 靠拢 · 不实施）

> 对应 [round-5-execution-plan.md §四 T2](./round-5-execution-plan.md#四t2--orm--分层评估报告朝-kratos-靠拢--不实施)。  
> **决策方向（[§一附 5](./round-5-execution-plan.md#一附2026-08-03-决策修订)）：** 朝 [go-kratos](https://go-kratos.dev/) 的分层习惯靠拢；model 期望**自动化生成**；把「模型 / 业务逻辑 / 数据访问」拆开。  
> **定位：** 本轮**只评估不实施**；产出结论进决策日志；实施另立项（Round 6+）。  
> **状态：** ⏳ 待写结论。

---

## 一、现状债

### 1.1 目录（分层混乱）

- [`internal/model/`](../internal/model/) · 35 文件，**同时承担**：ORM 结构体、`Find*` / `To*Pager` 查询、少量业务动作（如 `RecursiveDocument`、`ReleaseContent`）、`*Result` 展示对象
- [`internal/repository/`](../internal/repository/) · 5 文件，仅 `Document / Book / Member` 三个域做了薄封装（[`document_repo.go`](../internal/repository/document_repo.go) 等）
- [`internal/controller/`](../internal/controller/) · 17 文件，**Controller 直接 `model.NewDocument().Find(id)`**，绕过 Repo
- 无 `service` / `biz` 层；业务规则散落在 Controller 与 model

### 1.2 依赖分布

- **每个 model 文件都直接 import `github.com/beego/beego/v2/client/orm`**（30 处，见 `Grep orm.NewOrm|beego/orm`）
- Raw SQL 集中在 [`DocumentSearchResult.go`](../internal/model/DocumentSearchResult.go)（`o.Raw(sql2, ...).QueryRows(...)`）
- 表前缀依赖 `config.GetDatabasePrefix()`（不能直接下沉到生成器）

### 1.3 `*Result` 分布

| 文件 | 类型 |
|---|---|
| [`BookResult.go`](../internal/model/BookResult.go) | `BookResult` |
| [`BlogResult.go`](../internal/model/BlogResult.go) | `BlogResult` |
| [`MemberResult.go`](../internal/model/MemberResult.go) | `MemberRelationshipResult` / `SelectMemberResult` |
| [`DocumentSearchResult.go`](../internal/model/DocumentSearchResult.go) | `DocumentSearchResult` |
| [`ConvertBookResult.go`](../internal/model/ConvertBookResult.go) | `ConvertBookResult` |
| [`AttachmentResult.go`](../internal/model/AttachmentResult.go) | `AttachmentResult` |
| [`DocumentHistory.go`](../internal/model/DocumentHistory.go) | `DocumentHistorySimpleResult` |
| [`comment_result.go`](../internal/model/comment_result.go) | `CommentResult` |

这些是纯展示 / 视图 DTO，**含查询逻辑的**（如 `BookResult.FindByIdentify`）需先抽 Repo 才能迁 dto（配合 T8）。

### 1.4 文件命名不统一

[`internal/model/`](../internal/model/) 内文件名混用 **PascalCase** 与 **snake_case**，且同一语义后缀也不一致（有的带 `Model`，有的不带）：

| 风格 | 示例 |
|---|---|
| PascalCase（多数，偏 Java/C# 习惯） | `DocumentModel.go`、`BookResult.go`、`Member.go`、`AttachmentModel.go`、`Migrations.go` |
| snake_case（少数，偏 Go 习惯） | `book_model.go`、`book_query.go`、`book_import.go`、`comment.go`、`comment_result.go`、`comment_vote.go` |

**建议（Go 惯例）：**

1. **文件名**一律 `lower_snake_case.go`（见 [Effective Go](https://go.dev/doc/effective_go#package-names) / 社区惯例：包内文件小写 + 下划线，不用导出式 PascalCase 文件名）。  
2. **类型名**仍用导出 PascalCase（`Document`、`BookResult`），与文件名无关。  
3. **去冗余后缀**：实体文件优先 `document.go` 而非 `document_model.go`（包名已是 `model`）；查询/导入等横切文件可用 `book_query.go`、`book_import.go`。  
4. **Result / 展示型**：统一 `book_result.go`（与 T8 迁 `dto/` 后的 `book.go` 不冲突——包不同）。

#### 目标命名示意（节选）

| 现状 | 建议 |
|---|---|
| `DocumentModel.go` | `document.go` |
| `AttachmentModel.go` | `attachment.go` |
| `LabelModel.go` | `label.go` |
| `BookResult.go` | `book_result.go`（或 T8 后迁出） |
| `DocumentSearchResult.go` | `document_search_result.go` |
| `MemberApiToken.go` | `member_api_token.go` |
| `DocumentHistory.go` | `document_history.go` |
| `book_model.go` | `book.go`（与上条「去 Model 后缀」对齐） |
| 已是 snake_case 的 `comment_*.go` / `book_query.go` 等 | 保持，仅统一同域前缀 |

#### 实施建议

- **纯 `git mv` + 确认无 `//go:embed` / 脚本写死路径**；不改类型名则编译引用面为 0。  
- 可与 **T8**（Result 迁 dto）同批：先迁再按新包规范命名，避免搬两次。  
- 若本轮不做 ORM 切换（推荐 A），**文件重命名仍建议做**：低风险、改善可读性，且让后续 `internal/data/` 生成物从一开始就跟 snake_case 一致。  
- **本报告范围：** 把「model 文件命名统一」列为评估结论中的**明确建议项**；是否在 Round 5 单开小 PR 实施，写入决策日志即可（不阻塞 T2「只评估不换 ORM」）。

---

## 二、评估目标（本轮共识）

1. **架构方向：** 向 kratos 分层习惯靠拢（不必本轮引入完整 kratos 运行时）
2. **模型可生成：** 少手写 CRUD 胶水
3. **拆分：** `internal/model` → 生成实体 / 手写领域逻辑 / Repo 三层
4. **与 Beego 关系：** Web 层是否留 Beego，仅数据/业务层 kratos 化？

### 2.1 目标分层示意

```
internal/
├── data/            # ← 现 internal/repository/ 扩面 + ORM/生成代码
│   ├── ent/         # 或 model 生成物
│   ├── document_repo.go
│   ├── book_repo.go
│   └── member_repo.go
├── biz/             # ← 业务逻辑（现散在 controller/model 里的规则）
│   ├── document.go
│   ├── book.go
│   └── member.go
├── service/         # ← 面向 Web/MCP 的用例层（薄）
│   ├── document_web.go
│   └── document_mcp.go
├── dto/             # ← 已存在（mcpdto）；T8 后再加 web dto
└── model/           # ← 逐步降级为「共用领域实体」，Controller 不再直调
```

### 2.2 与 Beego 关系

| 折中层次 | 说明 | 破坏面 |
|---|---|---|
| ① **仅数据层 kratos 化** | ORM 换 `ent`/`sqlc`，Repo 走 kratos 的 `data` 布局；Web 仍 Beego | 小；模板绑定不变 |
| ② **业务层 kratos 化** | 在 ① 之上加 `biz` / 薄 `service`；Controller / MCP 只做编解码与鉴权入口 | 中；Session / i18n / 权限 middleware 仍可留 Beego |
| ③ **全量 kratos** | HTTP + middleware + wire 替换 Beego | 大；模板系统需替换或桥接；**不作为默认目标** |

**推荐方向（已拍板意图）：先做 ①，且必须带着 ② 的演进规划一起设计；③ 长期不追。**

要点：

- ① **可以先落地**（单域试点、破坏面可控）。  
- ② **不是「做完 ① 再看要不要」**，而是 Round 6+ 路线图里的**既定下一阶段**；① 的接口与目录必须为 ② 预留，避免 data 层泄漏业务规则、或 Repo 被 Controller 直接当作用例层用死。  
- ③ 仅当 Beego 出现硬阻塞时再单独立项。

---

## 三、候选对比

### 3.1 ORM / 生成器

| 候选 | 生成方式 | 优点 | 缺点 | 与现状匹配度 |
|---|---|---|---|---|
| **① 维持 `beego/orm` + 扩 Repo** | 无生成 | 零破坏；渐进 | 手写 CRUD 胶水多；无 kratos 味道 | ⭐⭐⭐ |
| **② [`ent`](https://entgo.io/)** | schema-first（Go DSL） | 关系强类型、图查询、Hook、支持 SQLite/MySQL | 学习曲线；主键/时间列约定与现表可能冲突；需迁移策略 | ⭐⭐⭐⭐ |
| **③ [`gorm gen`](https://gorm.io/gen)** | DB-first（反向生成） | 从现表反向生成，落地快；社区大 | 依赖 gorm 生态；与 `beego/orm` 并存有心智负担 | ⭐⭐⭐ |
| **④ [`sqlc`](https://sqlc.dev/)** | SQL-first | SQL 明确、类型安全；性能可控；无 ORM 神秘 | 需手写 SQL；关系导航不方便；SQLite/MySQL 双方言维护 | ⭐⭐⭐⭐ |
| **⑤ protobuf + 生成** | proto + wire | 与 kratos 原生对齐 | 侵入面最大；本项目非 gRPC；ROI 低 | ⭐ |

> **本轮拍板倾向：** 实施期（Round 6+ 阶段①）大概率在 **② ent** 与 **③ gorm gen** 二选一；① 仅作 Round 5 过渡，④/⑤ 不作为主候选（除非二选一出现硬阻塞再回看 sqlc）。

#### 3.1.1 ent vs gorm gen · 对照（决策用）

| 维度 | **ent** | **gorm gen** | 对本项目含义 |
|---|---|---|---|
| **范式** | Schema-first：用 Go DSL 写 schema → 生成 client/query | DB-first：连现库 / 读表结构 → 生成 model + 类型安全 DAO（`DO`） | 我们已有多年 beego 表 + `Migrations.go`；**gen 上手更快**，ent 要先「把现表翻译成 schema」或 `entimport` |
| **与现表契合** | 可用 [`entimport`](https://github.com/ariga/entimport) / atlas 从 DB 导入，再人工修关系与边；字段类型、默认值、联合索引常需手调 | 直接扫 MySQL/SQLite 表生成，字段名/tag 贴近现状；自定义 `TableName`、序列化可后补 | **存量迁移：gen 摩擦更小**；长期演进：ent schema 更像「单一真相」 |
| **查询体验** | 强类型谓词、边（Edge）预加载、图遍历清晰；复杂关联（Book↔Document↔Member）表达力强 | 生成 `Where`/`Join` 的类型安全 API，底层仍是 GORM 习惯；关联靠 GORM association / 手写 join | Document 树、权限关系多：两者都能做；**深关系/组合查询 ent 更顺手** |
| **代码生成物** | `ent/` 下 client、各实体、mutation、hook；生成量大，但边界清晰 | `model` + `query`（DO）两套；风格接近「增强版 GORM」 | 都适合放进 `internal/data/`；都不要让 Controller 直依赖生成包 |
| **Hook / 拦截** | 一等公民（Op、隐私层、拦截器） | GORM callbacks / gen 插件；能力够用但分散 | 审计字段、软删：两者都行；**业务规则仍禁止进 Hook**（见 §6.2） |
| **迁移 / schema 演进** | 官方推 [Atlas](https://atlasgo.io/) / `ent migrate`；与「显式迁移文件」结合好 | 多靠 GORM AutoMigrate 或外部 goose/migrate；gen 本身不负责版本迁移 | 项目已有 `Migrations.go`：**无论选谁，都建议迁到 goose/atlas 显式 SQL**，不要靠 AutoMigrate 上生产 |
| **表前缀 `md_`** | `sql.WithTablePrefix` 或 schema 注解 | GORM `NamingStrategy.TablePrefix` + gen 配置 | 两者都可；需在 `data` 初始化统一注入 `config.GetDatabasePrefix()` |
| **SQLite + MySQL** | 官方双驱；部分方言差异要测（JSON、布尔、自增） | GORM 双驱成熟；SQLite 细节仍要回归 | **必须**在 CI 对两适配器跑 Document 域单测 |
| **事务** | `client.Tx` / 回调式 | GORM `Transaction`；与现 Repo `tx.go` 易对齐 | 乐观锁更新（MCP）两者都能包一层 |
| **学习 / 招聘** | 概念多（schema/edge/mutation），曲线陡 | 国内资料多，会 GORM 即可读 gen 代码 | 团队若已熟 GORM → gen；想要更强领域建模 → ent |
| **依赖体积 / 生态** | ent 独立栈，不绑 GORM | 引入 `gorm.io/gorm` + driver；与残留 `beego/orm` **双 ORM 期**更明显 | 双栈期都痛苦；gen 的「心智」更接近传统 ORM，ent 是另一套 |
| **与 kratos `data` 示例** | 社区有 ent 示例，但 kratos 官方示例偏 gorm 更多 | 与许多 kratos 教程（gorm）同路，复制成本低 | **贴 kratos 样板：gen/gorm 略占优**；贴「干净领域模型」：ent 略占优 |
| **乐观锁 / 版本列** | 可用 schema 字段 + 手写 mutation 条件或拦截 | GORM 原生支持 version 标签 / 条件更新 | 现有 `UpdateMarkdownWithVersion`：两者都在 Repo 包一层即可 |
| **Raw SQL 逃生舱** | `client.QueryContext` / sql 驱动 | `db.Raw` / `Clauses` | 搜索 LIKE（T3 冻结期）继续 raw 没问题 |

#### 3.1.2 场景化建议（仍待 Round 6 开工前终裁）

| 若更在意… | 更倾向 |
|---|---|
| **尽快**从现表切入 Document 域、少写 schema 翻译 | **gorm gen** |
| 长期 **schema 即文档**、边关系清晰、少「GORM 魔法」 | **ent** |
| 与网上 **kratos + gorm** 教程对齐、降低新人成本 | **gorm gen** |
| 想把「生成实体」与「手写 biz」分得更开、避免 data 层长成 ActiveRecord | **ent**（强制经 client，不易在 model 上堆方法） |
| 双 ORM（beego 残留 + 新栈）过渡期可接受度 | 两者都要接受过渡；**gen 看起来更像旧 ORM**，迁移心理成本略低 |

**默认建议（可被数据打脸）：** Round 6 Document 试点优先评估 **gorm gen**（存量 DB-first、kratos 资料近）；若试点中发现关联查询/生成 API 别扭，或团队更想 schema-first，再 **换 ent 成本可控**——前提是 §6.2：`biz` 只依赖 Repo **接口**，不依赖具体生成包。

**明确不绑死：** 终裁前用 Document 三张表做 0.5~1 天 spike（同一组用例：按 identify 查、乐观锁更新、按 book 列目录），对比生成代码可读性与测试通过率后再写入决策日志。

### 3.2 SQL 方言与 SQLite 兼容

- 表结构现由 [`beego/orm` 自动建表](../internal/model/DocumentModel.go) + [`Migrations.go`](../internal/model/Migrations.go) 维护
- 无论 **ent** 或 **gorm gen**，生产环境都建议落到显式迁移（[`golang-migrate/migrate`](https://github.com/golang-migrate/migrate) / [`pressly/goose`](https://github.com/pressly/goose) / Atlas），避免 AutoMigrate 与历史 `Migrations.go` 双轨
- SQLite / MySQL：**spike 与 CI 双跑** Document 域关键路径

### 3.3 表前缀

- 当前 `config.GetDatabasePrefix()` 影响所有 `TableNameWithPrefix()`
- **ent**：`sql.WithTablePrefix` 或等价 middleware  
- **gorm gen / GORM**：`NamingStrategy{TablePrefix: ...}` 在打开 DB 时注入  
- **sqlc**（若回退）：query 内替换，稍麻烦  
- **不影响二选一**，但是阶段① 开工检查清单必选项

---

## 四、工作量粗估（按域）

假设阶段①采用 **ent 或 gorm gen**（二选一）+ Repo 扩面 + 保留 Beego Web：

| 域 | 表数 | 现有 Repo | 迁移工作量（data） |
|---|---|---|---|
| Document | 3（document、history、attachment） | ✅ 部分 | 3~5 天（含生成器 spike 摊销） |
| Book | 3（book、relationship、items） | ✅ 部分 | 3~5 天 |
| Member | 4（member、token、team、team_member） | ✅ 部分 | 3~5 天 |
| Blog / Comment | 3 | ❌ | 2~4 天 |
| Options / Logs / Migrations | 3（元数据/内部） | ❌ | 1~2 天 |

**说明：** gorm gen 往往在「首域接现表」上比 ent **少 0.5~1 天** schema 翻译；ent 可能在「复杂关联整理」上更省后续返工。表内按中位估算，不按品牌拆两列。

**单域全迁工作量约 3~5 天 × 5 域 ≈ 3~4 周**，不含并行的业务修复。**建议按域分 sprint**（Round 6 起，先 Document）。  
阶段②（biz）按域另估约 **2~4 天/域**（规则搬迁 + Web/MCP 改调 service），不计入上表「仅 data」人天。

---

## 五、破坏面清单（迁移期必须兼顾）

| 项 | 现状 | 迁移影响 |
|---|---|---|
| Beego 模板 | 直接吃 `*model.Document` 字段 | 生成实体字段名/tag 变化 → 模板要么保持字段名一致，要么中间加 view struct |
| MCP DTO | 已用 [`internal/dto/mcpdto`](../internal/dto/mcpdto/) | 相对独立，只改 Repo 侧调用 |
| Session | Beego session 表 | 独立表，不动 |
| `GetDatabasePrefix()` | 全 model 依赖 | 需在生成器 middleware 中兼容 |
| Cache key | `Document.Id.<id>` 等硬编码 | 尽量不变；改在 Repo 内部 |
| Raw SQL 搜索 | [`DocumentSearchResult.go`](../internal/model/DocumentSearchResult.go) | 与搜索方案（T3 冻结）绑定；生成器不覆盖 raw 查询 |

---

## 六、推荐

### 6.1 本轮结论（建议）

**A：维持 `beego/orm` + 渐进扩 Repo（Round 5 内的默认动作）**

- T9 里已经在做（`internal/repository/`）
- 命名与目录布局**朝未来 `internal/data/` / `internal/biz/` 靠拢**（避免下轮改名）
- Round 5 **不实施** ①/②；只把分阶段规划与接口约束写清（本文）
- **附带建议：** [`internal/model/`](../internal/model/) 文件名统一为 Go 惯例 `lower_snake_case`（见 §1.4）；可单独小 PR，或与 T8 合并，**不**算 ORM 切换

### 6.2 Round 6+ 分阶段规划（既定路径，非可选项）

总目标：**① 数据层 → ② 业务层**；按域推进，首域 **Document**。Web 继续 Beego，直至有理由才碰 ③。

```text
Round 6     阶段① 数据层 kratos 化（Document 试点）
            └─ 产出可复用的 data 边界，为 biz 留接口

Round 6~7   阶段② 业务层 kratos 化（仍先 Document）
            └─ 规则进 biz；Web/MCP 经 service 调用

Round 7+    按域复制 ①→②（Book → Member → …）
            └─ ③ 全量 kratos HTTP：默认不做
```

#### 阶段① · 数据层（可先做）

| 项 | 内容 |
|---|---|
| 做什么 | 引入 **ent 或 gorm gen**（§3.1.1 二选一，开工前 spike 终裁）；生成 `internal/data/...`；`DocumentRepo`（及同域表）只做持久化 |
| 谁调用 | 过渡期：现有 Controller / MCP / 薄 Repo 门面仍可调 `data`；**禁止**在 `data` 内写发布/权限/乐观锁编排等业务规则 |
| 预留 | Repo 接口放在可被 `biz` 依赖的位置（如 `internal/biz` 定义 iface、`data` 实现；或先 `repository` 接口、下阶段挪到 biz 依赖倒置） |
| 验收 | Document 读写经 `data`；行为与现网一致；文件 snake_case；有「业务规则仍在外」的代码审查清单 |

#### 阶段② · 业务层（① 之后的既定规划，须在 ① 设计时就对齐）

| 项 | 内容 |
|---|---|
| 做什么 | `internal/biz/document.go`：发布、乐观锁冲突处理、删除级联、历史快照等；`internal/service/document_{web,mcp}.go`：用例编排 |
| 谁调用 | Controller / MCP **只**调 `service`（或直接调 `biz` 若用例极薄）；**不再**直调 `data` / `model` |
| 从 ① 迁出 | 若过渡期业务规则仍散落在 Controller/MCP，② 开工时集中搬入 `biz`；① 留下的「临时编排」列进迁出清单 |
| 验收 | Document 域 Web + MCP 主路径经 biz/service；权限校验边界清晰（可留在 service 入口或 middleware）；单测可绕过 HTTP 测 biz |

#### ①→② 衔接约束（阶段① 就遵守，避免返工）

1. **`data` 不依赖** `controller` / `mcp` / Beego context；只依赖标准库 + ORM/驱动 + 本域实体。  
2. **业务规则不进生成代码 / data 实现**（Hook 仅限审计字段、软删等持久化策略）。  
3. **T9 的 `internal/service/`（若本轮已建）** 命名与职责按「未来 usecase」来，不要变成第二套 Repo。  
4. **按域切完 ① 再开同域 ②**，不要全站 data 迁完再一次性灌 biz（风险过大）；允许「Document ①→② 闭环后再开 Book ①」。  
5. **接口稳定优先于换 ORM 品牌**：阶段① 在 ent ↔ gorm gen 间调整时，只要 Repo 接口不变，② 不受损（故 spike 成本可承受）。

### 6.3 Round 6+ 远期

**C：全量 kratos HTTP（层次 ③）** — 仅当 Beego 遇到硬阻塞（安全/性能/生态断代）时才考虑；本报告**不推荐**主动做，也**不**阻挡 ①→②。

---

## 七、验收

- [ ] 报告合入 `docs/`
- [ ] [决策日志](./round-5-execution-plan.md#十四追踪表) / roadmap §八写入拍板结论：本轮 A；Round 6+ 走 **①→②** 分阶段（非「① 后再议要不要 ②」）
- [ ] **本轮无** ORM / 框架切换实施 PR
- [ ] T9 命名与目录**已按未来 `internal/data/` / `internal/biz/` 方向对齐**
- [ ] §1.4 文件命名建议已记入决策：Round 5 是否单开 `git mv` PR（建议做）
- [ ] §6.2 衔接约束已同步到 [T9 细化方案](./round-5-t9-repo-service.md)（避免 service 变成 Repo 克隆）

---

## 八、参考

- [Effective Go — 命名](https://go.dev/doc/effective_go#names) · [Code Review Comments — package names](https://go.dev/wiki/CodeReviewComments#package-names)  
- [go-kratos 分层说明](https://go-kratos.dev/docs/getting-started/layout/)
- [`entgo.io`](https://entgo.io/) · [`sqlc`](https://sqlc.dev/) · [`gorm gen`](https://gorm.io/gen)
- [`internal/model/`](../internal/model/) · [`internal/repository/`](../internal/repository/)
- [round-5-execution-plan.md §四 T2](./round-5-execution-plan.md#四t2--orm--分层评估报告朝-kratos-靠拢--不实施)
- [round-5-t8-result-dto.md](./round-5-t8-result-dto.md) — 与 Result 文件重命名可合并
- [round-5-t9-repo-service.md](./round-5-t9-repo-service.md) — Repo 扩面须服从 ①→② 衔接约束
