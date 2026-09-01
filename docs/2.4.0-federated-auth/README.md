# 2.4.0 · 联邦登录重构

> 对应 Round 5 **T16**。实施文档在本目录；路线图任务索引见 [../round-5/round-5-t16-oauth2.md](../round-5/round-5-t16-oauth2.md)（跳转页）。  
> 口径来源：知识库 [身份认证](https://doc.itopcms.com/docs/backend/backend-1f3sb20d3sq42) 及同章 OAuth / OIDC / Token；2026-09-01 已拍板。  
> **状态：** 📝 文档已出，**未改业务代码**。用户明确说「落实 / 按文档改」后再编码。

## 一句话目标

Doc 作为 **Relying Party**：外部 IdP 认人 → `member_identities` 绑到本地 `Member` → **只发本站 Session**。钉钉、企微走同一套管线；不兼容共用临时读者、不保留旧钉钉路由。

## 完成进度

| 项 | 状态 | 说明 |
|----|------|------|
| 口径确认 | 已确认 | 见 [01](./01-需求与口径.md) |
| 现状对照 | 已确认 | 见 [02](./02-现状分析.md) |
| 改造方案 | 已确认 | 见 [03](./03-改造方案.md) |
| F0 内核（接口 / Binder / 表 / 路由 / state） | 待落实 | 见 [04](./04-文件清单与代码要点.md) |
| F1 钉钉 Provider（删除 tmp_reader、旧路由、`thirdparty/dingtalk`） | 待落实 | 见 04 |
| F2 企业微信 Provider | 待落实 | 见 04 |
| 验收 | 未开始 | 见 [05](./05-验收.md) |
| 通用 OIDC / SAML / 换绑 UI | 不做 | 见 [06](./06-后续待办.md) |

## 本迭代明确不做

- 通用「只配 issuer 即可」的 OIDC 产品化（接口按此预留，实现进 06）
- SAML、JWT 替代浏览器 Session、自建授权服务器
- 改 LDAP / 本地密码 / `[oauth] http_login_url` 主路径
- 已有账号换绑 UI、邮箱自动抢绑
- 保留 `dingtalk_tmp_reader`、`TmpLogin` 联邦路径、`/dingtalk_login`、`/qrlogin/:app`、`internal/thirdparty/dingtalk` 包装层
- 新建 `internal/service/`（绑定编排放 `internal/auth/federated`）
- MCP 协议字段变更（登录成功仍只写 Web Session；MCP Token 仍登录后在 `/member/api-tokens` 签发）

## 文档索引

| 文件 | 内容 |
|------|------|
| [01-需求与口径.md](./01-需求与口径.md) | 已确认 / 边界 |
| [02-现状分析.md](./02-现状分析.md) | 现网链路与问题 |
| [03-改造方案.md](./03-改造方案.md) | 三层架构、表、配置、安全 |
| [04-文件清单与代码要点.md](./04-文件清单与代码要点.md) | 动作表 + 可粘贴代码 |
| [05-验收.md](./05-验收.md) | Web / 单测 / 回归 |
| [06-后续待办.md](./06-后续待办.md) | OIDC、SAML、T11 CSP |

## 切片（落实时）

| 切片 | 分支建议 | 内容 |
|------|----------|------|
| F0 | `feat/r5-t16-federated-core` | 表、接口、Binder、Controller、路由、state；登录页按 Registry 渲染 |
| F1 | `feat/r5-t16-dingtalk` | 钉钉适配；HTTP 客户端内聚进 `federated/dingtalk.go`；删除 `thirdparty/dingtalk`、旧方法与 `tmp_reader` |
| F2 | `feat/r5-t16-wework` | 企微网页授权；example 与部署说明 |

可与 T14 并行。未获「落实」指示前不改代码、不提交。
