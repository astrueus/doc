# Round 5 · T7 · 测试工程化 — 细化方案

> 对应 [round-5-execution-plan.md §九 T7](./round-5-execution-plan.md#九t7--测试工程化1~2-天)。  
> 脚本落点以 [T15](./round-5-scripts-layout.md) 为准（**方案 A** → `deployments/scripts/test.sh`；本地可用根目录 `make test` / `just test`）。  
> **状态：** ⏳ 待实施。

---

## 一、现状

- **无** 测试入口脚本 / Makefile 测试目标 / Gitea·GitHub Actions  
- 已有 `*_test.go` 约 18 个：主要在 `pkg/*`、`internal/{auth,errs,i18n,logging,mcp,repository}`  
- 覆盖率基线文档：[round-4-coverage.md](./round-4-coverage.md)（若存在则继承；否则本任务新建快照）

---

## 二、脚本

### 路径（T15 = A）

| 文件 | 用途 |
|---|---|
| `deployments/scripts/test.sh` | Linux/macOS / CI |
| `deployments/scripts/test.ps1`（或 `test.bat`） | Windows 本地 |

> 建议在 T15 搬迁 PR 之后（或同一 PR）新增；**不要**先写到根 `scripts/`。

### 行为

```bash
# 默认：race + cover，包白名单起步
./deployments/scripts/test.sh

# 可选环境变量
TEST_PKGS='./pkg/... ./internal/errs/... ...'   # 覆盖默认白名单
COVER_PROFILE=coverage.out
COVER_MIN=0          # 百分比整数；0 表示只生成报告不闸（首版可 0）
```

默认白名单（与执行计划一致）：

```text
./pkg/...
./internal/errs/...
./internal/auth/...
./internal/logging/...
./internal/i18n/...
./internal/repository/...
```

可选追加（不阻塞首版）：`./internal/mcp/...`、`./internal/cache/...`。

首版命令形态：

```bash
go test -race -count=1 -coverprofile="$COVER_PROFILE" -covermode=atomic $TEST_PKGS
go tool cover -func=coverage.out | tee coverage.txt
# 若 COVER_MIN>0：解析 total 行，低于则 exit 1
```

---

## 三、CI

### 建议文件

- 若用 Gitea Actions：`.gitea/workflows/test.yml`  
- 或 GitHub 兼容：`.github/workflows/test.yml`（按实际托管二选一，文档写清）

### Job 要点

| 项 | 建议 |
|---|---|
| 触发 | `push` / `pull_request` 到主分支 |
| Go 版本 | 与 `go.mod` 一致 |
| 步骤 | checkout → setup-go → `bash deployments/scripts/test.sh` |
| 缓存 | Go modules cache |
| 产物 | 上传 `coverage.out` / `coverage.txt`（optional） |

首版 **硬闸白名单包必须绿**；全仓 `./...` 可作 nightly / 手动，不挡 PR。

---

## 四、覆盖率门槛

1. 跑一次白名单，把 `total` 写入 `docs/round-5-coverage-baseline.md`（或更新 round-4 文档）。  
2. 门槛策略（首版）：**不低于基线 − 0**（不回退）；暂不设「≥ 40%」总目标（留给 Round 6+）。  
3. 实现：脚本解析 `go tool cover -func` 的 `total:` 行与基线文件比对（简单 awk/python 一行即可）。

---

## 五、文档改动

- [`deployments/scripts/README.md`](../deployments/scripts/README.md)（迁后）：增加「测试」小节  
- 根 [`Makefile`](../Makefile) / [`justfile`](../justfile)：`test` 目标  
- [`AGENTS.md`](../AGENTS.md) / 根 [`README.md`](../README.md)：一键跑测（`make test` / `just test` 或权威路径）  
- 本任务验收勾选写入 [执行计划 §九](./round-5-execution-plan.md#九t7--测试工程化1~2-天)

---

## 六、PR 拆分

| PR | 内容 |
|---|---|
| T7-a | `deployments/scripts/test.sh` (+ ps1) + README/AGENTS  
| T7-b | CI workflow + 覆盖率基线文档 + 可选门槛 |

可与 T15 搬迁 PR 合并或紧随其后。

---

## 七、验收

- [ ] 本地一键：`deployments/scripts/test.sh` 绿  
- [ ] CI 对白名单包硬闸且绿  
- [ ] 覆盖率快照入库；门槛策略写明  
- [ ] 路径与 T15 方案 A 一致  

---

## 八、参考

- [round-5-scripts-layout.md](./round-5-scripts-layout.md)  
- [round-4-coverage.md](./round-4-coverage.md)  
- [`deployments/scripts/`](../deployments/scripts/)（搬迁后）  
