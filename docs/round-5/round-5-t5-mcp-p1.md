# Round 5 · T5 · MCP 体验 P1 — 细化方案

> 对应 [round-5-execution-plan.md §七 T5](./round-5-execution-plan.md#七t5--mcp-体验-p10.5~1-天)。  
> 承接 Round 3 §十七 P1，并**纳入 Book 写工具最小集**（修订原 §17.3「当前阶段不做」——本轮做 create/update；delete 仍默认不做）。  
> **状态：** ⏳ 待实施。

---

## 一、现状

| 项 | 位置 | 说明 |
|---|---|---|
| 工具注册 | [`internal/mcp/server.go`](../../internal/mcp/server.go) | 10 工具；无 upsert / `if_exists`；无 Book 写 |
| 读 | [`internal/mcp/tools_read.go`](../../internal/mcp/tools_read.go) | `list_books` / `get_document` / `search_document` 等 |
| 写 | [`internal/mcp/tools_write.go`](../../internal/mcp/tools_write.go) | 仅 Document 写 |
| DTO | [`internal/dto/mcpdto/`](../../internal/dto/mcpdto/) | 已有 `BookBrief`；缺 Create/Update Book In/Out |
| 乐观锁 | `ExpectVersion` + `DocumentRepo.UpdateMarkdownWithVersion` | 冲突码 `6100`（文档） |
| 权限 | [`internal/mcp/authz.go`](../../internal/mcp/authz.go) | `canReadBook` / `ensureWritable`（书内编辑）；**无**「能否建书」校验 |
| 搜索 Out | `DocumentBrief` | **无** `book_identify` / `doc_identify` |

---

## 二、P1-1 · upsert / `if_exists=update`

### 推荐做法

**在 `create_document` 上增加可选参数 `if_exists`**，不新增独立工具名（减少工具面膨胀；Cursor/客户端只需多传一个字段）。

| 值 | 行为 |
|---|---|
| 缺省 / `error` | 现状：同书下 `identify` 已存在 → 返回业务错误 |
| `update` | 已存在 → 走「更新内容」路径；不存在 → 创建 |

### 输入 / 语义

```text
create_document:
  book_identify   string  required
  doc_identify    string  required
  title           string  optional（update 时缺省保留原标题）
  markdown        string  optional
  parent_identify string  optional（仅 create 路径生效）
  if_exists       string  optional  // "error" | "update"
  expect_version  int     optional  // 仅 if_exists=update 时生效；与 update_document_content 一致
```

### 实现要点

1. 先 `FindByIdentify(book, doc)`（经 Repo，配合 T9）。  
2. 不存在 → 现有 `create` 逻辑。  
3. 存在 + `if_exists=update` → 调与 `update_document_content` 相同的写路径（含 `ensureWritable`、乐观锁）。  
4. **不做**：改 `parent`、改 `identify`、静默覆盖无版本号的「强制写」（除非显式不传 `expect_version`，与现网 `update` 行为对齐）。  
5. 文档：[`mcp-integration.md`](../mcp-integration.md) 增示例与错误码说明。

### 验收

- [ ] 新 identify → 创建成功  
- [ ] 已存在 + `if_exists=update` → 内容更新；版本冲突返回 `6100`  
- [ ] 已存在 + 缺省 → 仍报错，不静默覆盖  

---

## 三、P1-2 · `get_document` 截断

### 参数

| 字段 | 类型 | 默认 | 说明 |
|---|---|---|---|
| `max_chars` | int | `0`（不截断） | 按 **Unicode 字符数**（`[]rune`）截断正文 |
| `include_truncated` | bool | `true` | 截断时是否在正文末追加 `…[truncated]` 标记 |

可选后续（本切片可不做）：`max_bytes`（UTF-8 字节上限）。

### 输出约定

- 增加布尔字段 `truncated: bool`（或 `content_truncated`），便于客户端区分「全文」与「摘要」。  
- 元数据（title / version / identify）**始终完整返回**；只截 `markdown` / `content`。  
- 上限建议在实现里硬顶一层（如 `max_chars > 200_000` 拒绝或钳制），防止恶意超大参数。

### 验收

- [ ] 短文：`truncated=false`，正文完整  
- [ ] 长文 + `max_chars=500`：长度合规且带标记  
- [ ] 权限与现网 `get_document` 一致  

---

## 四、P1-3 · `search_document` 带回 identify

### 改动

在 [`mcpdto`](../../internal/dto/mcpdto/) 的 `DocumentBrief`（或等价 Out）增加：

```go
BookIdentify string `json:"book_identify"`
DocIdentify  string `json:"doc_identify"`  // 文档 identify，非数字 id
```

### 实现

- 搜索仍走现有 `LIKE`（T3 冻结，不换引擎）。  
- 查询结果若已有 `identify` 列则直接映射；若只有数字 id，则在 Repo/model 层补齐 join（避免 N+1：一次查询带出）。  
- Web UI 搜索行为不变；仅 MCP Out 增字段（向后兼容）。

### 验收

- [ ] 公开书命中项同时含 `book_identify` + `doc_identify`  
- [ ] 客户端可凭二者直接调 `get_document` / `update_*`  
- [ ] 私有书仍受登录/权限约束  

---

## 五、P1-4 · Book 写工具（最小集）

> 承接 Round 3 [§17.3](../round-1-4/round-3-execution-plan.md#173-决策是否增加-create_book--update_book) 的「若将来做」最小集；**本轮改为实施**。  
> 定位：支撑「AI/CI 一键建空项目再灌文档」；封面、成员转让、高级导出等**仍走 Web**。

### 5.1 工具一览

| 工具 | 本轮 | 说明 |
|---|---|---|
| `create_book` | ✅ 做 | 建空项目；创建者成为创始人关系 |
| `update_book` | ✅ 做 | 仅元数据（标题/描述/公开私有等） |
| `delete_book` | ❌ 默认不做 | 误删整棵文档树代价高；若未来做须 `confirm=book_identify` + 仅创始人/超管 |

工具总数：现 10 + 本切片 Document P1 不增工具名 + **Book 2 个** → 注册表约 **12** 工具（`create_document` 仍是原名）。

### 5.2 `create_book`

#### 输入

```text
create_book:
  title        string  required   // 项目名称
  identify     string  required   // 唯一标识；非法字符/冲突按 Web 同规则报错
  private      bool    optional   // 默认 false=公开；true=私有（映射 PrivatelyOwned）
  description  string  optional   // 简介
  item_identify string optional  // 所属空间；缺省用站点默认空间（与 Web 创建一致）
```

#### 行为

1. **权限：** 与 Web `BookController` 创建一致——尊重 `member_general_can_create_book`（若配置存在）；管理员始终可建；禁止的角色返回 `403` 类业务码。  
2. `identify` 全局唯一；冲突返回明确错误（勿静默改名）。  
3. 写入 Book + Relationship（创建者 = 创始人）；可顺带建默认空白首页文档（**若 Web 创建也会建**，则 MCP 对齐；否则只建空书，由后续 `create_document` 灌文）。  
4. **不做：** 上传封面、复制书、导入 zip、改编辑器类型以外的重选项（除非 Web 创建表单的必填默认值必须带上——实现时对照 `SaveBook`，缺省用与 Web 相同的默认）。  

#### 输出

返回 `BookBrief`（或等价：`book_id` / `identify` / `title` / `private` / `description`），便于紧接着 `create_document`。

### 5.3 `update_book`

#### 输入

```text
update_book:
  book_id        int     optional  // 与 book_identify 二选一
  book_identify  string  optional
  title          string  optional  // 未传则不改
  description    string  optional
  private        bool    optional  // 未传则不改；注意 bool 零值：用指针或 *bool / 三态，避免「省略」与 false 混淆
```

#### 行为

1. **权限：** `ensureBookAdmin`（建议：**创始人 / 管理员**；普通 BookEditor **不可**改书级私有与 identify）。若现网 Web 允许编辑者改简介，可降级为「改 title/description 需 Editor+；改 private 需 Admin+」——实现前对照 `BookController` 设置页，**以 Web 为准**。  
2. **本轮允许改：** `title`、`description`、`private`。  
3. **本轮不改：** `identify`（改标识易断外链/MCP 脚本）、封面、成员、空间迁移。  
4. 走 `BookRepo`（T9 可先薄封装 `Create` / `UpdateMeta`）。

#### 输出

更新后的 `BookBrief`。

### 5.4 权限与 authz 扩展

在 [`authz.go`](../../internal/mcp/authz.go) 增加：

| 函数 | 语义 |
|---|---|
| `ensureCanCreateBook(m)` | 对齐 Web 建书开关与角色 |
| `ensureBookMetaWritable(m, bookID)` | update_book：至少 Editor 或 Admin（按字段细分见上） |

Token / stdio 身份与现网一致；私有书创建后，`list_books` 应对创建者可见。

### 5.5 DTO / 文档 / 注册

- [`internal/dto/mcpdto/book.go`](../../internal/dto/mcpdto/book.go)：`CreateBookIn` / `CreateBookOut` / `UpdateBookIn` / `UpdateBookOut`  
- [`server.go`](../../internal/mcp/server.go) 注册两工具；Description 标明权限  
- [`mcp-integration.md`](../mcp-integration.md)：工具表 + 「先 create_book 再灌文档」示例；更新「Book 写工具」进度说明  
- 决策日志：Round 3 §17.3 / roadmap「当前不做」→ **Round 5 T5 起实施最小集**

### 5.6 验收

- [ ] `create_book` 成功返回 identify；创建者可 `list_books` 见到  
- [ ] identify 冲突、无建书权限 → 明确错误  
- [ ] `update_book` 只改传入字段；私有/公开切换后读权限行为正确  
- [ ] 无 `delete_book` 工具  
- [ ] 既有 Document 10 工具 + P1-1~3 不回归  
- [ ] `mcp-integration.md` 与 schema 单测已更新  

---

## 六、PR 与测试

| PR | 内容 |
|---|---|
| T5-a | P1-1~3（document upsert / 截断 / search identify）+ 文档 + 单测 |
| T5-b | P1-4 `create_book` / `update_book` + authz + BookRepo 薄封装 + 文档 |

测试建议：

- 表驱动：`if_exists` 三态、截断边界、`DocumentBrief` JSON 字段  
- Book：建书成功/冲突/无权限；update 部分字段；Editor 越权改 private（若策略禁止）  
- 回归：原 10 工具 + P0 权限用例不红  

---

## 七、明确不做

- **`delete_book`**（本轮）  
- Book：封面上传、成员管理、空间转让、复制/导入项目  
- 搜索引擎升级（FULLTEXT / 倒排）  
- MCP 二进制附件上传  
- 改 `book.identify`（本轮 update 不开放）  

---

## 八、参考

- [mcp-integration.md](../mcp-integration.md)  
- [round-3-execution-plan.md §十七 / §17.3](../round-1-4/round-3-execution-plan.md#十七后续规划mcp-实测反馈与体验增强)  
- [`internal/mcp/`](../../internal/mcp/) · [`internal/dto/mcpdto/`](../../internal/dto/mcpdto/)  
- Web：`BookController` 创建/设置（权限与默认值对齐）  
