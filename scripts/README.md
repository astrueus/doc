# scripts/

构建与发版脚本：

- `build.bat` / `build.sh` — 多平台编译
- `release.ps1` / `release.bat` — 本地发版（编译 → zip → Gitea Release）
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

更多说明见 [docs/release-local.md](../docs/release-local.md)。

Round 2 起部署相关文件已迁到：

- `deployments/spug/spug_run.sh`
- `deployments/systemd/doc.service`
- `deployments/Dockerfile` / `docker-compose.yml` / `start.sh` / `sync_host.sh`

Spug 后置脚本路径请改为仓库内 `deployments/spug/spug_run.sh`（或发布包解压后的同路径）。
