# 上游 MinDoc 提交跟进清单

本文档基于 [mindoc-org/mindoc](https://github.com/mindoc-org/mindoc) 近期提交历史，对照当前 `doc` 项目整理，用于评估哪些上游改动值得跟进。

**文档生成依据：** 上游 GitHub 提交记录 + 本仓库代码现状（2026-06）。

---

## 一、当前基线

| 维度 | 当前 `doc` 状态 |
|------|----------------|
| 搜索 | SQL `LIKE`，见 `models/DocumentSearchResult.go` |
| 编辑器 | editor.md + wangEditor，**无** Cherry / Draw.io |
| 系统角色 | 3 种（超管/管理员/普通），见 `conf/enumerate.go` |
| 数据库 | MySQL + SQLite，**无** PostgreSQL |
| CLI | `install` / `version` / `password` / `migrate` / `service`，**无** `reindex` |
| MCP | 无 |
| 分词/索引 | 无 `lib/jieba/`、`utils/segmenter/` |
| 可执行名 | `doc`（上游为 `mindoc`） |
| 模块路径 | `git.itopcms.com/jackliu/doc`（上游为 `github.com/mindoc-org/mindoc`） |

---

## 二、推荐实施路线图

```text
第 1 周（稳）
  ├─ #1036 WorkingDirectory
  ├─ #1031 资源路径（若准备上搜索）
  ├─ #1029 / #1023 时区
  └─ #1026 / #1034 前端快捷键（可选）

第 2~3 周（核心）
  ├─ #1027 倒排索引 + jieba + 新表
  ├─ #1034 搜索重构 + doc reindex
  └─ 搜索回归测试

第 4 周（体验）
  ├─ #1035 editor.md / mermaid / KaTeX
  └─ #1036 编辑/预览 URL

按需并行
  ├─ #992 只读角色（有权限需求）
  ├─ #1010 MCP（有 AI 需求，依赖搜索）
  ├─ #792 PG（有 PG 需求）
  └─ Cherry 全家桶（单独大项目）
```

---

## 三、阶段 0：基础设施（建议最先做）

### 0.1 WorkingDirectory 与资源路径

| 项 | 内容 |
|----|------|
| 上游 PR | [#1036](https://github.com/mindoc-org/mindoc/pull/1036)、[#1031](https://github.com/mindoc-org/mindoc/pull/1031) |
| 日期 | 2026-03-23 ~ 2026-04-01 |
| 问题 | 单文件/非常规目录部署时找不到词典、静态资源；编辑/预览 URL 不正确 |
| 上游改动 | `conf/enumerate.go`、`commands/command.go` 中 WorkingDirectory 判定；资源随发行包分发 |
| 当前项目 | `commands/command.go` 仅用 `filepath.Dir(os.Args[0])` 设工作目录 |
| 建议动作 | 1. 对照 #1036 改 WorkingDirectory 逻辑<br>2. 若后续做搜索，同步 #1031 的 `lib/jieba/` 打包方式 |
| 工作量 | 小（0.5~1 天） |
| 前置依赖 | 无 |
| 验证 | 从非项目根目录启动 `doc`；`doc version`；后续 `doc reindex` 能找到词典 |

---

### 0.2 时区修复

| 项 | 内容 |
|----|------|
| 上游 PR | [#1029](https://github.com/mindoc-org/mindoc/pull/1029)、[#1023](https://github.com/mindoc-org/mindoc/pull/1023) |
| 日期 | 2026-02-27 ~ 2026-03-23 |
| 问题 | Blog 时间显示与本地不一致；Termux 等环境时区异常 |
| 上游改动 | `models/Blog.go`、时间格式化相关 |
| 当前项目 | 有 `orm.DefaultTimeLoc` 和 `ZONEINFO` 说明，Blog 时间逻辑可能仍偏旧 |
| 建议动作 | cherry-pick 时间处理；核对 `models/Blog.go`、模板中的 `date_format` |
| 工作量 | 小 |
| 验证 | Blog 列表创建/修改时间与系统时区一致 |

---

### 0.3 大文件上传

| 项 | 内容 |
|----|------|
| 上游 Commit | [4cdafc8](https://github.com/mindoc-org/mindoc/commit/4cdafc8)（2023-05） |
| 问题 | 上传 >1GB 报错 |
| 建议动作 | 对照 `DocumentController` / `BlogController` 上传大小解析逻辑 |
| 工作量 | 极小 |
| 是否必跟 | 有超大附件上传需求时建议跟 |

---

## 四、阶段 1：搜索体系（收益最高，改动最大）

### 1.1 倒排索引 + TF-IDF 初版

| 项 | 内容 |
|----|------|
| 上游 PR | [#1027](https://github.com/mindoc-org/mindoc/pull/1027) |
| 日期 | 2026-03-18 |
| 核心能力 | 倒排索引表、jieba 分词、TF-IDF 打分、文档/Blog 索引维护 |

**上游新增/大改文件：**

| 类别 | 文件 |
|------|------|
| 模型 | `models/ContentReverseIndex.go`（约 498 行，新表） |
| 分词 | `utils/segmenter/segmenter.go` |
| 词典 | `lib/jieba/*.utf8`（约 60 万行词典文件） |
| 控制器 | `controllers/SearchController.go`（`IndexV2`、`SearchV2`、`PerformSearchV2Raw`） |
| 文档保存 | `models/DocumentModel.go` 保存/删除时建索引 |
| Blog | `models/Blog.go` 保存/删除时建索引 |
| 删项目 | `models/BookModel.go` 级联删索引 |
| 启动 | `commands/command.go` 注册模型 + `InitializeMissingIndexes()` |

| 项 | 内容 |
|----|------|
| 依赖 | `github.com/yanyiwu/gojieba` |
| 当前项目 | 仅有 `models/DocumentSearchResult.go` 的 SQL 搜索 |
| 工作量 | 大（3~5 天，含迁移与测试） |
| 数据库 | 需新表（如 `md_content_reverse_index`，以实际上游为准） |

**搜索移植检查表：**

- [ ] 新增 `lib/jieba/` 词典目录
- [ ] 新增 `utils/segmenter/`
- [ ] 新增 `models/ContentReverseIndex.go`
- [ ] 修改 `models/DocumentModel.go`（保存/删除/发布时更新索引）
- [ ] 修改 `models/Blog.go`
- [ ] 修改 `models/BookModel.go`（删书时清索引）
- [ ] 重写/扩展 `controllers/SearchController.go`
- [ ] `commands/command.go` 注册模型与启动初始化
- [ ] 新增 `commands/reindex.go`（命令名改为 `doc reindex`）
- [ ] `go.mod` 增加 gojieba
- [ ] `conf/app.conf` 可选：是否启用倒排索引搜索
- [ ] `routers/router.go` 增加 SearchV2 路由（若上游有）
- [ ] 数据库迁移脚本

---

### 1.2 搜索重构 + reindex

| 项 | 内容 |
|----|------|
| 上游 PR | [#1034](https://github.com/mindoc-org/mindoc/pull/1034) |
| 日期 | 2026-03-31 |
| 在 #1027 基础上增强 | 见下表 |

| # | 增强项 |
|---|--------|
| 1 | 先取候选、统一加权排序后再分页（避免高相关结果被截断） |
| 2 | 标题命中、正文精确匹配、查询覆盖率 boost |
| 3 | 批量回表 + 统一权限过滤 |
| 4 | 空内容占位索引 |
| 5 | 新增 CLI：`doc reindex`（上游为 `mindoc reindex`） |
| 6 | 技术词白名单（`linux`、`grep` 等不被停用词过滤） |
| 7 | SQL 搜索增加标题相关性排序（兜底） |

| 项 | 内容 |
|----|------|
| 上游改动文件 | `models/ContentReverseIndex.go`、`utils/segmenter/`、`commands/reindex.go`（或同类）、`SearchController.go` |
| 建议动作 | 在 #1027 之后合并 #1034，不要只跟一半 |
| 工作量 | 中（1~2 天，依赖 #1027） |
| 验证 | `./doc reindex`；搜 `linux`、技术术语；大库分页结果排序合理 |

---

### 1.3 全局搜索前端体验

| 项 | 内容 |
|----|------|
| 上游 PR | [#1026](https://github.com/mindoc-org/mindoc/pull/1026)、#1034 部分 |
| 问题 | 焦点在输入框时仍触发全局快捷键；Ctrl+F 被拦截；ESC 关闭搜索异常 |
| 上游改动 | 前端 JS（全局快捷键检测）、阅读页搜索面板 |
| 当前项目 | `views/search/index.tpl` 存在，未见对应快捷键逻辑 |
| 工作量 | 小~中 |
| 可独立跟进 | 是（不依赖倒排索引，但常与搜索改版一起出现） |

---

## 五、阶段 2：编辑器与阅读体验

### 2.1 Editor.md + Mermaid + KaTeX

| 项 | 内容 |
|----|------|
| 上游 PR | [#1035](https://github.com/mindoc-org/mindoc/pull/1035) |
| 日期 | 2026-03-31 |
| 内容 | editor.md 升级；mermaid 8.13→10.9；KaTeX auto-render；#909/#948 修复 |
| 上游改动 | `static/editor.md/**`、`static/js/markdown.js`、`views/document/markdown_edit_template.tpl` |
| 当前项目 | 有 `static/js/markdown.js`、`views/document/markdown_edit_template.tpl`，KaTeX 无 auto-render |
| 工作量 | 中（1~2 天，需回归公式/流程图/序列图） |
| 风险 | 静态资源体积大，与现有 wangEditor 并存需分别测 |
| 可独立跟进 | 是 |

---

### 2.2 Cherry Markdown 全家桶

| 项 | 内容 |
|----|------|
| 上游 Commit | [21fe4b6](https://github.com/mindoc-org/mindoc/commit/21fe4b631a23f3b2ff1ea913b1bb8e34e8b9ddc1) 及后续（#1028、#1034） |
| 时间 | 2023-07 起，持续维护至 2026 |
| 内容 | 第二套 Markdown 编辑器；Draw.io；代码块复制；阅读页目录滚动；编辑目标保持 |
| 上游新增 | `static/cherry-markdown/**`、新模板、项目级编辑器选择 |
| 当前项目 | **完全没有** Cherry 相关代码 |
| 工作量 | 很大（5~10 天+） |
| 是否必跟 | 仅当需要 Cherry/Draw.io 时 |
| 可拆子项 | 仅跟 #1028 阅读页目录滚动 CSS（若仍用 editor.md） |

---

### 2.3 阅读页 URL 与编辑跳转

| 项 | 内容 |
|----|------|
| 上游 PR | [#1036](https://github.com/mindoc-org/mindoc/pull/1036)、#1034 |
| 内容 | 浏览器地址栏与预览链接更新；从阅读页进入编辑时保留目标文档 |
| 建议动作 | 与 #1036、编辑器选型一起评估 |
| 工作量 | 小~中 |

---

### 2.4 评论头像

| 项 | 内容 |
|----|------|
| 上游 PR | [#977](https://github.com/mindoc-org/mindoc/pull/977) |
| 日期 | 2024-08 |
| 内容 | 评论列表显示用户头像 |
| 工作量 | 极小 |
| 是否必跟 | 低优先级 UI 优化 |

---

## 六、阶段 3：权限与账号体系

### 3.1 只读用户角色

| 项 | 内容 |
|----|------|
| 上游 PR | [#992](https://github.com/mindoc-org/mindoc/pull/992) |
| 日期 | 2024-12 |
| 内容 | 系统级只读用户：不能创建/编辑，只能做项目观察者 |

**上游改动范围：**

| 类别 | 内容 |
|------|------|
| 枚举 | `conf/enumerate.go` 增加 `MemberReadOnlyRole` |
| 管理 | `controllers/ManagerController.go` 用户创建/编辑 |
| 权限 | 各 Controller 写操作拦截 |
| 模板 | 用户管理页、角色下拉 |
| CSS | `markdown.preview.css`、Cherry 阅读宽度（TOC 隐藏时 100%） |
| PDF | 发布者默认取公司名称 |

| 项 | 内容 |
|----|------|
| 当前项目 | `ManagerController` 仅允许 `MemberAdminRole` / `MemberGeneralRole` |
| 工作量 | 中（2~3 天） |
| 是否必跟 | 有「只读账号」需求时建议跟 |

---

### 3.2 OAuth2 登录重写

| 项 | 内容 |
|----|------|
| 上游 PR | [#851](https://github.com/mindoc-org/mindoc/pull/851) |
| 日期 | 2023-04 |
| 内容 | 企业微信、钉钉登录逻辑重写 |
| 当前项目 | 有钉钉（`AccountController` + `utils/dingtalk`），**无**企业微信 |
| 工作量 | 中~大 |
| 是否必跟 | 仅当 SSO 有问题或要加企微时 |

---

### 3.3 LDAPS

| 项 | 内容 |
|----|------|
| 上游 Commit | [a2202f8](https://github.com/mindoc-org/mindoc/commit/a2202f887888e481cd6bf43c5972408618c36710) |
| 内容 | LDAP over TLS |
| 当前项目 | `utils/ldap.go`、`models/Member.go`，未见 ldaps |
| 工作量 | 小 |
| 是否必跟 | 有 LDAPS 需求时 |

---

## 七、阶段 4：国际化与 MCP

### 4.1 i18n 可配置 + 俄语

| 项 | 内容 |
|----|------|
| 上游 PR | [#987](https://github.com/mindoc-org/mindoc/pull/987)、[#1013](https://github.com/mindoc-org/mindoc/pull/1013) |
| 日期 | 2024-11 ~ 2025-10 |
| 内容 | 语言列表可配置；新增俄语；修复语言设置问题 |
| 工作量 | 中 |
| 是否必跟 | 多语言/可配置语言时需要 |

---

### 4.2 MCP Server

| 项 | 内容 |
|----|------|
| 上游 PR | [#1010](https://github.com/mindoc-org/mindoc/pull/1010)、[#1012](https://github.com/mindoc-org/mindoc/pull/1012) |
| 日期 | 2025-09 ~ 2025-10 |
| 内容 | MCP 文档全局检索；配置文档 |
| 上游新增 | `mcp/` 包、`mark3labs/mcp-go` 依赖 |
| 与搜索关系 | #1027 后 MCP 改用 `PerformSearchV2Raw`（倒排索引） |
| 建议动作 | 若要 AI 集成，建议在搜索体系完成后移植 |
| 工作量 | 中（2~3 天，强依赖搜索） |

---

## 八、阶段 5：数据库与部署

### 5.1 PostgreSQL 支持

| 项 | 内容 |
|----|------|
| 上游 PR | [#792](https://github.com/mindoc-org/mindoc/pull/792)、[#986](https://github.com/mindoc-org/mindoc/pull/986) |
| 日期 | 2023-04 ~ 2024-11 |
| 内容 | 增加 PostgreSQL；修复 `LIMIT ?,?` 等语法 |
| 当前项目 | 仅 mysql/sqlite；`DocumentSearchResult.go` 使用 `LIMIT ?, ?` |
| 工作量 | 中~大 |
| 是否必跟 | 仅用 MySQL/SQLite 可跳过 |

---

### 5.2 Docker / PDF 导出

| 项 | 内容 |
|----|------|
| 上游 PR | [#994](https://github.com/mindoc-org/mindoc/pull/994)、[#1001](https://github.com/mindoc-org/mindoc/pull/1001) |
| 内容 | Dockerfile 升级、Calibre 版本、PDF 导出依赖 |
| 当前项目 | 有 `Dockerfile`（Go 1.25 + Ubuntu focal） |
| 是否必跟 | 用 Docker 部署或 PDF 导出有问题时 |

---

## 九、阶段 6：历史 Bug 修复（按需 cherry-pick）

| PR/Commit | 日期 | 内容 | 建议 |
|-----------|------|------|------|
| [#850](https://github.com/mindoc-org/mindoc/issues/850) / [c8f7a2a](https://github.com/mindoc-org/mindoc/commit/c8f7a2a54478388c938612a796f45a4440ac7a76) | 2023-06 | 附件不能下载、非 Markdown 文档附件问题 | 对照 `DocumentController` 是否已修 |
| [#849](https://github.com/mindoc-org/mindoc/issues/849) / [452577c](https://github.com/mindoc-org/mindoc/commit/452577ca3d015454b9bb3aefe4e5736a14eff1f2) | 2023-06 | 加密文章访问权限 | 若有加密文档功能则核对 |
| [#1024](https://github.com/mindoc-org/mindoc/issues/1024) / [bec1763](https://github.com/mindoc-org/mindoc/commit/bec17630274e44ce431e54871a892603bf9c32e6) | 2026-03 | 未公开 issue 修复 | 需看具体 diff 再定 |
| [d7547a8](https://github.com/mindoc-org/mindoc/commit/d7547a85df16d236b86dfcdcefbfb65de8ea0951) | 2023-06 | MTE 表格编辑增强 | 依赖 editor.md，可选 |
| [cc34c63](https://github.com/mindoc-org/mindoc/commit/cc34c6309e975611da9247895bf02601d2ce0922) | 2023-07 | Draw.io 支持 | 依赖 Cherry，可选 |

---

## 十、不建议跟或低优先级

| 项 | 原因 |
|----|------|
| [#1019](https://github.com/mindoc-org/mindoc/pull/1019) 官网链接 | 品牌不同，保持 `git.itopcms.com/jackliu/doc` |
| [30a1e87](https://github.com/mindoc-org/mindoc/commit/30a1e87068052c4f1c0fe77f11e001b2843fed6a) Docker 镜像版本号 | 文档类，非代码 |
| [b9f1381](https://github.com/mindoc-org/mindoc/commit/b9f13815e81421d8bec677b36d850df5c130615b) login.tpl 小改 | 需对比本仓库登录页 |
| dependabot 依赖升级 | 本仓库已 Go 1.25 + Beego v2，按需单独升 |

---

## 十一、每次 cherry-pick 的通用适配清单

- [ ] import 路径：`github.com/mindoc-org/mindoc` → `git.itopcms.com/jackliu/doc`
- [ ] CLI 帮助文案：`mindoc` → `doc`
- [ ] 表前缀 `md_` 是否一致（使用 `GetDatabasePrefix()`，一般兼容）
- [ ] Beego v2 API 差异（本仓库已迁 v2，比老 mindoc 冲突少）
- [ ] 模板中的 MinDoc 文案是否改为 Doc（按产品要求）
- [ ] `conf/app.conf` 新配置项是否合并进 `app.conf.example`
- [ ] 数据库迁移是否写入 `commands/migrate` 或 `install`

---

## 十二、快速决策表

| 你的需求 | 建议跟的 PR |
|----------|-------------|
| 搜索慢、结果差 | #1027 + #1034 |
| 部署/路径问题 | #1036 + #1031 |
| 公式/流程图问题 | #1035 |
| 只读账号 | #992 |
| AI 检索文档 | #1010（先完成搜索） |
| 钉钉/企微登录 | #851 |
| PostgreSQL | #792 + #986 |
| 新编辑器体验 | Cherry 大套件（2023 起） |
| 小修小补 | #1023、#1026、#1029、附件/加密等 |

---

## 十三、上游近期提交速查（2024-2026）

| 日期 | PR/Commit | 摘要 | 优先级 |
|------|-----------|------|--------|
| 2026-04 | #1036 | WorkingDirectory + 编辑/预览 URL | 高 |
| 2026-03 | #1035 | Editor.md / mermaid / KaTeX 升级 | 中 |
| 2026-03 | #1034 | 搜索重构 + reindex + 前端体验 | 高 |
| 2026-03 | #1031 | 二进制资源路径 / jieba 词典 | 高（配合搜索） |
| 2026-03 | #1029 | Termux 时区 | 低 |
| 2026-03 | #1028 | Cherry 阅读页目录滚动 | 低（需 Cherry） |
| 2026-03 | #1027 | 倒排索引 + TF-IDF | 高 |
| 2026-03 | #1026 | 全局快捷键输入框检测 | 中 |
| 2026-02 | #1023 | Blog 时间显示 | 低 |
| 2025-10 | #1013 | 语言设置修复 | 按需 |
| 2025-10 | #1012 | MCP 配置文档 | 按需 |
| 2025-09 | #1010 | MCP Server + i18n | 按需 |
| 2025-03~04 | #994/#1001 | Docker / PDF | 按需 |
| 2024-12 | #992 | 只读用户角色 | 按需 |
| 2024-12 | #989 | 语言修复 | 按需 |
| 2024-11 | #987 | i18n 可配置 + 俄语 | 按需 |
| 2024-11 | #986 | PostgreSQL SQL 兼容 | 按需 |
| 2024-08 | #977 | 评论头像 | 低 |
