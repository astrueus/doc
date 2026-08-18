# Round 5 · 环境变量 `MINDOC_*` → `DOC_*` — 细化方案

> 对应 [round-5-execution-plan.md §十六 冒烟清单](./round-5-execution-plan.md#十六上线--冒烟清单)「环境变量命名」项。  
> **决策：** **不再兼容** `MINDOC_*`；配置与部署一律改为 `DOC_*`（Breaking）。  
> 背景对照见 [refactor-roadmap.md](../refactor-roadmap.md) Step 4（原「双读兼容」设想**本轮否决**）。  
> **状态：** ✅ 已落地（example / 部署样例 / 代码默认值）；升级侧清 session/cache 与编排改名仍须验收勾选。

---

## 一、现状（落地前）

| 类别 | 落地前 | 落地后 |
|---|---|---|
| [`conf/app.conf.example`](../../conf/app.conf.example) | 主体 `${MINDOC_XXX\|\|default}` | 全部 `${DOC_*\|\|…}` |
| 已用 `DOC_` | `DOC_HOME`；`DOC_LOG_FORMAT`；`DOC_MCP_*` | 不变 |
| Docker Compose / sync 脚本 | `MINDOC_*` / `MINDOC_SYNC` | `DOC_*` / `DOC_SYNC` |
| 应用内默认标识 | `app_key=mindoc`、`sessionname=mindoc_id`、代码默认 `mindoc::cache` | 一律 `doc` / `doc_id` / `doc::cache` |

---

## 二、目标策略

**只认 `DOC_*`：example、代码占位、部署样例全部切换；不提供 `MINDOC_*` 回退映射。**

1. **`app.conf.example` / `app.conf.dev.example`**：所有 `${MINDOC_XXX||…}` 改为 `${DOC_XXX||…}`（规则见下）。  
2. **不写**启动期「MINDOC → DOC」环境映射函数。  
3. **部署资产与文档**同步改名；CHANGELOG 标 **Breaking**。  
4. **本轮同步改**应用内默认品牌标识（见 §二.2）；升级须清 session / cache。

### 2.1 环境变量命名规则

| 规则 | 说明 |
|---|---|
| 默认 | `MINDOC_` → `DOC_`，**后缀不变**（如 `MINDOC_DB_HOST` → `DOC_DB_HOST`） |
| 监听 | `MINDOC_ADDR` / `MINDOC_PORT` → **`DOC_ADDR` / `DOC_PORT`**（不用 `DOC_HTTP_*`） |
| 例外 | `MINDOC_EXPIRED` → **`DOC_MAIL_EXPIRED`**（与 `mail_expired` 语义对齐） |
| 部署同步 | `MINDOC_SYNC` → **`DOC_SYNC`**（`deployments/sync_host.sh` / `start.sh`） |
| 已是新名 | `DOC_HOME` / `DOC_MCP_*` / `DOC_LOG_FORMAT`；T14 仅 `DOC_STORAGE_*` |

### 2.2 应用内默认标识（本轮 Breaking）

| 配置项 | 旧默认 | 新默认 | 升级影响 |
|---|---|---|---|
| `app_key` | `mindoc` | `doc` | 「记住我」等依赖 AppKey 的 SecureCookie 失效 |
| `sessionname` | `mindoc_id` | `doc_id` | 旧 session cookie 名失效，须重新登录 |
| `cache_redis_prefix` | `mindoc::cache`（代码默认；example 曾为 `doc::cache`） | `doc::cache` | 旧 Redis 缓存键不再命中；建议清空旧前缀或接受冷启动 |

代码回退默认见 `internal/config/config.go`（与 example 一致）。

---

## 三、完整键名对照

> 规则：`MINDOC_X` → `DOC_X`，下列「例外 / 监听」已单独标注。

| 旧（废弃） | 新（唯一） | 备注 |
|---|---|---|
| `MINDOC_ADDR` | `DOC_ADDR` | 监听地址 |
| `MINDOC_PORT` | `DOC_PORT` | 监听端口 |
| `MINDOC_RUN_MODE` | `DOC_RUN_MODE` | |
| `MINDOC_BASE_URL` | `DOC_BASE_URL` | |
| `MINDOC_CONFIG_AUTO_DELAY` | `DOC_CONFIG_AUTO_DELAY` | |
| `MINDOC_HIGHLIGHT_STYLE` | `DOC_HIGHLIGHT_STYLE` | |
| `MINDOC_ENABLE_XSRF` | `DOC_ENABLE_XSRF` | |
| `MINDOC_SESSION_PROVIDER` | `DOC_SESSION_PROVIDER` | |
| `MINDOC_SESSION_PROVIDER_CONFIG` | `DOC_SESSION_PROVIDER_CONFIG` | |
| `MINDOC_SESSION_MAX_LIFETIME` | `DOC_SESSION_MAX_LIFETIME` | |
| `MINDOC_DB_ADAPTER` | `DOC_DB_ADAPTER` | |
| `MINDOC_DB_HOST` | `DOC_DB_HOST` | |
| `MINDOC_DB_PORT` | `DOC_DB_PORT` | |
| `MINDOC_DB_DATABASE` | `DOC_DB_DATABASE` | |
| `MINDOC_DB_USERNAME` | `DOC_DB_USERNAME` | |
| `MINDOC_DB_PASSWORD` | `DOC_DB_PASSWORD` | |
| `MINDOC_DB_PREFIX` | `DOC_DB_PREFIX` | 表前缀默认仍为 `md_`（未改） |
| `MINDOC_ENABLE_MAIL` | `DOC_ENABLE_MAIL` | |
| `MINDOC_MAIL_NUMBER` | `DOC_MAIL_NUMBER` | |
| `MINDOC_SMTP_USER_NAME` | `DOC_SMTP_USER_NAME` | |
| `MINDOC_SMTP_HOST` | `DOC_SMTP_HOST` | |
| `MINDOC_SMTP_PASSWORD` | `DOC_SMTP_PASSWORD` | |
| `MINDOC_SMTP_PORT` | `DOC_SMTP_PORT` | |
| `MINDOC_FORM_USERNAME` | `DOC_FORM_USERNAME` | |
| `MINDOC_EXPIRED` | `DOC_MAIL_EXPIRED` | **例外** |
| `MINDOC_MAIL_SECURE` | `DOC_MAIL_SECURE` | |
| `MINDOC_ENABLE_EXPORT` | `DOC_ENABLE_EXPORT` | |
| `MINDOC_EXPORT_PROCESS_NUM` | `DOC_EXPORT_PROCESS_NUM` | |
| `MINDOC_EXPORT_LIMIT_NUM` | `DOC_EXPORT_LIMIT_NUM` | |
| `MINDOC_EXPORT_QUEUE_LIMIT_NUM` | `DOC_EXPORT_QUEUE_LIMIT_NUM` | |
| `MINDOC_EXPORT_OUTPUT_PATH` | `DOC_EXPORT_OUTPUT_PATH` | |
| `MINDOC_CDN_URL` | `DOC_CDN_URL` | |
| `MINDOC_CDN_JS_URL` | `DOC_CDN_JS_URL` | |
| `MINDOC_CDN_CSS_URL` | `DOC_CDN_CSS_URL` | |
| `MINDOC_CDN_IMG_URL` | `DOC_CDN_IMG_URL` | |
| `MINDOC_CACHE` | `DOC_CACHE` | |
| `MINDOC_CACHE_PROVIDER` | `DOC_CACHE_PROVIDER` | |
| `MINDOC_CACHE_MEMORY_INTERVAL` | `DOC_CACHE_MEMORY_INTERVAL` | |
| `MINDOC_CACHE_FILE_PATH` | `DOC_CACHE_FILE_PATH` | |
| `MINDOC_CACHE_FILE_SUFFIX` | `DOC_CACHE_FILE_SUFFIX` | |
| `MINDOC_CACHE_FILE_DIR_LEVEL` | `DOC_CACHE_FILE_DIR_LEVEL` | |
| `MINDOC_CACHE_FILE_EXPIRY` | `DOC_CACHE_FILE_EXPIRY` | |
| `MINDOC_CACHE_MEMCACHE_HOST` | `DOC_CACHE_MEMCACHE_HOST` | |
| `MINDOC_CACHE_REDIS_HOST` | `DOC_CACHE_REDIS_HOST` | |
| `MINDOC_CACHE_REDIS_DB` | `DOC_CACHE_REDIS_DB` | |
| `MINDOC_CACHE_REDIS_PASSWORD` | `DOC_CACHE_REDIS_PASSWORD` | |
| `MINDOC_CACHE_REDIS_PREFIX` | `DOC_CACHE_REDIS_PREFIX` | 默认值改为 `doc::cache` |
| `MINDOC_LOG_PATH` | `DOC_LOG_PATH` | |
| `MINDOC_LOG_MAX_LINES` | `DOC_LOG_MAX_LINES` | |
| `MINDOC_LOG_MAX_SIZE` | `DOC_LOG_MAX_SIZE` | |
| `MINDOC_LOG_DAILY` | `DOC_LOG_DAILY` | |
| `MINDOC_LOG_MAX_DAYS` | `DOC_LOG_MAX_DAYS` | |
| `MINDOC_LOG_LEVEL` | `DOC_LOG_LEVEL` | |
| `MINDOC_LOG_IS_ASYNC` | `DOC_LOG_IS_ASYNC` | |
| （已是新名） | `DOC_LOG_FORMAT` | |
| `MINDOC_DINGTALK_CORPID` | `DOC_DINGTALK_CORPID` | T16 企微仅新名 `DOC_WEWORK_*` |
| `MINDOC_DINGTALK_APPKEY` | `DOC_DINGTALK_APPKEY` | |
| `MINDOC_DINGTALK_APPSECRET` | `DOC_DINGTALK_APPSECRET` | |
| `MINDOC_DINGTALK_READER` | `DOC_DINGTALK_READER` | |
| `MINDOC_DINGTALK_QRKEY` | `DOC_DINGTALK_QRKEY` | |
| `MINDOC_DINGTALK_QRSECRET` | `DOC_DINGTALK_QRSECRET` | |
| （已是新名） | `DOC_MCP_*` / `DOC_HOME` | |
| （无旧名） | `DOC_STORAGE_*` | T14 |
| `MINDOC_SYNC` | `DOC_SYNC` | 部署同步脚本 |

---

## 四、改动面

| 位置 | 动作 |
|---|---|
| `conf/app.conf.example` / `app.conf.dev.example` | 占位符全部 `DOC_*`；`app_key` / `sessionname` / redis 默认前缀改 `doc*` |
| `internal/config/config.go` | 上述三项代码默认值与 example 对齐 |
| `deployments/docker-compose.yml` | 环境变量改 `DOC_*`；sqlite 示例库名 `doc.db` |
| `deployments/sync_host.sh` / `start.sh` | `DOC_SYNC` |
| `deployments/systemd/doc.service` | 示例 `Environment=` 用 `DOC_*` |
| README | 前缀说明改为 `DOC_`；Docker 示例改名 |
| T14 `[storage]` | 只用 `DOC_STORAGE_*` |
| CHANGELOG | Breaking：env 硬切 + 默认标识改名 + 清 session/cache |

**不在本任务范围：** 表前缀 `md_`、示例书 identify、历史文档中的上游 MinDoc 专名。

---

## 五、Breaking 说明

| 项 | 策略 |
|---|---|
| Round 5 | **硬切** `DOC_*`；不设兼容期 |
| 升级动作（env） | 编排 / systemd / Spug / `.env` 中 `MINDOC_*` → `DOC_*`（含 `EXPIRED`→`MAIL_EXPIRED`、`SYNC`→`DOC_SYNC`）后重启 |
| 升级动作（标识） | **清空 session 存储**；视旧「记住我」cookie 失效；Redis 若用旧前缀可删 `mindoc::cache*` 或接受冷缓存 |
| CHANGELOG | 显式 Breaking + 指向本文对照表 |

---

## 六、验收

- [x] `app.conf.example` 中无 `${MINDOC_`
- [x] 仅设旧 `MINDOC_DB_HOST`、**未**设对应 `DOC_*` 时：**不能**再靠旧变量生效（代码核验：无 `os.Getenv("MINDOC_*")`；example 仅 `${DOC_*}`）
- [x] `DOC_HOME` 由 `ResolveWorkingDirectory` 读取；`DOC_ADDR` / `DOC_PORT` 仅经 conf 占位注入（与 example 一致）
- [x] 默认 `sessionname=doc_id`、`app_key=doc`（example + `config.go`）；升级后旧登录态失效且可重新登录（运行时验证）
- [x] compose / sync / systemd / README 示例已为 `DOC_*`
- [x] CHANGELOG Breaking 已写
- [ ] 仅设 `DOC_DB_HOST` 等关键项时可启动并连库（部署侧仍须勾）
- [ ] 冒烟清单对应项可勾选

---

## 七、PR 建议

可单 PR 落地（env + 默认标识 + 文档）；若过大可拆：

| PR | 内容 |
|---|---|
| env-a | `app.conf*.example` + `config.go` 默认值 |
| env-b | deployments / README / CHANGELOG / 本文 |

可与 T14-a 配置段合并（storage 键一起加）。

---

## 八、参考

- [refactor-roadmap.md](../refactor-roadmap.md) Step 4（历史双读设想；本轮改为硬切）  
- [`conf/app.conf.example`](../../conf/app.conf.example)  
- [round-5-t14-object-storage.md](./round-5-t14-object-storage.md)  
- [round-5-t16-oauth2.md](./round-5-t16-oauth2.md)  
