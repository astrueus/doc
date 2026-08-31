# Round 5 · T1 · 缓存完全重构评估（高性能 / 可扩展 / 防雪崩）

> 对应 [round-5-execution-plan.md §三 T1](./round-5-execution-plan.md#三t1--缓存方案评估报告0.5~1-天)。  
> **决策前提（2026-08-05）：** **不考虑**与现有 `beego/cache` / 旧 key / 旧接口兼容；以**完全重构**为目标，追求：  
> 极致读性能、高并发、易扩展、开发体验好、**自动重建（soft refresh）**、系统防护击穿 / 穿透 / 雪崩。  
> 落地切片见 [round-5-t12-cache-impl.md](./round-5-t12-cache-impl.md)。  
> **状态：** ✅ 评估结论已定（见 §六）。

---

## 一、现状债（为何要推倒重来）

| 问题 | 现状 | 影响 |
|---|---|---|
| 框架绑死 | [`beego/cache`](../../internal/cache/beego_adapter.go) 适配 | 与 Beego 生命周期耦合；能力停在「裸 K/V」 |
| 无回源编排 | 调用方各自 `Get` miss 再查库 | 高并发下易**击穿** |
| 无负缓存 | 查不到也反复打库 | **穿透** |
| TTL 齐步失效 | 文档类统一 1h | 热点 key 同时过期 → **雪崩** |
| 无分组失效 | 无 tag / 索引 | 发布文档、改书需手搓删 key |
| 无观测 | 无 hit/miss/load | 无法压测与容量规划 |
| API 偏底层 | `Get/Set/Delete` | 业务易写错「先查再载」；难统一防护 |

调用点虽少（Document / Blog / MCP Token），但都是**读多写少、权限敏感**路径——正好适合做「一次做对」的缓存基座，而不是打补丁。

---

## 二、目标能力清单（验收级）

| # | 能力 | 说明 |
|---|---|---|
| G1 | **L1 + L2** | 进程内微秒级 + Redis 跨实例共享 |
| G2 | **GetOrLoad 一等公民** | 业务只写 loader，框架处理 miss/合并/回填 |
| G3 | **防击穿** | 进程内 `singleflight`；多实例可选 Redis 租约（热点 key） |
| G4 | **防穿透** | 空值 / not-found 占位短 TTL；可选布隆（大表场景再上） |
| G5 | **防雪崩** | TTL = base ± jitter；禁止大量 key 同一秒过期 |
| G6 | **自动重建** | Soft-TTL / stale-while-revalidate：未过 hard TTL 时返回旧值，后台刷新 |
| G7 | **语义失效** | Tag / 前缀版本号：`book:{id}` 一键失效；写后主动 Invalidate |
| G8 | **跨实例刷 L1** | Redis Pub/Sub（或 keyspace）通知他节点删本地 |
| G9 | **可观测** | hit/miss/load/err/shared、p99 load、L1/L2 分层命中；可接 Prometheus |
| G10 | **开发体验** | 类型安全（泛型）、统一 key builder、禁止业务手搓 Redis |

**非目标（本轮不做）：**  
Session 存储替换（可继续 Beego session）；分布式事务缓存；全站 HTTP CDN 缓存正文。

---

## 三、市场常用方案与最佳实践（2025–2026）

### 3.1 业界共识架构

几乎所有高并发读服务最终收敛到同一形态：

```text
请求 → L1 本地缓存（Ristretto / TinyLFU / FreeCache）
         ↓ miss
       L2 Redis（go-redis）
         ↓ miss
       singleflight 合并回源 → DB / RPC
         ↓
       回填 L2 + L1（TTL + jitter；可选负缓存）
```

配套实践（多家工程博文 / Redis 模式一致）：

1. **Cache-Aside** 为主（应用管 miss），写路径 **主动失效** 优于超长 TTL 碰运气。  
2. **L1 TTL ≪ L2 TTL**（如 L1 10–30s，L2 5–15min），限制跨实例脏读窗口。  
3. **singleflight 是底线**；跨机「一击穿一 query」用短租约，不必默认上沉重分布式锁。  
4. **Soft refresh / early refresh**：热 key 在过期前异步重建，把 P99 从「等 DB」打成「本地命中」。  
5. **观测优先于玄学调参**：没有 hit rate 就不要谈「极致」。

### 3.2 候选库对照（完全重构视角）

| 方案 | 定位 | L1+L2 | 击穿 | 穿透 | 自动刷新 | Tag/失效 | 扩展性 | 社区 / 风险 |
|---|---|---|---|---|---|---|---|---|
| **自研 Facade + Ristretto + go-redis** | 薄内核、自控 | ✅ 自建 | singleflight + 可选租约 | 负缓存 | Soft-TTL 自实现 | Tag Set / 版本号 | **最高**（接口自有） | 要自己写测试；依赖面清晰 |
| **[mgtv-tech/jetcache-go](https://github.com/mgtv-tech/jetcache-go)** | 生产级二层框架（类 Java JetCache） | ✅ | ✅ Once/singleflight | ✅ not-found | ✅ auto-refresh | 有本地失效广播 | 高（接口驱动） | 国内生产验证；绑 go-redis |
| **[eko/gocache](https://github.com/eko/gocache)** | 多 store / chain / tag | ✅ chain | 需自包一层 | 需自做 | 弱 | ✅ 原生 tag | 中高 | 国际活跃；刷新策略要补 |
| **[viccon/sturdyc](https://github.com/viccon/sturdyc)** | 进程内抗雪崩 + early refresh | 偏 L1 | ✅ 强 | ✅ | ✅ 很强 | 弱（无 Redis 原生） | 中（多实例要再叠 Redis） | 刷新体验极好；跨机要自己拼 |
| **go-redis/cache 裸用** | Redis 旁路 | 仅 L2 | 需自包 | 需自做 | 需自做 | 手写 | 低 | 太薄 |
| **bigcache / 纯 map** | 极致本地吞吐 | 仅 L1 | 无 | 无 | 无 | 无 | 低 | 不适合多实例文档站 |
| **继续 beego/cache** | 遗留 | 弱 | 无 | 无 | 无 | 无 | 低 | **否决** |

### 3.3 与本项目的匹配

| 需求 | 含义 | 最贴方案 |
|---|---|---|
| 极致性能 | 热点读尽量不打 Redis/DB | L1 Ristretto + Soft refresh |
| 高并发 | 失效瞬间不打爆 DB | singleflight（+ 可选 Redis 租约） |
| 多实例 / 后续水平扩展 | 节点间共享 + 本地加速 | **L2 Redis 必选** |
| 方便开发 | 一行 GetOrLoad | 统一 `cache.Aside[T]` / `Once` API |
| 自动重建 | 热数据后台刷 | Soft-TTL 或 jetcache-go refresh / sturdyc early refresh |
| 最佳扩展 | 换 L1 实现、加 metrics、接 kratos data | **自有 Facade**，引擎可换 |

---

## 四、推荐架构（完全重构）

### 4.1 结论一句话

**采用「自研 `internal/cache` Facade + Ristretto(L1) + go-redis(L2) + 内置防护套件」为默认路线；**  
实现时可**对照 / 局部借用** jetcache-go 的 Once、负缓存、统计思路，但**业务只依赖我们自己的接口**，避免框架锁死。

若人力极紧、希望开箱即用：允许 **T12-a 直接以 jetcache-go 为引擎**，外包一层 `internal/cache`，日后仍可替换实现。

### 4.2 逻辑架构

```text
                    ┌──────────────────────────────────────┐
  biz / repository  │  cache.Aside[T].GetOrLoad(ctx, key,   │
                    │       opts, loader)                   │
                    └───────────────┬──────────────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              ▼                     ▼                     ▼
         KeyBuilder            SoftTTL/Jitter        Metrics
         TagIndex              NegCache              Tracer
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
              L1 Ristretto                    L2 Redis
              (短 TTL)                        (长 TTL + msgpack/JSON)
                    ▲                               │
                    └──── Pub/Sub invalidate ───────┘
```

### 4.3 读写语义

| 操作 | 行为 |
|---|---|
| **读** | L1 → L2 →（miss）singleflight(loader) → 写 L2+L1；若 Soft 过期但仍在 hard TTL 内：返回旧值 + 异步 refresh |
| **写后** | 业务更新 DB 成功后 `Invalidate(keys...)` 或 `InvalidateTag("book:"+id)`；发布 Pub/Sub 刷他机 L1 |
| **空结果** | `ErrNotFound` → 缓存 `null` 哨兵，TTL 短（如 30–60s） |
| **错误** | loader 临时错误**不**缓存（或极短熔断），避免毒缓存 |

### 4.4 关键 API（开发体验）

```go
// 业务侧只看到这些（示意）
type Aside[T any] interface {
    GetOrLoad(ctx context.Context, key string, opt Options, load func(context.Context) (T, error)) (T, error)
    Set(ctx context.Context, key string, v T, opt Options) error
    Delete(ctx context.Context, keys ...string) error
    InvalidateTag(ctx context.Context, tags ...string) error
}

opt := cache.Options{
    TTL:        10 * time.Minute,
    SoftTTL:    8 * time.Minute,  // 超过后后台重建
    Jitter:     0.1,             // ±10%
    Tags:       []string{"book:12"},
    CacheNull:  true,
}
doc, err := documentCache.GetOrLoad(ctx, key, opt, func(ctx context.Context) (Document, error) {
    return repo.Find(ctx, id)
})
```

禁止业务直接 `redis.Get`；禁止再出现「手写 Get + 判空 + Set」散落各处。

### 4.5 Key / Tag 规范（全新前缀）

```text
doc:v1:document:id:{id}
doc:v1:document:book:{bookId}:ident:{identify}
doc:v1:blog:id:{id}
doc:v1:mcp:token:{sha256}
tag → Redis SET  doc:v1:tag:book:{bookId}  members=keys
```

版本位 `v1` 便于将来破坏性变更时整前缀废弃，无需兼容旧 `Document.Id.*`。

### 4.6 防护映射

| 漏洞 | 机制 |
|---|---|
| **击穿**（热 key 过期） | singleflight；Soft-TTL 后台刷新使热 key「几乎不过期」；可选 Redis `SET NX` 短租约限制跨实例回源 |
| **穿透**（查不存在） | 负缓存；非法 id 在业务层直接拒（参数校验） |
| **雪崩**（大批同时过期） | TTL jitter；错峰 Soft refresh；Redis 连接池 / 限流回源 |
| **毒缓存** | loader error 不写正缓存；解码失败删 key |
| **大 key / 污染** | L1 cost 按 payload 估算；禁止缓存超大 Markdown 全文时可改为缓存「元数据 + 短摘要」，全文另策略（见 §五） |

### 4.7 文档正文是否进缓存

| 策略 | 说明 |
|---|---|
| **A. 元数据进缓存，正文短 TTL 或按版本号** | 推荐：title/identify/version 长缓存；`markdown`/`release` 用 `version` 参与 key 或短 Soft-TTL |
| **B. 全文进 L2** | 实现简单，内存/带宽压力大；L1 慎入 |
| **C. 仅 Token / 权限对象** | 太保守，浪费重构红利 |

**默认建议：Document 缓存「结构化视图」+ version；正文命中用 version 校验，变更即 InvalidateTag。**

---

## 五、方案比选与终裁

| 维度 | 自研 Facade + Ristretto + Redis | 直接 jetcache-go | gocache chain | sturdyc(+Redis) |
|---|---|---|---|---|
| 性能上限 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐（本地） |
| 防护完备 | 自建齐套 | 开箱齐 | 需补刷新/负缓存 | 刷新强，跨机弱 |
| 扩展 / 换引擎 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| 开发便利 | API 自定最贴业务 | Once API 成熟 | 中 | GetOrFetch 很好 |
| 交付风险 | 中（要写对 SoftTTL） | 低 | 中 | 中高（拼 L2） |
| 与 kratos data 层 | 天然适配 | 适配一层即可 | 可 | 可 |

### 终裁

| 项 | 结论 |
|---|---|
| **路线** | **完全重构**：废弃 beego 适配为业务缓存后端 |
| **默认实现** | **自研 Facade**；L1=Ristretto，L2=go-redis，防护= singleflight + Soft-TTL + 负缓存 + jitter + Tag + Pub/Sub |
| **备选加速** | 若 T12 工期紧：引擎先用 **jetcache-go**，Facade 包一层，接口不变 |
| **否决** | 维持 A、仅 singleflight 补丁、file cache、无 Redis 的多实例方案 |
| **T12** | **升格为实施项（非「维持 A 则关闭」）**；见实施文档 |

---

## 六、对执行计划的影响

- T1：本报告结论 = **全面重构**，不再走「观测一周再决定维持 A」。  
- T12：按 [round-5-t12-cache-impl.md](./round-5-t12-cache-impl.md) 分 PR 落地；可与 T9（Repo）协同——**缓存调用下沉到 repository / service**，Controller/model 不再直操 cache。  
- Session：可暂留 Beego；与业务缓存分离配置（`[cache]` vs session）。  
- 环境变量：Redis 等键走 `DOC_CACHE_*` / `DOC_REDIS_*`（[env 硬切](./round-5-env-mindoc-to-doc.md)）。

---

## 七、验收（评估文档本身）

- [x] 市场方案与威胁模型写清  
- [x] 终裁：完全重构 + Facade + L1/L2 + 防护套件  
- [x] 决策记入执行计划 §一附 / 追踪表  
- [x] T12 按新实施文档开工（T12-a 2026-08-31）  

---

## 八、参考

- [Building a High-Performance Cache Layer in Go with Redis (2026)](https://dev.to/young_gao/building-a-high-performance-cache-layer-in-go-2ejd)  
- [Caching Strategies at Scale](https://backendbytes.com/articles/caching-strategies-at-scale/)  
- [singleflight 与 stampede](https://medium.com/pickme-engineering-blog/singleflight-in-go-a-clean-solution-to-cache-stampede-02acaf5818e3)  
- [Ristretto](https://github.com/dgraph-io/ristretto) · [jetcache-go](https://github.com/mgtv-tech/jetcache-go) · [sturdyc](https://github.com/viccon/sturdyc) · [eko/gocache](https://github.com/eko/gocache) · [go-redis](https://github.com/redis/go-redis)  
- 现状：[`internal/cache/`](../../internal/cache/)  
- 实施：[round-5-t12-cache-impl.md](./round-5-t12-cache-impl.md)  
