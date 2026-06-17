# 上游 MinDoc 提交跟进清单

本文档基于 [mindoc-org/mindoc](https://github.com/mindoc-org/mindoc) 近期提交历史，对照当前 `doc` 项目整理，用于评估哪些上游改动值得跟进。

**文档生成依据：** 上游 GitHub 提交记录 + 本仓库代码现状（2026-06）。

---

## 一、当前基线


| 维度    | 当前 `doc` 状态                                                                |
| ----- | -------------------------------------------------------------------------- |
| 搜索    | SQL `LIKE`，见 `models/DocumentSearchResult.go`                              |
| 编辑器   | editor.md + wangEditor，**无** Cherry / Draw.io                              |
| 系统角色  | 3 种（超管/管理员/普通），见 `conf/enumerate.go`                                       |
| 数据库   | MySQL + SQLite，**无** PostgreSQL                                            |
| CLI   | `install` / `version` / `password` / `migrate` / `service`，**无** `reindex` |
| MCP   | 无                                                                          |
| 分词/索引 | 无 `lib/jieba/`、`utils/segmenter/`                                          |
| 可执行名  | `doc`（上游为 `mindoc`）                                                        |
| 模块路径  | `git.itopcms.com/jackliu/doc`（上游为 `github.com/mindoc-org/mindoc`）          |


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


| 项     | 内容                                                                                                              |
| ----- | --------------------------------------------------------------------------------------------------------------- |
| 上游 PR | [#1036](https://github.com/mindoc-org/mindoc/pull/1036)、[#1031](https://github.com/mindoc-org/mindoc/pull/1031) |
| 日期    | 2026-03-23 ~ 2026-04-01                                                                                         |
| 问题    | 单文件/非常规目录部署时找不到词典、静态资源；编辑/预览 URL 不正确                                                                            |
| 上游改动  | `conf/enumerate.go`、`commands/command.go` 中 WorkingDirectory 判定；资源随发行包分发                                        |
| 当前项目  | `commands/command.go` 仅用 `filepath.Dir(os.Args[0])` 设工作目录                                                       |
| 建议动作  | 1. 对照 #1036 改 WorkingDirectory 逻辑 2. 若后续做搜索，同步 #1031 的 `lib/jieba/` 打包方式                                        |
| 工作量   | 小（0.5~1 天）                                                                                                      |
| 前置依赖  | 无                                                                                                               |
| 验证    | 从非项目根目录启动 `doc`；`doc version`；后续 `doc reindex` 能找到词典                                                            |


**本地最小升级方案**

- **核心思路**：保留现有 `filepath.Dir(os.Args[0])` 行为，仅在 `conf.WorkingDir`/`commands.ResolveWorkingDir` 之类的入口增加优先级：`DOC_HOME` 环境变量 > 进程目录 > `os.Getwd()` 回退。不引入上游的"嵌入资源 + 自展开"复杂逻辑。
- **改动文件**：`commands/command.go`（约 20 行）、`conf/enumerate.go` 中 `WorkingDir()` 函数（加 env 探测）、`conf/app.conf.example` 加一行注释说明 `DOC_HOME`。
- **放弃的上游能力**：systemd 服务模式下的工作目录自动判定、上游把 `lib/jieba` 等资源打进二进制的方案。
- **工作量**：0.5 天。
- **验证**：`DOC_HOME=/tmp/doc ./doc`、从 `/` 目录启动、Windows 服务方式启动均能读到 `static/`、`views/`、`conf/`。
- **升级路径**：未来若上 #1031（资源随包分发），只需在该入口新增"嵌入资源回退"分支即可，不破坏 env 优先级。

---

### 0.2 时区修复


| 项     | 内容                                                                                                              |
| ----- | --------------------------------------------------------------------------------------------------------------- |
| 上游 PR | [#1029](https://github.com/mindoc-org/mindoc/pull/1029)、[#1023](https://github.com/mindoc-org/mindoc/pull/1023) |
| 日期    | 2026-02-27 ~ 2026-03-23                                                                                         |
| 问题    | Blog 时间显示与本地不一致；Termux 等环境时区异常                                                                                  |
| 上游改动  | `models/Blog.go`、时间格式化相关                                                                                        |
| 当前项目  | 有 `orm.DefaultTimeLoc` 和 `ZONEINFO` 说明，Blog 时间逻辑可能仍偏旧                                                           |
| 建议动作  | cherry-pick 时间处理；核对 `models/Blog.go`、模板中的 `date_format`                                                         |
| 工作量   | 小                                                                                                               |
| 验证    | Blog 列表创建/修改时间与系统时区一致                                                                                           |


**本地最小升级方案**

- **核心思路**：不动 ORM 配置（`orm.DefaultTimeLoc` 已设 `time.Local`），只在模板/JSON 输出层补一道兜底：模型读出时间后统一 `t.In(time.Local).Format("2006-01-02 15:04:05")`。Blog 列表与文档列表分别核对一次即可。
- **改动文件**：`models/Blog.go`（`Find*` 类方法返回前调 `.In(time.Local)`）、`models/Document.go` 同上；模板里 `date_format` 调用统一传 `"2006-01-02 15:04"`。
- **放弃的上游能力**：Termux 等特殊环境的 `ZONEINFO` 探测、容器内时区自检脚本。
- **工作量**：0.5 天（多数时间在回归 Blog / 文档列表 / 评论 / 历史版本）。
- **验证**：宿主机 `TZ=Asia/Shanghai` 与容器 `TZ=UTC` 各跑一次，Blog 创建时间显示一致；MySQL 字段类型保持 `datetime` 不动。
- **升级路径**：如真出现 Termux 场景，再单独打补丁判断 `time.Local` 是否为 UTC 兜底。

---

### 0.3 大文件上传


| 项         | 内容                                                                      |
| --------- | ----------------------------------------------------------------------- |
| 上游 Commit | [4cdafc8](https://github.com/mindoc-org/mindoc/commit/4cdafc8)（2023-05） |
| 问题        | 上传 >1GB 报错                                                              |
| 建议动作      | 对照 `DocumentController` / `BlogController` 上传大小解析逻辑                     |
| 工作量       | 极小                                                                      |
| 是否必跟      | 有超大附件上传需求时建议跟                                                           |


**本地最小升级方案**

- **核心思路**：把"最大上传体积"做成 `conf/app.conf` 的显式配置项，启动时一次性写到 Beego `web.BConfig.MaxMemory` 与 `web.BConfig.MaxUploadSize`，避免硬编码与默认 1GB 限制。Controller 不动。
- **改动文件**：`conf/app.conf.example` 新增 `http_max_memory_mb`、`http_max_upload_mb`；`commands/command.go` 启动初始化时读取并赋值；`controllers/DocumentController.go`/`BlogController.go` 上传处补一条配置项校验（仅日志，不强行截断）。
- **放弃的上游能力**：分片上传、断点续传。
- **工作量**：极小（2~3 小时）。
- **验证**：上传 1.5GB 附件不报 `http: request body too large`；超过配置上限时返回友好提示。

---

## 四、阶段 1：搜索体系（收益最高，改动最大）

### 1.1 倒排索引 + TF-IDF 初版


| 项     | 内容                                                      |
| ----- | ------------------------------------------------------- |
| 上游 PR | [#1027](https://github.com/mindoc-org/mindoc/pull/1027) |
| 日期    | 2026-03-18                                              |
| 核心能力  | 倒排索引表、jieba 分词、TF-IDF 打分、文档/Blog 索引维护                   |


**上游新增/大改文件：**


| 类别   | 文件                                                                           |
| ---- | ---------------------------------------------------------------------------- |
| 模型   | `models/ContentReverseIndex.go`（约 498 行，新表）                                  |
| 分词   | `utils/segmenter/segmenter.go`                                               |
| 词典   | `lib/jieba/*.utf8`（约 60 万行词典文件）                                              |
| 控制器  | `controllers/SearchController.go`（`IndexV2`、`SearchV2`、`PerformSearchV2Raw`） |
| 文档保存 | `models/DocumentModel.go` 保存/删除时建索引                                          |
| Blog | `models/Blog.go` 保存/删除时建索引                                                   |
| 删项目  | `models/BookModel.go` 级联删索引                                                  |
| 启动   | `commands/command.go` 注册模型 + `InitializeMissingIndexes()`                    |



| 项    | 内容                                           |
| ---- | -------------------------------------------- |
| 依赖   | `github.com/yanyiwu/gojieba`                 |
| 当前项目 | 仅有 `models/DocumentSearchResult.go` 的 SQL 搜索 |
| 工作量  | 大（3~5 天，含迁移与测试）                              |
| 数据库  | 需新表（如 `md_content_reverse_index`，以实际上游为准）    |


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

**本地最小升级方案**

- **核心思路**：暂不引入 `gojieba` 与新表，在现有 `models/DocumentSearchResult.go` 的 SQL `LIKE` 基础上做三件事：① 标题命中加权（`ORDER BY (title LIKE ? )*10 + (content LIKE ?) DESC`）；② MySQL 给 `md_documents(document_name, release)` 加 `FULLTEXT` 索引并新增 `MATCH ... AGAINST` 查询分支；③ SQLite 走 FTS5 虚拟表（可选，迁移时通过 `commands/migrate` 建立）。
- **改动文件**：`models/DocumentSearchResult.go`（SQL 重写 + 引擎分支）、`commands/migrate/`（FULLTEXT/FTS5 DDL）、`conf/app.conf.example`（开关 `search_engine = like|fulltext`）、`controllers/SearchController.go`（仅按开关切换查询方法）。
- **放弃的上游能力**：jieba 中文分词、`ContentReverseIndex` 表、TF-IDF 打分、`PerformSearchV2Raw`、`IndexV2`、`SearchV2` 路由。
- **适用场景**：文档总量 < 5 万；中文短句子串搜索可接受；不需要"AI/MCP 检索"。
- **工作量**：1~2 天。
- **验证**：标题命中排前；翻页结果总数稳定；不同引擎开关切换可热替换；与现有 SQL `LIKE` 结果做对比。
- **升级路径**：保持 `SearchController` 的对外接口不变，未来切到 #1027/#1034 时只替换内部实现，前端 URL、模板都不动。

---

### 1.2 搜索重构 + reindex


| 项             | 内容                                                      |
| ------------- | ------------------------------------------------------- |
| 上游 PR         | [#1034](https://github.com/mindoc-org/mindoc/pull/1034) |
| 日期            | 2026-03-31                                              |
| 在 #1027 基础上增强 | 见下表                                                     |



| #   | 增强项                                        |
| --- | ------------------------------------------ |
| 1   | 先取候选、统一加权排序后再分页（避免高相关结果被截断）                |
| 2   | 标题命中、正文精确匹配、查询覆盖率 boost                    |
| 3   | 批量回表 + 统一权限过滤                              |
| 4   | 空内容占位索引                                    |
| 5   | 新增 CLI：`doc reindex`（上游为 `mindoc reindex`） |
| 6   | 技术词白名单（`linux`、`grep` 等不被停用词过滤）            |
| 7   | SQL 搜索增加标题相关性排序（兜底）                        |



| 项      | 内容                                                                                                  |
| ------ | --------------------------------------------------------------------------------------------------- |
| 上游改动文件 | `models/ContentReverseIndex.go`、`utils/segmenter/`、`commands/reindex.go`（或同类）、`SearchController.go` |
| 建议动作   | 在 #1027 之后合并 #1034，不要只跟一半                                                                           |
| 工作量    | 中（1~2 天，依赖 #1027）                                                                                   |
| 验证     | `./doc reindex`；搜 `linux`、技术术语；大库分页结果排序合理                                                           |


**本地最小升级方案**

- **核心思路**：在 1.1 的"FULLTEXT/FTS5"最小方案上加 `doc reindex` 命令，仅用于"重建 FULLTEXT 索引"或"重建 FTS5 虚拟表"；不要求实现倒排表 + 加权打分。技术词白名单仅作用于现有 SQL `LIKE` 的禁词剔除。
- **改动文件**：`commands/reindex.go`（新建，约 60 行）、`utils/stopwords.go`（停用词 + 技术词白名单）、`commands/command.go`（注册命令）。
- **放弃的上游能力**：批量回表、文档级 TF-IDF、查询覆盖率 boost、空内容占位索引。
- **工作量**：0.5~1 天（依赖 1.1 已落地）。
- **验证**：`./doc reindex` 不报错；搜 `linux`/`grep` 不被剔除；命令重复执行幂等。

---

### 1.3 全局搜索前端体验


| 项     | 内容                                                               |
| ----- | ---------------------------------------------------------------- |
| 上游 PR | [#1026](https://github.com/mindoc-org/mindoc/pull/1026)、#1034 部分 |
| 问题    | 焦点在输入框时仍触发全局快捷键；Ctrl+F 被拦截；ESC 关闭搜索异常                            |
| 上游改动  | 前端 JS（全局快捷键检测）、阅读页搜索面板                                           |
| 当前项目  | `views/search/index.tpl` 存在，未见对应快捷键逻辑                            |
| 工作量   | 小~中                                                              |
| 可独立跟进 | 是（不依赖倒排索引，但常与搜索改版一起出现）                                           |


**本地最小升级方案**

- **核心思路**：不重写搜索面板。新增一个共享守卫函数 `function shouldIgnoreShortcut(e){var t=e.target;return !!(t && (t.tagName==='INPUT'||t.tagName==='TEXTAREA'||t.isContentEditable));}`，在 `kancloud.js` 与各页面的全局快捷键处理入口先调用；Ctrl+F 仅在 `#searchPanel` 隐藏时拦截；ESC 关闭仅在搜索面板可见时生效。
- **改动文件**：`static/js/kancloud.js`（新增守卫 + 改 keydown 监听，约 15 行）、`views/search/index.tpl` 与 `views/document/default_read.tpl` 的内联 JS。
- **放弃的上游能力**：搜索面板 UI 重设计、搜索结果高亮组件升级。
- **工作量**：半天。
- **验证**：在标题输入框里能正常输入 `/`、`?`；阅读页 Ctrl+F 唤起自定义搜索；ESC 只关搜索面板不影响其它弹层。

---

## 五、阶段 2：编辑器与阅读体验（已处理）

### 2.1 Editor.md + Mermaid + KaTeX


| 项     | 内容                                                                                        |
| ----- | ----------------------------------------------------------------------------------------- |
| 上游 PR | [#1035](https://github.com/mindoc-org/mindoc/pull/1035)                                   |
| 日期    | 2026-03-31                                                                                |
| 内容    | editor.md 升级；mermaid 8.13→10.9；KaTeX auto-render；#909/#948 修复                             |
| 上游改动  | `static/editor.md/`**、`static/js/markdown.js`、`views/document/markdown_edit_template.tpl` |
| 当前项目  | 有 `static/js/markdown.js`、`views/document/markdown_edit_template.tpl`，KaTeX 无 auto-render |
| 工作量   | 中（1~2 天，需回归公式/流程图/序列图）                                                                    |
| 风险    | 静态资源体积大，与现有 wangEditor 并存需分别测                                                             |
| 可独立跟进 | 是                                                                                         |


**当前差距速查（核查于 2026-06）：**


| 组件                              | 仓库现状                                                                    | 上游目标                             | 影响                                                                     |
| ------------------------------- | ----------------------------------------------------------------------- | -------------------------------- | ---------------------------------------------------------------------- |
| `static/editor.md/editormd.js`  | v1.5.0（行 5/62）                                                          | v1.7.17                          | 编辑预览 mermaid 用 `mermaid.init()`，与 mermaid 10 不兼容                       |
| `static/editor.md/lib/mermaid/` | 3.5.6，仅 `mermaid.slim.js`/`mermaidAPI.slim.js`，**无** `mermaid.min.js`   | 10.x                             | 编辑预览只能渲染极旧语法                                                           |
| `static/katex/`                 | 仅 `katex.js`/`katex.css`，**无** `katex.min.*`、**无** `auto-render.min.js` | 含 min + auto-render              | `default_read.tpl:29`、`blog/index.tpl:24` 写死 `katex.min.css` → **404** |
| 阅读页 JS                          | `default_read.tpl` 只引 mermaid/katex **CSS**，未引 JS                       | 应引 mermaid + katex + auto-render | 公式/流程图阅读页**完全不渲染**                                                     |
| `kancloud.js`                   | `articleOpen` 仅调 `initHighlighting()`                                   | 切文档后需补 mermaid/katex 重渲染         | AJAX 切文档后公式/图无效                                                        |
| `views/widgets/scripts.tpl`     | **缺失**（但 `BaseController.go:79` 已读取注入 `{{.Scripts}}`）                   | —                                | 现成的"全局脚本注入点"未利用                                                        |
| `static/js/markdown.js`         | 有 `tex:true` `flowChart` `sequenceDiagram`，未显式 `mermaid:true`           | 显式 `mermaid:true`                | 与 `blog.js` 不一致                                                        |


**本地最小升级方案**

按"组件升级 + 最小本地接线"四步走，不合并上游 Go/模板：

1. **补 KaTeX 资源（≈30 分钟）**
  - 把 KaTeX 官方发布包里的 `katex.min.js`、`katex.min.css`、`contrib/auto-render.min.js` 放入 `static/katex/`。
  - 保留旧的 `katex.js` / `katex.css`，原因：`views/document/markdown_edit_template.tpl:13` 把 `window.katex` 作为**前缀字符串**传给 editormd，editormd 内部会再拼 `.js/.css`（不带 min），删了会破坏编辑预览。
2. **升 Mermaid（二选一）**
  - **路径 A（推荐）**：整目录替换 `static/editor.md/` 为上游 v1.7.x 或 [ibm-skills-network/editor.md](https://github.com/ibm-skills-network/editor.md) 包。mermaid、editormd 一起升级，几乎不用改 JS 逻辑。
  - **路径 B（仅当 editor.md 有定制）**：只放新的 `static/editor.md/lib/mermaid/mermaid.min.js`，并手改 `editormd.js` 中所有 `mermaid.init(` → `mermaid.run(`、修正 mermaid 加载路径。
3. **新建 `views/widgets/scripts.tpl`**（核心杠杆，一次注入两页）

```html
<script src="{{cdnjs "/static/editor.md/lib/mermaid/mermaid.min.js"}}"></script>
<script src="{{cdnjs "/static/katex/katex.min.js"}}"></script>
<script src="{{cdnjs "/static/katex/contrib/auto-render.min.js"}}"></script>
<script>
  if (window.mermaid && mermaid.initialize) {
    mermaid.initialize({ startOnLoad: false, securityLevel: 'loose' });
  }
  window.renderReadPageMath = function (root) {
    root = root || document.getElementById('page-content');
    if (!root) return;
    if (typeof renderMathInElement === 'function') {
      renderMathInElement(root, {
        delimiters: [
          { left: '$$', right: '$$', display: true },
          { left: '$',  right: '$',  display: false },
          { left: '\\(', right: '\\)', display: false },
          { left: '\\[', right: '\\]', display: true }
        ],
        ignoredTags: ['script','noscript','style','textarea','pre','code'],
        throwOnError: false
      });
    }
    if (window.mermaid && mermaid.run) {
      var nodes = root.querySelectorAll('.lang-mermaid:not([data-processed])');
      if (nodes.length) mermaid.run({ nodes: nodes }).catch(console.warn);
    }
  };
  document.addEventListener('DOMContentLoaded', function () {
    window.renderReadPageMath(document.getElementById('page-content'));
  });
</script>
```

1. **改 `static/js/kancloud.js` 一处**（AJAX 切文档后重渲染）
  在 `articleOpen`（第 2–24 行）中 `initHighlighting()` 调用之后追加：

```javascript
if (typeof window.renderReadPageMath === 'function') {
    window.renderReadPageMath(document.getElementById('page-content'));
}
```

**改动面汇总：**


| 区域                                | 改动                                                 | 量级         |
| --------------------------------- | -------------------------------------------------- | ---------- |
| `static/katex/`                   | 新增 3 个文件                                           | 仅资源        |
| `static/editor.md/`               | 整目录替换（路径 A）/ 局部替换（路径 B）                            | 大文件 / 中等手改 |
| `views/widgets/scripts.tpl`       | 新建                                                 | ~30 行      |
| `static/js/kancloud.js`           | 加 1 处调用                                            | +3 行       |
| `static/js/markdown.js`           | 可选显式 `mermaid:true`                                | +1 行       |
| `views/document/default_read.tpl` | 可选删除已 404 的 `katex.min.css` 行（补齐 `.min.css` 后保留也行） | 0~1 行      |
| Go 后端                             | **不动**                                             | 0          |


**放弃的上游能力：** `markdown_edit_template.tpl` 工具栏重排、Cherry 联动、i18n 文案调整、上游对 `saveHTMLToTextarea` 的二次处理。

**风险与对策：**


| 风险                                                       | 触发场景                                            | 对策                                              |
| -------------------------------------------------------- | ----------------------------------------------- | ----------------------------------------------- |
| 旧 release HTML 是已渲染的 `<div class="mermaid"><svg/></div>` | 历史文档                                            | 渲染时只挑 `.lang-mermaid:not([data-processed])`，不冲突 |
| 旧 release HTML 是未渲染的 `<pre class="lang-mermaid"><code>`  | mindoc release 把 `saveHTMLToTextarea` 后 HTML 存库 | 阅读页 JS 会补渲染，反而是修复                               |
| 行内 `$` 误识别（如价格 `$100`）                                   | KaTeX auto-render                               | 上面 `delimiters` 可改为仅 `$$` + `\(..\)`，更安全        |
| editor.md v1.7.x 工具栏 DOM 改了                              | 路径 A                                            | 回归 `markdown_edit_template.tpl` 工具栏样式，必要时退回路径 B |
| 导出 PDF                                                   | `models/BookResult.go` 复制 `mermaid.css`         | 路径 A 确认 `mermaid.css` 仍在；路径 B 不受影响              |


**工作量：** 0.5~1.5 天（路径 A 偏少，路径 B 偏多）。

**实施顺序（按"看得见效果"排序）：**

```text
Day 0  10 min : 补 katex.min.js/css           → 阅读页 CSS 404 消失
Day 0  30 min : 补 auto-render + 建 scripts.tpl → 阅读页公式渲染
Day 0  10 min : kancloud.js 加一行            → AJAX 切文档后仍渲染
─── 阅读页 OK 后 ───
Day 1         : 升 editor.md（路径 A 或 B）   → 编辑预览新 Mermaid / 行内公式
Day 1 下午    : 回归（编辑/阅读/Blog/PDF）
```

---

### 2.2 Cherry Markdown 全家桶


| 项         | 内容                                                                                                               |
| --------- | ---------------------------------------------------------------------------------------------------------------- |
| 上游 Commit | [21fe4b6](https://github.com/mindoc-org/mindoc/commit/21fe4b631a23f3b2ff1ea913b1bb8e34e8b9ddc1) 及后续（#1028、#1034） |
| 时间        | 2023-07 起，持续维护至 2026                                                                                             |
| 内容        | 第二套 Markdown 编辑器；Draw.io；代码块复制；阅读页目录滚动；编辑目标保持                                                                    |
| 上游新增      | `static/cherry-markdown/**`、新模板、项目级编辑器选择                                                                         |
| 当前项目      | **完全没有** Cherry 相关代码                                                                                             |
| 工作量       | 很大（5~10 天+）                                                                                                      |
| 是否必跟      | 仅当需要 Cherry/Draw.io 时                                                                                            |
| 可拆子项      | 仅跟 #1028 阅读页目录滚动 CSS（若仍用 editor.md）                                                                              |


**本地最小升级方案**

- **核心思路**：**不引入** Cherry / Draw.io。如果只想要"阅读页目录跟随滚动"（#1028 子项），用原生 IntersectionObserver 监听 `#page-content` 内的 `h1,h2,h3`，命中后调 jstree `select_node` 同步左侧目录、再 `scrollIntoView` 滚动目录列表。
- **改动文件**：`static/js/kancloud.js`（新增 TOC 同步函数，约 50 行）、`static/css/kancloud.css`（高亮当前节点样式）。
- **放弃的上游能力**：Cherry 编辑器、Draw.io、代码块复制按钮、项目级"选择编辑器"。
- **工作量**：1 天。
- **验证**：阅读页向下滚动，左侧目录跟随高亮；点击目录后阅读区滚到对应章节；jstree 折叠状态正确恢复。

---

### 2.3 阅读页 URL 与编辑跳转


| 项     | 内容                                                            |
| ----- | ------------------------------------------------------------- |
| 上游 PR | [#1036](https://github.com/mindoc-org/mindoc/pull/1036)、#1034 |
| 内容    | 浏览器地址栏与预览链接更新；从阅读页进入编辑时保留目标文档                                 |
| 建议动作  | 与 #1036、编辑器选型一起评估                                             |
| 工作量   | 小~中                                                           |


**本地最小升级方案**

- **核心思路**：在 `kancloud.js:articleOpen` 已有 `pushState` 的基础上把 `$url` 标准化为 `/:key/:id`；阅读页右上角"编辑"按钮 href 追加 `?doc_id={当前文档ID}`；编辑页加载脚本读 query，自动调 `loadDocument(...)` 定位文档。
- **改动文件**：`static/js/kancloud.js`（URL 规整 + 编辑按钮 href 动态更新）、`views/document/default_read.tpl`（编辑按钮 href 模板）、`views/document/markdown_edit_template.tpl` 顶部内联 JS。
- **放弃的上游能力**：上游 `editor/` URL 全量重排、路由层调整。
- **工作量**：0.5 天。
- **验证**：阅读 → 切到子文档 → URL 同步更新；浏览器后退正确；点编辑按钮直达对应文档；编辑页刷新仍停留同一文档。

---

### 2.4 评论头像


| 项     | 内容                                                    |
| ----- | ----------------------------------------------------- |
| 上游 PR | [#977](https://github.com/mindoc-org/mindoc/pull/977) |
| 日期    | 2024-08                                               |
| 内容    | 评论列表显示用户头像                                            |
| 工作量   | 极小                                                    |
| 是否必跟  | 低优先级 UI 优化                                            |


**本地最小升级方案**

- **核心思路**：评论模型已有 `Member.Avatar` 字段，模板直接渲染 `<img class="avatar" src="{{.Avatar}}">`；为空时回退到 Gravatar（`https://www.gravatar.com/avatar/{md5(email)}?d=identicon`）或本地默认头像。CSS 加圆形样式 `border-radius: 50%`。
- **改动文件**：评论列表模板（`views/document/default_read.tpl` 或 `views/comment/list.tpl`）、`static/css/kancloud.css`。
- **放弃的上游能力**：用户主页跳转、头像缓存代理。
- **工作量**：1~2 小时。

---

## 六、阶段 3：权限与账号体系

### 3.1 只读用户角色


| 项     | 内容                                                    |
| ----- | ----------------------------------------------------- |
| 上游 PR | [#992](https://github.com/mindoc-org/mindoc/pull/992) |
| 日期    | 2024-12                                               |
| 内容    | 系统级只读用户：不能创建/编辑，只能做项目观察者                              |


**上游改动范围：**


| 类别  | 内容                                               |
| --- | ------------------------------------------------ |
| 枚举  | `conf/enumerate.go` 增加 `MemberReadOnlyRole`      |
| 管理  | `controllers/ManagerController.go` 用户创建/编辑       |
| 权限  | 各 Controller 写操作拦截                               |
| 模板  | 用户管理页、角色下拉                                       |
| CSS | `markdown.preview.css`、Cherry 阅读宽度（TOC 隐藏时 100%） |
| PDF | 发布者默认取公司名称                                       |



| 项    | 内容                                                              |
| ---- | --------------------------------------------------------------- |
| 当前项目 | `ManagerController` 仅允许 `MemberAdminRole` / `MemberGeneralRole` |
| 工作量  | 中（2~3 天）                                                        |
| 是否必跟 | 有「只读账号」需求时建议跟                                                   |


**本地最小升级方案**

- **核心思路**：不引入新角色枚举 `MemberReadOnlyRole`。复用现有"普通用户 + 项目级观察者权限"。在 `conf/app.conf` 加 `member_general_can_create_book = true|false`；当 false 时，`BookController.Create`/`SaveBook` 对 `MemberGeneralRole` 用户拒绝；管理员仍可手动创建项目并把目标用户设为"观察者"。
- **改动文件**：`conf/app.conf.example`、`controllers/BookController.go`（写操作拦截）、`controllers/ManagerController.go` 用户管理页加一个全局开关 UI。
- **放弃的上游能力**：用户管理页角色下拉新增、各 Controller 写操作的"系统级只读用户"全链路改造、PDF 默认发布者取公司名。
- **工作量**：0.5 天。
- **验证**：开关关闭后，新建普通用户**无法**新建项目、但**能**被管理员加入项目作为观察者；现有用户不受影响。
- **升级路径**：未来真要"系统级只读"再迁到 #992 完整方案，引入新角色枚举即可，本地改动可平滑覆盖。

---

### 3.2 OAuth2 登录重写


| 项     | 内容                                                    |
| ----- | ----------------------------------------------------- |
| 上游 PR | [#851](https://github.com/mindoc-org/mindoc/pull/851) |
| 日期    | 2023-04                                               |
| 内容    | 企业微信、钉钉登录逻辑重写                                         |
| 当前项目  | 有钉钉（`AccountController` + `utils/dingtalk`），**无**企业微信 |
| 工作量   | 中~大                                                   |
| 是否必跟  | 仅当 SSO 有问题或要加企微时                                      |


**本地最小升级方案**

- **核心思路**：保留现有钉钉登录（`AccountController` + `utils/dingtalk`），不重写公共 OAuth 框架。若要新增企业微信，单独新增 `utils/wework/` 与 `controllers/WeWorkController.go`，与钉钉并行；`AccountController` 仅在登录页加一个企微入口按钮。
- **改动文件**：`utils/wework/`（新建）、`controllers/WeWorkController.go`（新建）、`routers/router.go`（路由）、`views/account/login.tpl`（登录按钮）、`conf/app.conf.example`（`wework_corp_id` 等配置项）。
- **放弃的上游能力**：统一 OAuth 接口抽象、provider 抽象层、钉钉登录的连带优化。
- **工作量**：1~2 天（仅添加企微时）。
- **验证**：钉钉登录回归通过；企微扫码可走完登录流程；两套登录互不影响。

---

### 3.3 LDAPS


| 项         | 内容                                                                                              |
| --------- | ----------------------------------------------------------------------------------------------- |
| 上游 Commit | [a2202f8](https://github.com/mindoc-org/mindoc/commit/a2202f887888e481cd6bf43c5972408618c36710) |
| 内容        | LDAP over TLS                                                                                   |
| 当前项目      | `utils/ldap.go`、`models/Member.go`，未见 ldaps                                                     |
| 工作量       | 小                                                                                               |
| 是否必跟      | 有 LDAPS 需求时                                                                                     |


**本地最小升级方案**

- **核心思路**：在 `utils/ldap.go` 给 `LDAPClient.Connect` 增加 `useTLS bool` 参数：true 时走 `ldap.DialTLS("tcp", addr, &tls.Config{InsecureSkipVerify: skip})`，否则保持现有 `ldap.Dial`。`conf/app.conf` 新增 `ldap_tls`、`ldap_tls_skip_verify`、`ldap_ca_file`（可选）。
- **改动文件**：`utils/ldap.go`（约 20 行）、`models/Member.go` 调用处传参、`conf/app.conf.example`。
- **放弃的上游能力**：证书指纹校验、双向 mTLS。
- **工作量**：2 小时。
- **验证**：`ldap_tls=true` + 自签证书时 `ldap_tls_skip_verify=true` 能登录；正式证书无需 skip 也能登录；ldap 普通连接回归通过。

---

## 七、阶段 4：国际化与 MCP

### 4.1 i18n 可配置 + 俄语


| 项     | 内容                                                                                                            |
| ----- | ------------------------------------------------------------------------------------------------------------- |
| 上游 PR | [#987](https://github.com/mindoc-org/mindoc/pull/987)、[#1013](https://github.com/mindoc-org/mindoc/pull/1013) |
| 日期    | 2024-11 ~ 2025-10                                                                                             |
| 内容    | 语言列表可配置；新增俄语；修复语言设置问题                                                                                         |
| 工作量   | 中                                                                                                             |
| 是否必跟  | 多语言/可配置语言时需要                                                                                                  |


**本地最小升级方案**

- **核心思路**：`conf/app.conf` 加 `enabled_langs = zh-cn,en-us,zh-tw`；`controllers/BaseController.go` 在 `Prepare` 中读出并塞到 `c.Data["EnabledLangs"]`；语言切换下拉模板改循环 `range .EnabledLangs`，文案查 `i18n` 现有 keys。不主动新增俄语 messages 文件，按真实需求再补。
- **改动文件**：`conf/app.conf.example`、`controllers/BaseController.go`、语言切换 widget 模板。
- **放弃的上游能力**：后台"语言列表 UI 管理"、俄语翻译。
- **工作量**：0.5 天。
- **验证**：注释/调整 `enabled_langs` 后下拉变化；切换语言生效；未启用语言不会出现在下拉里但 URL 强访问不报错（回退默认语言）。

---

### 4.2 MCP Server


| 项     | 内容                                                                                                              |
| ----- | --------------------------------------------------------------------------------------------------------------- |
| 上游 PR | [#1010](https://github.com/mindoc-org/mindoc/pull/1010)、[#1012](https://github.com/mindoc-org/mindoc/pull/1012) |
| 日期    | 2025-09 ~ 2025-10                                                                                               |
| 内容    | MCP 文档全局检索；配置文档                                                                                                 |
| 上游新增  | `mcp/` 包、`mark3labs/mcp-go` 依赖                                                                                  |
| 与搜索关系 | #1027 后 MCP 改用 `PerformSearchV2Raw`（倒排索引）                                                                       |
| 建议动作  | 若要 AI 集成，建议在搜索体系完成后移植                                                                                           |
| 工作量   | 中（2~3 天，强依赖搜索）                                                                                                  |


**本地最小升级方案**

- **核心思路**：MCP 实现强依赖优质搜索。若**不**上倒排索引（即 1.1/1.2 走的也是最小方案），可以做一个**最小可用 MCP shim**：基于 `github.com/mark3labs/mcp-go` 的 stdio server，工具仅暴露 `search_document(query, limit)`、`get_document(id)` 两个，内部直接调现有 `DocumentSearchResult.FindToPager` 与 `models/Document.FindById`，权限按公开文档过滤。
- **改动文件**：`mcp/server.go`（新建，约 150 行）、`commands/mcp.go`（新增子命令 `doc mcp`）、`go.mod` 加 `mark3labs/mcp-go`。
- **放弃的上游能力**：HTTP 模式 MCP、文档级权限二次校验、上游配置文档示例、依赖倒排的相关性排序。
- **工作量**：1 天（不依赖搜索 V2 时）。
- **验证**：Claude Desktop / Cursor 配置 `doc mcp` 后能列出工具；`search_document("foo")` 返回与 web 搜索一致的前 N 条。
- **升级路径**：等 1.1/1.2 升级到 #1027/#1034，把内部调用换成 `PerformSearchV2Raw` 即可，不影响 MCP 协议层。

---

## 八、阶段 5：数据库与部署

### 5.1 PostgreSQL 支持


| 项     | 内容                                                                                                          |
| ----- | ----------------------------------------------------------------------------------------------------------- |
| 上游 PR | [#792](https://github.com/mindoc-org/mindoc/pull/792)、[#986](https://github.com/mindoc-org/mindoc/pull/986) |
| 日期    | 2023-04 ~ 2024-11                                                                                           |
| 内容    | 增加 PostgreSQL；修复 `LIMIT ?,?` 等语法                                                                            |
| 当前项目  | 仅 mysql/sqlite；`DocumentSearchResult.go` 使用 `LIMIT ?, ?`                                                    |
| 工作量   | 中~大                                                                                                         |
| 是否必跟  | 仅用 MySQL/SQLite 可跳过                                                                                         |


**本地最小升级方案**

- **核心思路**：暂不引入 PG 驱动；仅做"为未来铺路"——把 `models/` 中所有 `LIMIT ?, ?` 全局改为 `LIMIT ? OFFSET ?`（MySQL/SQLite/PG 都兼容），消除将来切 PG 时最大的语法阻塞。
- **改动文件**：`models/DocumentSearchResult.go`、其它 `LIMIT ?, ?` 出现的模型文件（可全局检索 `LIMIT \?\s*,\s*\?`）。
- **放弃的上游能力**：`go-sql-driver/pq` 引入、Adapter 抽象层、PG 专属 SQL（如 `RETURNING`）的优化。
- **工作量**：0.5 天。
- **验证**：MySQL/SQLite 跑现有分页用例全部通过；`EXPLAIN` 看执行计划未变化。
- **升级路径**：未来切 PG 时只需新增驱动注册与 conf 配置块。

---

### 5.2 Docker / PDF 导出


| 项     | 内容                                                                                                            |
| ----- | ------------------------------------------------------------------------------------------------------------- |
| 上游 PR | [#994](https://github.com/mindoc-org/mindoc/pull/994)、[#1001](https://github.com/mindoc-org/mindoc/pull/1001) |
| 内容    | Dockerfile 升级、Calibre 版本、PDF 导出依赖                                                                             |
| 当前项目  | 有 `Dockerfile`（Go 1.25 + Ubuntu focal）                                                                        |
| 是否必跟  | 用 Docker 部署或 PDF 导出有问题时                                                                                       |


**本地最小升级方案**

- **核心思路**：保持现 `Dockerfile` 结构（Go 1.25 + Ubuntu focal）。仅针对 PDF 导出失败的两类常见原因打小补丁：① 安装 `fonts-noto-cjk`（中文字体）；② 安装 `tzdata` 并设置 `ENV TZ=Asia/Shanghai`；③ 锁定 Calibre 版本以稳定 `ebook-convert` 行为。
- **改动文件**：`Dockerfile`（+3~5 行）、可选 `docker-entrypoint.sh` 兜底设 `TZ`。
- **放弃的上游能力**：多阶段构建重排、Calibre 源切换、最新基础镜像跨度大版本升。
- **工作量**：1~2 小时。
- **验证**：容器内 `ebook-convert --version` 正常；中文 PDF 不出现"豆腐块"；时间戳显示与宿主一致。

---

## 九、阶段 6：历史 Bug 修复（按需 cherry-pick）


| PR/Commit                                                                                                                                                   | 日期      | 内容                       | 建议                           |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | ------------------------ | ---------------------------- |
| [#850](https://github.com/mindoc-org/mindoc/issues/850) / [c8f7a2a](https://github.com/mindoc-org/mindoc/commit/c8f7a2a54478388c938612a796f45a4440ac7a76)   | 2023-06 | 附件不能下载、非 Markdown 文档附件问题 | 对照 `DocumentController` 是否已修 |
| [#849](https://github.com/mindoc-org/mindoc/issues/849) / [452577c](https://github.com/mindoc-org/mindoc/commit/452577ca3d015454b9bb3aefe4e5736a14eff1f2)   | 2023-06 | 加密文章访问权限                 | 若有加密文档功能则核对                  |
| [#1024](https://github.com/mindoc-org/mindoc/issues/1024) / [bec1763](https://github.com/mindoc-org/mindoc/commit/bec17630274e44ce431e54871a892603bf9c32e6) | 2026-03 | 未公开 issue 修复             | 需看具体 diff 再定                 |
| [d7547a8](https://github.com/mindoc-org/mindoc/commit/d7547a85df16d236b86dfcdcefbfb65de8ea0951)                                                             | 2023-06 | MTE 表格编辑增强               | 依赖 editor.md，可选              |
| [cc34c63](https://github.com/mindoc-org/mindoc/commit/cc34c6309e975611da9247895bf02601d2ce0922)                                                             | 2023-07 | Draw.io 支持               | 依赖 Cherry，可选                 |


---

## 十、不建议跟或低优先级


| 项                                                                                                            | 原因                                    |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------- |
| [#1019](https://github.com/mindoc-org/mindoc/pull/1019) 官网链接                                                 | 品牌不同，保持 `git.itopcms.com/jackliu/doc` |
| [30a1e87](https://github.com/mindoc-org/mindoc/commit/30a1e87068052c4f1c0fe77f11e001b2843fed6a) Docker 镜像版本号 | 文档类，非代码                               |
| [b9f1381](https://github.com/mindoc-org/mindoc/commit/b9f13815e81421d8bec677b36d850df5c130615b) login.tpl 小改 | 需对比本仓库登录页                             |
| dependabot 依赖升级                                                                                              | 本仓库已 Go 1.25 + Beego v2，按需单独升         |


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


| 你的需求       | 建议跟的 PR                  |
| ---------- | ------------------------ |
| 搜索慢、结果差    | #1027 + #1034            |
| 部署/路径问题    | #1036 + #1031            |
| 公式/流程图问题   | #1035                    |
| 只读账号       | #992                     |
| AI 检索文档    | #1010（先完成搜索）             |
| 钉钉/企微登录    | #851                     |
| PostgreSQL | #792 + #986              |
| 新编辑器体验     | Cherry 大套件（2023 起）       |
| 小修小补       | #1023、#1026、#1029、附件/加密等 |


---

## 十三、上游近期提交速查（2024-2026）


| 日期         | PR/Commit  | 摘要                             | 优先级         |
| ---------- | ---------- | ------------------------------ | ----------- |
| 2026-04    | #1036      | WorkingDirectory + 编辑/预览 URL   | 高           |
| 2026-03    | #1035      | Editor.md / mermaid / KaTeX 升级 | 中           |
| 2026-03    | #1034      | 搜索重构 + reindex + 前端体验          | 高           |
| 2026-03    | #1031      | 二进制资源路径 / jieba 词典             | 高（配合搜索）     |
| 2026-03    | #1029      | Termux 时区                      | 低           |
| 2026-03    | #1028      | Cherry 阅读页目录滚动                 | 低（需 Cherry） |
| 2026-03    | #1027      | 倒排索引 + TF-IDF                  | 高           |
| 2026-03    | #1026      | 全局快捷键输入框检测                     | 中           |
| 2026-02    | #1023      | Blog 时间显示                      | 低           |
| 2025-10    | #1013      | 语言设置修复                         | 按需          |
| 2025-10    | #1012      | MCP 配置文档                       | 按需          |
| 2025-09    | #1010      | MCP Server + i18n              | 按需          |
| 2025-03~04 | #994/#1001 | Docker / PDF                   | 按需          |
| 2024-12    | #992       | 只读用户角色                         | 按需          |
| 2024-12    | #989       | 语言修复                           | 按需          |
| 2024-11    | #987       | i18n 可配置 + 俄语                  | 按需          |
| 2024-11    | #986       | PostgreSQL SQL 兼容              | 按需          |
| 2024-08    | #977       | 评论头像                           | 低           |


---

## 十四、本地最小升级方案总览

本节汇总各小节的"本地最小升级方案"。**指导思想**：当前 `doc` 项目已与上游 mindoc 在模块路径、品牌、Beego 版本、编辑器组合等方面差异较大，**不再做完整 cherry-pick**，而是**仅吸收功能要点**：要么升级第三方静态组件、要么用最少的本地代码达到等价效果，避免把上游的业务逻辑/路由/模型耦合带进来。

### 14.1 总览表


| 小节  | 主题                      | 本地最小方案要点                                                                                      | 改动量       | 放弃的上游能力                  |
| --- | ----------------------- | --------------------------------------------------------------------------------------------- | --------- | ------------------------ |
| 0.1 | WorkingDirectory        | 加 `DOC_HOME` env 优先级，回退 `os.Getwd()`                                                          | 0.5 天     | 嵌入资源/单二进制分发              |
| 0.2 | 时区                      | 模型返回时间统一 `.In(time.Local)`                                                                    | 0.5 天     | Termux/ZONEINFO 特殊探测     |
| 0.3 | 大文件上传                   | `app.conf` 加 `http_max_upload_mb`，启动赋值                                                        | 2 小时      | 分片/断点续传                  |
| 1.1 | 搜索打底                    | 标题加权 + MySQL FULLTEXT / SQLite FTS5                                                           | 1~2 天     | jieba、倒排表、TF-IDF         |
| 1.2 | reindex                 | `doc reindex` 重建 FULLTEXT/FTS5 + 技术词白名单                                                       | 0.5~1 天   | 批量回表加权                   |
| 1.3 | 快捷键                     | 共享 `shouldIgnoreShortcut` 守卫                                                                  | 半天        | 搜索面板 UI 重设计              |
| 2.1 | Editor.md+Mermaid+KaTeX | 补 KaTeX min/auto-render + 新建 `widgets/scripts.tpl` + `kancloud.js` 加 1 行 + (可选)整目录换 editor.md | 0.5~1.5 天 | 上游编辑模板重排                 |
| 2.2 | Cherry                  | **不引入**；如需 TOC 滚动用 IntersectionObserver 自写                                                    | 1 天       | Cherry/Draw.io           |
| 2.3 | URL 与编辑跳转               | 利用现有 `pushState`，编辑按钮带 `?doc_id`                                                              | 0.5 天     | 路由层 `editor/` 重排         |
| 2.4 | 评论头像                    | 模板直渲染 `Member.Avatar`，CSS 圆形                                                                  | 1~2 小时    | 用户主页跳转                   |
| 3.1 | 只读账号                    | 用 `app.conf` 开关禁普通用户建项目，复用观察者角色                                                               | 0.5 天     | `MemberReadOnlyRole` 全链路 |
| 3.2 | OAuth2                  | 并行新增 `WeWorkController`，保留钉钉不重写                                                               | 1~2 天     | 统一 OAuth 抽象              |
| 3.3 | LDAPS                   | `utils/ldap.go` 加 `useTLS` 参数                                                                 | 2 小时      | mTLS、证书指纹                |
| 4.1 | i18n                    | `app.conf` 加 `enabled_langs`，模板循环                                                             | 0.5 天     | 后台 UI、俄语翻译               |
| 4.2 | MCP                     | mcp-go stdio shim，仅 `search_document`/`get_document`                                          | 1 天       | HTTP MCP、依赖倒排排序          |
| 5.1 | PostgreSQL              | 仅把 `LIMIT ?, ?` → `LIMIT ? OFFSET ?` 铺路                                                       | 0.5 天     | PG 驱动                    |
| 5.2 | Docker/PDF              | `Dockerfile` 加 `fonts-noto-cjk` + `tzdata` + 锁 Calibre                                        | 1~2 小时    | 多阶段重排                    |


### 14.2 推荐组合（按场景选）

```text
组合 A：「公式 + 流程图」轻量化升级（半天 ~ 1 天）
  └─ 2.1（KaTeX + Mermaid + scripts.tpl + kancloud.js）

组合 B：「搜索体验」最小提升（1~2 天）
  ├─ 1.1（FULLTEXT/FTS5 + 标题加权）
  ├─ 1.2（doc reindex + 技术词白名单）
  └─ 1.3（快捷键守卫）

组合 C：「部署稳定性」（半天）
  ├─ 0.1（DOC_HOME）
  ├─ 0.2（时区兜底）
  ├─ 0.3（上传大小）
  └─ 5.2（Dockerfile 字体/TZ）

组合 D：「为未来铺路」（1 天）
  ├─ 5.1（LIMIT 语法统一）
  ├─ 1.1 接口层抽象（便于切倒排）
  └─ 2.3（编辑跳转 query 参数）

不建议在最小方案阶段做：
  ├─ 1.1 完整版（jieba + 倒排）
  ├─ 2.2（Cherry）
  ├─ 3.2 OAuth2 重写
  └─ 4.2 MCP HTTP 模式
```

### 14.3 通用原则

- **只动会带来用户可见改进**的部分；其余保留现状。
- **不引入新依赖**优先；必须引入时（如 mcp-go、KaTeX）选用稳定独立模块。
- **保留上升路径**：每个最小方案都要让以后可以平滑切换到上游完整实现，不引入"反向锁定"。
- **不动 Go 后端核心**：能用前端/模板/配置解决就不改 Controller、Model。
- **可独立验证**：每一步都能单独跑回归，避免"动了 A 才能测 B"。

