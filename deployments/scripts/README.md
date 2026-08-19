# deployments/scripts/

构建、测试与发版脚本（Round 5 T15：由仓库根 `scripts/` 迁入）。日常入口优先用仓库根 **`make`** / **`just`**。

| 文件 | 用途 |
|------|------|
| `build.sh` / `build.bat` | 多平台编译（Zig / MinGW） |
| `test.sh` / `test.ps1` | 白名单包 `go test` + 覆盖率门槛（T7） |
| `release.sh` / `release.bat` / `release.ps1` | 一键发版（编译 → 打包 → tag → Gitea Release） |
| `lib/json.sh` / `lib/json_test.sh` | 纯 bash JSON 解析 + 单测 |
| `.env.release.example` | 发版环境变量模板 |

## 快捷入口

```bash
make help          # 或 just --list
make build         # just build
make test          # just test
make release VERSION=1.0.0
just release 1.0.0
```

Windows 未装 Make 时：`just test` 或直接 `powershell -File deployments\scripts\test.ps1`。

CI 直接调用权威路径：`bash deployments/scripts/test.sh`（见 `.gitea/workflows/test.yml`）。

## 本地发版（Windows）

```bat
REM 1) 准备环境文件（一次性）
copy deployments\scripts\.env.release.example deployments\scripts\.env.release
REM 编辑 deployments\scripts\.env.release，填入 GITEA_TOKEN 等

REM 2) 只编译+打包（不需要 Token）
deployments\scripts\release.bat 0.0.1-test windows --dry-run

REM 3) 草稿发版
deployments\scripts\release.bat 0.0.1-test windows --draft
```

PowerShell：

```powershell
powershell -ExecutionPolicy Bypass -File deployments\scripts\release.ps1 -Version 0.0.1-test -Target windows -EnvFile deployments\scripts\.env.release -Draft
```

## 本地发版（Linux / macOS）

```bash
cp deployments/scripts/.env.release.example deployments/scripts/.env.release
# 编辑 deployments/scripts/.env.release，填入 GITEA_TOKEN 等

./deployments/scripts/release.sh 0.0.1-test --dry-run
./deployments/scripts/release.sh 0.0.1-test linux --draft
./deployments/scripts/release.sh 1.0.0 all
```

依赖：`bash`、`curl`、`tar`；打 Windows zip 时需 `zip`。  
JSON 解析使用 `deployments/scripts/lib/json.sh`（纯 bash）。单测：`bash deployments/scripts/lib/json_test.sh`。

更多说明见 [docs/release/release-local.md](../../docs/release/release-local.md)。  
产物目录：`release/`（已 gitignore）。

部署相关仍在：

- `deployments/spug/spug_pre.sh`
- `deployments/spug/spug_run.sh`
- `deployments/systemd/doc.service`
- `deployments/Dockerfile` / `docker-compose.yml` / `start.sh` / `sync_host.sh`

Spug 自定义发布：前置 [`spug_pre.sh`](../spug/spug_pre.sh)、后置 [`spug_run.sh`](../spug/spug_run.sh)（或发布包解压后的同路径）。服务器上 `$REPO/scripts/` 是运维习惯目录，与本目录无耦合。
