# Round 2 升级须知（Breaking）

> 权威步骤亦见 [round-2-execution-plan.md §七](./round-2-execution-plan.md)。
> **现网从 Round 1 / 旧布局升到 Round 2 时必须执行**：清 session + 清 gob 文件缓存，否则登录态 / 缓存反序列化可能直接 500。

## 原因

`encoding/gob` 把类型名写成 **`"包路径.类型名"`**。目录搬迁后例如：

- `models.Member` → `internal/model.Member`
- `git.itopcms.com/astrueus/doc/models` → `.../internal/model`

旧 session、旧 file cache 无法再解码。

## 升级步骤

1. **停服**（预留短暂停机窗口）
2. 备份：配置目录（`conf/` 或仍存在的 `configs/`）、`runtime/session/`、`runtime/cache/`（及仍可能存在的旧根目录 `cache/`）
3. **清 session**（按 `sessionprovider`）
   - `file` → 删除 `runtime/session/*`
   - `redis` → 按业务 prefix 删 key；**勿**在共用 Redis 上整库 `FLUSHDB`
   - `mysql` → `TRUNCATE` session 表（表名以配置/前缀为准，常见含 `session`）
4. **清 gob / 文件缓存**（按 `cache_provider`）
   - `file` → 删除 `runtime/cache/*`（若仍有根目录 `cache/` 一并清空）
   - `redis` → 按 prefix 清
   - `memory` → 重启即清
5. 部署 Round 2 产物（`cmd/doc`、`conf/`、`web/`、`deployments/`）
6. **配置目录**：若现网仍是 `configs/`，整目录迁到 `conf/`（含 `app.conf`、`lang/`）；正式布局对齐 Beego 默认 `./conf/app.conf`
7. 启动并检查首页、静态资源、登录（stderr 不应再出现 `open conf/app.conf`）
8. 通知用户：**需要重新登录**

## 目录结构变更速览

| 旧 | 新（Round 2 收尾后） |
| --- | --- |
| 根目录 `conf/`（曾含 Go 源码） | Go → `internal/config/`；**配置文件仍在 `conf/`**（曾短暂迁到 `configs/`，现已改回以消 Beego 噪音） |
| Round 1/2 中期的 `configs/` | **`conf/`**（请迁移） |
| `static/` / `views/` | `web/static/` / `web/views/` |
| 根目录运行时 `cache/` | `runtime/cache/` |
| 路由 `/api/:key/edit` 等页面 | `/book/:key/edit` 等（JSON `/api/*` 仍保留） |

## 相关

- [Round 2 执行计划](./round-2-execution-plan.md)（含 [§十五 收尾 A](./round-2-execution-plan.md#十五收尾工作round-2-宣布收工前)）
- [CHANGELOG.md](../CHANGELOG.md)
