#!/bin/bash
#
# spug_unpack.sh — Spug 常规发布「应用发布前」（目标机，待发布版本目录）
# 按 Git Tag 从 Gitea Release 下载 tar.gz，解压到当前目录（不要解到 WWW）。
#
# 钩子里写：  bash ./deployments/spug/spug_unpack.sh
# 发布方式：  必须「基于 Tag」（使用 $SPUG_GIT_TAG）
# 应用发布后：bash ./deployments/spug/spug_run.sh
#
# Token 与 spug_pre.sh 相同：配置中心不会自动 export GITEA_TOKEN。
#
set -euo pipefail

log() { echo "[spug-unpack] $*"; }

TAG="${TAG-}"
if [ -z "$TAG" ]; then
  TAG="${SPUG_GIT_TAG-}"
fi
if [ -z "$TAG" ]; then
  TAG="${SPUG_RELEASE-}"
fi
TAG="${TAG:?常规发布请基于 Tag，需要 SPUG_GIT_TAG}"
case "$TAG" in
  v*) ;;
  *) TAG="v$TAG" ;;
esac

TOKEN="${GITEA_TOKEN-}"
if [ -z "$TOKEN" ]; then
  API_TOKEN="${SPUG_API_TOKEN-}"
  if [ -n "$API_TOKEN" ]; then
    SPUG_BASE="${SPUG_URL:-https://spug.itopcms.com}"
    log "GITEA_TOKEN 未注入，从配置中心拉取"
    curl -fsS \
      "${SPUG_BASE}/api/apis/config/?apiToken=${API_TOKEN}&format=env&noPrefix=1" \
      -o /tmp/spug.env
    set -a
    # shellcheck disable=SC1091
    . /tmp/spug.env
    set +a
    rm -f /tmp/spug.env
    TOKEN="${GITEA_TOKEN-}"
  fi
fi

OWNER="${GITEA_OWNER:-astrueus}"
GITEA_REPO="${GITEA_REPO:-doc}"
BASE="${GITEA_URL:-https://git.itopcms.com}"
VERSION="${TAG#v}"
PKG="doc_${VERSION}_linux_amd64.tar.gz"
URL="$BASE/$OWNER/$GITEA_REPO/releases/download/$TAG/$PKG"
PKG_FILE="/tmp/${GITEA_REPO}_${TAG}_${PKG}"

log "TAG=$TAG VERSION=$VERSION cwd=$(pwd)"
log "TOKEN 长度: ${#TOKEN}"
log "下载 $URL"

CURL_OPTS=(-fL --retry 3 --retry-delay 2)
if [ -n "$TOKEN" ]; then
  CURL_OPTS+=(-H "Authorization: token $TOKEN")
fi
curl "${CURL_OPTS[@]}" "$URL" -o "$PKG_FILE"

log "解压到当前目录（待发布版本目录）"
tar -xzf "$PKG_FILE" -C .

if [ ! -f ./doc ]; then
  log "解压后找不到 ./doc，请检查包内容"
  exit 1
fi
chmod 755 ./doc

log "二进制自检"
./doc version
export TAG
log "解包完成"
