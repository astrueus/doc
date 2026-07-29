#!/bin/bash
#
# spug_run.sh — 部署后置脚本（Round 2：位于 deployments/spug/）
# 由 Spug 在每次发布后执行；处理软链、配置同步、systemd 服务。
#
set -euo pipefail

# ===== 路径变量 =====
WWW=/data/wwwroot/doc.itopcms.com
REPO=/data/repos/doc.itopcms.com/resource
SERVICE_NAME=doc.service
# systemd unit 仍落在 $REPO/scripts/（运维习惯与历史兼容）；源文件来自发布包 deployments/systemd/
SERVICE_SRC="$REPO/scripts/$SERVICE_NAME"
SERVICE_LINK="/etc/systemd/system/$SERVICE_NAME"
# RUN_USER=www
# RUN_GROUP=www

log() { echo "[spug] $*"; }

# ===== 0. Round 2 目录形态预检 =====
if [ ! -d "$WWW/configs" ] || [ ! -d "$WWW/web/static" ] || [ ! -d "$WWW/web/views" ]; then
  log "错误：WWW 缺少 Round 2 目录（需要 configs/、web/static/、web/views/）。请检查发布包结构。"
  exit 1
fi
if [ -e "$WWW/conf/app.conf" ]; then
  log "警告：检测到遗留 $WWW/conf/app.conf，请迁移到 configs/app.conf"
fi
if [ -d "$WWW/static" ] && [ ! -L "$WWW/static" ]; then
  log "警告：检测到遗留目录 $WWW/static，请迁移到 web/static/"
fi
if [ -d "$WWW/views" ] && [ ! -L "$WWW/views" ]; then
  log "警告：检测到遗留目录 $WWW/views，请迁移到 web/views/"
fi

# ===== 1. 持久化目录（uploads / runtime）=====
mkdir -p "$REPO/uploads" "$REPO/runtime" "$REPO/scripts"

ln -sfn "$REPO/uploads" "$WWW/uploads"
ln -sfn "$REPO/runtime" "$WWW/runtime"

# ===== 2. 应用配置 app.conf =====
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
mkdir -p "$REPO/scripts"
if [ -d "$WWW/deployments/spug" ]; then
  cp -rf "$WWW/deployments/spug/." "$REPO/scripts/"
fi
if [ -f "$WWW/deployments/systemd/$SERVICE_NAME" ]; then
  cp -f "$WWW/deployments/systemd/$SERVICE_NAME" "$REPO/scripts/$SERVICE_NAME"
elif [ -f "$WWW/scripts/$SERVICE_NAME" ]; then
  # 兼容旧包仍把 unit 放在 scripts/
  cp -f "$WWW/scripts/$SERVICE_NAME" "$REPO/scripts/$SERVICE_NAME"
fi

if [ ! -e "$SERVICE_SRC" ]; then
  log "[$SERVICE_NAME] 不存在，请确认发布包含 deployments/systemd/$SERVICE_NAME"
  exit 1
fi

# ===== 4. 权限 =====
chmod 755 "$WWW/doc"

# ===== 5. 注册 / 重启 systemd 服务 =====
if systemctl cat "$SERVICE_NAME" >/dev/null 2>&1; then
  SERVICE_PATH=$(systemctl show -p FragmentPath --value "$SERVICE_NAME" 2>/dev/null || true)
  if [ -z "$SERVICE_PATH" ]; then
    SERVICE_PATH=$(systemctl show -p FragmentPath "$SERVICE_NAME" | sed 's/^FragmentPath=//')
  fi
  LOADED_PATH=$(readlink -f "$SERVICE_PATH" 2>/dev/null || echo "$SERVICE_PATH")
  EXPECTED_PATH=$(readlink -f "$SERVICE_SRC" 2>/dev/null || echo "$SERVICE_SRC")
  log "FragmentPath=$SERVICE_PATH"
  log "LoadedPath  =$LOADED_PATH"
  log "ExpectedPath=$EXPECTED_PATH"

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
