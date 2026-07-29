# scripts/

构建脚本仍放此处：

- `build.bat` / `build.sh` — 多平台编译

Round 2 起部署相关文件已迁到：

- `deployments/spug/spug_run.sh`
- `deployments/systemd/doc.service`
- `deployments/Dockerfile` / `docker-compose.yml` / `start.sh` / `sync_host.sh`

Spug 后置脚本路径请改为仓库内 `deployments/spug/spug_run.sh`（或发布包解压后的同路径）。
