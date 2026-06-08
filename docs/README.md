# Doc 项目文档

本目录存放与项目维护、上游同步相关的文档。

## 文档索引

| 文档 | 说明 |
|------|------|
| [upstream-mindoc-checklist.md](./upstream-mindoc-checklist.md) | 上游 [mindoc-org/mindoc](https://github.com/mindoc-org/mindoc) 提交跟进清单 |

## 上游关系

- 上游项目：[mindoc-org/mindoc](https://github.com/mindoc-org/mindoc)
- 本仓库：`git.itopcms.com/jackliu/doc`
- 可执行文件：`doc`（上游为 `mindoc`）
- 模块路径：`git.itopcms.com/jackliu/doc`（上游为 `github.com/mindoc-org/mindoc`）

## 同步建议

1. 添加 upstream remote：`git remote add upstream https://github.com/mindoc-org/mindoc.git`
2. 按功能 cherry-pick，避免整库 merge
3. 每次移植后改 import 路径与 CLI 文案（`mindoc` → `doc`）
4. 优先从「阶段 0 基础设施」和「阶段 1 搜索」开始
