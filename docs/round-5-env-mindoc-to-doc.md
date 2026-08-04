# Round 5 · 环境变量 `MINDOC_*` → `DOC_*` — 细化方案

> 对应 [round-5-execution-plan.md §十六 冒烟清单](./round-5-execution-plan.md#十六上线--冒烟清单)「环境变量命名」项。  
> **决策：** **不再兼容** `MINDOC_*`；配置与部署一律改为 `DOC_*`（Breaking）。  
> 背景对照见 [refactor-roadmap.md](./refactor-roadmap.md) Step 4（原「双读兼容」设想**本轮否决**）。  
> **状态：** ⏳ 待实施（可与 T7/T14/文档任务并行）。

---

## 一、现状

| 类别 | 现状 |
|---|---|
| [`conf/app.conf.example`](../conf/app.conf.example) | 主体仍 `${MINDOC_XXX\|\|default}` |
| 已用 `DOC_` | `DOC_HOME`；`DOC_LOG_FORMAT`；MCP：`DOC_MCP_*` |
| Docker Compose | 仍见 `MINDOC_DB_DATABASE` 等 |
| 目标态示例 | roadmap 中有完整 `DOC_*` 表 |

---

## 二、目标策略

**只认 `DOC_*`：example、代码占位、部署样例全部切换；不提供 `MINDOC_*` 回退映射。**

1. **`app.conf.example` / `app.conf.dev.example`**：所有 `${MINDOC_XXX||…}` 改为 `${DOC_XXX||…}`。  
2. **不写**启动期「MINDOC → DOC」环境映射函数。  
3. **部署资产与文档**同步改名；CHANGELOG 标 **Breaking**：沿用旧变量名的环境需一次性改环境变量 / 编排文件。  
4. 实施 PR 按 `app.conf.example` **逐行**核对新旧键名（完整对照表可放 PR 描述或本文附录，实施时补齐）。

### 常见键名对照（示意，实施时补全）

| 旧（废弃） | 新（唯一） |
|---|---|
| `MINDOC_ADDR` | `DOC_HTTP_ADDR` 或与 example 统一的 `DOC_ADDR`（实施时定一名） |
| `MINDOC_PORT` | `DOC_HTTP_PORT` |
| `MINDOC_RUN_MODE` | `DOC_RUN_MODE` |
| `MINDOC_BASE_URL` | `DOC_BASE_URL` |
| `MINDOC_DB_*` | `DOC_DB_*` |
| `MINDOC_SESSION_*` | `DOC_SESSION_*` |
| `MINDOC_*`（邮件 / CDN / cache 等） | 对应 `DOC_*` |
| （无） | `DOC_STORAGE_*`（T14 新增，仅新名） |
| （已是新名） | `DOC_HOME` / `DOC_MCP_*` / `DOC_LOG_FORMAT` |

---

## 三、改动面

| 位置 | 动作 |
|---|---|
| `conf/app.conf.example` / `app.conf.dev.example` | 占位符全部改为 `DOC_*`；删除「仍可读 MINDOC」类注释 |
| `deployments/docker-compose.yml` | 环境变量改 `DOC_*` |
| `deployments/systemd/doc.service` | 示例 `Environment=` 用 `DOC_*` |
| Spug / 部署文档 | 更新说明；写明旧 `MINDOC_*` **立即失效** |
| README / AGENTS | 一句话：环境变量前缀为 `DOC_` |
| T14 `[storage]` | 只用 `DOC_STORAGE_*` |
| 仓库内其它 `${MINDOC_` 引用 | 全文检索替换 |
| 默认值里的品牌字符串 | **另议**：`app_key=mindoc`、`sessionname=mindoc_id`、`cache_redis_prefix=mindoc::cache` 是否改 `doc` —— **本任务默认只改「环境变量前缀」**；应用内默认标识改名易导致 session/cache 全失效，需单独 Breaking 公告 |

---

## 四、Breaking 说明

| 项 | 策略 |
|---|---|
| Round 5 | **硬切** `DOC_*`；不设兼容期 |
| 升级动作 | 运维把编排 / systemd / Spug / `.env` 中的 `MINDOC_*` 改为 `DOC_*` 后重启 |
| CHANGELOG | 显式 Breaking + 对照表示例 |

若同时改 `sessionname` / redis prefix：另开 Breaking + 清 session/cache（与冒烟清单一致）；**不与本任务默认范围捆绑**。

---

## 五、验收

- [ ] `app.conf.example` 中无 `${MINDOC_`  
- [ ] 仅设 `DOC_DB_HOST` 等关键项时可启动并连库  
- [ ] 仅设旧 `MINDOC_DB_HOST`、**未**设对应 `DOC_*` 时：**不能**再靠旧变量生效（符合硬切预期）  
- [ ] `DOC_HOME` 行为不回归  
- [ ] compose / systemd / 部署文档示例已改为 `DOC_*`  
- [ ] CHANGELOG Breaking 已写  
- [ ] 冒烟清单对应项可勾选  

---

## 六、PR 建议

| PR | 内容 |
|---|---|
| env-a | `app.conf*.example` 占位符切换 + 仓库内引用清理 |
| env-b | deployments / 文档 / CHANGELOG Breaking |

可与 T14-a 配置段合并（storage 键一起加）。

---

## 七、参考

- [refactor-roadmap.md](./refactor-roadmap.md) Step 4（历史双读设想；本轮改为硬切）  
- [`conf/app.conf.example`](../conf/app.conf.example)  
- [round-5-t14-object-storage.md](./round-5-t14-object-storage.md)  
