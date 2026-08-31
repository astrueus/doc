# Round 5 · T12 · 缓存完全重构实施

> 对应 [round-5-execution-plan.md §十三附 T12](./round-5-execution-plan.md#十三附t12--缓存-b-实施闸门)。  
> **依据：** [round-5-cache-evaluation.md](./round-5-cache-evaluation.md)（2026-08-05：完全重构，不兼容旧方案）。  
> **状态：** 🔶 T12-a/b 已落地（2026-08-31）；T12-c/d 业务接入未做。  
> 业务路径仍走旧 `beego/cache` 适配，本切片**不删** `beego_adapter.go`，**未改** `RegisterCache()`。

---

## 一、目标与范围

### 做

1. 新建缓存内核：`Aside[T]` + L1 Ristretto + L2 Redis + singleflight + Soft-TTL + 负缓存 + TTL jitter + Tag + Pub/Sub。  
2. **删除**业务对 `beego/cache` 适配的依赖（[`beego_adapter.go`](../../internal/cache/beego_adapter.go) 可删或仅留测试桩）。  
3. 迁移调用点：Document / Blog / MCP Token → 全部走 `GetOrLoad`；写路径显式 `Invalidate` / `InvalidateTag`。  
4. 配置、指标、文档、压测脚本齐套。

### 不做

- Session 后端替换（可继续 Beego session）。  
- 布隆过滤器（首版负缓存足够；QPS 与扫库打穿再开）。  
- 缓存超大附件二进制（属 T14 对象存储）。

---

## 二、目标目录

```text
internal/cache/
├── aside.go              # Aside[T]、GetOrLoad / Set / Delete / InvalidateTag
├── runtime.go            # Open(ctx, Settings) → Runtime；NewAsideFrom
├── key.go                # KeyBuilder / 前缀 doc:v1:
├── options.go            # TTL / SoftTTL / Jitter / Tags / CacheNull
├── coalesce.go           # singleflight
├── soft_ttl.go           # stale-while-revalidate
├── null_value.go         # 负缓存哨兵
├── metrics.go            # 计数；Snapshot.Map() 日志字段
├── pubsub.go             # Broadcaster + MemoryBus / NoopBus
├── pubsub_redis.go       # Redis Pub/Sub
├── tag.go                # TagIndex + 进程内实现
├── tag_redis.go          # Redis SET tag→keys
├── store/
│   ├── store.go          # Store 接口 Get/Set/Del
│   ├── memory.go         # 单测 L1/L2
│   ├── ristretto.go      # L1
│   └── redis.go          # L2（go-redis；key 不再叠前缀）
├── codec/
│   └── msgpack.go
└── cachetest/            # 内存双层 fake，单测用

# 删除或移出业务路径（T12-c/d）：
# beego_adapter.go / 旧全局 Get/Set 包装（可留 Deprecated 一版编译期删干净）
```

**T12-a/b 已落地（2026-08-31）：** 上表文件均已存在。`Open` / `Runtime` / `NewAsideFrom` 在 `runtime.go`；现网仍走 `beego_adapter.go`，调用方未切。

业务接入示例：

```text
internal/repository/document_repo.go   # Find 外包 GetOrLoad；Update 后 InvalidateTag
internal/mcp/http_auth.go              # token → member
```

---

## 三、核心行为规格

### 3.1 GetOrLoad

```text
1. L1 Get → hit 且未 Soft 过期 → 返回（metric l1_hit）
2. L1 hit 但 Soft 过期 → 返回旧值，并触发异步 refresh（单 flight 去重）
3. L1 miss → L2 Get
4. L2 hit → 回填 L1（短 TTL）→ 若 Soft 过期则同上异步 refresh → 返回
5. L2 miss → sf.Do(key):
     5.1 双检 L1/L2（防等待期间已被他人填充）
     5.2 loader()
     5.3 not found → 写负缓存（短 TTL）并返回哨兵错误
     5.4 ok → 写 L2（TTL±jitter）+ L1 → 返回
6. loader 临时错误 → 不写正缓存；可返回 stale（若有）或 error
```

### 3.2 Soft-TTL / 自动重建

| 字段 | 含义 |
|---|---|
| `TTL` | hard 过期；超过则必须回源 |
| `SoftTTL` | `SoftTTL < TTL`；超过后**仍返回旧值**，后台 loader 重建 |
| 并发 | 同一 key 同时只允许一个 refresh in-flight |

热门文档 / Token：建议 `SoftTTL ≈ 0.8 * TTL`，使峰值流量几乎不撞 hard miss。

### 3.3 失效

```text
Invalidate(keys):
  Del L2 keys + L1 keys + Pub/Sub {op:del, keys:[...]}

InvalidateTag(tag):
  SMEMBERS tagset → Invalidate(members) → DEL tagset
  （写 key 时 SADD tagset key，并设 tagset TTL ≥ max(key TTL)）
```

文档发布 / 更新 / 删除：**必须**打 `tag=book:{bookId}`（及 `document:{id}`）。

### 3.4 默认 TTL 建议

| 域 | L2 TTL | SoftTTL | L1 TTL | 负缓存 |
|---|---|---|---|---|
| Document 元数据 | 10m ±10% | 8m | 20s | 45s |
| Blog | 10m ±10% | 8m | 20s | 45s |
| MCP Token→Member | 5m ±10% | 4m | 15s | 30s |

可配置：`[cache]` section + `DOC_CACHE_*`。

---

## 四、配置

T12-b **增量**写进现有 `[cache]`，保留 `cache_provider` / file / `cache_redis_prefix=doc::cache` 直到 T12-c 切调用方。Aside 前缀与 beego 前缀分开，避免现网 Redis 键冲突。

```ini
[cache]
# —— 现网 beego/cache（T12-c 前仍生效）——
cache = "${DOC_CACHE||false}"
cache_provider = "${DOC_CACHE_PROVIDER||file}"
cache_redis_host = "${DOC_CACHE_REDIS_HOST||127.0.0.1:6379}"
cache_redis_prefix = "${DOC_CACHE_REDIS_PREFIX||doc::cache}"

# —— T12 Aside（Open / Runtime；默认 local，不强制现网 Redis）——
cache_mode = "${DOC_CACHE_MODE||local}"
cache_l1_max_cost = "${DOC_CACHE_L1_MAX_COST||67108864}"
cache_l1_num_counters = "${DOC_CACHE_L1_NUM_COUNTERS||1000000}"
cache_redis_addr = "${DOC_CACHE_REDIS_ADDR}"   # 空则回退 cache_redis_host
cache_pubsub_channel = "${DOC_CACHE_PUBSUB_CHANNEL||doc:cache:invalidate}"
cache_default_jitter = "${DOC_CACHE_DEFAULT_JITTER||0.1}"
cache_aside_prefix = "${DOC_CACHE_ASIDE_PREFIX||doc:v1:}"
```

| `cache_mode` | L1 | L2 | Tag | Bus |
|---|---|---|---|---|
| `local`（默认） | Ristretto | 无 | 进程内 | Noop |
| `redis` | Redis | 无 | Redis SET | Redis Pub/Sub |
| `chain` | Ristretto | Redis | Redis SET | Redis Pub/Sub |

- Redis 数据 key **不再叠一层前缀**（`KeyBuilder` 已是 `doc:v1:...`）。Tag key 形如 `doc:v1:tag:book:12`。  
- `Open` 已实现；**bootstrap 仍未调用**（T12-c 再接）。

---

## 五、PR 拆分

| PR | 内容 | 验收要点 |
|---|---|---|
| **T12-a** | Store + Aside 内核 + 单测（含 stampede / 负缓存 / soft refresh） | **2026-08-31 已完成** |
| **T12-b** | Redis L2 + Tag + Pub/Sub；conf；metrics；`Open` | **2026-08-31 已完成**（miniredis / MemoryBus 单测；业务未切） |
| **T12-c** | Document / Blog 接入；model 内旧 cache 调用删除 | 阅读 / 发布冒烟；按书失效正确 |
| **T12-d** | MCP Token 接入 + 压测脚本 / 文档 | 登录态 / MCP 不回归；README 运维说明 |

可选并行：**T12-engine-jetcache**：若选 jetcache-go 作引擎，在 a 中换 store 实现，业务 API 不变。

---

## 六、与分层（T9）的衔接

| 规则 | 说明 |
|---|---|
| 缓存边界 | **Repository 或薄 service** 内 GetOrLoad；Controller / MCP handler **不**直接碰 cache |
| model | 删除 `DocumentModel` 内嵌 cache 读写，避免双路径 |
| 测试 | Repo 单测注入 `cachetest` 内存实现 |

---

## 七、压测与观测（上线门槛）

### 指标

- T12-b：`MetricsSnapshot.Map()` → `cache_l1_hit` / `cache_l2_hit` / `cache_miss` / `cache_load` / `cache_load_shared` / `cache_load_err` / `cache_null_hit` / `cache_soft_refresh`（写日志即可）  
- T12-d 可选：Prometheus 刮取、`cache_load_seconds` histogram、L1 cost / Redis pool 等待  

### 压测场景

1. 单 key 1k 并发 miss → DB QPS ≈ 1（或实例数，若未开跨机租约）。  
2. Soft-TTL 窗口内 QPS 稳定，load 以 refresh 计数为主。  
3. 随机不存在 id 穿透 → DB QPS 被负缓存削峰。  
4. 整点大量 key 过期模拟 → jitter 后回源曲线平滑。

### 上线

- 新前缀 `doc:v1:`，无需迁移旧 key；部署后旧 `Document.Id.*` 自然失效。  
- CHANGELOG Breaking：缓存实现与配置项变更。  
- 回滚：`mode=local` 或临时关闭 cache（Aside no-op）开关。

---

## 八、验收清单

- [ ] 业务路径零 `beego/cache` 依赖（Session 除外）— **T12-c/d**  
- [ ] Document / Blog / Token 均经 `GetOrLoad`；写后 Invalidate/Tag 有单测 — **T12-c/d**  
- [x] 击穿 / 穿透 / Soft refresh / jitter 有自动化测试（`aside_test.go`；压测记录仍待 T12-d）  
- [x] Pub/Sub 多实例 L1 失效（`TestAsidePubSubFlushesPeerL1` / `TestOpenChainPubSubFlushesPeerL1`；手册见下文附录）  
- [x] metrics 日志字段：`MetricsSnapshot.Map()`（`cache_l1_hit` 等）；Prometheus 刮取非必须，未做  
- [x] conf.example + `DOC_CACHE_MODE` 等新键 + env 文档 — **T12-b**  

---

## 九、工作量粗估

| 切片 | 人天 |
|---|---|
| T12-a 内核 | 2~3 |
| T12-b Redis/Tag/PubSub | 1~2 |
| T12-c/d 业务接入 + 压测文档 | 1~2 |
| **合计** | **4~7 天** |

（若直接嵌 jetcache-go，a+b 可压缩约 1~1.5 天，但要验收 Facade 可替换性。）

---

## 附录 · 双实例 L1 失效手册（T12-b）

T12-c 之前 **bootstrap 不调用 `Open`**，现网双进程不会走新内核。验证以单测为准；手工 compose 可选。

### 已覆盖的自动化

| 测试 | 说明 |
|---|---|
| `TestAsidePubSubFlushesPeerL1` | 共享 Memory L2 + MemoryBus；A `Delete` 后 B 的 L1 为空 |
| `TestOpenChainPubSubFlushesPeerL1` | 两套 `Open(mode=chain)` + miniredis；同上 |

有 Bus 的 `Aside` / `Runtime` 须 `Close()`（单测已 `t.Cleanup`）。

### 手工（T12-c 接入后）

1. Redis：`docker run --rm -p 6379:6379 redis:7`  
2. 两份进程同一 `DOC_CACHE_MODE=chain`、`DOC_CACHE_REDIS_ADDR`  
3. 实例 A 走写路径（会 `Delete` / `InvalidateTag`）  
4. 实例 B 下一读不应继续命中本机 L1（可对日志字段 `cache_l1_hit` / `cache_miss`）

不必为验证 T12-b 真跑双容器。

---

## 十、参考

- [round-5-cache-evaluation.md](./round-5-cache-evaluation.md)  
- [dgraph-io/ristretto](https://github.com/dgraph-io/ristretto)  
- [redis/go-redis](https://github.com/redis/go-redis)  
- [mgtv-tech/jetcache-go](https://github.com/mgtv-tech/jetcache-go)（可选引擎）  
- [`golang.org/x/sync/singleflight`](https://pkg.go.dev/golang.org/x/sync/singleflight)  
