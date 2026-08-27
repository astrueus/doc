# 001 · MCP 写入与 Web 发布不对齐

> 2026-08-27 在书籍「后端开发」上用 MCP 批量补文时发现。  
> 本文只收问题，**不排期、不改代码**。实施方案另开。  
> 目录约定见 [README](./README.md)。

## 背景（三列）

`md_documents` 里和正文相关的是三列，不是一列：

| 列 | 含义 | 读者 / 编辑器 |
|----|------|----------------|
| `markdown` | 源稿 | Markdown 编辑器、MCP 读写 |
| `content` | 未发布的 HTML 草稿 | HTML 编辑器；Web 保存时写入预览 HTML |
| `release` | 已发布 HTML | 阅读页实际渲染 |

Web 保存：同时写 `markdown` + `content`（前端 `html`）。发布：把 `content` 拷到 `release`，再加工。  
MCP 写：目前几乎只动 `markdown`。

## 问题清单

| # | 现象 | 原因（简要） | 影响 |
|---|------|----------------|------|
| 1 | MCP 写完后 `content` 常为空 | `UpdateMarkdownWithVersion` 只更新 `markdown` / `version` / `modify_at` | 阅读页靠 `release` 仍可能正常；HTML 编辑器、整本「发布项目」会踩空 |
| 2 | `auto_release` / `release_document` 不回写 `content` | `releaseOneDocument` 在内存里 Markdown→HTML，`ReleaseContent` 只 `InsertOrUpdate("release")` | 库里 `content` 与 `release` 长期不一致 |
| 3 | 阅读页没有文末「作者 / 创建时间 / 最后编辑」 | 作者栏是发布时插入的 `div.wiki-bottom`，且必须挂到 `.markdown-article` 或 `article.markdown-article-inner` 上。MCP 用 blackfriday 碎片 HTML，没有这些包装，插入被丢掉 | 读者可见正文，没有页脚元信息 |
| 4 | 网页点「发布项目」可能把 MCP 已发布的正文盖空 | 整本发布读库里的 `content` 再写入 `release`；`content` 空则 `release` 变空 | **操作风险**：在修好 #1/#2 前不要对 MCP 写过的书点整本发布 |
| 5 | MCP 更新后 `modify_time` 可能仍是旧值 | 乐观锁走 `orm.Params` 更新，未带 `modify_time`；Beego `auto_now` 只在整结构体 Update 时生效 | 即便补上 wiki-bottom，「更新时间」也可能不准 |

## 现状对照

| 路径 | markdown | content | release | wiki-bottom |
|------|----------|---------|---------|-------------|
| Web 编辑器保存并发布 | 有 | 有 | 有 | 通常有 |
| MCP 写入 + `auto_release` | 有 | 常空 | 有（无页脚） | 无 |
| 只建空文档、从未保存 | 空 | 空 | 空 | 无 |

## 未修之前

- 需要读者看见正文：MCP 继续用 `auto_release: true` 或 `release_document`（单篇），**不要**对同一本书用网页「发布项目」。
- 不要用 HTML 编辑器打开 MCP 写过的文档当源稿（`content` 可能是空的）。
- 已写入的「后端开发」等书：等迭代后再统一补 `content` / 重发 `release`，不必手工逐篇用网页点发布。

## 后续迭代（占位）

建议打成一轮「MCP 发布与 Web 对齐」，而不是零散补丁。方向（不定案）：

1. 发布时同时持久化 `content` 与 `release`。
2. 转 HTML 后包上 `article.markdown-article-inner`（或 Processor 找不到包装层时把 wiki-bottom 挂到 body）。
3. MCP 更新顺带刷新 `modify_time`。
4. 回归：MCP 写一篇 → 阅读页有正文和作者栏；网页「发布项目」不把 `release` 清空。

契约（是否对外暴露 `content`、是否改 MCP 字段）拍板后再写方案。
