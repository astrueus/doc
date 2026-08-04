# scripts/

构建与发版脚本：

- `build.bat` / `build.sh` — 多平台编译
- `release.ps1` / `release.bat` — 本地发版（Windows）
- `release.sh` — 本地发版（Linux / macOS）
- `lib/json.sh` — 小型 JSON 解析库（`release.sh` 等复用；零第三方依赖）
- `lib/json_test.sh` — `json.sh` 单测（`bash scripts/lib/json_test.sh`）
- `.env.release.example` — 发版环境变量模板；复制为 `.env.release` 后填写（**已 gitignore，勿提交**）

## 本地发版（Windows）

```bat
REM 1) 准备环境文件（一次性）
copy scripts\.env.release.example scripts\.env.release
REM 编辑 scripts\.env.release，填入 GITEA_TOKEN 等

REM 2) 只编译+打包（不需要 Token）
scripts\release.bat 0.0.1-test windows --dry-run

REM 3) 草稿发版（默认读取 scripts\.env.release）
scripts\release.bat 0.0.1-test windows --draft

REM 或显式指定 env 文件
scripts\release.bat 0.0.1-test windows --env=scripts\.env.release --draft
```

PowerShell：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\release.ps1 -Version 0.0.1-test -Target windows -EnvFile scripts\.env.release -Draft
```

## 本地发版（Linux / macOS）

```bash
# 1) 准备环境文件（一次性）
cp scripts/.env.release.example scripts/.env.release
# 编辑 scripts/.env.release，填入 GITEA_TOKEN 等

# 2) 只编译+打包（不需要 Token；默认 target=linux）
./scripts/release.sh 0.0.1-test --dry-run
./scripts/release.sh 0.0.1-test linux --dry-run

# 3) 草稿发版
./scripts/release.sh 0.0.1-test linux --draft

# 正式双平台（Windows 交叉编译需 Zig 或 MinGW，且需本机有 zip）
./scripts/release.sh 1.0.0 all
```

依赖：`bash`、`curl`、`tar`；打 Windows zip 时需 `zip`。  
发版脚本的 JSON 解析使用仓库内 `scripts/lib/json.sh`（纯 bash，无需 jq/python/go）。

其它脚本如需通用 JSON 解析，可 `source scripts/lib/json.sh`（`json_flatten` / `json_get` / `json_find_index` 等）；单测：`bash scripts/lib/json_test.sh`。

更多说明见 [docs/release-local.md](../docs/release-local.md)。  
产物目录：`release/doc_<version>_windows_amd64.zip`、`release/doc_<version>_linux_amd64.tar.gz`（已 gitignore）。

Round 2 起部署相关文件已迁到：

- `deployments/spug/spug_run.sh`
- `deployments/systemd/doc.service`
- `deployments/Dockerfile` / `docker-compose.yml` / `start.sh` / `sync_host.sh`

Spug 后置脚本路径请改为仓库内 `deployments/spug/spug_run.sh`（或发布包解压后的同路径）。
