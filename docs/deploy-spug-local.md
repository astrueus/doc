# 本地发版 + Spug 上线

> 配合 [`release-local.md`](./release-local.md)。本地脚本把 zip 推到 Gitea Release，Spug 从 Gitea 拉取并部署到生产服务器。

## 适用场景

- 已经按 `release-local.md` 实现本地一键发版
- 服务器侧用 Spug 统一管理上线/回滚
- 不依赖 Gitea Actions Runner

---

## 一、总链路

```mermaid
flowchart LR
    A[开发者本机<br/>scripts/release.ps1] --> B[Gitea Release<br/>doc_linux_amd64.zip]
    B --> C[Spug 发布任务<br/>下载 + 解压]
    C --> D[Spug 前置脚本<br/>spug_pre.sh]
    D --> E[Spug 后置脚本<br/>deployments/spug/spug_run.sh]
    E --> F[doc.service<br/>systemd 重启]
    F --> G[健康检查<br/>http://:8181]
```

| 节点 | 负责人 | 工件 |
|------|--------|------|
| 编译/发版 | 开发者本机 | `doc_linux_amd64.zip` 上传到 Gitea Release |
| 拉取/部署 | Spug | 把 zip 解压到 `WWW` |
| 软链/服务 | `deployments/spug/spug_run.sh` | 持久化目录、`app.conf`、systemd |
| 运行 | `doc.service` | 启动 `doc` |

---

## 二、服务器目录约定（与现有 `spug_run.sh` 一致）

```text
/data/wwwroot/doc.itopcms.com/         ← WWW：每次发布覆盖
  doc                                  ← 可执行文件（必须叫 doc）
  conf/app.conf                        ← 每次从 REPO 覆盖
  conf/app.conf.example                ← 仓库自带
  web/static/  web/views/              ← 静态资源与模板（Round 2）
  uploads -> /data/repos/.../uploads   ← 软链
  runtime -> /data/repos/.../runtime   ← 软链
  deployments/
    spug/spug_run.sh                   ← Spug 后置脚本
    systemd/doc.service                ← systemd unit 源文件
  dist/doc_linux_amd64                 ← 解压时存在；前置脚本会拷成 doc

/data/repos/doc.itopcms.com/resource/  ← REPO：持久化数据，不随发布覆盖
  app.conf                             ← 权威配置（运维维护）
  uploads/                             ← 用户上传文件
  runtime/                             ← Beego 运行时（日志/缓存）
  scripts/
    doc.service                        ← systemd unit（由 spug_run.sh 从 deployments/systemd 同步）
  backup/<TAG>/                        ← 二进制备份（用于回滚）
```

> **核心约束**：`doc.service` 的 `ExecStart` 指向 `/data/wwwroot/doc.itopcms.com/doc`，因此必须保证 `WWW/doc` 始终是新版本可执行文件。

---

## 三、一次性准备（首次上线前手工执行）

在目标服务器 root 用户下：

```bash
# 1. 目录
mkdir -p /data/wwwroot/doc.itopcms.com
mkdir -p /data/repos/doc.itopcms.com/resource/{uploads,runtime,scripts,backup}

# 2. 准备初始 app.conf（首次部署 spug_run.sh 也会自动从 example 复制）
cp /path/to/app.conf.example /data/repos/doc.itopcms.com/resource/app.conf
vi /data/repos/doc.itopcms.com/resource/app.conf
# 至少修改：
#   httpport     = "8181"
#   db_adapter   = mysql 或 sqlite3
#   db_host/db_user/db_password/db_database
#   site_name / app_key 等

# 3. 必要工具
yum install -y unzip curl   # 或 apt install -y unzip curl
```

### Spug 端

1. 主机管理：纳管目标服务器，确保 SSH 可执行命令
2. 应用管理：新建应用「Doc 文档系统」
3. 发布参数：建议增加一个变量 `TAG`，默认值 `v1.0.0`
4. 发布配置：本文方案使用「自定义发布」流程（前置 + 后置脚本）

---

## 四、Spug 前置脚本：`scripts/spug_pre.sh`

> **职责**：从 Gitea Release 下载 zip，解压到 `WWW`，把 `dist/doc_linux_amd64` 复制为 `WWW/doc`。
>
> 解决 `spug_run.sh` 中 `chmod 755 "$WWW/doc"` 必须先有 `doc` 文件的前提。

将以下内容保存为 `scripts/spug_pre.sh`（与 `spug_run.sh` 同目录）：

```bash
#!/bin/bash
# spug_pre.sh — 部署前置脚本
# 由 spug 在文件分发前/后执行，从 Gitea Release 下载对应版本的 zip 并解压到 WWW。
#
# 期望环境变量（在 Spug 应用变量中配置）：
#   TAG          发布的 tag，例如 v1.0.0
#   GITEA_OWNER  默认 jackliu
#   GITEA_REPO   默认 doc
#   GITEA_URL    默认 https://git.itopcms.com
#   GITEA_TOKEN  可选；私有仓库下载时需要
#
set -euo pipefail

WWW=/data/wwwroot/doc.itopcms.com
TAG="${TAG:?TAG 未设置，例如 v1.0.0}"
OWNER="${GITEA_OWNER:-jackliu}"
REPO="${GITEA_REPO:-doc}"
BASE="${GITEA_URL:-https://git.itopcms.com}"
TOKEN="${GITEA_TOKEN:-}"

PKG_NAME=doc_linux_amd64.zip
PKG_URL="$BASE/$OWNER/$REPO/releases/download/$TAG/$PKG_NAME"
PKG_FILE="/tmp/${REPO}_${TAG}_${PKG_NAME}"

log() { echo "[spug-pre] $*"; }

log "TAG=$TAG, 下载 $PKG_URL"
mkdir -p "$WWW"

CURL_OPTS=(-fL --retry 3 --retry-delay 2)
if [ -n "$TOKEN" ]; then
  CURL_OPTS+=(-H "Authorization: token $TOKEN")
fi

curl "${CURL_OPTS[@]}" "$PKG_URL" -o "$PKG_FILE"

log "解压到 $WWW"
unzip -oq "$PKG_FILE" -d "$WWW"

# build.sh --mode=release 产物路径：dist/doc_linux_amd64
if [ -f "$WWW/dist/doc_linux_amd64" ]; then
  cp -f "$WWW/dist/doc_linux_amd64" "$WWW/doc"
elif [ -f "$WWW/doc_linux_amd64" ]; then
  cp -f "$WWW/doc_linux_amd64" "$WWW/doc"
else
  log "找不到 doc_linux_amd64，请检查 zip 内容"
  exit 1
fi

chmod 755 "$WWW/doc"

# 自检版本号
log "二进制自检"
"$WWW/doc" version || { log "doc version 执行失败"; exit 1; }

log "前置完成"
```

---

## 五、Spug 后置脚本：`deployments/spug/spug_run.sh`（已存在）

仓库已有完整 `deployments/spug/spug_run.sh`，本节是增强建议（按需追加，不强制修改）。

### 建议增强项

```bash
# 顶部：并发锁，防止 Spug 同时触发多次发布
LOCK=/var/lock/doc-deploy.lock
exec 9>"$LOCK"
flock -n 9 || { log "另一个发布任务进行中"; exit 1; }

# 末尾：版本备份 + 健康检查
TAG="${TAG:-unknown}"
mkdir -p "$REPO/backup/$TAG"
cp -f "$WWW/doc" "$REPO/backup/$TAG/doc" || true

# 健康检查：等待 systemd 拉起后端口可访问
PORT=$(grep -E '^httpport' "$WWW/conf/app.conf" | head -1 | sed -E 's/.*= *"?([0-9]+)"?.*/\1/')
PORT="${PORT:-8181}"

for i in 1 2 3 4 5 6 7 8 9 10; do
  sleep 2
  if curl -fsS "http://127.0.0.1:$PORT/" >/dev/null 2>&1; then
    log "服务就绪 :$PORT"
    log "当前版本: $($WWW/doc version 2>&1 | head -1)"
    exit 0
  fi
  log "等待服务就绪 ($i/10)"
done

log "健康检查失败：journalctl -u $SERVICE_NAME --since '1 min ago'"
journalctl -u "$SERVICE_NAME" --since "1 min ago" | tail -50
exit 1
```

> 不修改仓库现有 `spug_run.sh` 也可工作；上述只是稳健性补强建议。

---

## 六、`doc.service` 调整建议

仓库 `deployments/systemd/doc.service` 默认 root 启动，且 `After=mysqld.service`。生产可调整：

```ini
[Unit]
Description=doc
After=network.target
# 仅当用 MySQL 且服务名为 mysqld 时启用：
# After=mysqld.service
# Wants=mysqld.service

[Service]
Type=simple
User=www
Group=www
WorkingDirectory=/data/wwwroot/doc.itopcms.com
ExecStart=/data/wwwroot/doc.itopcms.com/doc
Restart=always
RestartSec=3
# 可选：显式指定项目根（与 -dir 等效；未设置时默认用可执行文件所在目录）
# Environment=DOC_HOME=/data/wwwroot/doc.itopcms.com
# 若需要监听 < 1024 端口（默认 8181 不需要）：
# AmbientCapabilities=CAP_NET_BIND_SERVICE

LimitNOFILE=65535
NoNewPrivileges=true
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
```

> 改为 `User=www` 后，需要把 `WWW` 与 `REPO` 目录 chown 为 www，并把 `spug_run.sh` 里的 `chown -R "$RUN_USER:$RUN_GROUP" ...` 那行注释打开。

---

## 七、Spug 控制台配置示例

### 应用变量（每次发布可填）

| 变量 | 示例 | 说明 |
|------|------|------|
| `TAG` | `v1.0.0` | 必填，要部署的 Release tag |
| `GITEA_TOKEN` | （Spug 应用密文） | 私有仓库下载使用 |

### 发布步骤（按顺序）

1. **前置（在目标主机执行）**
   ```bash
   bash /data/wwwroot/doc.itopcms.com/scripts/spug_pre.sh
   ```
   > 首次发布时 `WWW` 还没有 `spug_pre.sh`。两种做法：
   > - **方案A**：把 `spug_pre.sh` 内容直接粘贴到 Spug 「自定义命令」里执行（不依赖文件存在）
   > - **方案B**：首次手工 scp 一份到服务器，后续每次发布由 `spug_pre.sh` 解压时一并覆盖

2. **后置（在目标主机执行）**
   ```bash
   bash /data/wwwroot/doc.itopcms.com/deployments/spug/spug_run.sh
   ```

### 推荐：直接在 Spug「自定义命令」里串成一条

```bash
set -e
export TAG="$SPUG_VERSION"          # 或者发布参数里直接取
export GITEA_OWNER=jackliu
export GITEA_REPO=doc
export GITEA_URL=https://git.itopcms.com
# export GITEA_TOKEN=...            # 若仓库为私有，从 Spug 密文注入

# 1) 拉包（脚本内联，不依赖远端文件）
WWW=/data/wwwroot/doc.itopcms.com
mkdir -p "$WWW"
curl -fL ${GITEA_TOKEN:+-H "Authorization: token $GITEA_TOKEN"} \
  "$GITEA_URL/$GITEA_OWNER/$GITEA_REPO/releases/download/$TAG/doc_linux_amd64.zip" \
  -o /tmp/doc.zip
unzip -oq /tmp/doc.zip -d "$WWW"
[ -f "$WWW/dist/doc_linux_amd64" ] && cp -f "$WWW/dist/doc_linux_amd64" "$WWW/doc"
chmod 755 "$WWW/doc"

# 2) 后置（仓库自带）
bash "$WWW/deployments/spug/spug_run.sh"
```

---

## 八、完整上线流程

```text
本机：
  scripts\release.bat 1.0.0      # 编译 + 上传 Gitea Release

Spug：
  打开应用 → 新建发布
  发布参数：TAG = v1.0.0
  点击发布 → 选目标主机 → 执行
  日志中观察：
    [spug-pre] TAG=v1.0.0, 下载 ...
    [spug-pre] 解压到 /data/wwwroot/doc.itopcms.com
    [spug-pre] 二进制自检
    [spug]    重新加载并重启 doc.service
    [spug]    服务就绪 :8181
    [spug]    当前版本: 1.0.0
```

---

## 九、回滚

### 方案 A：发布旧 tag（最简单）

```text
Spug 发布参数 TAG = v0.9.9 → 重新发布即可
```

`spug_pre.sh` 会下载 v0.9.9 的 zip 覆盖；`spug_run.sh` 重启服务；数据目录（`uploads`/`runtime`/`app.conf`）持久化不变。

### 方案 B：用本地备份的二进制（快速回退，但仅回退可执行文件）

`scripts/spug_rollback.sh`（手工执行）：

```bash
#!/bin/bash
set -euo pipefail
TAG="${1:?Usage: spug_rollback.sh <TAG>}"
WWW=/data/wwwroot/doc.itopcms.com
REPO=/data/repos/doc.itopcms.com/resource
BAK="$REPO/backup/$TAG/doc"

[ -f "$BAK" ] || { echo "no backup for $TAG"; exit 1; }

cp -f "$BAK" "$WWW/doc"
chmod 755 "$WWW/doc"
systemctl restart doc.service
echo "rolled back to $TAG"
"$WWW/doc" version
```

执行：

```bash
bash /data/repos/doc.itopcms.com/resource/scripts/spug_rollback.sh v0.9.9
```

> 注意：方案 B 只回滚可执行文件；若新版本改动了 `static`/`views`，需用方案 A 才能完整回退。

---

## 十、数据备份建议（首次上线就配上）

无论是否用 Spug，强烈建议每次发布前做一次数据快照（可写在 `spug_run.sh` 顶部或新建独立脚本）：

```bash
BAK_ROOT=/data/repos/doc.itopcms.com/resource/backup
SNAP="$BAK_ROOT/$(date +%Y%m%d_%H%M%S)_${TAG:-manual}"
mkdir -p "$SNAP"

# 1. 配置
cp -f /data/repos/doc.itopcms.com/resource/app.conf "$SNAP/" 2>/dev/null || true

# 2. SQLite
if [ -f /data/repos/doc.itopcms.com/resource/database.db ]; then
  cp -f /data/repos/doc.itopcms.com/resource/database.db "$SNAP/"
fi

# 3. MySQL（按需）
# mysqldump -h HOST -u USER -pPASS DB > "$SNAP/mindoc.sql"

# 4. 旧二进制
[ -f /data/wwwroot/doc.itopcms.com/doc ] && cp -f /data/wwwroot/doc.itopcms.com/doc "$SNAP/doc.prev"

# 5. 保留最近 10 份
ls -1dt "$BAK_ROOT"/*/ 2>/dev/null | tail -n +11 | xargs -r rm -rf
```

---

## 十一、常见问题

### Q1：`spug_run.sh` 第 54 行 `chmod 755 "$WWW/doc"` 报「No such file」
- `spug_pre.sh` 没跑 / zip 内可执行文件未复制为 `doc`
- 检查 `dist/doc_linux_amd64` 是否在 zip 中

### Q2：systemd 报「指向 X 与本项目 Y 不一致」
- 服务器上之前手动装过 `doc.service` 指向别的路径
- 处理：`systemctl disable --now doc.service && rm /etc/systemd/system/doc.service`，再重新发布

### Q3：服务起来但端口不通
- `journalctl -u doc.service --since "5 min ago"`
- 检查 `app.conf` 的 `httpport` 与防火墙
- 检查 DB 连接

### Q4：用户上传文件丢失
- 一定要把 `uploads` 软链到 `$REPO/uploads`（脚本已做）
- 检查 `WWW/uploads` 是软链还是真实目录：`ls -l "$WWW/uploads"`

### Q5：app.conf 改了不生效
- 改的应该是 `$REPO/app.conf`（权威配置），不是 `$WWW/conf/app.conf`
- 改后需要再 spug 发布一次或手工 `cp -f` 再 `systemctl restart`

### Q6：私有仓库下载 401
- 在 Spug 应用密文中加 `GITEA_TOKEN`
- 确认 token 有 `read:repository` 权限

### Q7：Spug 显示成功但服务异常
- 当前 `spug_run.sh` 没做健康检查；按第五节追加 `curl 自检` 段落

---

## 十二、验证清单

每次上线后逐项过一遍：

- [ ] `systemctl is-active doc.service` 输出 `active`
- [ ] `curl -fsS http://127.0.0.1:8181/` 返回 200
- [ ] `/data/wwwroot/doc.itopcms.com/doc version` 与 TAG 一致
- [ ] 浏览器登录后台，新建/查看文档正常
- [ ] `ls -l /data/wwwroot/doc.itopcms.com/uploads` 为软链
- [ ] `journalctl -u doc.service` 没有报错堆栈
- [ ] 备份目录 `$REPO/backup/<TAG>/` 已生成

---

## 十三、相关文档

- [release-local.md](./release-local.md)：本地发版脚本
- [deploy-spug-actions.md](./deploy-spug-actions.md)：如果改用 Gitea Actions 自动发版，再走 Spug 部署
