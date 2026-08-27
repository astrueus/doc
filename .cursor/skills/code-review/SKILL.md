---
name: code-review
description: >-
  Reviews Go/Beego/MCP code in this doc repository against project-standards.
  Use when reviewing pull requests, examining diffs, git changes, or when the
  user asks for a code review or 代码审查.
---

# Code Review

按 `@project-standards.mdc` 审查。协作约定见 [AGENTS.md](../../../AGENTS.md)。

范围：用户指定，否则当前文件或 `git diff`。分层对错以项目执行标准为准；不要把存量 `model.Find*` 直调在无关改动里全部标成 Critical。

## 反馈

- **Critical**：正确性、安全、未确认的契约/鉴权、新的反向依赖或 SQL 拼接
- **Suggestion**：可维护性、测试不足
- **Nice to have**：命名统一等

```markdown
- **Critical**：MCP 工具新增必填字段未确认（决策确认）。
- **Suggestion**：新查询写在 model 上，建议补 Repo 方法（项目执行标准）。
```
