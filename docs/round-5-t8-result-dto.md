# Round 5 · T8 · `*Result` → `internal/dto/` — 细化方案

> 对应 [round-5-execution-plan.md §十 T8](./round-5-execution-plan.md#十t8--result--internaldto2~4-天)。  
> 承接 Round 2 收尾 B1。  
> **状态：** ⏳ 待实施。

---

## 一、目标

把「纯展示 / 视图」结构从 [`internal/model/`](../internal/model/) 迁到 [`internal/dto/`](../internal/dto/)，打断 model ↔ 展示层循环依赖；**含查询逻辑的方法**先抽到 [`internal/repository/`](../internal/repository/)（可与 T9 串联）。

---

## 二、清单与分类

| 类型 | 文件 | 分类 | 处理 |
|---|---|---|---|
| `BookResult` | `BookResult.go` | 混合（结构 + `Find*`） | 结构 → dto；查询 → BookRepo |
| `BlogResult` | `BlogResult.go` | 混合 | 同上 |
| `MemberRelationshipResult` / `SelectMemberResult` | `MemberResult.go` | 混合 | 同上 |
| `DocumentSearchResult` | `DocumentSearchResult.go` | 混合 + raw SQL | 结构 → dto；查询暂留 Repo/搜索模块（T3 冻结期不动引擎） |
| `ConvertBookResult` | `ConvertBookResult.go` | 偏纯展示 | → dto |
| `AttachmentResult` | `AttachmentResult.go` | 混合 | 结构 → dto；查询 → Repo |
| `DocumentHistorySimpleResult` | `DocumentHistory.go` | 偏纯展示 | → dto（可与 History 同 PR） |
| `CommentResult` | `comment_result.go` | 混合 | 结构 → dto；查询 → Repo |

现有 [`internal/dto/mcpdto/`](../internal/dto/mcpdto/) **不动**；新建例如：

```text
internal/dto/
├── mcpdto/           # 已有
├── book.go           # BookResult
├── blog.go
├── member.go
├── attachment.go
├── document_search.go
├── document_history.go
├── comment.go
└── convert_book.go
```

包名建议：`dto`（与 mcpdto 并列），类型名可保留 `BookResult` 以降低模板/调用改动面。

文件名遵循 Go 惯例 `lower_snake_case`（见 [T2 §1.4](./round-5-orm-migration-evaluation.md#14-文件命名不统一)）；若同批整理 `internal/model/`，Result 迁出后再对残留实体做 `git mv`，避免搬两次。

---

## 三、迁移步骤（每类型）

1. **盘点引用**：`rg BookResult`（controller / model / mcp / tests）。  
2. **若仅有字段 + 无 DB 方法**：移动类型到 `internal/dto/`，改 import。  
3. **若带 `Find*` / raw SQL**：  
   - 把方法挪到对应 Repo（或临时 `internal/repository/xxx_query.go`）  
   - Repo 返回 `dto.Xxx` 或继续返回组合结构  
   - model 文件删除或只留 ORM 实体  
4. **编译通过 → 冒烟**（相关页面 / MCP）。  
5. **单独 PR**，避免大爆炸。

### 循环依赖规避

- `dto` **禁止** import `model`（若需嵌套实体，改为 dto 内平铺字段，或只引用基础类型）。  
- `model` **禁止** import `dto`。  
- `repository` 可依赖 `model` + `dto`。  
- `controller` / `mcp` 依赖 `dto` + `repository`。

---

## 四、PR 建议顺序

| 顺序 | PR | 理由 |
|---|---|---|
| 1 | `ConvertBookResult` / `DocumentHistorySimpleResult` | 最接近纯展示 |
| 2 | `AttachmentResult` / `CommentResult` | 中等 |
| 3 | `BlogResult` / `Member*Result` | 调用面略广 |
| 4 | `BookResult` | Web 高频 |
| 5 | `DocumentSearchResult` | raw SQL，风险最高，放最后 |

---

## 五、验收

- [ ] 上表类型均已迁出或文档标明「暂留原因」  
- [ ] `internal/model` 不再被纯展示 dto 反向依赖  
- [ ] 相关页面 / MCP 冒烟通过  
- [ ] Round 2 B1 / 本追踪表更新  

---

## 六、明确不做

- 借机换 ORM（属 T2 / Round 6+）  
- 重写搜索 SQL（T3 冻结）  
- 一次 PR 搬完所有 Result  

---

## 七、参考

- [round-2-execution-plan.md](./round-2-execution-plan.md) B1  
- [round-5-orm-migration-evaluation.md](./round-5-orm-migration-evaluation.md) §1.3  
- [`internal/model/`](../internal/model/) · [`internal/dto/`](../internal/dto/)  
