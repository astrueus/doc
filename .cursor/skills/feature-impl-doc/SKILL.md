---
name: feature-impl-doc
description: >-
  Writes and updates feature implementation docs under docs/{version}-{slug}/
  (README with progress, 01口径, 02现状, 03方案, 04文件与代码要点, 05验收, 06待办),
  then implements only after the user says to. Use when the user pastes a
  product doc, asks for 实施文档, 改造方案, 先出文档, 先不改代码, 按文档落实, 更新进度,
  or wants docs split into a folder with concrete code samples before coding.
---

# 需求实施文档

默认 **先文档、后代码**。用户没说「落实 / 按文档改代码 / 帮我修改」时，只写文档，不动业务代码。

落盘遵循 [AGENTS.md](../../../AGENTS.md)「文档输出目录」（本仓默认 `docs/`）。目录名：`docs/{预定版本}-{英文短名}/`，例如 `docs/2.4.0-object-storage/`。预定版本用 SemVer、**不带 `v`**（与 CHANGELOG / git-workflow 一致），不是 Round 编号。用户指定了路径则以其为准。版本或短名不清时先确认。先打开同版本已有目录当样例，再新建。不要把新功能实施文档写进 `docs/round-5/`（那是路线图任务）。

写代码时遵守 `@project-standards.mdc`。契约 / 鉴权 / 迁移见决策确认。

## 1. 写文档（先出文档 / 先看看 / 先不改代码）

1. 读需求链接或粘贴正文，对照代码现状，标出已确认 / 未决 / 明确不做。
2. 按 [reference.md](reference.md) 建文件。过长再拆；没有待办可以不建 `06`。
3. `04` 必须落到文件、方法、插入点，给出可粘贴代码，禁止只写「在某某方法里处理」。
4. README 必须有 **完成进度表** 和 **本迭代明确不做**。
5. 未决口径列出选项（推荐项标出），等用户拍板后再改文档，不要擅自定案后写代码。改 HTTP/MCP 字段含义须先确认，不能写进方案当既定事实。

## 2. 改文档

用户改口径、说「补充文档 / 更新进度 / 这个不管」时：只改正文和进度表，不借机改代码。进度表与 `04` 文件清单保持一致。

## 3. 按文档落实（用户明确说落实 / 按文档改）

1. 只读对应目录的 README 进度表和 `04`，跳过「已完成 / 用户已手动 / 不做」。
2. 按清单改，不发明术（未列出的接口、缓存、分层重构）。
3. 改完回写 README 与 `04` 的进度。
4. 超出文档的改动先问。
