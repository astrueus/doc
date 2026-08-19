# 常规发布（Git Tag + Release 包 + 保留历史版本）

> 自定义发布见 [`deploy-spug-local.md`](./deploy-spug-local.md)。  
> 本方案用 Spug **常规发布**自动保留 N 个版本目录；真正运行的仍是 Gitea Release 的 `tar.gz`，**不要**把 Git 源码当网站内容。

## 脚本（一眼对应钩子）

| 文件 | 钩子 | 做什么 |
|------|------|--------|
| [`spug_strip.sh`](../../deployments/spug/spug_strip.sh) | **代码迁出后** | 清空 Git 检出，只留 `deployments/spug/` |
| [`spug_unpack.sh`](../../deployments/spug/spug_unpack.sh) | **应用发布前** | 按 `$SPUG_GIT_TAG` 下包，解压到**当前目录** |
| [`spug_run.sh`](../../deployments/spug/spug_run.sh) | **应用发布后** | 软链、权威 `app.conf`、systemd |

自定义发布仍用 [`spug_pre.sh`](../../deployments/spug/spug_pre.sh)（解压到固定 `WWW`），不要和本方案混用。

## Spug 发布配置

1. 类型：**常规发布**，Git 仓库填本仓库地址  
2. 申请时：**基于 Tag**（否则没有 `$SPUG_GIT_TAG`）  
3. 目标主机仓库路径：例如 `/data/spug/versions/doc`（历史版本目录）  
4. 目标主机部署路径：`/data/wwwroot/doc.itopcms.com`（软链指向当前版本）  
5. 保留历史版本数量：例如 `3`  
6. **文件过滤：关闭**（空「仅包含」= 全量源码；不存在的路径 = 打包失败。源码靠 `spug_strip.sh` 清掉）

钩子（检出后脚本还在，必须用仓库路径调用，不要把 `strip` 全文粘贴）：

```bash
# 代码迁出后（Spug 服务器）
bash ./deployments/spug/spug_strip.sh

# 应用发布前（目标机，cwd = 待发布版本目录）
bash ./deployments/spug/spug_unpack.sh

# 应用发布后（cwd = 已发布目录 / WWW 软链）
bash ./deployments/spug/spug_run.sh
```

`spug_unpack.sh` 能跑起来，是因为 `strip` 故意留下了 `deployments/spug/`。请先把这两份脚本合进 `master` 并打 tag，再用该 tag 做常规发布。

## 现网注意

- 部署路径若已是**真实目录**，Spug 会拒绝接管。先备份，删掉该目录后再发（`uploads` / `runtime` / `app.conf` 必须已在 REPO）。  
- `spug_unpack.sh` 解压到 `.`，不要改成 `$WWW`。  
- 回滚走 Spug「回滚」切软链；旧版本目录里应已有当时解过的包。发布后钩子仍应跑 `spug_run.sh` 以便重启服务。
