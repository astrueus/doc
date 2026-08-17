# 搜索方案重定义（Search Redesign）

> **状态：** 📝 草案（2026-08-05）· **引擎选型未决**（候选见 §五，待 spike / 评审终裁）  
> **用途：** 取代 Round 3/5 原定的「MySQL FULLTEXT / SQLite FTS5 + 标题加权」路线；作为 [round-5-execution-plan.md §五 T3](./round-5-execution-plan.md#五t3--搜索最小方案-fulltext--fts5--⏸-暂不实施) / [§六 T4](./round-5-execution-plan.md#六t4--倒排--向量检索评估--⏸-冻结) 的**解冻前提文档**。  
> **上游对照：** [upstream-mindoc-checklist.md §1.1 / §1.2](../upstream-mindoc-checklist.md)（#1027 自研倒排 + #1034 reindex）——**不作为终态跟进目标**。  
> **现状：** Web（`SearchController`）与 MCP（`internal/mcp/search_provider.go`）均为 SQL `LIKE`，排序弱、中文差、两套实现易漂移。

---

## 一、背景与问题

| 问题 | 说明 |
| --- | --- |
| 召回质量 | `document_name` / `release` 上 `LIKE %q%`，无分词、无相关度，长库慢 |
| 双实现 | Web 与 MCP 各写一套 SQL，权限与排序语义易不一致 |
| 旧方案不足 | FULLTEXT/FTS5 + 标题加权：中文分词弱、SQLite/MySQL 能力分叉、扩展（同义词/语义）空间小 |
| 上游自研倒排 | gojieba（CGO）+ 应用内 TF-IDF + 倒排表：可跑，但维护/扩展/交叉编译成本高，非市场主流形态 |
| 演进需求 | MCP / AI 检索要求「同一套高质量召回」；远期可能要混合语义检索 |

**已达成共识（与具体引擎无关）：** 搜索应作为**旁路子系统**，经统一 `SearchProvider` 接入；主库只做权威数据与权限真相源，不做全文排序引擎。  
**尚未决策：** 一期具体引擎（Meilisearch / Typesense / OpenSearch / Bleve 等），见 §五分析与 §六待决。

---

## 二、目标与非目标

### 2.1 目标（一期）

1. **统一检索入口**：Web 全局搜、书内搜、MCP `search_*` 全部走同一 `SearchProvider`。  
2. **可替换引擎**：业务只依赖 Provider；候选见 §五（专用引擎 / 内嵌 / 二期混合等）。  
3. **中文可用**：短句、标题优先、常见技术术语（如 `grep` / `C++`）可检索；有可重复的 spike 验收集。  
4. **权限不泄漏**：私有书 / 团队成员关系下，未授权文档不得出现在结果中（含分页与 MCP）。  
5. **可运维**：配置开关、全量 `reindex`、增量同步、索引落后可观测。  
6. **易维护与迭代**：优先选市场成熟、可扩展的方案；避免自研倒排打分成为长期主路径。

### 2.2 非目标（一期不做）

- 不实施 MySQL FULLTEXT / SQLite FTS5 / 标题加权 SQL 排序。  
- 不移植上游 `ContentReverseIndex` + gojieba + TF-IDF 作为终态。  
- 不借搜索强制迁移 PostgreSQL。  
- 不改前端搜索 URL 契约（除非引擎能力需要新增可选参数）。  
- **向量 / 混合检索**：倾向二期（见 §五 G、§十），一期是否预留接口可随引擎终裁一并定。

---

## 三、评估维度（选型共用）

无论选哪家引擎，需同时满足：

| # | 维度 | 说明 |
| --- | --- | --- |
| 1 | 中文分词与相关性 | 标题、正文、标签；短 query / 技术术语 |
| 2 | 权限安全 | 公开书 / 私有书 / 团队；不能「先露出再过滤漏网」 |
| 3 | Web 与 MCP 同源 | 同一套召回与排序，禁止双 SQL |
| 4 | 索引与主库解耦 | 保存时增量同步 + 全量 `reindex` |
| 5 | 运维负担 | 单机 Docker 可跑 vs 独立集群 |
| 6 | 演进空间 | 同义词、facet、向量混合、多租户 |

**最佳实践骨架（引擎无关，建议先定）：**

```text
写：Document/Blog 变更 → Outbox/Hook → Indexer → 搜索引擎
读：Query → SearchProvider.Search(ctx, q, acl) → 引擎拿排序 ID → 主库 hydrate
运维：doc search reindex / 后台补缺索引
```

---

## 四、目标架构（引擎无关）

```text
                    ┌──────────────────────────────────────┐
  Web / MCP / CLI   │         SearchProvider（唯一入口）      │
                    └───────────────┬──────────────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              ▼                     ▼                     ▼
        Query 规范化           引擎 Client            主库 Hydrate
        （裁剪/长度）         （候选见 §五）           （标题/摘要/链）
              │                     │                     │
              │              filter(ACL)+排序              │
              └─────────────────────┴─────────────────────┘
                                    │
                                    ▼
                         有序 Document/Blog DTO

写路径：
  Document/Blog/Book 变更
       → Indexer（同步或 Outbox）
       → 引擎 Upsert / Delete
       →（失败）记 lag / 重试

运维：
  doc search reindex [--book=] [--since=]
  启动可选：补缺索引（落后文档）
```

### 4.1 包落点（建议）

```text
internal/search/
├── provider.go          # SearchProvider 接口与请求/响应 DTO
├── acl.go               # 可见性 filter 构造（member → bookIDs / filter 表达式）
├── indexer.go           # 增量索引：UpsertDocument / Delete / UpsertBlog
├── reindex.go           # 全量重建
├── config.go            # [search] 配置解析
├── engine/
│   ├── engine.go        # Engine 接口（Index/Delete/Search）
│   ├── meilisearch.go   # 候选 A
│   ├── typesense.go     # 候选 A'
│   ├── opensearch.go    # 候选 B
│   └── bleve.go         # 候选 C（无旁路进程）
└── noop.go              # engine=off / like 时回退 LIKE（过渡期）
```

> 具体实现文件随终裁引擎增减。MCP 现有 `searchProvider` / `sqlLikeProvider` 应删除或改为适配本包。

### 4.2 Provider 接口（草案）

```go
type Mode string // keyword | hybrid（二期）

type Query struct {
    Text     string
    MemberID int    // 0 = 匿名
    BookID   int    // 0 = 全局；>0 = 书内
    Types    []string // document | blog | book（可选）
    Page     int
    PageSize int
    Mode     Mode
}

type Hit struct {
    ContentType string  // document | blog | book
    ContentID   int
    BookID      int
    Score       float64
    Title       string
    Snippet     string  // 可选高亮片段
}

type Provider interface {
    Search(ctx context.Context, q Query) (hits []Hit, total int, err error)
}

type Indexer interface {
    UpsertDocument(ctx context.Context, doc *model.Document, book *model.Book) error
    DeleteDocument(ctx context.Context, documentID int) error
    UpsertBlog(ctx context.Context, blog *model.Blog) error
    DeleteBlog(ctx context.Context, blogID int) error
    DeleteByBook(ctx context.Context, bookID int) error
    Reindex(ctx context.Context, opts ReindexOptions) error
}
```

---

## 五、市场方案深度分析

> 以下为选型调研（2026-08），**不代表已终裁**。  
> 明确**不采用**「标题加权 + MySQL FULLTEXT / SQLite FTS5」；也不绑定必须跟上游自研倒排。

### 5.1 方案总览

| # | 方案 | 成熟度 | 运维 | 中文 | 扩展性 | 与「最佳实践 / 易维护」契合 |
| --- | --- | --- | --- | --- | --- | --- |
| A | Meilisearch / Typesense | 高（产品化） | 低 | 需配置/验证 | 高 | ★★★★★ 强候选 |
| B | OpenSearch / Elasticsearch + IK | 极高 | 中高 | 强（IK） | 极高 | ★★★★★ 体量大时 |
| C | Bleve（进程内） | 中高（Go 生态） | 很低 | 弱～中（自接分词） | 中 | ★★★ 强约束单进程时 |
| D | 上游式自研倒排 + gojieba | 低（定制） | 中 | 可控 | 差 | ★★ 不推荐终态 |
| E | PostgreSQL + zhparser / pg_jieba | 中高 | 中（换库） | 中～高 | 中 | ★★★ 仅当主库战略=PG |
| F | 云搜索 SaaS | 高 | 极低 | 看厂商 | 高（易锁定） | ★★★ 私有化默认慎用 |
| G | 关键词 + 向量混合 | 高（趋势） | 中～高 | 看嵌入模型 | 极高 | ★★★★★ 作二期演进 |

---

### 5.2 方案 A：Meilisearch / Typesense（专用轻量搜索引擎）

**是什么：** 独立进程 + HTTP API；倒排 + typo + 可配排序/过滤；常见「应用主库 + 旁路搜索」形态。

**怎么接：** 保存文档时 upsert（字段含 `doc_id, book_id, title, body, type, 可见性/可过滤 ACL`）；查询用 **filter** 表达可见性，再按相关度排序。

#### 优点

- **产品完成度高**：分页、高亮、typo、同义词、可过滤属性、多 index，比自研 TF-IDF 省大量坑。  
- **运维轻**：单 Docker、内存型索引，中小库（约十万级文档）通常轻松。  
- **易迭代**：调 ranking rules / 同义词 / stop words，少改 Go 业务。  
- **与 MCP 契合**：同一 Provider，延迟通常远低于扫 `release LIKE`。  
- **可扩展**：书内搜、标签 facet 等成本低；Provider 稳定后可迁 B。

#### 缺点

- **多一个运行时依赖**（部署、备份、版本升级）。  
- **中文不如 ES+IK 开箱即战**：需 spike 验证分词、词典、术语（`grep` / `C++`）。  
- **权限模型要设计好**：ACL 进 filterable，或「引擎粗筛 + 主库精滤」；私有书占比高时后者可能浪费召回名额。  
- SQLite 单机小部署时，多一个服务有心理负担（可用 Compose 一锅端缓解）。

#### Meilisearch vs Typesense

| | Meilisearch | Typesense |
| --- | --- | --- |
| DX / 文档 | 很友好 | 友好 |
| 过滤 / 多租户 | 强 | 强 |
| 中文社区/文档站实践 | 渐多 | 相对少 |
| API 习惯 | 自有风格 | 更接近 Algolia 思路 |
| 许可 / 商业边界 | 选型时核对版本功能 | 开源核心相对清晰 |

**适用判断：** 要「最佳实践 + 可维护 + 可换引擎」时的**第一梯队候选**；是否采用取决于中文 spike。

---

### 5.3 方案 B：OpenSearch / Elasticsearch + IK 分词

**是什么：** 行业标准全文检索；国内文档/Wiki 大量使用 **ES/OpenSearch + analysis-ik**。

#### 优点

- **中文与相关性最成熟**：IK、同义词、自定义词典、高亮、聚合、复杂 query DSL。  
- **扩展天花板最高**：多字段 boosting、拼音、补全；后续接向量（dense_vector / knn）路径清晰。  
- **生态与人才多**：问题好查、好交接。  
- **适合会长大的知识库**：书多、正文长、检索分析需求多时不易很快撞墙。

#### 缺点

- **运维与资源重**：JVM、磁盘、集群、升级、mapping 变更——对小团队是长期税。  
- **心智负担高**：analyzer、refresh、bulk、别名切换需要规范。  
- **小规模可能杀鸡用牛刀**：文档很少时收益常被运维成本抵消。  
- OpenSearch vs ES：许可与发行版需提前定，避免后期扯皮。

**适用判断：** 明确要做企业知识库 / 复杂检索 / 长期平台 → 强候选；中小团队 Wiki → 可作 A 不达标后的升级路径。

---

### 5.4 方案 C：Bleve（Go 内嵌搜索）

**是什么：** 纯 Go 全文库，索引落本地目录，无独立搜索进程。

#### 优点

- **部署最简单**：跟 `doc` 二进制走，适合单机包 / 强约束「不能加第二进程」。  
- **无网络 hop**：Provider 实现直观。  
- 与「少组件」哲学、SQLite 用户友好。

#### 缺点

- **中文是最大短板**：默认分析器偏英文；接 gse/jieba 等等于自维「分词插件 + 词典路径」。  
- **扩展与运维经验弱于专用引擎**：多实例、水平扩展、词典热更新、观测都要自建。  
- **并发写 / 大索引**需小心（锁、合并、磁盘）。  
- 同义词管理、typo 策略等「搜索产品能力」不如 Meili/ES。

**适用判断：** 仅当「绝对不能多进程」时的务实选项；若接受 Compose 多一个服务，长期通常不如 A/B。

---

### 5.5 方案 D：上游自研倒排 + gojieba + TF-IDF（#1027 / #1034）

#### 优点

- 与 MinDoc 行为可对齐；无新中间件。  
- 中文分词路径清晰（词典随包）。  
- 逻辑全在仓库内，可抠 MCP 场景。

#### 缺点

- **可维护性差**：倒排表膨胀、更新一致性、分页截断、打分调参皆为自研债务。  
- **可扩展性差**：同义词、拼音、向量、多语言、高亮片段每一项都像新项目。  
- **gojieba = CGO**：交叉编译、Windows/Docker 构建痛。  
- **非市场主流形态**：交接成本高，难招「会维护内部搜索引擎」的人。  
- TF-IDF 只是基线；专用引擎的 BM25 + 产品规则通常更省心。

**适用判断：** ❌ **不推荐作为终态**；最多作行为对照或过渡参考。与本文「最佳实践 / 易迭代」目标冲突。

---

### 5.6 方案 E：PostgreSQL + 中文全文（zhparser / pg_jieba）

#### 优点

- 数据与搜索同库，事务一致性好。  
- PG + GIN 生态成熟。  
- 若未来主库战略本就是 PG，可顺路做搜索。

#### 缺点

- **当前栈是 MySQL + SQLite**：为搜索换主库，范围远超搜索。  
- SQLite 用户如何覆盖？双引擎或放弃一端。  
- 分词扩展在托管/精简镜像中安装常很烦。  
- typo、同义词运营、混合检索仍弱于专用引擎。  
- 复杂 ACL 仍易在 SQL 里拧成难维护状态。

**适用判断：** 仅当「主库战略 = PG」时并入；**不要为了搜索单独迁库**。

---

### 5.7 方案 F：云搜 SaaS（Algolia、Elastic Cloud、阿里云开放搜索等）

#### 优点

- 运维接近零；功能与中文（国内云）往往很强。  
- 控制台配同义词等，迭代快。

#### 缺点

- **数据出域 / 专有云 / 费用**；内网文档站经常一票否决。  
- 供应商锁定；本地 SQLite 小部署难对齐。  
- 私有化发行版叙事不友好。

**适用判断：** ⏸ 公有云商业版可另议；**自建 `doc` 默认发行版不建议绑死**。

---

### 5.8 方案 G：混合检索（关键词 + 向量）——倾向二期

**形态：** Meili / OpenSearch / Bleve 做精确与术语；Qdrant / pgvector / ES knn 做语义；RRF 或加权融合。

#### 优点

- 对「换说法也能搜到」、AI/MCP RAG 最友好。  
- 知识库长期演进方向正确。

#### 缺点

- 嵌入模型、费用、索引体积、延迟、评测集——复杂度上一个数量级。  
- **没有稳定关键词底座时先上向量，会两边都调不明白。**

**适用判断：** Provider 可预留 `SearchMode = keyword|hybrid`；**一期先把关键词引擎做对**，二期再加向量。不要与「倒排 vs FULLTEXT」绑成同一评估项。

---

### 5.9 横向对比（决策用）

| 维度 | A Meili/Typesense | B OpenSearch/ES | C Bleve | D 自研倒排 | E PG 全文 | G 混合 |
| --- | --- | --- | --- | --- | --- | --- |
| 最佳实践契合 | ★★★★★ | ★★★★★ | ★★★ | ★★ | ★★★ | ★★★★★（阶段） |
| 易维护 | ★★★★★ | ★★★ | ★★★ | ★★ | ★★★ | ★★ |
| 易迭代（产品能力） | ★★★★★ | ★★★★★ | ★★★ | ★★ | ★★★ | ★★★★ |
| 可扩展 | ★★★★ | ★★★★★ | ★★★ | ★★ | ★★★ | ★★★★★ |
| 中文开箱 | ★★★～★★★★ | ★★★★★ | ★★～★★★ | ★★★★ | ★★★★ | 看模型 |
| 部署成本 | 低 | 高 | 极低 | 中 | 中高 | 高 |
| 与现栈耦合 | 低（旁路） | 低 | 低 | 高（表+CGO） | 高（换库） | 中 |

### 5.10 分析结论（供评审，非终裁）

1. **明确淘汰：** 标题加权 + MySQL FULLTEXT / SQLite FTS5；以「跟上游 #1027」当终态。  
2. **架构层已定：** Provider + 旁路（或内嵌）引擎 + reindex；与具体品牌无关。  
3. **引擎层未定：**  
   - 接受旁路进程 → 在 **A（Meili/Typesense）** 与 **B（OpenSearch）** 间 spike 后二选一（或 A 起步、B 升级）。  
   - 禁止第二进程 → **C Bleve**，并接受能力上限。  
4. **D / E / F：** 不作默认主路径（理由见上）。  
5. **G：** 二期；T4 解冻后改为「在已选关键词引擎上的 hybrid 评估」。  
6. **不要并行开三条实现线**；先书面终裁引擎，再编码。

---

## 六、引擎选型状态与 Spike

### 6.1 当前状态

| 项 | 状态 |
| --- | --- |
| 一期引擎 | ⏳ **未决**（候选 A / A' / B / C） |
| 升级路径是否写死「A→B」 | ⏳ 待决后决定（可作为推荐叙述，非强制） |
| 向量 / hybrid | ⏭ 倾向二期 |
| 自研倒排 / FULLTEXT | ❌ 否决作主路径 |

### 6.2 建议的 Spike 顺序（可改）

```text
1）若接受 Docker 旁路进程：
     先 Meilisearch（或 Typesense）中文 + ACL + reindex
     硬性不达标 → 改测 OpenSearch + IK
2）若禁止第二进程：
     直接 Bleve + 中文分析器 spike
3）出一份一页纸终裁：引擎名、部署形态、不做清单
```

### 6.3 Spike 验收集（对入选引擎都必须过）

用真实中文语料（建议 ≥ 500 篇文档，含术语与私有书）：

1. **中文短句 / 中英混排 / 技术词**（`linux`、`grep`、带符号词）召回可接受。  
2. **权限**：无权限书的文档 ID **永不**出现在最终结果（引擎 filter 或强制二次校验）。  
3. **一致性**：改标题后在约定 SLA 内可搜到；删除后不可搜到。  
4. **运维**：空库全量 reindex 时间、内存/磁盘、备份方式可文档化。

---

## 七、索引模型与权限

### 7.1 建议索引字段

| 字段 | 类型 | 用途 |
| --- | --- | --- |
| `id` | string | 主键，如 `doc:{document_id}` / `blog:{blog_id}` |
| `content_type` | string | `document` \| `blog` \| `book`（book 可选一期） |
| `content_id` | int | 主库 ID |
| `book_id` | int | 书内搜 / 过滤 |
| `title` | string | 高权重搜索 |
| `body` | string | 去 HTML 后的正文（`release` / markdown 择一权威） |
| `labels` | string[] | 可选 |
| `privately_owned` | bool/int | 书是否私有 |
| `is_public` | bool | 匿名是否可见（冗余便于 filter） |
| `member_ids` | int[] | 可选：显式授权成员（关系表展开，有上限要防爆） |
| `team_ids` | int[] | 可选：团队可见 |
| `updated_at` | int/unix | 排序辅助、增量 |

> **正文入库前**统一 `StripTags` / 控制最大长度（超长截断策略需定：按字符截断并记 metric）。

### 7.2 ACL 策略（二选一，推荐组合）

| 策略 | 做法 | 优点 | 风险 |
| --- | --- | --- | --- |
| **P1 引擎内 filter（主）** | 查询时带 `is_public = true OR book_id IN :readableBooks` | 分页准确、少回表浪费 | 需维护成员可读 `book_id` 列表 |
| **P2 主库精滤（辅）** | 引擎多取 → 主库校验可见性 → 不足再补页 | 防引擎字段过期 | 私有书多时可能多次往返 |

**推荐：** P1 为主（登录用户先解析可读 `book_id` 集合再 filter）；P2 作安全网（抽样或严格模式可开）。  
**禁止：** 先返回未授权 hit 再靠前端隐藏。

### 7.3 与现网权限对齐

现网逻辑（公开书 / 私有书 + relationship / team）须在 `acl.go` **单点实现**，供 Web 与 MCP 共用；禁止在 Controller 与 MCP 各写一份 bookID 列表。

---

## 八、读写路径与一致性

### 8.1 增量索引触发点

| 事件 | 动作 |
| --- | --- |
| 文档保存 / 发布 | UpsertDocument（标题 + 发布正文） |
| 文档删除 | DeleteDocument |
| Blog 保存 / 删除 | UpsertBlog / DeleteBlog |
| 书删除 | DeleteByBook |
| 书权限变更（公开↔私有、成员变更） | 更新该书下文档的 ACL 字段，或按 book 批量 reindex ACL |

> 权限变更若只改正文索引、不刷 ACL 字段，会导致**越权或漏搜**——必须纳入设计。

### 8.2 同步 vs 异步

| 模式 | 适用 | 说明 |
| --- | --- | --- |
| 同步 Upsert | 一期可默认 | 实现简单；引擎短暂不可用时写库成功但搜索落后，需记错误日志 |
| Outbox / 队列 | 增强项 | 表 `search_outbox` 或借现有任务机制，提升韧性 |

一期可同步 + 「失败入补缺队列」；SLA 目标：**准实时（秒级）**，不保证跨机房强一致。

### 8.3 全量 reindex

```text
doc search reindex
doc search reindex --book=12
doc search reindex --type=document
```

- 建议：**重建临时 index → 别名切换**（Meili/OS 类引擎），避免清空窗口内搜索空洞。  
- 若引擎不支持别名：文档化「维护窗口」与重试。  
- 与上游 `mindoc reindex` 语义对齐（运维习惯），实现不跟其倒排表。

### 8.4 配置（草案，engine 值待终裁）

```ini
[search]
# off | like | meilisearch | typesense | opensearch | bleve
engine = off
# 引擎不可用或未配置时是否回退 LIKE（建议生产 false，避免静默降质）
fallback_like = false
# 以下按所选引擎启用
meili_host = http://127.0.0.1:7700
meili_api_key =
meili_index = doc
# opensearch_url = https://127.0.0.1:9200
# bleve_path = data/search.bleve
max_body_chars = 100000
```

环境变量前缀与全局一致：`DOC_SEARCH_*`（硬切，不兼容旧名）。

---

## 九、与 MCP / Web 的一致性

| 项 | 要求 |
| --- | --- |
| 召回 | MCP 与 Web 同一 `Provider.Search` |
| 权限 | 同一 `acl`；MCP Token 对应用户 = MemberID |
| 排序 | 均以引擎相关度为主；禁止 MCP 再用 `ORDER BY modify_time` 当主排序 |
| 字段 | MCP 工具仍返回文档元数据；snippet 可选增强 |
| 开关 | `engine=off/like` 时行为写进 MCP 文档，避免 AI 侧误判「语义搜索」 |

相关文档待引擎落地后修订：[mcp-integration.md](../mcp-integration.md)。

---

## 十、二期：混合检索（T4 方向）

在一期关键词引擎稳定后评估（对应 §五 G）：

```text
keyword（已选引擎） + vector（Qdrant / OS knn / pgvector 等）
→ RRF 或加权融合 → 同一 Provider，Mode=hybrid
```

| 触发条件（示例） | 动作 |
| --- | --- |
| 召回投诉「说法不同搜不到」占比高 | 开 hybrid 评估 |
| MCP RAG 场景成为主路径 | 优先向量侧评测集 |
| 关键词引擎运维尚未稳定 | **禁止**先上向量 |

T4 解冻后应改为 **「在已选关键词引擎上的 hybrid 评估」**，不再评估「自研倒排 vs FULLTEXT」。

---

## 十一、迁移与发布节奏

```text
阶段 0  本文评审 + 候选引擎 spike + 一页纸终裁
阶段 1  internal/search Provider + Indexer + reindex CLI；engine 可切
阶段 2  写路径挂钩；双写（引擎 + LIKE 开关）阴影对比
阶段 3  Web / MCP 切引擎；生产建议 fallback_like=false
阶段 4  删除业务路径对搜索 SQL 的依赖；LIKE 仅留 noop/测试
阶段 5（可选）hybrid
```

**阴影对比（阶段 2）：** 同一 query 记录引擎 topN vs LIKE topN 的重叠率，作切流依据，不作为永久双读。

---

## 十二、验收清单（解冻 / 一期完成）

### 12.1 文档与立项（解冻 T3）

- [x] 本文草案落地（含市场方案分析）  
- [ ] 评审通过：**引擎终裁**写入本文 §六（改「未决」为选定项）  
- [ ] Spike 报告附中文样例与权限用例结果  
- [ ] [round-5-execution-plan.md](./round-5-execution-plan.md) 将 T3/T4 改回可排期或指定后续 Round  

### 12.2 一期功能验收

- [ ] `SearchProvider` 单入口；MCP 无独立 SQL 搜索实现  
- [ ] 增量索引 + `doc search reindex`  
- [ ] 匿名 / 登录 / 私有书 / 团队 权限用例通过  
- [ ] 配置 `[search]` + `DOC_SEARCH_*`  
- [ ] 引擎宕机行为符合配置（报错或显式 fallback，不静默错结果）  
- [ ] 基础指标：索引滞后、search QPS/延迟、reindex 耗时（可先日志后 Prometheus）

### 12.3 明确不做（回归时检查）

- [ ] 未引入 FULLTEXT/FTS5 业务路径  
- [ ] 未引入 gojieba / `t_content_reverse_index` 作为主引擎  

---

## 十三、工期粗估（终裁后按引擎调整）

| 阶段 | 内容 | 粗估 |
| --- | --- | --- |
| Spike | 入选候选的中文 + ACL + 样例集 | 2～3 天/引擎 |
| Provider + 引擎 Client + reindex | 核心链路 | 4～6 天（轻量引擎） / 6～10 天（OpenSearch） |
| 写路径挂钩 + Web/MCP 切换 | 含回归 | 2～3 天 |
| 阴影对比与切流 | 可选但推荐 | 1～2 天 |
| Bleve 路径 | 若选 C，省容器运维、多分词打磨 | 合计视中文插件而定 |
| Hybrid（二期） | 另立项 | 不计入一期 |

---

## 十四、风险与待决

| # | 项 | 说明 | 状态 |
| --- | --- | --- | --- |
| 1 | **一期引擎终裁** | A Meili / A' Typesense / B OpenSearch / C Bleve | ⏳ 未决 |
| 2 | 发行版是否允许旁路进程 | 决定能否优先 A/B | ⏳ 未决 |
| 3 | `member_ids` 膨胀 | 大书成员很多时 filter 字段过大 | 建议优先 `book_id IN readable` |
| 4 | 正文体积 | 大文档截断丢尾部命中 | 定 `max_body_chars` + 监控截断率 |
| 5 | 回退自研倒排/FULLTEXT 的压力 | 与目标冲突 | ❌ 坚持否决主路径 |
| 6 | 与缓存 T12 | 搜索结果是否缓存 | 一期可不缓存 hit；ACL 失效复杂 |
| 7 | OpenSearch vs ES 发行版 | 若选 B 需先定 | 随 B 决策 |

---

## 十五、相关文档

| 文档 | 关系 |
| --- | --- |
| [round-5-execution-plan.md](./round-5-execution-plan.md) | T3/T4 冻结与解冻条件 |
| [upstream-mindoc-checklist.md](../upstream-mindoc-checklist.md) | 上游 #1027/#1034 对照；搜索终态转向本文 |
| [mcp-integration.md](../mcp-integration.md) | MCP 工具与配置；引擎落地后需同步 |
| Round 3 执行计划 §六 | FULLTEXT 历史底稿，**勿按之开工** |

---

## 十六、修订记录

| 日期 | 说明 |
| --- | --- |
| 2026-08-05 | 初稿：否决 FULLTEXT/自研倒排；定 Provider + 旁路架构 |
| 2026-08-05 | 并入市场方案深度分析（A～G）；标明引擎选型未决，弱化「默认 Meili」表述 |
