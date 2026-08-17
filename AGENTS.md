# AI / Agent 协作约定

本文件是仓库级约定，供 Cursor、Claude Code、Codex、Copilot 等工具与人工协作者共同遵循。工具专用入口（如 `.cursor/rules/`、`CLAUDE.md`）应指向本文，避免多处全文复制后漂移。

## 语言：简体中文

本仓库面向中文协作。写**文档、注释、提交说明**以及与用户沟通时，默认使用**简体中文**。

### 必须用中文

- 用户可见说明、`docs/**`、`.docs/**`、CHANGELOG、执行计划等 Markdown
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

## 文档输出目录

生成、整理、导出 Markdown/说明类文档时，按以下顺序决定目标目录（相对仓库根）：

1. **仅存在 `.docs`** → 写入 `.docs/`
2. **仅存在 `docs`** → 写入 `docs/`
3. **两者都不存在** → **自动创建 `.docs/`**，再写入其中
4. **两者都存在** → 询问用户放到哪个目录（提问方式见「决策确认」），**等用户确认后再写文件**

### 约束

- 用户已明确指定路径时，以用户指定为准
- 不要把业务文档默认写到项目根目录或随意子目录
- 目录检测以仓库根为准（存在即为目录存在，不必非空）

### 篇幅与拆分

单篇可能过长、难以阅读或评审时，在已选定的输出根目录下**按主题拆成子目录 + 多篇 Markdown**，不要写成一篇巨文。

- 按独立主题 / 章节拆分，不要无意义切成碎片
- 子目录内用 `README.md`（或索引）串阅读顺序与文件职责
- 用户指定了单一路径且明确要求不分文件 → 以用户为准
- 是否拆分、怎么拆有多种合理解法且影响后续维护 → 先按「决策确认」询问

本仓库当前仅有 `docs/`，因此默认写入 `docs/`。

## 决策确认

影响对外契约、鉴权、数据或生产配置时先确认；小改动直接做。  
提问：优先用可点选的选项工具（Cursor 为 AskQuestion）；不可用则改用同类工具；都没有则用文本列出——**问题用序号，选项用 A/B**（写清差异，必要时附少量代码），等确认后再做。方案对比不清楚时，提供「先对照代码/示例再决定」选项。  
冲突裁决与提问细则见 `.cursor/rules/decision-confirm.mdc`。

## 相关入口

| 路径 | 用途 |
|------|------|
| `AGENTS.md`（本文） | 跨工具权威约定 |
| `CLAUDE.md` | Claude Code 等入口，指向本文 |
| `.cursor/rules/chinese-communication.mdc` | 中文表达（Cursor 摘要） |
| `.cursor/rules/doc-output-location.mdc` | 文档输出目录（Cursor 摘要） |
| `.cursor/rules/decision-confirm.mdc` | 决策确认（Cursor 全文） |
