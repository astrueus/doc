# Round 5 · T1 · 缓存方案评估报告

> 对应 [round-5-execution-plan.md §三 T1](./round-5-execution-plan.md#三t1--缓存方案评估报告0.5~1-天)。  
> **定位：** 只出评估结论；若结论为「维持 A」，则关闭 T12；若结论为「上 B」，则 T12 依此实施。  
> **状态：** ⏳ 待写结论。

---

## 一、现状（方案 A）

### 1.1 抽象层

- 接口 [`internal/cache/iface.go`](../internal/cache/iface.go)：`Cache` 提供 `Get / Set / Delete / IsExist / Incr / Decr / Clear`
- 适配层 [`internal/cache/beego_adapter.go`](../internal/cache/beego_adapter.go)：包装 `beego/cache` 后端；**值统一用 msgpack 编解码**（脱离 gob，跨版本重构可控）
- 全局入口 [`internal/cache/cache.go`](../internal/cache/cache.go)：`cache.Default` + 包级函数
- 空实现 [`internal/cache/cache_null.go`](../internal/cache/cache_null.go)：便于禁用/测试

### 1.2 后端

由 `beego/cache` 提供 `memory / file / memcache / redis` 四种，配置见 [`internal/config/config.go` §CacheSection](../internal/config/config.go)（`cache_provider` / `cache_redis_*` 等）。

### 1.3 当前调用点

| 位置 | key 形态 | TTL | 用途 |
|---|---|---|---|
| [`internal/model/DocumentModel.go:193/197/208/211/219/240`](../internal/model/DocumentModel.go) | `Document.Id.<id>` / `Document.BookId.<bid>.Identify.<ident>` | 1h | 文档缓存 |
| [`internal/model/Blog.go:124/137/257`](../internal/model/Blog.go) | Blog key | 1h | 博客详情 |
| [`internal/mcp/http_auth.go:48/80~108`](../internal/mcp/http_auth.go) | `tokenCacheKey(hash)` | 5min | API Token → Member 缓存 |

### 1.4 已知问题 / 缺口

1. **无 singleflight**：热点 key 失效瞬间可能穿透，多请求并发查库 + 回填。  
2. **无 tag / 分组失效**：一本书下所有文档改标题，需要遍历删 key；MCP `release_document` 后未做批量失效。  
3. **msgpack 兼容性隐忧**：结构体字段增删需靠 `msgpack` 标签或字段名对齐；漏改导致解码 error（已见于历史 log）。  
4. **`cache.Default == nil` 保护**：调用点分散，各自判空（`http_auth.go` 有，`DocumentModel.go` 无）；`null` 后端应作为默认注入以简化调用侧。  
5. **观测缺失**：无命中率、无回源耗时、无穿透计数；无法客观判断「上 B」的收益。

---

## 二、候选 B

### 2.1 [`eko/gocache/v3`](https://github.com/eko/gocache)

- **能力**：多 store（memory/redis/…）、chain、tag、metrics 钩子
- **契合点**：tag 能一次失效「同本书所有文档」；chain 可做 L1(memory) + L2(redis) 组合
- **代价**：
  - 引入新依赖树；`beego/cache` 大概率仍留（Session / 现有代码），双栈期成本
  - 需要把 msgpack marshal 层保留（`gocache` 值不做序列化约定）
  - API 与现有 `cache.Cache` 差异大，需要在适配层重新实现或渐进迁移

### 2.2 [`singleflight`](https://pkg.go.dev/golang.org/x/sync/singleflight) + 手工 tag

- **能力**：解决穿透；tag 用「本地 map + delete key list」自实现
- **契合点**：改动最小；只在读路径包一层 `sf.Do(key, load)`，写路径统一走 `invalidateByTag`
- **代价**：tag 需要自己写 + 维护一致性（key ↔ tag 双向索引）；进程内 tag 在多实例场景失效需要 pub-sub 兜底

### 2.3 对照表

| 维度 | 方案 A（现状） | A + singleflight | 上 gocache |
|---|---|---|---|
| 抗穿透 | ❌ | ✅ | ✅ |
| tag 失效 | 手写循环 delete | 手写 tag map | 原生 |
| 观测钩子 | ❌ | 手动打点 | 中间件式 |
| 多实例一致性 | 靠 redis 后端 | 需 pub-sub | chain + redis |
| 依赖增量 | 0 | `x/sync`（已有） | `eko/gocache/v3` + 各 store |
| 学习/改造成本 | 0 | 极小 | 中 |
| 与 msgpack 序列化 | 已有 | 保留 | 保留（自行包装） |

---

## 三、观测建议（上 B 前必做）

拍板上 B 之前应有一份**至少 1 周**的观测数据：

1. **命中率**：按 key 前缀（`Document.Id.*` / `Blog.*` / `Token.*`）分类
2. **回源耗时**：p50 / p95 / p99
3. **穿透次数**：相同 key 在窗口内并发回源次数
4. **失效频率**：单位时间内 `Delete` 调用次数
5. **进程内存/GC**：memory 后端下的对象数与内存占用

**实施建议：** 在 [`internal/cache/beego_adapter.go`](../internal/cache/beego_adapter.go) 里加 4 行 counter（可复用未来 metrics 出口），成本 <0.5 天。

---

## 四、结论矩阵

| 观测结果 | 建议 |
|---|---|
| 命中率 >90%，穿透 <10/日，无 tag 需求 | **维持 A**；关闭 T12 |
| 命中率 >90%，穿透突刺明显 | **A + singleflight**（小 T12） |
| 命中率 <70% 或 tag 强需求（如 T14 后附件缓存） | **上 gocache**（完整 T12） |
| 无稳定观测数据 | **维持 A + 补观测**；T12 挂起 |

---

## 五、验收

- [ ] 报告合入 `docs/`
- [ ] [round-5-execution-plan.md §八 决策日志](./round-5-execution-plan.md#十四追踪表) 记一笔
- [ ] 结论：
  - 维持 A → 关闭 / ⏭ T12
  - 上 A + singleflight → T12 定义为「加 singleflight + 观测埋点」
  - 上 gocache → T12 定义为「gocache/chain + tag」，须给最小切片（先做 `Document.*` 一族）

---

## 六、默认建议（未观测前）

**A + singleflight（含最小观测埋点）**。理由：

1. 现有调用点少（3 个模块），穿透风险主要在 token 缓存与文档热点；singleflight 的 ROI 最高。
2. tag 失效在 T14 对象存储 CDN 场景才明显有价值，本轮不见得需要。
3. 保留继续升级到 gocache 的空间；不设置技术债路障。

---

## 七、参考

- [`internal/cache/`](../internal/cache/) — 现有实现
- [round-5-execution-plan.md §三 T1 / §十三附 T12](./round-5-execution-plan.md)
- [eko/gocache/v3](https://github.com/eko/gocache)
- [`golang.org/x/sync/singleflight`](https://pkg.go.dev/golang.org/x/sync/singleflight)
