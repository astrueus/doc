# deployments/scripts/

构建、测试、开发启动与发版脚本（Round 5 T15：由仓库根 `scripts/` 迁入）。日常入口优先用仓库根 **`make`** / **`just`**。

| 文件 | 用途 |
|------|------|
| `build.sh` / `build.bat` | 多平台编译（Zig / MinGW） |
| `test.sh` / `test.ps1` | 白名单包 `go test` + 覆盖率门槛（T7） |
| `run.sh` / `run.ps1` | 开发启动：`go run ./cmd/doc --dir <仓库根>`（不落盘二进制，不加热重载） |
| `release.sh` / `release.bat` / `release.ps1` | 一键发版（编译 → 打包 → tag → Gitea Release；可选 `--github`） |
| `lib/json.sh` / `lib/json_test.sh` | 纯 bash JSON 解析 + 单测 |
| `.env.release.example` | 发版环境变量模板 |

## 快捷入口

```bash
make help          # 或 just --list
make build         # just build
make test          # just test
make run           # just run（开发启动）
make run ARGS=install
just run install
make release VERSION=1.0.0
just release 1.0.0
```

Windows 未装 Make 时：`just test` / `just run`，或直接 `powershell -File deployments\scripts\test.ps1` / `run.ps1`。

`go run` 的二进制在临时目录，脚本会固定 `--dir` 为仓库根；不要用裸 `go run ./cmd/doc`。sqlite 需要 CGO；Windows 未设 `CC` 时优先 `gcc`，否则 Zig。just 自身的 flag 写成 `just run -- --help`。

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

REM 4) Gitea 成功后再发 GitHub（需 .env.release 里 GITHUB_TOKEN；等镜像 tag）
deployments\scripts\release.bat 2.3.2 linux --github

REM 5) 只补 GitHub（Gitea 上已有该 tag 和包）
deployments\scripts\release.bat 2.3.2 linux --github-only
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
./deployments/scripts/release.sh 2.3.2 linux --github
./deployments/scripts/release.sh 2.3.2 linux --github-only
```

依赖：`bash`、`curl`、`tar`；打 Windows zip 时需 `zip`。  
JSON 解析使用 `deployments/scripts/lib/json.sh`（纯 bash）。单测：`bash deployments/scripts/lib/json_test.sh`。

更多说明见 [docs/release/release-local.md](../../docs/release/release-local.md)。  
产物目录：`release/`（已 gitignore）。

部署相关仍在：

- `deployments/spug/spug_pre.sh` — 自定义发布：下包解压到 WWW
- `deployments/spug/spug_strip.sh` — 常规发布：代码迁出后去掉 Git 源码
- `deployments/spug/spug_unpack.sh` — 常规发布：应用发布前解开 Release 包
- `deployments/spug/spug_run.sh` — 两种方式共用：软链 / app.conf / systemd
- `deployments/systemd/doc.service`
- `deployments/Dockerfile` / `docker-compose.yml` / `start.sh` / `sync_host.sh`

自定义发布：[`spug_pre.sh`](../spug/spug_pre.sh) + [`spug_run.sh`](../spug/spug_run.sh)。  
常规发布：[`spug_strip.sh`](../spug/spug_strip.sh) + [`spug_unpack.sh`](../spug/spug_unpack.sh) + [`spug_run.sh`](../spug/spug_run.sh)。  
服务器上 `$REPO/scripts/` 是运维习惯目录，与本目录无耦合。
