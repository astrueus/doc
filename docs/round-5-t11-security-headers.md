# Round 5 · T11 · 安全头（CORS / CSP / HSTS）— 细化方案（可选）

> 对应 [round-5-execution-plan.md §十三 T11](./round-5-execution-plan.md#十三t11--安全头可选)。  
> **建议：** T6（Vite）后再收紧 CSP。  
> **状态：** ⏳ 可选。

---

## 一、现状

- [`internal/middleware/headers.go`](../internal/middleware/headers.go)：已有 `TopDoc-Version` / `TopDoc-Site` / `X-XSS-Protection`  
- **无** CORS / CSP / HSTS 实现  

---

## 二、配置设计

`conf/app.conf` 增加 `[security]`（名称可调），强类型进 `config.Config`：

```ini
[security]
enable_cors = false
cors_allow_origins =                    # 逗号分隔；空 = 不反射 Origin
cors_allow_methods = GET,POST,PUT,DELETE,OPTIONS
cors_allow_headers = Authorization,Content-Type,X-Requested-With
cors_allow_credentials = false

enable_csp = false
csp_policy = default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'

enable_hsts = false
hsts_max_age = 31536000
hsts_include_subdomains = false
```

环境变量：与 [MINDOC→DOC](./round-5-env-mindoc-to-doc.md) 对齐，优先 `DOC_SECURITY_*` / `DOC_ENABLE_CSP` 等（具体键名在实施时列入 `app.conf.example`）。

**默认全关**，避免上线打断现网（内联脚本、CDN、MCP 跨域）。

---

## 三、中间件行为

| 头 | 行为 |
|---|---|
| CORS | 仅当 `enable_cors`；校验 Origin 白名单；处理 `OPTIONS` 预检；**`/mcp` 可单独更严或更松**（建议同开关 + 可选 `cors_mcp_origins`） |
| CSP | `Content-Security-Policy`（报告模式可先用 `Content-Security-Policy-Report-Only` 一版） |
| HSTS | 仅 HTTPS 响应时加；HTTP 本地开发不加 |

注册点：现有 [`register.go`](../internal/middleware/) 链路上追加，顺序在 Session/Auth 外层即可。

---

## 四、CSP 与前端协作

| 阶段 | 策略 |
|---|---|
| T6 前 | `enable_csp=false`；或 Report-Only + 极宽松 |
| T6 后 | 逐步去掉 `'unsafe-inline'`（内联 JS 已抽离的页面） |
| CDN | 若使用 `cdn` / `cdnjs`，把 CDN host 写入 `script-src` / `style-src` |
| Vite dev | `connect-src` / `script-src` 含 dev server origin |

editor.md / katex / mermaid 常需 `'unsafe-eval'` 或 blob —— **首版允许**，文档标明技术债。

---

## 五、验收

- [ ] 三开关默认可关；主站登录 / 编辑冒烟  
- [ ] `/mcp` HTTP 工具在开关开/关下行为符合预期  
- [ ] HSTS 在本地 HTTP 不误加  
- [ ] `app.conf.example` + 部署文档说明  

---

## 六、明确不做

- WAF / 完整安全审计  
- 强制开启 CSP 导致编辑器不可用  

---

## 七、参考

- [`internal/middleware/headers.go`](../internal/middleware/headers.go)  
- [round-5-t6-vite.md](./round-5-t6-vite.md)  
