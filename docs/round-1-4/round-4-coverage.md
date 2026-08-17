# Round 4 覆盖率快照（T10）

Round 4 T10 阶段产出：覆盖 `pkg/` 以及 `internal/errs`、`auth`、`logging`、`i18n`、`repository` 等包的基础单测。

> **后续：** `scripts/test.sh` / CI 硬闸 / 覆盖率门槛 📦 **已移交 [Round 5 T7](../round-5/round-5-execution-plan.md#九t7--测试工程化1~2-天)**。

## 如何刷新

```powershell
go test ./pkg/... ./internal/errs/... ./internal/auth/... ./internal/logging/... ./internal/i18n/... ./internal/repository/... -cover
```

## 新增/扩充测试的包（2026-08-03）

| 包 | 说明 |
|---|---|
| `pkg/cryptil` | 加解密往返、哈希、随机串 |
| `pkg/sqltil` | EscapeLike |
| `pkg/urlutil` | JoinURI |
| `pkg/krand` | 各 kind 随机串 |
| `pkg/password` | 哈希与校验 |
| `pkg/filetil` | FormatBytes、BOM 读取、CopyFile、IsImageExt |
| `pkg/htmlutil` | StripTags、AutoSummary |
| `pkg/gob` | msgpack Encode/Decode |
| `pkg/gopool` | LoadOrStore |
| `internal/errs` | BizError |
| `internal/auth` | MemberIDFromSession |
| `internal/logging` | ANSI 剥离与双通道写出 |
| `internal/i18n` | ini 的 SetMessage / Tr |
| `internal/repository` | DocumentRepo（含乐观锁） |

计划目标：`pkg/*` 行覆盖长期 ≥ 40%；本文档为基线快照，不作 CI 硬门槛。
