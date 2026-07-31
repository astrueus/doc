# Round 3 · 执行文档（MCP Server + 搜索基础）

> 本文是 [refactor-roadmap.md §五 Round 3](./refactor-roadmap.md#🥉-round-3mcp--搜索基础2~3-周) 与 [§2.1 目标一](./refactor-roadmap.md#21-目标一mcp-serverai-接入) 的**可执行分解**。
> 目标：让 AI 助手（Claude Desktop / Cursor / 其他 MCP 客户端）通过 MCP 协议对 doc 项目做**读 + 写**操作，含创建/更新/删除文档。**采用官方** `modelcontextprotocol/go-sdk` **v1.x**（非社区 `mark3labs/mcp-go`，见 [§八 决策 2026-07-23](./refactor-roadmap.md#八决策记录decision-log)）。
> **代码直接落在 Round 2 完成的** `internal/mcp/` — 零重复搬迁。
>
> **进度标记（2026-07-31）：** T1 ⏸ 暂缓；T2–T7 ✅（MCP MVP 闭环）。`search_document` 过渡期使用 `LIKE`。  
> **实测后续：** Cursor MCP 批量写入 `docs/` 后的体验缺口与是否做 Book 级工具，见 [§十七](#十七后续规划mcp-实测反馈与体验增强)。

---



## 一、范围与不做清单



### 本轮做


| 序号  | 任务                                                      | 工作量          | 上线感知                               |
| --- | ------------------------------------------------------- | ------------ | ---------------------------------- |
| T1  | 搜索最小方案（MySQL FULLTEXT / SQLite FTS5 + 标题加权）             | 2~3 天        | ⏸ **暂缓**（2026-07-30：方案待再评估；不挡 T2+） |
| T2  | MCP MVP · stdio · 4 个读工具                                | 2 天          | 新增 CLI `doc mcp`                   |
| T3  | MCP MVP · stdio · 6 个写工具（含乐观锁 + 确认参数）                   | 2~3 天        | 同上                                 |
| T4  | `internal/model/MemberApiToken` 新表 + 后台管理页              | 2 天          | 新增管理页 `/member/api-tokens`         |
| T5  | MCP Streamable HTTP + Bearer 鉴权 + 限流                    | 2 天          | 新增 HTTP endpoint `/mcp/*`          |
| T6  | `internal/dto/mcpdto/`（工具 In/Out struct）                | 与 T2/T3 同 PR | —                                  |
| T7  | `docs/mcp-integration.md`（Claude Desktop / Cursor 接入指南） | 0.5 天        | —                                  |


**总工期：** 10~~14 个工作日（2~~3 周）。

### 本轮**不做**（明确排除）

- ❌ **不上向量检索**（bleve / meilisearch / qdrant）— Round 4 或独立立项
- ❌ **不做倒排索引服务化**（如 elasticsearch）— 同上
- ❌ **不做 SSE 传输**（Streamable HTTP 已覆盖大部分场景，SSE 需要时另加）
- ❌ **不复用** `MemberToken` **表做 API Token**（[§六 风险 11](./refactor-roadmap.md#六关键风险清单)明确禁止）
- ❌ **不做 MCP Prompts / Resources**（本轮只做 Tools，SDK 支持 Prompts/Resources 但暂无用户场景）
- ❌ **不做 MCP 端到端集成测试**（框架搭好 + `mcp` cmdline 手工调用即可，自动化测试 Round 4 补）

---



## 二、前置条件（Round 2 已完成）

- ✅ `internal/mcp/` 空目录（Round 2 T8 已建）
- ✅ `internal/dto/mcpdto/` 空目录（同上）
- ✅ `internal/config/config.go` 有 `MCPConfig` 字段（Round 2 T4）
- ✅ `conf/app.conf` 有 `[mcp]` section（Round 2 T3 / 收尾 A 后路径为 `conf/`）
- ✅ `internal/router/api.go`（或 `router.go`）可挂 `/mcp/*` handler（Round 2 T6）
- ✅ `internal/middleware/ratelimit.go` 占位存在（Round 2 T7）
- ✅ `internal/errs/` 已存在（Round 1 T7）
- ✅ `internal/cache/` 抽好 `Cache` 接口（Round 1 T5）
- ✅ `spf13/cobra` 已引入，`doc mcp` 子命令 stub 已存在（Round 1 T4）
- ✅ 强类型 `config.Global.MCP.XXX` 可读（Round 2 T4）

**分支：** `feature/round-3-mcp`（内部按 T1~T7 拆 7 个 PR）

---



## 三、工具矩阵（10 个）


| 类型  | 工具名                       | 权限门槛                   | 关键保护                                            |
| --- | ------------------------- | ---------------------- | ----------------------------------------------- |
| 读   | `search_document`         | 无（尊重公开/私有）             | 结果加 `bookRole` 过滤                               |
| 读   | `get_document`            | 无                      | 私有 book 需 `BookRole ≥ Observer`                 |
| 读   | `list_books`              | 无                      | 只返回当前身份可访问的 book                                |
| 读   | `list_document_tree`      | 无                      | 同 get_document                                  |
| 写   | `create_document`         | `BookRole ≥ Editor(2)` | —                                               |
| 写   | `update_document_content` | 同上                     | **必带** `expect_version` **乐观锁**                 |
| 写   | `append_document_content` | 同上                     | 无乐观锁（幂等追加更少见冲突）                                 |
| 写   | `update_document_meta`    | 同上                     | 改 title/identify 等元信息                           |
| 写   | `release_document`        | 同上                     | 触发 `ReleaseContent()`（Markdown → HTML）          |
| 写   | `delete_document`         | 同上                     | **必带** `confirm: true` + 写快照到 `DocumentHistory` |


**权限对照**（`conf/enumerate.go:43-49`）：

```go
BookFounder BookRole = iota  // 0
BookAdmin                    // 1
BookEditor                   // 2  ← 写权限门槛
BookObserver                 // 3
```

---



## 四、关键数据结构



### 4.1 `internal/model/MemberApiToken.go`（T4 新表）

```go
package model

type MemberApiToken struct {
    TokenId      int       `orm:"column(token_id);pk;auto"`
    MemberId     int       `orm:"column(member_id);type(int);index"`
    TokenHash    string    `orm:"column(token_hash);size(64);unique"`  // sha256(token) hex
    Name         string    `orm:"column(name);size(100)"`               // 用户命名 "Claude Desktop"
    Scopes       string    `orm:"column(scopes);size(255)"`             // 逗号分隔 "read,write,admin"
    ExpiresAt    time.Time `orm:"column(expires_at);null"`              // 可为空 = 永不过期
    LastUsedAt   time.Time `orm:"column(last_used_at);null"`
    LastUsedIP   string    `orm:"column(last_used_ip);size(45);null"`
    RevokedAt    time.Time `orm:"column(revoked_at);null"`
    CreatedAt    time.Time `orm:"column(created_at);auto_now_add"`
}

func (m *MemberApiToken) TableName() string           { return "member_api_tokens" }
func (m *MemberApiToken) TableEngine() string         { return "INNODB" }
func (m *MemberApiToken) TableNameWithPrefix() string { return config.GetDatabasePrefix() + m.TableName() }
```

**与** `MemberToken` **的关键差异**（对比参考 `models/MemberToken.go`）：


| 字段   | `MemberToken`（邮箱验证码）                             | `MemberApiToken`（本表）          |
| ---- | ------------------------------------------------ | ----------------------------- |
| 主用途  | 找回密码 / 邮箱验证                                      | MCP HTTP Bearer 鉴权            |
| 明文存储 | `Token` 明文（因为要邮件里显示链接）                           | `TokenHash` = `sha256(token)` |
| 有效期  | `ValidTime`（单次 30 分钟）                            | `ExpiresAt`（长期，可空）            |
| 使用次数 | `SendTime` 限流发送                                  | `LastUsedAt` 记录审计             |
| 关联   | `Email` 字段绑邮箱                                    | 只关联 `MemberId`                |
| 是否复用 | ❌ 不复用（[§六 风险 11](./refactor-roadmap.md#六关键风险清单)） | —                             |


**DDL（迁移文件）：**

```sql
CREATE TABLE `md_member_api_tokens` (
  `token_id`     INT NOT NULL AUTO_INCREMENT,
  `member_id`    INT NOT NULL,
  `token_hash`   VARCHAR(64) NOT NULL,
  `name`         VARCHAR(100) NOT NULL DEFAULT '',
  `scopes`       VARCHAR(255) NOT NULL DEFAULT 'read',
  `expires_at`   DATETIME NULL,
  `last_used_at` DATETIME NULL,
  `last_used_ip` VARCHAR(45) NULL,
  `revoked_at`   DATETIME NULL,
  `created_at`   DATETIME NOT NULL,
  PRIMARY KEY (`token_id`),
  UNIQUE KEY `uk_token_hash` (`token_hash`),
  KEY `idx_member_id` (`member_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

SQLite 版本用相同字段结构，去掉 ENGINE/CHARSET。

### 4.2 工具 In/Out DTO（`internal/dto/mcpdto/`）

**示例 ·** `update_document_content`**：**

```go
// internal/dto/mcpdto/document.go
package mcpdto

type UpdateDocumentContentIn struct {
    DocumentID    int    `json:"document_id"    jsonschema:"required,description=文档ID"`
    ExpectVersion int64  `json:"expect_version" jsonschema:"required,description=期望的 Document.Version（时间戳），乐观锁用"`
    Markdown      string `json:"markdown"       jsonschema:"required,description=完整 Markdown 内容（覆盖式写入）"`
    AutoRelease   bool   `json:"auto_release,omitempty" jsonschema:"description=写入后是否立即 release（Markdown→HTML），默认 false"`
}

type UpdateDocumentContentOut struct {
    DocumentID int    `json:"document_id"`
    Version    int64  `json:"version"`     // 更新后的新 version
    Message    string `json:"message"`     // 人类可读
}
```

**示例 ·** `delete_document`**：**

```go
type DeleteDocumentIn struct {
    DocumentID int  `json:"document_id" jsonschema:"required"`
    Confirm    bool `json:"confirm"     jsonschema:"required,description=必须为 true 才会真删；给 AI 一个显式意图确认"`
}
```

其余 8 个工具 In/Out 见 T2/T3 章节末。

### 4.3 `conf/app.conf` `[mcp]` section（Round 2 已加）

```ini
[mcp]
mcp_enable         = ${DOC_MCP_ENABLE||false}
mcp_listen         = ${DOC_MCP_LISTEN||127.0.0.1:8280}
mcp_stdio_member   = ${DOC_MCP_STDIO_MEMBER||admin}       # stdio 模式使用哪个 member 身份
mcp_token_required = ${DOC_MCP_TOKEN_REQUIRED||true}      # HTTP 模式是否强制 Bearer
mcp_rate_limit     = ${DOC_MCP_RATE_LIMIT||60}            # 写工具 req/min per token
```

---



## 五、目标目录结构

```
internal/mcp/
├─ doc.go              # package mcp（Round 2 T8 占位；本轮扩充）
├─ server.go           # NewStdioServer() / NewHTTPServer() 组装
├─ authz.go            # Bearer 校验 + memberFromCtx()
├─ errors.go           # mcpError → JSON-RPC error 映射（含 VERSION_CONFLICT / CONFIRM_REQUIRED 等）
├─ ratelimit.go        # 官方 SDK 的 middleware，golang.org/x/time/rate
├─ search_provider.go  # searchProvider 接口 + sqlLikeProvider（默认）
├─ convert.go          # model.Document → dto 输出格式转换
├─ tools_read.go       # search_document / get_document / list_books / list_document_tree
├─ tools_write.go      # create/update/append/update_meta/release/delete
├─ http.go             # http.Handler (Beego 挂 /mcp/*)
└─ stdio.go            # stdin/stdout 通道封装

internal/dto/mcpdto/
├─ doc.go              # package mcpdto
├─ document.go         # Document 相关 In/Out
├─ book.go             # Book 相关
└─ search.go           # search_document 相关

internal/cli/
└─ mcp.go              # cobra 子命令：doc mcp [--http]（替换 Round 1 的 stub）

internal/controller/
└─ MemberApiTokenController.go   # 后台 Token 管理（列表/新建/撤销）

web/views/member/
└─ api_tokens.tpl                # Token 管理页
```

---



## 六、T1 · 搜索最小方案（2~3 天）

> **状态（2026-07-30）：⏸ 暂缓 / 方案待再评估。**  
> 曾尝试按本节落地后已还原，不纳入当前 Round 3 合入范围。待评估点包括：索引列 `markdown` vs 现网 `release`、迁移形态（Go `Migration` vs `*.up.sql`）、锁表风险、是否与 MCP 解耦等。  
> **过渡：** T2 `search_document` 可先基于现有 `DocumentSearchResult` / `LIKE`；本节方案保留作后续评估底稿，勿删。



### 现状

`internal/model/DocumentSearchResult.go`（原 `models/DocumentSearchResult.go`，10.8KB）用 SQL `LIKE` 匹配 `title` 和 `markdown`，权限走 `book_id IN (...)`。

### 方案

**MySQL：** 加 FULLTEXT 索引

```sql
ALTER TABLE `md_documents`
  ADD FULLTEXT INDEX `ft_title_markdown` (`document_name`, `markdown`) WITH PARSER ngram;
-- ngram 是为了中文分词；MySQL 5.7.6+ 支持。ngram_token_size 建议 = 2
```

**SQLite：** 建 FTS5 影子表

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
    document_name, markdown, content='md_documents', content_rowid='document_id'
);
-- 触发器同步（略）
```

**加权：** `search_document` 里 SQL 用 `MATCH ... AGAINST (? IN BOOLEAN MODE)` + `title` 命中权重 3 倍：

```sql
SELECT d.*,
    (MATCH(d.document_name) AGAINST (? IN BOOLEAN MODE) * 3 +
     MATCH(d.markdown)      AGAINST (? IN BOOLEAN MODE)) AS score
FROM md_documents d
WHERE MATCH(d.document_name, d.markdown) AGAINST (? IN BOOLEAN MODE)
  AND d.book_id IN (?)
ORDER BY score DESC
LIMIT ?
```



### `searchProvider` 抽象（为 Round 4 上倒排/向量铺路）

```go
// internal/mcp/search_provider.go
package mcp

type searchProvider interface {
    Search(ctx context.Context, q string, bookIDs []int, limit int) ([]*model.Document, error)
}

type sqlFulltextProvider struct{}  // 本轮实现
type sqlLikeProvider struct{}      // 降级实现（无 FULLTEXT 索引时）
// 未来：bleveProvider / meilisearchProvider ...

func newSearchProvider() searchProvider {
    if hasFulltextIndex() { return &sqlFulltextProvider{} }
    return &sqlLikeProvider{}
}
```



### 迁移

- 新增 `internal/migrate/round3_search_index.up.sql` / `.down.sql`
- `doc install --upgrade` 或首次启动自动执行（评估：老库有大量数据时 `ALTER TABLE ADD FULLTEXT` 会锁表，建议手工执行）
- README 加"升级到 Round 3 需执行搜索索引迁移"



### 验收

- 老库执行 DDL 后，`SELECT MATCH(...) AGAINST(...)` 能返回结果
- MCP `search_document("test")` 与 web UI 搜索结果**基本一致**（顺序可能因加权略不同）
- SQLite 场景走 FTS5 也能通

---



## 七、T2 · MCP MVP stdio · 4 个读工具（2 天）



### `commands/mcp.go`（Round 1 stub）→ `internal/cli/mcp.go`（本轮实现）

```go
// internal/cli/mcp.go
var mcpCmd = &cobra.Command{
    Use:   "mcp",
    Short: "Start MCP server (stdio by default)",
    RunE: func(cmd *cobra.Command, args []string) error {
        if httpMode {
            return mcp.RunHTTP(context.Background(), &config.Global.MCP)  // T5
        }
        return mcp.RunStdio(context.Background(), &config.Global.MCP)     // T2
    },
}
var httpMode bool

func init() {
    mcpCmd.Flags().BoolVar(&httpMode, "http", false, "Serve MCP over Streamable HTTP")
    rootCmd.AddCommand(mcpCmd)
}
```

`cmd/doc/main.go` 调用 `cli.Execute()` 触发 cobra，`mcp.RunStdio` 里：

1. `app.Bootstrap()` — 加载 config / DB / cache / models（复用 web 启动逻辑，跳过 `web.Run()`）
2. 用 `config.Global.MCP.StdioMember` 查出 `Member`，注入到 `ctx`
3. 建 `modelcontextprotocol/go-sdk` 的 `Server`，注册 10 个工具
4. `server.Serve(stdio.NewChannel(os.Stdin, os.Stdout))`



### 工具 In/Out & Handler 骨架

`search_document`**：**

```go
// internal/dto/mcpdto/search.go
type SearchDocumentIn struct {
    Query   string `json:"query"          jsonschema:"required,description=全文搜索关键字"`
    BookID  int    `json:"book_id,omitempty" jsonschema:"description=限定 book，0=全部可见"`
    Limit   int    `json:"limit,omitempty"   jsonschema:"minimum=1,maximum=50,description=返回条数，默认 10"`
}
type SearchDocumentOut struct {
    Total int             `json:"total"`
    Items []DocumentBrief `json:"items"`
}
type DocumentBrief struct {
    ID       int    `json:"id"`
    BookID   int    `json:"book_id"`
    Title    string `json:"title"`
    Snippet  string `json:"snippet"`  // 摘要，从 Markdown 抽 200 字
    Version  int64  `json:"version"`
}
```

```go
// internal/mcp/tools_read.go
func handleSearchDocument(ctx context.Context, req *mcp.CallToolRequest, in mcpdto.SearchDocumentIn) (*mcp.CallToolResult, mcpdto.SearchDocumentOut, error) {
    member := memberFromCtx(ctx)
    if member == nil { return nil, mcpdto.SearchDocumentOut{}, errs.New(errs.CodeUnauthorized, "unauthorized") }

    bookIDs, err := visibleBookIDs(member, in.BookID)
    if err != nil { return nil, out, err }

    docs, err := searchProvider.Search(ctx, in.Query, bookIDs, defaultLimit(in.Limit, 10))
    if err != nil { return nil, out, errs.Wrap(errs.CodeInternal, "search failed", err) }

    out := convertToSearchOut(docs)
    return nil, out, nil
}
```

**4 个读工具 In 汇总：**


| 工具                   | 关键参数                                         |
| -------------------- | -------------------------------------------- |
| `search_document`    | `query`、`book_id?`、`limit?`                  |
| `get_document`       | `document_id` 或 `book_identify+doc_identify` |
| `list_books`         | `page?`、`page_size?`（默认 20，最大 100）           |
| `list_document_tree` | `book_id` 或 `book_identify`                  |


**注册（官方 SDK 泛型 API）：**

```go
// internal/mcp/server.go
srv := mcp.NewServer(&mcp.Implementation{Name: "doc-mcp", Version: "1.0.0"}, nil)

mcp.AddTool(srv, &mcp.Tool{
    Name: "search_document",
    Description: "Search documents by keyword; returns brief items with book/title/snippet.",
}, handleSearchDocument)

// ... 其余 9 个
```



### 验收（T2）

- `./doc mcp` 启动进入 stdio 模式
- 用 `[@modelcontextprotocol/inspector](https://github.com/modelcontextprotocol/inspector)` 或手工 JSON-RPC 调用：
  - `tools/list` 返回 10 个工具（本 PR 完成后为 4 个）
  - `tools/call search_document` 返回带 `snippet` 的列表
- Claude Desktop 添加 MCP server 后能问出"帮我搜索关于 XXX 的文档"

---



## 八、T3 · MCP MVP stdio · 6 个写工具（2~3 天）



### 写工具设计要点（对齐 [§2.1 关键设计点](./refactor-roadmap.md#关键设计点)）

1. **权限**：所有写工具进入 handler 第一件事 → `checkWritePermission(ctx, bookID)` → 要求 `BookRole ≥ BookEditor(2)`
2. **只写 Markdown**：写工具只更新 `Document.Markdown`；`Content` / `Release`（HTML）不动，由 `release_document` 或 `auto_release=true` 触发 `ReleaseContent()`
3. **乐观锁**（`update_document_content` 独占）：
  - 客户端从 `get_document` 拿到 `version`
  - `update_document_content(document_id, expect_version, markdown)`
  - Handler 内 `UPDATE ... SET markdown=?, version=? WHERE document_id=? AND version=?`
  - `RowsAffected == 0` → 返回 `VERSION_CONFLICT`（`errs.CodeVersionConflict = 6100`）
  - AI 侧收到冲突 → 重新 `get_document` 后 diff/merge 重试
4. **删除保护**（`delete_document`）：
  - 参数 `confirm bool` **必须为 true**，否则返回 `CONFIRM_REQUIRED`
  - 删前把当前 markdown 快照到 `DocumentHistory`
  - 走现有 `DocumentModel.RecursiveDocument` 逻辑
5. **限流**：写工具与删除工具的分类限流由 `mcp/ratelimit.go` 实现（见 T5）



### 6 个写工具 In/Out（简表）


| 工具                        | In 关键字段                                                           | Out                                    |
| ------------------------- | ----------------------------------------------------------------- | -------------------------------------- |
| `create_document`         | `book_id`, `parent_id`, `title`, `identify?`, `markdown?`         | `document_id`, `version`               |
| `update_document_content` | `document_id`, `expect_version`, `markdown`, `auto_release?`      | `document_id`, `version`, `message`    |
| `append_document_content` | `document_id`, `markdown_append`                                  | `document_id`, `version`               |
| `update_document_meta`    | `document_id`, `title?`, `identify?`, `order_sort?`, `parent_id?` | `document_id`                          |
| `release_document`        | `document_id` 或 `book_id`（批量 release 整个 book）                     | 受影响文档数                                 |
| `delete_document`         | `document_id`, `confirm=true`                                     | `deleted_count`, `snapshot_history_id` |




### 关键 handler 示例

```go
// internal/mcp/tools_write.go
func handleUpdateDocumentContent(ctx context.Context, _ *mcp.CallToolRequest, in mcpdto.UpdateDocumentContentIn) (*mcp.CallToolResult, mcpdto.UpdateDocumentContentOut, error) {
    m := memberFromCtx(ctx)
    doc, err := model.NewDocument().Find(in.DocumentID)
    if err != nil { return nil, out, errs.Wrap(errs.CodeNotFound, "document not found", err) }

    if err := ensureWritable(m, doc.BookId); err != nil { return nil, out, err }

    newVersion := time.Now().Unix()
    aff, err := orm.NewOrm().QueryTable(doc.TableNameWithPrefix()).
        Filter("document_id", in.DocumentID).
        Filter("version",     in.ExpectVersion).      // 乐观锁
        Update(orm.Params{
            "markdown": in.Markdown,
            "version":  newVersion,
        })
    if err != nil { return nil, out, errs.Wrap(errs.CodeInternal, "db error", err) }
    if aff == 0 {
        return nil, out, errs.New(errs.CodeVersionConflict, "version conflict: please refetch with get_document and retry")
    }

    if in.AutoRelease {
        _ = doc.ReleaseContent()  // 复用现有 Markdown→HTML
    }
    return nil, mcpdto.UpdateDocumentContentOut{
        DocumentID: in.DocumentID, Version: newVersion, Message: "updated",
    }, nil
}
```



### 快照写 History（`delete_document`）

```go
h := model.DocumentHistory{
    DocumentId: doc.DocumentId,
    Markdown:   doc.Markdown,
    Content:    doc.Content,
    Version:    doc.Version,
    ModifyAt:   m.MemberId,
    ModifyName: m.Account,
    HistoryType: "mcp_delete",
}
_ = orm.NewOrm().Insert(&h)
```



### 验收（T3）

- Inspector 调 `create_document(book_id, "hello world")` → 数据库多出 doc
- `get_document(id)` → 拿到 `version`
- `update_document_content(id, expect_version=拿到的, markdown="x")` → 200，返回新 version
- 再 `update_document_content(id, expect_version=旧的, markdown="y")` → `VERSION_CONFLICT`
- `delete_document(id)`（不带 confirm）→ `CONFIRM_REQUIRED`
- `delete_document(id, confirm=true)` → doc 被删，`document_history` 表多一条快照

---



## 九、T4 · `MemberApiToken` + 后台管理页（2 天）



### 数据库

- `internal/model/MemberApiToken.go`（本文 §4.1）
- 迁移：`internal/migrate/migrate_round3_member_api_token.go`（版本 `202607301700`），执行 `doc migrate`（与现有 Go Migration 一致；非独立 `*.up.sql`）



### 后端 controller

`internal/controller/MemberApiTokenController.go`：


| 方法       | 路由                                   | 功能                               |
| -------- | ------------------------------------ | -------------------------------- |
| `Index`  | GET `/member/api-tokens`             | 列表页                              |
| `Create` | POST `/member/api-tokens/create`     | 生成新 token（**只返回明文一次**，之后只存 hash） |
| `Revoke` | POST `/member/api-tokens/:id/revoke` | 撤销                               |


**生成逻辑：**

```go
raw := crypto.RandomString(48)               // 48 字节 = 384bit 熵，够
hash := sha256.Sum256([]byte(raw))
token := &model.MemberApiToken{
    MemberId:  m.MemberId,
    TokenHash: hex.EncodeToString(hash[:]),
    Name:      c.GetString("name"),
    Scopes:    c.GetString("scopes"),        // 默认 "read,write"
    ExpiresAt: parseExpires(c.GetString("expires_at")),
}
_, _ = orm.NewOrm().Insert(token)
// 只这一次返回明文
c.JsonResult(0, "ok", map[string]any{"token": "doc_" + raw, "token_id": token.TokenId})
```

**明文格式：** 前缀 `doc_` + `raw`，方便日志脱敏（正则匹配 `doc_[A-Za-z0-9]+`）。

### 前端页面

`web/views/member/api_tokens.tpl`：

- 表格：Name / Scopes / ExpiresAt / LastUsedAt / LastUsedIP / 撤销按钮
- "生成新 Token" 弹窗 → 提交后**弹窗显示明文**（只此一次）+ 强提示"关闭窗口后无法再看到"



### 路由

`internal/router/account.go`（或 `member.go`）加：

```go
web.Router("/member/api-tokens",              &controller.MemberApiTokenController{}, "get:Index")
web.Router("/member/api-tokens/create",       &controller.MemberApiTokenController{}, "post:Create")
web.Router("/member/api-tokens/:id/revoke",   &controller.MemberApiTokenController{}, "post:Revoke")
```

`internal/middleware/auth.go` 里已有登录校验，直接 cover。

### 验收（T4）

- 登录用户能看到自己的 Token 列表
- 新建 → 弹窗展示明文 → 刷新后列表出现新行、明文消失
- 撤销 → `revoked_at` 有值 → HTTP MCP 用该 token 请求返回 401

---



## 十、T5 · Streamable HTTP + Bearer + 限流（2 天）



### 挂到 Beego

`internal/router/api.go`：

```go
web.Handler("/mcp/*", mcp.NewHTTPHandler(), true)   // 第三参数 true 允许 catchall
```

或在 `internal/app/bootstrap.go` 直接 `web.Handler("/mcp/", mcpHandler, true)`。

### `internal/mcp/http.go`（handler 骨架）

```go
package mcp

func NewHTTPHandler() http.Handler {
    srv := buildServer()  // 与 stdio 共享工具注册
    return &streamableHTTP{server: srv}
}

func (h *streamableHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Bearer 校验（authz.go）
    m, err := verifyBearer(r)
    if err != nil { http.Error(w, err.Error(), http.StatusUnauthorized); return }

    // 2. 注入 member 到 ctx
    ctx := context.WithValue(r.Context(), memberCtxKey{}, m)

    // 3. 限流（ratelimit.go）
    if !rateLimiter.AllowByToken(m.TokenID, r.URL.Path) {
        http.Error(w, "rate limited", http.StatusTooManyRequests); return
    }

    // 4. 交给官方 SDK 的 streamable HTTP transport
    h.server.ServeStreamableHTTP(w, r.WithContext(ctx))
}
```



### `internal/mcp/authz.go`

```go
func verifyBearer(r *http.Request) (*model.Member, error) {
    auth := r.Header.Get("Authorization")
    if !strings.HasPrefix(auth, "Bearer ") { return nil, errUnauthorized }
    raw := strings.TrimPrefix(auth, "Bearer doc_")

    hash := sha256.Sum256([]byte(raw))
    hex := hex.EncodeToString(hash[:])

    // 缓存：命中 → 直接返回（cache.Cache Round 1 已抽好）
    var m model.Member
    if err := tokenCache.Get(r.Context(), "mcp:tok:"+hex, &m); err == nil {
        return &m, nil
    }

    var t model.MemberApiToken
    if err := orm.NewOrm().QueryTable((&t).TableNameWithPrefix()).
        Filter("token_hash", hex).Filter("revoked_at__isnull", true).One(&t); err != nil {
        return nil, errUnauthorized
    }
    if !t.ExpiresAt.IsZero() && t.ExpiresAt.Before(time.Now()) { return nil, errUnauthorized }

    member, err := model.NewMember().Find(t.MemberId)
    if err != nil { return nil, errUnauthorized }

    // 异步更新 last_used_at / last_used_ip
    go updateLastUsed(t.TokenId, r.RemoteAddr)

    _ = tokenCache.Set(r.Context(), "mcp:tok:"+hex, member, 5*time.Minute)
    return member, nil
}
```



### `internal/mcp/ratelimit.go`

```go
// 每 token 每分钟：读工具 60 次，写工具 30 次，delete 10 次
type limits struct {
    read, write, delete *rate.Limiter
}

func AllowByToken(tokenID int, toolKind string) bool {
    l := getOrCreate(tokenID)
    switch toolKind {
    case "delete": return l.delete.Allow()
    case "write":  return l.write.Allow()
    default:       return l.read.Allow()
    }
}
```

具体限额从 `config.Global.MCP.RateLimit`（默认 60/min）派生：`read = X`，`write = X/2`，`delete = X/6`。

### HTTPS 要求

- **强制**在部署文档说明 HTTP MCP 必须走 HTTPS（Nginx 反代或 Traefik）
- `internal/cli/mcp.go` 加 warn：若 `mcp_listen` 是 `0.0.0.0` 且未检测到前置 TLS，打印告警
- Bearer token 明文不能走裸 HTTP



### 验收（T5）

- 生成一个 API token（T4）
- `curl -H "Authorization: Bearer doc_XXX" http://127.0.0.1:8280/mcp/` 能拿到工具列表
- 错误 token → 401
- 60 req/min 之后返回 429
- 撤销 token 后 5 分钟内（缓存过期时间）内的请求仍能过；5 分钟后 401（可选：撤销时主动 cache.Delete）

---



## 十一、T6 · `internal/dto/mcpdto/` 与 T7 · 接入文档



### T6 · DTO 汇总

- `internal/dto/mcpdto/search.go` — Search 系列
- `internal/dto/mcpdto/document.go` — Get / Create / Update / Append / UpdateMeta / Release / Delete
- `internal/dto/mcpdto/book.go` — ListBooks / ListDocumentTree

**为什么 DTO 独立包：** Round 4 引入 Repository/Service 分层后，`mcpdto` 天然是 service→adapter 的输出契约；同时防止 `internal/mcp/` 与 `internal/model/` 循环 import。

### T7 · `docs/mcp-integration.md`

必写章节：

1. **前置**：生成 API Token（截图）
2. **stdio 接入 Claude Desktop**：
  ```json
   {
     "mcpServers": {
       "doc": {
         "command": "/path/to/doc",
         "args": ["mcp"],
         "env": { "DOC_MCP_STDIO_MEMBER": "admin" }
       }
     }
   }
  ```
3. **stdio 接入 Cursor**：`.cursor/mcp.json` 类似格式
4. **HTTP 接入**（更适合团队共享）：
  ```json
   {
     "mcpServers": {
       "doc-remote": {
         "url": "https://docs.example.com/mcp",
         "headers": { "Authorization": "Bearer doc_XXX" }
       }
     }
   }
  ```
5. **10 个工具速查表**（含每个工具的 In/Out 示例、常见错误码）
6. **常见问题**：
  - 遇到 `VERSION_CONFLICT` 怎么办
  - 遇到 429 rate limit 怎么办
  - stdio vs HTTP 怎么选
  - 如何撤销 token
7. **安全建议**：不要把 Token 提交到 git、每 token 只给必要 scope、定期轮转

---



## 十二、PR 拆分


| #   | PR                                                           | 内容                                | 大小  |
| --- | ------------------------------------------------------------ | --------------------------------- | --- |
| 1   | `feat(round3): search fulltext index + provider abstraction` | T1（⏸ 暂缓）                          | 中   |
| 2   | `feat(round3): MCP stdio server with 10 read/write tools`    | T2+T3+T6（已合入 `5b6ca51`）           | 中大  |
| 3   | `feat(round3): MemberApiToken table + management page`       | T4（已合入 `4c4f346`）                 | 中   |
| 4   | `feat(round3): MCP Streamable HTTP with Bearer & rate limit` | T5（已合入 `2fe3f6a`）                  | 中   |
| 5   | `docs(round3): MCP integration guide`                        | T7（已合入 `67b30d5`）                  | 小   |


**合入顺序：** ~~1 →~~ **2 → 3 → 4 → 5**（T1 / PR-1 暂缓，见 §六）。  
原计划「读/写拆两个 PR」已在实现时合并为一次 stdio 落地（见 §十四）。  
理由：T2 stdio 是最小可用产品；T1 不挡 MCP；T4 是 T5 的前置；T5 需要 T4 提供 token。  
T1 评估完成后再插回（可与 MCP 并行或作为独立 PR）。

---



## 十三、验收清单（全轮回归）

> **实测备注（2026-07-31）：** 本地测试环境经 Cursor 已配置的 HTTP MCP（`user-doc-remote`）完成读冒烟 + 将仓库 `docs/*.md` 写入私有项目 `test12`。下列 `[x]` 为该次可确认项；其余仍待专项/负例回归。



### 功能

- [ ] `search_document` 与 web UI 搜索结果基本一致（MCP 侧 LIKE 搜索可用；未与 Web UI 逐条对比）
- [x] `get_document` / `list_books` / `list_document_tree` 在有权限身份下工作正常（未做无权限负例；「权限过滤正确」完整验收待补）
- [x] `create_document` → 文档树可见新文档（`docs` 父文档及子文档；未单独打开 Web UI 肉眼核对，数据与 `list_document_tree` 一致）
- [ ] `update_document_content` 乐观锁生效（并发场景）（覆盖写已验通；6100 冲突并发未测）
- [x] `append_document_content` 追加正常（大文件分块追加可用；偶发长度略短见 §十七）
- [ ] `update_document_meta` 改标题后 web UI 立即可见
- [x] `release_document` 触发 Markdown→HTML（含 `auto_release`；抽查 `get_document.release` 有 HTML）
- [ ] `delete_document` 必须带 `confirm: true`，删前有 History 快照
- [ ] MCP HTTP 401 / 429 触发路径通



### 安全

- [ ] Token 数据库只存 hash，明文只出现在生成时的响应里
- [ ] Token 撤销后请求被拒（撤销后最长 5 分钟因缓存生效延迟，符合预期）
- [ ] 无写权限的 member 调 `create_document` 返回 `FORBIDDEN`
- [ ] `delete_document` 无 `confirm` 返回 `CONFIRM_REQUIRED`
- [ ] Rate limit 生效（可用 `ab` / `wrk` 短测）



### 集成

- [ ] Claude Desktop stdio 接入能问出"搜索 XXX"
- [x] Cursor HTTP 接入能问出同上（读工具冒烟 + 写文档全流程）
- [x] Inspector 里 10 个工具全部可见 + schema 正确（以 Cursor MCP 面板/工具调用等价确认；未单独跑 Inspector）



### 兼容

- [ ] 未启用 MCP（`mcp_enable=false`）时启动无副作用
- [ ] MCP 服务崩溃不影响 web 主服务（stdio 是独立进程，HTTP 是同进程但 handler 隔离）

---



## 十四、追踪表

> 更新日期：2026-07-31。已合入 `v2.2.1`。T2/T3/T6 合并在同一 commit（stdio 一次注册 10 工具）。


| #   | 任务                               | Commit           | 状态    | 备注                                                    |
| --- | -------------------------------- | ---------------- | ----- | ----------------------------------------------------- |
| T1  | 搜索 FULLTEXT/FTS5 + Provider      | —                | ⏸ 暂缓  | 2026-07-30 方案待再评估；不挡 MCP；过渡期 `search_document` 用 LIKE |
| T2  | MCP stdio · 4 读工具                | `5b6ca51`        | ✅     | 与 T3/T6 同 commit                                      |
| T3  | MCP stdio · 6 写工具（乐观锁 + confirm） | `5b6ca51`        | ✅     | 与 T2/T6 同 commit                                      |
| T4  | `MemberApiToken` + 后台管理页         | `4c4f346`        | ✅     | 迁移 `202607301700`；页面 `/member/api-tokens`             |
| T5  | MCP HTTP + Bearer + 限流           | `2fe3f6a`（以分支为准） | ✅     | `doc mcp --http` + `mcp_enable` 时挂 `/mcp`             |
| T6  | `internal/dto/mcpdto/`           | `5b6ca51`        | ✅     | 随 T2/T3；jsonschema 标签已按 go-sdk 规范修正                   |
| T7  | `docs/mcp-integration.md`        | `67b30d5`        | ✅     | Claude Desktop / Cursor / HTTP 接入 + 工具速查              |
| —   | MCP 体验增强（P0/P1）                  | —                | 📋 规划 | 见 [§十七](#十七后续规划mcp-实测反馈与体验增强)；可收尾小 PR 或 Round 4 T13   |


**合入进度：** T1 ⏭ → T2✅ → T3✅ → T4✅ → T5✅ → T6✅ → T7✅ → **§十七 体验项待做**

---



## 十五、Round 4 前置产物

- MCP 工具已在生产运行 ≥ 2 周，收集使用统计
- 决定：是否上倒排索引服务（bleve / meilisearch），基于 MCP 反馈的搜索质量数据
- `internal/dto/mcpdto/` 结构稳定，可作为 Round 4 Repository/Service 分层的输出契约
- 按 [§十七](#十七后续规划mcp-实测反馈与体验增强) 消化 P0/P1 体验项（或明确推迟到 Round 4 T13）

---



## 十六、参考

- [mcp-integration.md](./mcp-integration.md) — **接入指南（T7）**
- [refactor-roadmap.md §2.1](./refactor-roadmap.md#21-目标一mcp-serverai-接入) — MCP 目标详述
- [refactor-roadmap.md §六 风险 8/9/10/11](./refactor-roadmap.md#六关键风险清单) — MCP 专属风险
- [upstream-mindoc-checklist.md §4.2](./upstream-mindoc-checklist.md) — 上游 MinDoc MCP PR 参考（本项目实现方案已超越）
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) — 官方 Go SDK
- [MCP Go SDK Quick Start](https://go.sdk.modelcontextprotocol.io/quick_start/)
- [MCP 规范](https://modelcontextprotocol.io/)
- [round-4-execution-plan.md T13](./round-4-execution-plan.md#十三附t13--mcp-体验增强可选) — Round 4 可选承接本轮 §十七

---



## 十七、后续规划（MCP 实测反馈与体验增强）

> **背景（2026-07-31）：** 在本地测试环境通过 Cursor 已配置的 `user-doc-remote` MCP，完成读工具冒烟 + 将仓库 `docs/*.md`（16 篇）写入私有项目 `test12`。  
> **总判：** Round 3 的 10 工具已形成「搜 → 读 → 改 → 发布」闭环，**不必为功能完整再堆大量新工具**；后续以**体验缺口与缺陷**为主。  
> **落点：** P0/P1 优先作为 **Round 3 收尾小 PR**；来不及则并入 [Round 4 T13](./round-4-execution-plan.md#十三附t13--mcp-体验增强可选)。P2 与 Book 级工具默认延后。



### 17.1 实测暴露的问题


| 现象                                                     | 影响                                            | 归类                 |
| ------------------------------------------------------ | --------------------------------------------- | ------------------ |
| 大文档（约 30–60KB）一次 `update_document_content` 易被客户端/中间层卡住 | 只能「首段 update + 多次 append」；分块时曾出现约 2–5% 内容缺失风险 | 体验 / 约定            |
| `append_document_content` 无乐观锁、无 `auto_release`        | 连写不安全；分块写完还需再调 `release_document`             | P0 加固              |
| `doc mcp` stdio 启动时 bootstrap 日志打到 **stdout**          | MCP 握手失败（`invalid trailing data`）             | P0 缺陷              |
| `search_document` 仅 SQL `LIKE`                         | AI「找相关文档」质量一般                                 | 仍归 **T1** 评估，不挡收工  |
| 缺「按 identify 查找 / upsert」                              | 批量同步要先 `list_document_tree` 再对表               | P1 可选              |
| 无批量/事务型写                                               | 多篇文档需大量 tool call                             | P2 延后              |
| 无 `create_book` / `update_book`                        | 建项目仍走 Web                                     | **明确暂不做**（见 §17.3） |




### 17.2 优先级与建议项



#### P0 — 建议尽快补（缺陷 / 体验，不算扩工具面）


| #    | 项                                | 说明                                                                                                           |
| ---- | -------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| P0-1 | **stdio 启动静默 stdout**            | 在 `bootstrap` 之前关闭 console logger（或将 bootstrap 日志改 stderr），避免污染 MCP 协议。属真实可用性 bug。                           |
| P0-2 | **加固** `append_document_content` | 增加 `expect_version`（CAS）；支持 `auto_release`；返回最新 `version`（已有则保持）。                                            |
| P0-3 | **长文写入约定写入文档**                   | 在 [mcp-integration.md](./mcp-integration.md) 增补：客户端单次参数体限制、推荐「首段 update → 后续 append → 最后 release」、6100 冲突重试。 |




#### P1 — 按需小增强（仍围绕文档读写）


| #    | 项                                                            | 说明                                                             |
| ---- | ------------------------------------------------------------ | -------------------------------------------------------------- |
| P1-1 | `upsert_document`（或 `create_document` 增加 `if_exists=update`） | 按 `book_id + identify` 创建或覆盖，降低批量同步成本。                         |
| P1-2 | `get_document` **截断选项**                                      | 如 `include_release=false`、`markdown_max_chars`，避免大文档撑爆模型上下文。   |
| P1-3 | `search_document` **增强返回**                                   | 附带 `book_identify` / `doc_identify`，减少再调 `list_document_tree`。 |




#### P2 — 明确延后（不为测试场景过度设计）


| #    | 项                              | 说明                                       |
| ---- | ------------------------------ | ---------------------------------------- |
| P2-1 | `import_documents` / 批量 create | 限流、体积、部分失败语义复杂                           |
| P2-2 | MCP Resources / Prompts        | 对当前 Doc 工作流收益有限                          |
| P2-3 | 附件上传、历史 diff、评论                | Web 已有能力，MCP 暂不镜像                        |
| P2-4 | **T1 FULLTEXT/FTS5**           | 独立评估（见 §六），与「工具是否够用」解耦；质量数据见 Round 4 T11 |
| P2-5 | Book 级 MCP 写工具                 | 见下节决策                                    |




### 17.3 决策：是否增加 `create_book` / `update_book`


| 问题                  | 结论                                                                                                                                               |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Round 3 / 当前阶段要不要加？ | **不要。**                                                                                                                                          |
| 理由                  | ① 路线图 MVP 是「在已有项目里读写文档」，`list_books` 仅用于导航；② 实测批量写文档只需选定已有 book；③ Book 创建/保存涉及 identify、公开私有、封面、空间、转让等，MCP 要么字段不全难用，要么工具过重；④ 误操作成本高于单篇文档（整棵文档树）。 |
| 何时再评估               | 出现稳定需求：如「AI 一键拉起空项目 + 灌文档」、CI/租户初始化脚本化建书。                                                                                                        |
| 若将来做，最小集            | `create_book`：`title` + `identify` + `private` + 可选 `description`（封面/高级项仍走 Web）；`update_book`：仅元数据；**默认不做** `delete_book`（若做须强确认 + 更高权限）。        |


> 已记入 [refactor-roadmap.md §八 决策日志](./refactor-roadmap.md#八决策记录decision-log)（2026-07-31）。



### 17.4 与各轮次的关系

```
Round 3 MVP（T2–T7）✅
    │
    ├─► §十七 P0（收尾小 PR，优先）
    ├─► §十七 P1（可选同 PR 或紧随）
    │
    └─► Round 4
            ├─ T11  搜索后端（LIKE 不够时）
            └─ T13  MCP 体验增强（承接未做完的 P0/P1；不含 Book 写工具）
```



### 17.5 验收建议（P0 收尾时）

- [ ] `doc mcp`（stdio）冷启动后客户端可 `initialize`，stdout 无 beego/bootstrap 杂讯
- [ ] `append_document_content` 在错误 `expect_version` 时返回 6100；`auto_release=true` 后 Web 可读到 HTML
- [ ] `mcp-integration.md` 含「长文分块写入」小节
- [ ] 回归：既有 10 工具 schema 与权限行为不变；**仍无** `create_book` / `update_book`