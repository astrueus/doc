#!/bin/bash
#
# spug_strip.sh — Spug 常规发布「代码迁出后」（Spug 服务器）
# 删掉 Git 检出里的源码，只留下 deployments/spug/，避免源码被打进版本包。
#
# 钩子里写：  bash ./deployments/spug/spug_strip.sh
# 文件过滤：  关闭（清空后再打包，几乎只剩本目录下的脚本）
#
set -euo pipefail

log() { echo "[spug-strip] $*"; }

KEEP_REL=deployments/spug
if [ ! -d "$KEEP_REL" ]; then
  log "当前目录没有 $KEEP_REL，请确认钩子在检出目录执行"
  exit 1
fi

tmp="$(mktemp -d /tmp/spug_strip.XXXXXX)"
cp -a "$KEEP_REL" "$tmp/spug"

log "清空检出，仅保留 $KEEP_REL/"
find . -mindepth 1 -maxdepth 1 -exec rm -rf {} +
mkdir -p deployments
cp -a "$tmp/spug" "$KEEP_REL"
rm -rf "$tmp"

log "清空完成，剩余："
find . -type f | sort
