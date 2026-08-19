#!/bin/bash
#
# spug_pre.sh — Spug 自定义发布「目标主机」前置脚本
# 从 Gitea Release 下载 tar.gz，解压到 WWW，并在网站根目录做二进制自检。
#
# 版本号（自定义发布不要用 SPUG_GIT_TAG，那个只有常规发布基于 Tag 才有值）：
#   TAG            可选覆盖
#   SPUG_RELEASE   申请单里填的版本，例如 v2.3.0 或 2.3.0
#
# Token（配置中心的 GITEA_TOKEN 不会自动变成环境变量）：
#   GITEA_TOKEN    自定义全局变量或已 export 时可用；set -u 下必须带默认值
#   SPUG_API_TOKEN Spug 钩子内置；为空时用它拉配置中心
#   SPUG_URL       配置中心地址，默认 https://spug.itopcms.com（目标机要能访问）
#
set -euo pipefail

log() { echo "[spug-pre] $*"; }

WWW=/data/wwwroot/doc.itopcms.com
OWNER="${GITEA_OWNER:-astrueus}"
GITEA_REPO="${GITEA_REPO:-doc}"
BASE="${GITEA_URL:-https://git.itopcms.com}"
SPUG_BASE="${SPUG_URL:-https://spug.itopcms.com}"

TAG="${TAG-}"
if [ -z "$TAG" ]; then
  TAG="${SPUG_RELEASE-}"
fi
TAG="${TAG:?请填写版本号：申请单 SPUG_RELEASE 或变量 TAG，例如 v2.3.0}"
case "$TAG" in
  v*) ;;
  *) TAG="v$TAG" ;;
esac

TOKEN="${GITEA_TOKEN-}"
if [ -z "$TOKEN" ]; then
  API_TOKEN="${SPUG_API_TOKEN-}"
  if [ -n "$API_TOKEN" ]; then
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

VERSION="${TAG#v}"
PKG="doc_${VERSION}_linux_amd64.tar.gz"
URL="$BASE/$OWNER/$GITEA_REPO/releases/download/$TAG/$PKG"
PKG_FILE="/tmp/${GITEA_REPO}_${TAG}_${PKG}"

log "TAG=$TAG VERSION=$VERSION"
log "TOKEN 长度: ${#TOKEN}"
log "下载 $URL"
mkdir -p "$WWW"

CURL_OPTS=(-fL --retry 3 --retry-delay 2)
if [ -n "$TOKEN" ]; then
  CURL_OPTS+=(-H "Authorization: token $TOKEN")
fi
curl "${CURL_OPTS[@]}" "$URL" -o "$PKG_FILE"

if [ -f "$WWW/doc" ]; then
  cp -f "$WWW/doc" "/tmp/doc.bak.$(date +%s)" || true
fi

log "解压到 $WWW"
tar -xzf "$PKG_FILE" -C "$WWW"

if [ ! -f "$WWW/doc" ]; then
  log "解压后找不到 $WWW/doc，请检查包内容"
  exit 1
fi
chmod 755 "$WWW/doc"

# Spug 默认 cwd 是 /tmp；doc 的 init 按 cwd 找 web/static/fonts，必须先 cd
log "二进制自检"
cd "$WWW"
./doc version

export TAG
log "前置完成"
