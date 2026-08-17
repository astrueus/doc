# Doc MCP 接入指南

> Round 3 · AI 助手通过 [MCP](https://modelcontextprotocol.io/) 读写 Doc 文档。  
> 实现基于官方 [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)。  
> 相关执行计划：[round-3-execution-plan.md](./round-1-4/round-3-execution-plan.md)。

本文说明如何把 Doc 接到 **Claude Desktop / Cursor** 等 MCP 客户端，并给出 10 个工具的速查与排错。

---

## 1. 前置条件

1. 已部署含 Round 3 MCP 的 `doc` 二进制，工作目录含 `conf/app.conf`、数据库可用。
2. **HTTP 模式**还需：
   - 执行过 `doc migrate`（创建 `md_member_api_tokens` 表，迁移版本 `202607301700`）
   - 在个人中心生成 API Token（见下）
3. **stdio 模式**需配置里的 `mcp_stdio_member`（或环境变量 `DOC_MCP_STDIO_MEMBER`）对应一个真实账号（默认 `admin`）。

### 1.1 生成 API Token（HTTP 必备）

1. 浏览器登录 Doc。
2. 打开个人中心侧栏 **API Token**，或直接访问：`/member/api-tokens`。
3. 点击 **生成新 Token** → 填写名称（如 `Cursor`）→ Scopes 默认 `read,write` → 可选过期日。
4. 提交后弹窗会显示**明文一次**，格式形如：

   ```text
   doc_AbCdEf...（约 48 位字母数字）
   ```

5. **立即复制保存**。关闭弹窗后无法再查看明文（库中只存 SHA-256）。

撤销：列表中点「撤销」。撤销后该 Token 不可再用于 HTTP MCP（进程内缓存最长约 5 分钟才完全失效）。

> 截图建议：在本机打开 `/member/api-tokens`，对「生成弹窗 + 明文提示」自行截图附到团队 Wiki；仓库不强制入库截图。

### 1.2 相关配置（`conf/app.conf` `[mcp]`）

| 键 | 默认 | 说明 |
|---|---|---|
| `mcp_enable` | `false` | `true` 时 **Web 主进程**挂载 `/mcp`（与 `httpport` 同端口） |
| `mcp_listen` | `127.0.0.1:8280` | 仅 `doc mcp --http` 独立进程监听地址 |
| `mcp_stdio_member` | `admin` | stdio 模式使用的成员账号 |
| `mcp_token_required` | `true` | HTTP 是否强制 Bearer |
| `mcp_rate_limit` | `60` | 每 Token 每分钟读上限；写 ≈ `/2`，删 ≈ `/6` |

也可用环境变量：`DOC_MCP_ENABLE`、`DOC_MCP_LISTEN`、`DOC_MCP_STDIO_MEMBER`、`DOC_MCP_TOKEN_REQUIRED`、`DOC_MCP_RATE_LIMIT`。

---

## 2. stdio 接入 Claude Desktop

适合本机单用户、与 Doc 工作目录同机。

1. 确认能在终端跑通（工作目录指向含 `conf/` 的 Doc Home）：

   ```bash
   /path/to/doc mcp --dir /path/to/doc-home
   ```

   正常会阻塞在 stdio，等待 MCP 客户端；用 Ctrl+C 退出。

2. 编辑 Claude Desktop 配置（macOS 示例：`~/Library/Application Support/Claude/claude_desktop_config.json`；Windows：`%APPDATA%\Claude\claude_desktop_config.json`）：

   ```json
   {
     "mcpServers": {
       "doc": {
         "command": "D:/jcwork/doc/doc.exe",
         "args": ["mcp", "--dir", "D:/jcwork/doc"],
         "env": {
           "DOC_MCP_STDIO_MEMBER": "admin"
         }
       }
     }
   }
   ```

3. 重启 Claude Desktop，对话里应能看到 MCP 工具（如「搜索关于 XXX 的文档」）。

**注意：**

- `command` 用绝对路径；`--dir` 指向配置与数据所在工作目录。
- stdio **不使用** API Token，身份完全由 `DOC_MCP_STDIO_MEMBER` / `mcp_stdio_member` 决定。
- 不要把控制台日志打到 stdout（实现已关闭 console logger，避免污染协议）。

---

## 3. stdio 接入 Cursor

在项目或用户级 MCP 配置中增加（Cursor：Settings → MCP，或仓库 `.cursor/mcp.json`）：

```json
{
  "mcpServers": {
    "doc": {
      "command": "D:/jcwork/doc/doc.exe",
      "args": ["mcp", "--dir", "D:/jcwork/doc"],
      "env": {
        "DOC_MCP_STDIO_MEMBER": "admin"
      }
    }
  }
}
```

保存后在 MCP 面板启用 `doc`，确认 tools 列表有 10 个工具。

---

## 4. HTTP 接入（团队共享推荐）

HTTP 使用 **Streamable HTTP** + Bearer。有两种部署方式：

| 方式 | 何时用 | URL |
|---|---|---|
| **A. 挂在 Web 主进程** | `mcp_enable=true`，只跑 `doc` / `doc web` | `https://docs.example.com/mcp`（同主站端口） |
| **B. 独立进程** | `doc mcp --http` | `http://127.0.0.1:8280/mcp/`（`mcp_listen`） |

生产环境 **必须 HTTPS**（Nginx / Traefik 反代）；Bearer 明文不可走裸 HTTP。监听 `0.0.0.0` 时进程会打告警。

### 4.1 开启 Web 挂载（方式 A）

```ini
[mcp]
mcp_enable = true
mcp_token_required = true
```

重启 Web。日志应出现：`MCP HTTP handler mounted at /mcp`。

### 4.2 Cursor / Claude HTTP 配置示例

```json
{
  "mcpServers": {
    "doc-remote": {
      "url": "https://docs.example.com/mcp",
      "headers": {
        "Authorization": "Bearer doc_请替换为你的明文Token"
      }
    }
  }
}
```

### 4.3 手工 curl 自检

`/mcp` **不是网页**。浏览器直接打开常会看到 `jsonrpc` / `version tag` 类错误，属预期。

正确示例（先 `initialize`）：

```bash
curl -s https://docs.example.com/mcp \
  -H "Authorization: Bearer doc_XXX" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"curl","version":"0.1"}}}'
```

再 `tools/list`：

```bash
curl -s https://docs.example.com/mcp \
  -H "Authorization: Bearer doc_XXX" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

也可用 [MCP Inspector](https://github.com/modelcontextprotocol/inspector) 对接 HTTP 或 stdio。

---

## 5. 10 个工具速查

权限约定（项目角色数值越小权限越高）：

- **读**：公开项目任意人可读；私有项目需至少 Observer。
- **写**：需至少 Editor（含管理员 / 创始人）；系统管理员可写全部。

业务错误常以工具结果 `IsError` + JSON 文本返回，形如：`{"code":6100,"message":"..."}`。

### 5.1 读工具

#### `search_document`

当前搜索实现为 **SQL LIKE**（FULLTEXT 旧方案在 Round 5 **⏸ 暂不实施**，见 [round-5 §一附](./round-5/round-5-execution-plan.md#一附2026-08-03-决策修订)）。

```json
// In
{ "query": "安装说明", "book_id": 0, "limit": 10 }

// Out
{ "total": 1, "items": [{ "id": 12, "book_id": 3, "title": "...", "snippet": "...", "version": 1710000000 }] }
```

#### `get_document`

```json
// In（二选一）
{ "document_id": 12 }
{ "book_identify": "my-book", "doc_identify": "intro" }

// Out
{ "document_id": 12, "book_id": 3, "title": "...", "markdown": "...", "release": "...", "version": 1710000000, "parent_id": 0 }
```

#### `list_books`

```json
// In
{ "page": 1, "page_size": 20 }

// Out
{ "total": 5, "items": [{ "book_id": 3, "identify": "my-book", "title": "...", "private": false, "role_id": 2, "doc_count": 10 }] }
```

#### `list_document_tree`

```json
// In
{ "book_id": 3 }
// 或 { "book_identify": "my-book" }

// Out
{ "book_id": 3, "book_identify": "my-book", "nodes": [{ "document_id": 12, "parent_id": 0, "title": "...", "identify": "intro", "version": 1710000000, "order_sort": 1 }] }
```

### 5.2 写工具

#### `create_document`

```json
// In
{ "book_id": 3, "parent_id": 0, "title": "新文档", "identify": "", "markdown": "# Hello" }

// Out
{ "document_id": 99, "version": 1710000001 }
```

#### `update_document_content`（乐观锁）

先 `get_document` 取 `version`，再带 `expect_version` 覆盖写 Markdown。

```json
// In
{ "document_id": 99, "expect_version": 1710000001, "markdown": "# 更新后", "auto_release": false }

// Out
{ "document_id": 99, "version": 1710000100, "message": "updated" }
```

#### `append_document_content`（乐观锁）

先 `get_document` 取 `version`，再带 **必填** `expect_version` 追加 Markdown。错误 version → `6100`。

```json
// In
{ "document_id": 99, "expect_version": 1710000100, "markdown_append": "\n\n追加段落", "auto_release": false }

// Out
{ "document_id": 99, "version": 1710000200 }
```

> **Breaking（T13 P0）：** `expect_version` 现为必填，与 `update_document_content` 一致；旧客户端不传会失败。

#### `update_document_meta`

可选字段用指针语义：只传要改的键。

```json
{ "document_id": 99, "title": "新标题", "order_sort": 2 }
```

#### `release_document`

将 Markdown 渲染为 HTML 写入 `release`（单篇或整书）。

```json
{ "document_id": 99 }
// 或 { "book_id": 3 }
```

#### `delete_document`

必须 `confirm: true`；删前写入 `document_history` 快照（`action=mcp_delete`）。

```json
{ "document_id": 99, "confirm": true }
// Out: { "deleted_count": 1, "snapshot_history_id": 55 }
```

### 5.3 常见错误码

| Code | 常量 | 含义 | 处理 |
|---|---|---|---|
| 6002 | `CodeInvalidParam` | 参数缺失/非法 | 检查工具入参 |
| 6003 | `CodeUnauthorized` | 未登录 / Token 无效 | 检查 Bearer 或 stdio member |
| 6004 | `CodeForbidden` | 无读/写权限 | 调整项目成员角色 |
| 6005 | `CodeNotFound` | 文档/项目不存在 | 核对 id / identify |
| 6100 | `CodeVersionConflict` | 乐观锁冲突 | 重新 `get_document` 后合并再写 |
| 6200 | `CodeRateLimited` | 业务限流码（预留） | 降低频率 |
| HTTP 401 | — | Bearer 失败 | 换 Token / 检查 `doc_` 前缀 |
| HTTP 429 | — | 网关级限流 | 等待或调大 `mcp_rate_limit` |

工具错误体示例：

```json
{ "code": 6100, "message": "version conflict: please refetch with get_document and retry" }
```

---

## 6. 常见问题

### 遇到 `VERSION_CONFLICT`（6100）怎么办？

1. 再调 `get_document` 拿最新 `version` 与 `markdown`。
2. 在客户端做 diff/merge（若需要）。
3. 用新的 `expect_version` 再调 `update_document_content` 或 `append_document_content`。

### 长文怎么分块写入？

客户端 / 中间层对单次工具参数体有限制：约 **30–60KB** 的 Markdown 一次 `update_document_content` 容易卡住或截断。推荐：

1. **首段**：`update_document_content`（带 `expect_version`）写入开头内容。
2. **后续**：多次 `append_document_content`，每次带上一步返回的新 `version` 作为 `expect_version`；单块建议远小于 30KB。
3. **收尾**：最后一次 `auto_release: true`，或单独调 `release_document`，让 Web 可读到 HTML。

若中途 `6100`：重新 `get_document`，确认已写入内容后从断点继续 append（勿用过期 version 盲写）。

### 遇到 HTTP 429 / rate limit 怎么办？

- 默认每 Token：读 60/min、写 30/min、删 10/min（由 `mcp_rate_limit` 派生）。
- 降低调用频率，或临时提高 `mcp_rate_limit` 后重启。
- 不要多客户端共用同一 Token 打满限额。

### stdio vs HTTP 怎么选？

| | stdio | HTTP |
|---|---|---|
| 场景 | 本机 Claude / Cursor | 团队共享、远程服务 |
| 身份 | 配置里的 member 账号 | API Token → 对应用户 |
| 运维 | 客户端拉起子进程 | Web `/mcp` 或 `doc mcp --http` |
| 安全 | 本机文件权限 | **必须 HTTPS** + Token 轮转 |

### 如何撤销 Token？

`/member/api-tokens` → 对应行「撤销」。撤销后明文立即失效于库；若刚用过，缓存最多约 5 分钟内仍可能放行（撤销接口会主动删缓存）。

### `doc mcp` stdio 握手失败 / `invalid trailing data`？

stdio 模式 stdout 专供 MCP JSON-RPC。已在 bootstrap 前关闭 console logger，并跳过会 `fmt.Println` 的 preflight。若仍失败：确认使用含 T13 P0 的二进制；勿在 MCP 进程外把日志重定向到同一 stdout。

### `Forbidden: invalid Host header "example.com"`？

官方 go-sdk 默认拒绝「请求落在 127.0.0.1 但 Host 非 localhost」的请求（防 DNS 重绑定）。Nginx 反代到本机回环地址时会命中。本仓库已在 `StreamableHTTPOptions` 中设置 `DisableLocalhostProtection=true`（HTTP MCP 另有 Bearer）。若仍见此错误，确认已部署含该修复的版本并重启进程。

### `malformed payload: invalid message version tag ""; expected "2.0"`？

请求体不是合法 JSON-RPC 2.0（常见：空 body、浏览器 GET、缺少 `"jsonrpc":"2.0"`）。用上面的 curl 或 MCP 客户端重试。

### 搜索结果和网页不太一样？

T1 FULLTEXT 已移交 Round 5 后 **⏸ 暂不实施**（等搜索方案重定义），`search_document` 现为 LIKE，与 Web 全局搜索策略接近但不保证排序完全一致。

---

## 7. 安全建议

1. **永远不要**把 `doc_…` Token 提交到 git、贴到公开 Issue/截图。
2. 每个客户端单独 Token，名称写清用途；只开需要的 scopes（当前校验以项目角色为主，scopes 预留扩展）。
3. 定期轮转：生成新 Token → 更新客户端配置 → 撤销旧 Token。
4. 生产 HTTP 必须 TLS；反代后注意透传 `Authorization`。
5. stdio 的 `DOC_MCP_STDIO_MEMBER` 尽量用权限适中的专用账号，避免长期用超级管理员。
6. 日志脱敏：明文匹配 `doc_[A-Za-z0-9]+` 即可打码。

---

## 8. 相关命令速查

```bash
./doc migrate              # 含 member_api_tokens 等迁移
./doc mcp                  # stdio MCP
./doc mcp --http           # 独立 HTTP MCP（mcp_listen）
./doc                      # Web；mcp_enable=true 时同端口提供 /mcp
```

更多实现细节见 [round-3-execution-plan.md](./round-1-4/round-3-execution-plan.md)。  
后续体验增强与「是否做 Book 写工具」的规划见 [round-3-execution-plan.md §十七](./round-1-4/round-3-execution-plan.md#十七后续规划mcp-实测反馈与体验增强)。  
**进度：** §十七 **P0 已合入**（Round 4 T13）；**P1 📦 已移交 Round 5 T5**；Book 写工具当前不做。整体轮次进度见 [docs/README.md](./README.md) / [round-5-execution-plan.md](./round-5/round-5-execution-plan.md)。
