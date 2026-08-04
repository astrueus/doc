# Round 5 · T10 · DocumentController 域拆分 — 细化方案

> 对应 [round-5-execution-plan.md §十二 T10](./round-5-execution-plan.md#十二t10--controller-域拆分可选)。  
> **状态：** ⏸ **本轮暂不拆**（2026-08-04）。下文仅作**解冻后**的方向备忘，避免将来误用「平铺多文件」方案。

---

## 一、本轮决策

| 项 | 结论 |
|---|---|
| Round 5 是否实施 | ❌ **暂不拆**；`DocumentController.go` 维持现状 |
| 理由 | 体量大但非阻塞；与 T8/T9/T14 并行易冲突；先把 Repo/Service 与上传路径理顺再动 Controller 边界更划算 |
| 解冻条件 | T9（读路径经 Repo）告一段落，且有明确「按子域拆包」方案评审通过后再开 |

本轮验收：

- [x] 追踪表标 ⏸；无拆分 PR  

---

## 二、现状（备忘）

- [`internal/controller/DocumentController.go`](../internal/controller/DocumentController.go) ≈ **1296 行**  
- 方法大致分组：读页 / 写 / 附件 / 历史 / 导出 / 私有校验  

---

## 三、解冻后方向（禁止平铺）

### 3.1 明确不采用

**不要**在 `internal/controller/` 下平铺拆成：

```text
# ❌ 不采用
DocumentReadController.go
DocumentEditController.go
DocumentAttachController.go
...
```

也不推荐「同包 `package controller` + 仅物理拆多个 `.go` 但仍挤在同一目录」当成最终形态——可读性提升有限，包边界仍模糊。

### 3.2 推荐形态：子包按域拆（非平铺）

解冻后优先：

```text
internal/controller/
├── base.go                    # BaseController 等共用
├── book.go / member.go / …    # 其它控制器可暂留
└── document/                  # 子包：document 域 HTTP 适配
    ├── controller.go          # 类型、路由注册入口、共用依赖注入
    ├── read.go                # Index / Read / Content / Search / …
    ├── edit.go
    ├── attach.go
    ├── history.go
    └── export.go
```

要点：

1. **`package document`**（子包），不是把文件平铺回父目录。  
2. 路由仍由应用层显式注册；**URL / 方法名行为不变**（纯搬迁）。  
3. 与 [T2 / T9](./round-5-orm-migration-evaluation.md) 对齐：Controller 只做编解码与鉴权入口，业务逐步进 `service`/`biz`，避免「拆文件却继续堆逻辑」。  
4. 可选更进一步：按 **独立 Controller 类型**（`ReadController` / `EditController`）分结构体，而不是一个上帝类型 + 多文件——若 Beego 路由允许，解冻评审时再定。

### 3.3 与其它任务顺序

| 前置 | 说明 |
|---|---|
| T9 | MCP/Web 读路径经 Repo 后，搬迁时引用面更清晰 |
| T14-b | 上传点切换宜在拆包**前**或**严格串行**，避免双线改 `Upload` |
| T8 | Result→dto 可与拆包无关；不必绑死 |

---

## 四、解冻后验收（届时）

- [ ] 采用 **子包**（或等价非平铺结构），非根目录平铺多文件  
- [ ] 路由 URL 与 HTTP 状态与拆分前一致  
- [ ] 无行为变更（diff 以 move 为主）  
- [ ] 单文件行数可维护；包职责写进简短包注释  

---

## 五、明确不做（本轮 + 解冻约束）

- Round 5：**不实施**拆分 PR  
- 解冻后：**不采用** `DocumentXxxController.go` 平铺方案  
- 不借机改业务逻辑 / 换模板  
- 不借机全站 Controller 大拆  

---

## 六、参考

- [`internal/controller/DocumentController.go`](../internal/controller/DocumentController.go)  
- [round-5-t9-repo-service.md](./round-5-t9-repo-service.md)  
- [round-5-orm-migration-evaluation.md §6.2](./round-5-orm-migration-evaluation.md#62-round-6-分阶段规划既定路径非可选项)  
