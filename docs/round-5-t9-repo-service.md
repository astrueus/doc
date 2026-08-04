# Round 5 · T9 · Repository 扩面 + 可选 Service — 细化方案

> 对应 [round-5-execution-plan.md §十一 T9](./round-5-execution-plan.md#十一t9--repository-扩面--可选-service3~5-天)。  
> 方向对齐 [T2](./round-5-orm-migration-evaluation.md)：朝未来 `internal/data` / `internal/biz` 靠拢，**本轮不换 ORM**。  
> **状态：** ⏳ 待实施。

---

## 一、现状

| 项 | 状态 |
|---|---|
| [`internal/repository/document_repo.go`](../internal/repository/document_repo.go) | CRUD + 乐观锁较完整 |
| `book_repo.go` | 仅薄封装（如 `Find`） |
| `member_repo.go` | 有 |
| MCP 写路径 | 多经 `documentRepo()` |
| MCP 读 / authz / 搜索 | **仍直调** `model.*` |
| `internal/service/` | **不存在** |

---

## 二、本轮目标（可切片）

### 2.1 必做：MCP 读路径走 Repo

| 工具 / 模块 | 现状 | 目标 |
|---|---|---|
| `get_document` | model | `DocumentRepo` |
| `list_document_tree` | model | `DocumentRepo` |
| `search_document` | model raw | `DocumentRepo`（或 `SearchRepo` 薄封装，SQL 暂不动） |
| `list_books` | model | `BookRepo` 扩面 |
| `authz.go` 读 Book/Member | model | `BookRepo` / `MemberRepo` |

原则：Repo 方法语义对齐现有 model 方法，**先搬后清**，行为比特级一致。

### 2.2 建议：高频 Web 读试点（选 1）

优先其一，避免铺开：

- **A.** 文档阅读页（`DocumentController.Read` / `Content`）→ `DocumentRepo`  
- **B.** 图书首页 / 目录 → `BookRepo` + `DocumentRepo`

推荐 **A**（与 MCP 同一域，复用方法）。

### 2.3 可选：`internal/service/`（薄用例层）

仅当「Controller / MCP 出现重复编排」时引入：

```text
internal/service/
└── document.go    # 例：GetReadableDocument(ctx, member, bookIdent, docIdent)
```

命名约定（对齐 T2，避免下轮改名）：

| 本轮 | Round 6+ 可能演进 |
|---|---|
| `internal/repository` | → `internal/data`（可保留 repository 文件名作过渡） |
| `internal/service` | → 拆为 `biz`（领域）+ 薄 `service`（用例），或直接当 usecase |

**本轮不要**创建 `internal/biz/`（评估未实施）。

> 与 [T2 §6.2](./round-5-orm-migration-evaluation.md#62-round-6-分阶段规划既定路径非可选项) 对齐：Round 6+ 路径是 **① data → ② biz**（既定，非「做完 data 再议」）。本轮若引入 `service`，按 usecase 薄编排来写，**不要**把业务规则塞进 Repo/`data`，也**不要**让 `service` 沦为第二套 Repo。

---

## 三、Repo 扩面清单（Document / Book）

### DocumentRepo 建议补齐

- `FindById` / `FindByIdentify`  
- `ListByBookId` / 树形所需查询  
- `SearchLike(...)`（包装现有 LIKE，供 MCP/Web）  
- 写路径保持：`Create` / `UpdateMarkdownWithVersion` / `Delete` / `Release...`

### BookRepo 建议补齐

- `FindById` / `FindByIdentify`  
- `ListVisibleTo(member)`（供 `list_books`）  
- 权限判定所需的最小字段加载（避免 Controller 再拼 SQL）

### MemberRepo

- MCP token / stdio 身份解析已用到的读方法统一入口（若仍散落 model）

---

## 四、实施顺序

```text
1) DocumentRepo 补齐读方法 + MCP 读工具切换
2) BookRepo / MemberRepo 补齐 + authz / list_books
3)（可选）Web Document 阅读路径试点
4)（可选）抽 service.Document 去重 MCP/Web 编排
5) 单测：repository 表驱动；MCP 回归
```

与 **T8** 冲突时：先完成该 Result 的 Repo 抽取，再迁 dto（同域串行）。

---

## 五、验收

- [ ] MCP 读+写均经 Repo（例外须在本文或代码注释标明）  
- [ ] 权限 / 乐观锁行为与现网一致  
- [ ] `go test` 相关包不回退  
- [ ] **无** ORM 替换  

---

## 六、明确不做

- 引入 ent/sqlc/gorm  
- 全量 Controller 改走 Repo（留给后续 sprint）  
- 新建完整 kratos `biz/data` 目录树  

---

## 七、参考

- [round-5-orm-migration-evaluation.md](./round-5-orm-migration-evaluation.md)  
- [`internal/repository/`](../internal/repository/) · [`internal/mcp/`](../internal/mcp/)  
