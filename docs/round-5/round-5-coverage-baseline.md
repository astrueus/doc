# Round 5 覆盖率基线（T7）

> 对应 [round-5-t7-testing.md](./round-5-t7-testing.md)。  
> 机器可读数字：[`coverage-baseline.txt`](./coverage-baseline.txt)（`go tool cover -func` 的 **total** 百分比，不含 `%`）。

## 范围

与 `deployments/scripts/test.sh` 默认白名单一致：

```text
./pkg/...
./internal/errs/...
./internal/auth/...
./internal/logging/...
./internal/i18n/...
./internal/repository/...
```

## 快照

| 日期 | total | 说明 |
|------|-------|------|
| 2026-08-18 | **28.5%** | Windows 本机，`RACE=0`、`covermode=atomic`、`-p 1` |

门槛：**不低于本基线 28.5%**（脚本未设 `COVER_MIN` 时读取 `coverage-baseline.txt`）。暂不设全仓 ≥ 40%（Round 6+）。

CI（`.gitea/workflows/test.yml`）在 Linux 上默认开 `-race`。若首次 Runner 跑出来的 total 与 28.5 有小幅偏差，以 CI 实测微调基线文件，避免无意义抖动挡 PR。

继承：[round-4-coverage.md](../round-1-4/round-4-coverage.md)（包级说明，不作本门禁数字）。
