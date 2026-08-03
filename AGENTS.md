# AI / Agent 协作约定

本文件是仓库级约定，供 Cursor、Claude Code、Codex、Copilot 等工具与人工协作者共同遵循。工具专用入口（如 `.cursor/rules/`、`CLAUDE.md`）应指向本文，避免多处全文复制后漂移。

## 语言：简体中文

本仓库面向中文协作。写**文档、注释、提交说明**以及与用户沟通时，默认使用**简体中文**。

### 必须用中文

- 用户可见说明、`docs/**`、CHANGELOG、执行计划等 Markdown
- Go / 前端等源码中的**包注释、导出符号注释、关键说明**
- `git commit` 的标题与正文（说明「为什么」）

### 可以保留英文

- 标识符、包路径、配置键、协议字段、第三方库名
- 日志里的稳定机器字段名（如 `level`、`caller`）
- 对外协议 / API 若已规定英文，从其规定

### 示例

```text
# ❌ 提交说明
fix(round4): polish zap console UX and session login filter

# ✅ 提交说明
fix(round4): 修复日志控制台体验与 Session 登录过滤器误判
```

```go
// ❌
// MemberIDFromSession extracts member id from session value.

// ✅
// MemberIDFromSession 从 session 值中解析成员 ID。
```

新增或改动注释 / 文档时：不要为了「国际化注释」改回英文；与周围已有中文风格保持一致。

## 相关入口

| 路径 | 用途 |
|------|------|
| `AGENTS.md`（本文） | 跨工具权威约定 |
| `CLAUDE.md` | Claude Code 等入口，指向本文 |
| `.cursor/rules/chinese-communication.mdc` | Cursor 常驻规则，指向本文 |
