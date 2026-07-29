#!/bin/bash
#
# spug_run.sh — 部署后置脚本
# 由 spug 在每次发布后执行；处理软链、配置同步、systemd 服务。
#
set -euo pipefail

# ===== 路径变量 =====
WWW=/data/wwwroot/doc.itopcms.com
REPO=/data/repos/doc.itopcms.com/resource
SERVICE_NAME=doc.service
SERVICE_SRC="$REPO/scripts/$SERVICE_NAME"
SERVICE_LINK="/etc/systemd/system/$SERVICE_NAME"
# RUN_USER=www
# RUN_GROUP=www

log() { echo "[spug] $*"; }

# ===== 1. 持久化目录（uploads / runtime）=====
# 资源目录在仓库外持久化，软链回 wwwroot
mkdir -p "$REPO/uploads" "$REPO/runtime" "$REPO/scripts"

ln -sfn "$REPO/uploads" "$WWW/uploads"
ln -sfn "$REPO/runtime" "$WWW/runtime"

# ===== 2. 应用配置 app.conf =====
# 设计：$REPO/app.conf 是权威配置，wwwroot 每次发布从 repo 强制覆盖，
# 运维直接改 wwwroot 不生效；配置变更需修改 $REPO/app.conf 后通过 spug 重新发布。
if [ ! -e "$REPO/app.conf" ]; then
  if [ -e "$WWW/configs/app.conf.example" ]; then
    log "首次部署，使用 app.conf.example 初始化 app.conf"
    cp "$WWW/configs/app.conf.example" "$REPO/app.conf"
  else
    log "缺少 $WWW/configs/app.conf.example，无法初始化 app.conf"
    exit 1
  fi
fi
cp -f "$REPO/app.conf" "$WWW/configs/app.conf"

# ===== 3. 同步 systemd unit 到 resource =====
# 每次发布都覆盖，确保仓库里 doc.service 的改动能落到生效路径
mkdir -p "$REPO/scripts"
cp -rf "$WWW/scripts/." "$REPO/scripts/"

if [ ! -e "$SERVICE_SRC" ]; then
  log "[$SERVICE_NAME] 不存在，请先在仓库 scripts/ 中提交后再发布。"
  exit 1
fi

# ===== 4. 权限 =====
# chown 暂不开启，由 spug 侧或人工保证 owner 正确
# chown -R "$RUN_USER:$RUN_GROUP" "$WWW" "$REPO"
# 如果 spug 已经将 owner 设为 www，可改回 744
chmod 755 "$WWW/doc"

# ===== 5. 注册 / 重启 systemd 服务 =====
if systemctl cat "$SERVICE_NAME" >/dev/null 2>&1; then
  # 已存在：校验当前 unit 是否指向本项目维护的文件
  # systemctl --value 直接输出属性值（systemd 230+ 支持），避免用 awk 切串
  SERVICE_PATH=$(systemctl show -p FragmentPath --value "$SERVICE_NAME" 2>/dev/null || true)
  if [ -z "$SERVICE_PATH" ]; then
    SERVICE_PATH=$(systemctl show -p FragmentPath "$SERVICE_NAME" | sed 's/^FragmentPath=//')
  fi
  # 同时打印字符串路径与 canonical 路径，便于排查
  LOADED_PATH=$(readlink -f "$SERVICE_PATH" 2>/dev/null || echo "$SERVICE_PATH")
  EXPECTED_PATH=$(readlink -f "$SERVICE_SRC" 2>/dev/null || echo "$SERVICE_SRC")
  log "FragmentPath=$SERVICE_PATH"
  log "LoadedPath  =$LOADED_PATH"
  log "ExpectedPath=$EXPECTED_PATH"

  # 用 -ef 比较 inode，规避路径前缀含软链/绑定挂载导致字符串不等的问题
  if [ -e "$SERVICE_PATH" ] && [ -e "$SERVICE_SRC" ] && [ "$SERVICE_PATH" -ef "$SERVICE_SRC" ]; then
    log "重新加载并重启 $SERVICE_NAME"
    systemctl daemon-reload
    systemctl restart "$SERVICE_NAME"
  else
    log "已存在同名 [$SERVICE_NAME] 服务，但指向 $SERVICE_PATH，与本项目 $SERVICE_SRC 不一致。"
    log "请重命名服务或先卸载后再发布。"
    exit 1
  fi
else
  log "首次注册 $SERVICE_NAME"
  ln -sfn "$SERVICE_SRC" "$SERVICE_LINK"
  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME"
  systemctl start "$SERVICE_NAME"
fi

log "部署完成。"
