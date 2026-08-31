# Round 5 · T12 缓存运维与压测

> 实施细节见 [round-5-t12-cache-impl.md](./round-5-t12-cache-impl.md)。  
> 本文只写**怎么开、怎么看、怎么压、怎么回滚**。

---

## 一、开关

`[cache]` 里 `cache=true`（或环境变量 `DOC_CACHE=true`）才会 `Open` Aside。`cache=false` 时文档 / 博客 / MCP Token **直打库**。

| 键 | 建议 | 说明 |
|---|---|---|
| `cache_mode` | 单机 `local`；多实例 `chain` | `local` 不必 Redis |
| `cache_redis_addr` | 空则回退 `cache_redis_host` | 仅 `redis` / `chain` 需要 |
| `cache_aside_prefix` | 保持 `doc:v1:` | 不要改现网 `cache_redis_prefix` |
| `cache_provider` | 可忽略 | T12-d 起不再初始化 beego 业务缓存 |

启动日志：

- 开启：`Aside 缓存已就绪, mode= local`（或 `chain` / `redis`）
- 关闭：`Aside 缓存未开启（cache=false）。`
- Redis 连不上：`初始化 Aside 缓存失败，文档/博客/Token 将直打库`（进程不退出）

Session 仍走 Beego session（`sessionprovider`），与 Aside 无关。

热加载 `config_auto_delay` 会重跑 `RegisterCache` / `RegisterAside`；改 `cache_mode` 后更稳妥的是重启 `just run`。

---

## 二、缓存了什么

键前缀 `doc:v1:`（`cache_aside_prefix`）。

| 域 | key | TTL / Soft / L1 / 负缓存 |
|---|---|---|
| Document | `document:id:{id}`、`document:book:{bookId}:ident:{identify}` | 10m / 8m / 20s / 45s |
| Blog | `blog:id:{id}` | 同上 |
| MCP Token | `mcp:token:{sha256hex}` | 5m / 4m / 15s / 30s |

写路径必须失效：文档/博客走 model 钩子；Token **吊销**走 `repository.InvalidateAPIToken`。

未缓存：目录树、列表、搜索、历史、图书/成员、Session、附件、阅读次数。

Token 缓存不含密码；命中 L1/L2 时不写 `last_used`（与旧 beego 命中行为一致，避免每个 MCP 请求打库）。回源（miss / Soft 刷新）会异步更新 last_used。

---

## 三、指标

`cache.Kernel().Metrics.Snapshot().Map()` 字段：

`cache_l1_hit` / `cache_l2_hit` / `cache_miss` / `cache_load` / `cache_load_shared` / `cache_load_err` / `cache_null_hit` / `cache_soft_refresh`

本轮不刮 Prometheus。单测与压测脚本用上述计数验收击穿（并发 miss 回源 ≈ 1）。

---

## 四、压测

仓库脚本（无 Redis、不打真实端口）：

```bash
bash deployments/scripts/cache_load_test.sh
# Windows：
powershell -File deployments/scripts/cache_load_test.ps1
```

覆盖：`TestAsideStampedeSingleLoad`（64 并发 miss 回源 1 次）、负缓存、Soft-TTL、Document/Blog/Token 接入单测，以及 `BenchmarkAsideGetOrLoadParallel`。

手工 HTTP（进程已 `just run`，`cache=true`）：

1. 文档：连续打开同一 `/docs/{book}/{doc}`，第二次应明显更快；保存发布后再读应为新内容。  
2. MCP：同一 Bearer 连打 `/mcp`（`initialize` 即可）。吊销 Token 后下一请求须 401。  
3. 可选 `hey`：`hey -n 2000 -c 50 -m POST -H "Authorization: Bearer doc_..." http://127.0.0.1:8181/mcp`（路径以实际 MCP HTTP 为准）。观察库连接与日志，不应出现与并发同量级的 Token 查询。

多实例 L1 失效：`cache_mode=chain` + Redis，见实施文档附录。

---

## 五、回滚

1. `cache=false` 或 `DOC_CACHE=false`：全部直打库。  
2. 仅单机：`cache_mode=local`（关掉 Redis L2 / Pub/Sub）。  
3. 无需迁移旧 `Document.Id.*` / `mcp:tok:` 键；新前缀 naturally 冷启动。
