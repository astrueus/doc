# Round 5 · T9 · Repository 扩面 + 可选 Service — 细化方案

> 对应 [round-5-execution-plan.md §十一 T9](./round-5-execution-plan.md#十一t9--repository-扩面--可选-service3~5-天)。  
> 方向对齐 [T2](./round-5-orm-migration-evaluation.md)：朝未来 `internal/data` / `internal/biz` 靠拢，**本轮不换 ORM**。  
> **状态：** ✅ 已实施（与 T8 同批）。本轮**未**建 `internal/service/`。

---

## 一、现状

| 项 | 状态 |
|---|---|
| [`internal/repository/document_repo.go`](../../internal/repository/document_repo.go) | CRUD + 乐观锁较完整 |
| `book_repo.go` | 读列表 / 权限 / 展示 Result 已扩面 |
| `member_repo.go` | 含账号、API Token、项目成员查询 |
| MCP 写路径 | 经 `documentRepo()` / `bookRepo()` |
| MCP 读 / authz / 搜索 | **经 Repo**（`DocumentRepo` / `BookRepo` / `MemberRepo`） |
| `internal/service/` | **不存在**（本轮未建） |

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

### 2.4 衔接约束（T2 已同步，实施时遵守）

1. **`repository` 不依赖** `controller` / `mcp` / Beego `web.Context`（可传标准 `context.Context`）。  
2. **业务规则不进 Repo**（发布编排、权限决策、乐观锁冲突处理留在调用方或未来 `service`/`biz`）。  
3. 本轮若建 `internal/service/`，只做 **usecase 薄编排**，不要变成第二套 Repo。  
4. **本轮不建** `internal/biz/`（评估未实施）。  
5. `model` 文件 snake_case 重命名与 **T8** 合并，避免 Result 迁出前搬两次。

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

- [x] MCP 读+写均经 Repo（例外：`resolveItemID` 仍读 `Itemsets` 实体；导出 HTML 仍调用 beego `ExecuteViewPathTemplate`）
- [x] 权限 / 乐观锁行为与现网对齐（`FindForRoleId` / `UpdateMarkdownWithVersion` 原 SQL 搬迁）
- [x] `go test` 相关包不回退（`internal/repository`、`internal/mcp`）
- [x] **无** ORM 替换
- [x] Web 试点：`DocumentController.Content` 的文档 `Find` 走 `DocumentRepo`；阅读页 `Read` 仍走 model 缓存（待 T12）
- [x] 本轮验收以代码与 `go test ./internal/repository ./internal/mcp` 为准（2026-08-31 收口）；未建 `internal/service/`

---

## 六、明确不做

- 引入 ent/sqlc/gorm  
- 全量 Controller 改走 Repo（留给后续 sprint）  
- 新建完整 kratos `biz/data` 目录树  

---

## 七、参考

- [round-5-orm-migration-evaluation.md](./round-5-orm-migration-evaluation.md)  
- [`internal/repository/`](../../internal/repository/) · [`internal/mcp/`](../../internal/mcp/)  
