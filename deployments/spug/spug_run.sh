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

# 把 dest 变成指向 persist 的一层软链。
# 发布包常带空 uploads/ 目录，直接 ln -sfn 会变成 uploads/uploads 套娃。
ensure_persist_link() {
  local dest="$1"
  local persist="$2"
  local base nested n

  mkdir -p "$persist"
  base="$(basename "$dest")"

  if [ -L "$dest" ]; then
    ln -sfn "$persist" "$dest"
    log "软链 $dest -> $persist"
    return 0
  fi

  if [ -d "$dest" ]; then
    log "$dest 是真实目录，迁移到 $persist 后改为软链"
    for nested in "$dest"/* "$dest"/.[!.]* "$dest"/..?*; do
      if [ ! -e "$nested" ] && [ ! -L "$nested" ]; then
        continue
      fi
      n="$(basename "$nested")"
      # 跳过已套娃的同名软链（uploads/uploads -> persist）
      if [ -L "$nested" ] && [ "$n" = "$base" ]; then
        log "跳过套娃软链 $nested"
        continue
      fi
      cp -a "$nested" "$persist/"
    done
    rm -rf "$dest"
  elif [ -e "$dest" ]; then
    log "错误：$dest 存在且不是目录或软链"
    exit 1
  fi

  ln -sfn "$persist" "$dest"
  log "软链 $dest -> $persist"
}

# ===== 0. Round 2 目录形态预检 =====
if [ ! -d "$WWW/conf" ] || [ ! -d "$WWW/web/static" ] || [ ! -d "$WWW/web/views" ]; then
  log "错误：WWW 缺少 Round 2 目录（需要 conf/、web/static/、web/views/）。请检查发布包结构。"
  exit 1
fi
if [ -d "$WWW/configs" ]; then
  log "警告：检测到遗留 $WWW/configs/，请迁移到 conf/（Beego 默认路径）"
fi
if [ -d "$WWW/static" ] && [ ! -L "$WWW/static" ]; then
  log "警告：检测到遗留目录 $WWW/static，请迁移到 web/static/"
fi
if [ -d "$WWW/views" ] && [ ! -L "$WWW/views" ]; then
  log "警告：检测到遗留目录 $WWW/views，请迁移到 web/views/"
fi

# ===== 1. 持久化目录（uploads / runtime）=====
mkdir -p "$REPO/uploads" "$REPO/runtime" "$REPO/scripts"
ensure_persist_link "$WWW/uploads" "$REPO/uploads"
ensure_persist_link "$WWW/runtime" "$REPO/runtime"

# ===== 2. 应用配置 app.conf =====
# 权威配置只在 $REPO/app.conf（改数据库/端口请改这一份）。
# 每次发布用它覆盖 $WWW/conf/app.conf，避免被发布包里的 example 冲掉。
mkdir -p "$WWW/conf"
if [ ! -e "$REPO/app.conf" ]; then
  if [ -f "$WWW/conf/app.conf" ]; then
    log "首次部署：将现有 $WWW/conf/app.conf 收为权威配置"
    cp -f "$WWW/conf/app.conf" "$REPO/app.conf"
  elif [ -e "$WWW/conf/app.conf.example" ]; then
    log "首次部署：用 app.conf.example 初始化 $REPO/app.conf（请随后填写数据库密码、DOC_SESSION_KEY 等，example 无明文默认）"
    cp -f "$WWW/conf/app.conf.example" "$REPO/app.conf"
  else
    log "缺少权威配置且没有 app.conf.example，无法初始化 app.conf"
    exit 1
  fi
fi
log "同步权威配置 $REPO/app.conf -> $WWW/conf/app.conf"
cp -f "$REPO/app.conf" "$WWW/conf/app.conf"
if [ ! -f "$WWW/conf/app.conf" ]; then
  log "错误：同步后 $WWW/conf/app.conf 不存在"
  exit 1
fi

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
