# Round 5 · T16 · OAuth2 登录重写 — 细化方案

> 对应 [round-5-execution-plan.md](./round-5-execution-plan.md) T16。  
> 来源：[upstream-mindoc-checklist.md §3.2](./upstream-mindoc-checklist.md#32-oauth2-登录重写)（上游 [#851](https://github.com/mindoc-org/mindoc/pull/851)）；**本轮改为正式重写**（修订清单里「不重写公共 OAuth、仅并行加企微」的最小方案）。  
> **状态：** ⏳ 待实施（建议独立切片，可与 T14/T6 并行）。

---

## 一、现状

| 项 | 说明 |
|---|---|
| 钉钉 | [`AccountController`](../internal/controller/AccountController.go) + [`internal/thirdparty/dingtalk`](../internal/thirdparty/dingtalk/dingtalk.go)；路由 `/dingtalk_login` |
| 企业微信 | **无** |
| LDAP | 独立密码校验路径（[`internal/thirdparty/ldap`](../internal/thirdparty/ldap/ldap.go)），**不在**本任务 OAuth 抽象内（可并列存在） |
| 配置 | `[dingtalk]`（example 仍可能是 `MINDOC_DINGTALK_*`；与 [env 硬切](./round-5-env-mindoc-to-doc.md) 一并改 `DOC_DINGTALK_*`） |
| 问题 | 登录逻辑挤在 AccountController；无统一 Provider；加企微只能复制粘贴；state/回调校验与账号绑定分散 |

---

## 二、目标

1. **统一 OAuth2（含扫码类）Provider 抽象**：授权 URL / 用 code 换身份 / 映射到本地 `Member`。  
2. **钉钉迁入新框架**（行为回归：扫码登录、自动建用户策略与现网一致）。  
3. **新增企业微信 Provider**（至少一种扫码/网页授权流程可跑通）。  
4. Controller 变薄：通用回调入口 + 登录页展示已启用的 providers。  
5. 配置强类型进 `config.Config`；环境变量 **`DOC_*` 硬切**。

### 明确不做（本切片）

- 通用「任意 OIDC IdP」完整产品化（可留接口扩展点，本轮只落地钉钉 + 企微）  
- 改 LDAP / 本地密码登录主路径（仅保证互不破坏）  
- 绑定已有账号的复杂「换绑 UI」（最小：按第三方 userid/unionid 查找或自动注册，对齐现钉钉策略）

---

## 三、架构设计

```text
internal/
├── auth/oauth/                 # 或 internal/thirdparty/oauth/
│   ├── provider.go             # Provider 接口
│   ├── registry.go             # 按 conf 启用的 provider 列表
│   ├── session_bind.go         # code → Member → 写 session（共用）
│   ├── dingtalk/
│   │   └── provider.go         # 包装现有 dingtalk 客户端或内聚重写
│   └── wework/
│       └── provider.go
├── controller/
│   └── AccountController.go    # 登录页；OAuth 入口/回调变薄
└── router/
    └── account.go              # /login/oauth/:provider 等
```

### Provider 接口（示意）

```go
type Provider interface {
    Name() string                         // "dingtalk" | "wework"
    Enabled() bool
    // AuthURL：带 state 的跳转或前端扫码所需参数（钉钉 QR key 等）
    AuthURL(state string) (string, error)
    // Exchange：authorization code / 扫码 code → 规范化身份
    Exchange(ctx context.Context, code string) (*Identity, error)
}

type Identity struct {
    Provider   string
    Subject    string // 稳定唯一键（如 userid / openid）
    UnionID    string // 可选
    DisplayName string
    AvatarURL  string
    Email      string // 可选
}
```

### HTTP 流（建议）

| 步骤 | 路径（示例） | 说明 |
|---|---|---|
| 登录页 | `/login` | 渲染已启用 provider 按钮/二维码区 |
| 开始 | `/login/oauth/{provider}` | 生成 `state` 存 session；redirect 或返回 QR 参数 JSON |
| 回调 | `/login/oauth/{provider}/callback` | 校验 state；`Exchange`；绑定/创建 Member；写登录 session |

兼容：旧 `/dingtalk_login` 可 **301/内部转发** 到新回调一版本，再在 CHANGELOG 标明废弃。

### 账号绑定规则（对齐现钉钉）

1. 用 `(auth_method/provider, subject)` 或现有字段约定查找 Member。  
2. 若不存在：按现网「是否允许自动注册 / 临时读者」配置决定创建或拒绝。  
3. 写入 session 与本地密码登录同一套（`MemberIDFromSession` 等 Round 4 约定不变）。

---

## 四、配置

```ini
[oauth]
# 全局开关（可选）
enable = true

[oauth.dingtalk]
enable = true
corp_id = "${DOC_DINGTALK_CORPID}"
app_key = "${DOC_DINGTALK_APPKEY}"
app_secret = "${DOC_DINGTALK_APPSECRET}"
# 其它现有 dingtalk_* 迁入并改名

[oauth.wework]
enable = false
corp_id = "${DOC_WEWORK_CORPID}"
agent_id = "${DOC_WEWORK_AGENT_ID}"
secret = "${DOC_WEWORK_SECRET}"
```

- 与 [env 方案](./round-5-env-mindoc-to-doc.md) 一致：**不**再读 `MINDOC_DINGTALK_*`。  
- 旧 `[dingtalk]` section：迁移期可读一层并打 deprecate 日志，或一次性改 example + 部署文档（推荐硬切，同 env 决策）。

---

## 五、安全

| 项 | 要求 |
|---|---|
| `state` | 随机、单次、绑定 session；回调校验失败则拒 |
| HTTPS | 生产回调 URL 用 `baseurl`；文档写清企微/钉钉后台需配的 redirect |
| 密钥 | 仅 conf / env；不进日志 |
| 与 T11 | CSP 若启用，需放行钉钉/企微扫码相关 script/frame（T16 文档列白名单建议） |

---

## 六、PR 拆分

| PR | 内容 |
|---|---|
| T16-a | `auth/oauth` 接口 + registry + state；钉钉迁入；旧路由兼容；回归钉钉登录 |
| T16-b | 企业微信 Provider + 登录页入口 + conf example + 文档 |
| T16-c | 删除兼容层 / 清理 AccountController 内联钉钉大段逻辑（可与 a 合并若 diff 可控） |

---

## 七、验收

- [ ] 钉钉扫码/免登路径与重写前一致（或差异已文档化）  
- [ ] 企微在 `enable=true` 时可完成登录并建立 session  
- [ ] LDAP / 本地账号密码登录不回归  
- [ ] `state` 伪造或复用失败  
- [ ] conf / env 为 `DOC_*`；[`mcp-integration` 无关] 部署文档已更新  
- [ ] 上游清单 §3.2 状态改为「Round 5 T16 重写落地」  

---

## 八、工作量

约 **3~5 天**（含钉钉迁移回归 + 企微一条链路 + 文档）；不含把所有上游 #851 边角功能一次性搬完。

---

## 九、参考

- [upstream-mindoc-checklist.md §3.2](./upstream-mindoc-checklist.md#32-oauth2-登录重写)  
- 上游 [mindoc#851](https://github.com/mindoc-org/mindoc/pull/851)  
- [`internal/controller/AccountController.go`](../internal/controller/AccountController.go)  
- [`internal/thirdparty/dingtalk/`](../internal/thirdparty/dingtalk/)  
- [round-5-env-mindoc-to-doc.md](./round-5-env-mindoc-to-doc.md)  
