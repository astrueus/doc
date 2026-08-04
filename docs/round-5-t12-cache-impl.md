# Round 5 · T12 · 缓存方案 B 实施 — 细化方案（闸门）

> 对应 [round-5-execution-plan.md §十三附 T12](./round-5-execution-plan.md#十三附t12--缓存-b-实施闸门)。  
> **依赖：** [T1 评估报告](./round-5-cache-evaluation.md) + 至少约 1 周观测数据。  
> **状态：** ⏳ 闸门（无数据则 ⏭）。

---

## 一、触发条件

| T1 结论 | T12 动作 |
|---|---|
| 维持 A | **关闭 / 跳过** T12 |
| A + singleflight | 实施本文件 §二 |
| 上 gocache | 实施本文件 §三 |
| 无观测数据 | **挂起**；先做 T1 §三观测埋点 |

默认建议（无数据前）：**不要开工 T12 完整项**；若只差埋点，把「观测 counter」算作 T1 收尾而非 T12。

---

## 二、切片 S：`singleflight` + 观测（推荐小 T12）

### 做

1. 在 [`internal/cache`](../internal/cache/) 增加 `GetOrLoad(key, ttl, load func() (any, error))`，内部 `singleflight.Group.Do`。  
2. 热点调用点切换：  
   - `DocumentModel` 文档缓存  
   - `Blog` 详情  
   - MCP `http_auth` token → member  
3. 埋点：hit / miss / singleflight_shared / load_err（日志字段或原子计数；可后续接 metrics）。  
4. **不**引入 tag 系统（除非观测证明需要）。

### 验收

- [ ] 并发打同一 miss key，回源次数 ≈ 1  
- [ ] 功能回归：文档 / 博客 / MCP token  
- [ ] 无新依赖（除已有 `x/sync`）  

---

## 三、切片 G：`eko/gocache`（完整 T12）

仅当 T1 明确要求。

### 最小范围

1. 适配层：保留现有 `cache.Cache` 接口，底层换 gocache store（memory / redis）。  
2. **先只迁** `Document.*` 一族；Blog / Token 第二 PR。  
3. tag：`book:<id>` 绑文档 key；书内批量失效走 tag。  
4. msgpack 编解码仍放适配层。  
5. 双栈期：Session 等可继续 `beego/cache`，文档写清。

### 验收

- [ ] Document 缓存命中与失效正确  
- [ ] 按书 tag 失效验证  
- [ ] 性能不低于方案 A（同压测脚本）  

---

## 四、上线注意

- 清 file cache 目录 / redis 相关前缀（若 key 格式变）  
- CHANGELOG 注明  
- 冒烟清单勾选「若动缓存：清 session + cache」  

---

## 五、参考

- [round-5-cache-evaluation.md](./round-5-cache-evaluation.md)  
- [`internal/cache/`](../internal/cache/)  
- [eko/gocache](https://github.com/eko/gocache)  
