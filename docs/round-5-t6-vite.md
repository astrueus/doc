# Round 5 · T6 · 前端 P2 Vite — 细化方案

> 对应 [round-5-execution-plan.md §八 T6](./round-5-execution-plan.md#八t6--前端-p2-vite1~2-周)。  
> 底稿沿用 [round-4 §十 T9](./round-4-execution-plan.md#t9--p21~2-周)。  
> **不做：** Bootstrap 升级、Vue 3 SPA、Tailwind。  
> **状态：** ⏳ 待实施（建议独立 sprint）。

---

## 一、现状

- [`web/static/`](../web/static/)：bootstrap / jquery / editor.md / vuejs / katex / webuploader 等**已检入**第三方树  
- **无** 根级 `package.json` / Vite / `web-ui/`  
- 模板用 `{{cdnjs}}` / `{{cdncss}}` / `{{cdnimg}}` + `/static/...`  
- 静态挂载：[`internal/app/bootstrap.go`](../internal/app/bootstrap.go) — `/static` → `web/static`

---

## 二、目标架构

```
web-ui/                         # 【新增】Vite 源码
├── package.json
├── vite.config.ts
├── src/
│   ├── entries/                # 多入口（按页面）
│   │   ├── document-edit.ts
│   │   ├── document-read.ts
│   │   ├── book-setting.ts
│   │   └── ...
│   ├── styles/
│   └── lib/                    # 抽离的公共工具（原内联 JS）
└── README.md

web/static/
├── dist/                       # 【构建产物】gitignore 或发布时生成
│   ├── manifest.json
│   └── assets/*
├── editor.md/ …                # 过渡期保留；逐步改为 npm 依赖或仍 CDN
└── ...
```

### Vite 配置要点

- `build.manifest = true`；`outDir = ../web/static/dist`  
- `base = '/static/dist/'`（与 Beego 静态前缀一致）  
- 多 `rollupOptions.input` 对应 `src/entries/*`  
- 生产 hash；dev 走 Vite server

---

## 三、模板接入：`vite_asset`

在 BaseController / 模板函数注册处新增：

```text
vite_asset("document-edit") → /static/dist/assets/document-edit.<hash>.js
vite_css("document-edit")   → 对应 CSS（若拆出）
```

实现：

1. 读 `web/static/dist/manifest.json`（启动时或首次请求缓存；文件变更可 mtime 刷新）  
2. 找不到 entry → 打日志并返回空 / 降级旧 `/static/...` 路径（过渡期）  
3. 与现有 `cdnjs` **兼容**：`vite_asset` 返回的路径仍可走 CDN 前缀包装（若 `cdn` 已配置）

### 开发模式

- `npm run dev` 起 Vite（如 `:5173`）  
- `runmode=dev` 时 bootstrap 把 `/static/dist/*` **反代**到 Vite；或模板直接指 `http://127.0.0.1:5173/@vite/client` + entry（二选一，推荐反代以保持同源）  
- 生产：`npm run build` 后 Beego 直接 serve 静态文件  

---

## 四、内联 JS 抽离策略（渐进）

| 优先级 | 页面 / 模板 | 说明 |
|---|---|---|
| P0 | `document/markdown_edit_template.tpl` | 编辑器页，内联最多 |
| P1 | `document/*_read*.tpl` / compare | 阅读与对比 |
| P2 | `book/setting.tpl` / `manager/*` | 设置与后台 |
| P3 | 其余零散 `<script>` | 清单化后按需 |

每 PR 抽 **1~2 个模板**：删内联 → 入 `web-ui/src/entries/` → 模板改 `vite_asset` → 冒烟。

**过渡：** 未抽离页继续用旧 `/static/...`；不要求一次切完。

---

## 五、vendor 策略

| 库 | 本轮建议 |
|---|---|
| jquery / bootstrap 3 | **暂留** `web/static/`，不做 npm 化（避免 P3 范围蔓延） |
| editor.md / katex / mermaid | 优先保持现路径；若 entry 强依赖再考虑 npm |
| 新建业务 JS | **只**进 `web-ui/` |

---

## 六、构建与发布

1. 本地 / CI：`cd web-ui && npm ci && npm run build`  
2. 发布包：`scripts/release.*` 增加一步前端构建（或文档要求发版前必跑）  
3. `web/static/dist/`：建议 **构建产物入库或发版时生成**（二选一写进 `web-ui/README.md`；推荐 CI/发版生成 + `.gitignore dist`，避免 hash 噪音）

---

## 七、PR 拆分

| PR | 内容 |
|---|---|
| T6-a | `web-ui/` 脚手架 + `vite_asset` + 空 manifest 兜底 + README |
| T6-b | 第一个真实入口（建议 markdown 编辑页）+ 冒烟 |
| T6-c+ | 按页渐进抽离；每 PR 小而可回滚 |

---

## 八、验收

- [ ] `npm run build` 成功；`manifest.json` 生成  
- [ ] 登录页 / 文档阅读 / 文档编辑 关键资源 200  
- [ ] dev 反代或 HMR 可用（至少一种）  
- [ ] 内联 `<script>` 数量下降，或剩余项有清单  
- [ ] **无** Bootstrap / Vue SPA 改动  

---

## 九、参考

- [Vite](https://vitejs.dev/)  
- [round-4-execution-plan.md §十 T9](./round-4-execution-plan.md#t9--p21~2-周)  
- [`web/static/`](../web/static/) · [`internal/app/bootstrap.go`](../internal/app/bootstrap.go)  
