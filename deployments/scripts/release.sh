#!/usr/bin/env bash
# ============================================================
#  Doc 一键发版（Linux / macOS）：编译 → 打包 →（可选）tag / Gitea Release
#  默认只发 Gitea。--github 在 Gitea 成功后等镜像 tag 再发 GitHub。
#  --github-only 不编包：核 Gitea 的 tag/附件，下载后再发 GitHub。
#
#  Usage:
#    release.sh <version> [all|linux|windows] [options...]
#
#  Options:
#    --env=PATH           env 文件（默认：存在则用 deployments/scripts/.env.release）
#    --draft              创建草稿 Release
#    --dry-run            只编译+打包，不打 tag、不调发布 API
#    --skip-tag           跳过 git tag / push
#    --github             Gitea 发完后再发 GitHub（等 tag 镜像，commit 须一致）
#    --github-only        只发 GitHub（先确认 Gitea 已有 tag 和包）
#    --github-wait=SEC    等待 GitHub tag 的超时秒数，默认 90
#    -h|--help
#
#  Examples:
#    ./deployments/scripts/release.sh 0.0.1-test linux --dry-run
#    ./deployments/scripts/release.sh 2.3.2 linux --github
#    ./deployments/scripts/release.sh 2.3.2 linux --github-only
#    ./deployments/scripts/release.sh 1.0.0 linux
#
#  产物（与 Windows 脚本一致）：
#    release/doc_<version>_windows_amd64.zip
#    release/doc_<version>_linux_amd64.tar.gz
# ============================================================

set -euo pipefail

VERSION=""
TARGET="linux"
ENV_FILE=""
DRAFT=0
DRY_RUN=0
SKIP_TAG=0
GITHUB=0
GITHUB_ONLY=0
GITHUB_WAIT=90

die() { echo "[ERROR] $*" >&2; exit 1; }
log() { echo "$*"; }

usage() {
  cat <<'EOF'
Usage: release.sh <version> [all|linux|windows] [options...]

Options:
  --env=PATH           env file (default: deployments/scripts/.env.release if present)
  --draft              create draft release
  --dry-run            build+package only
  --skip-tag           skip git tag/push
  --github             after Gitea, wait for mirrored tag then publish GitHub
  --github-only        GitHub only (Gitea tag+assets must already exist)
  --github-wait=SEC    GitHub tag wait timeout seconds (default 90)
  -h|--help

Examples:
  ./deployments/scripts/release.sh 0.0.1-test linux --dry-run
  ./deployments/scripts/release.sh 2.3.2 linux --github
  ./deployments/scripts/release.sh 2.3.2 linux --github-only
EOF
  exit 0
}

# ---------- parse args ----------
if (( $# < 1 )); then
  usage
fi

case "$1" in
  -h|--help|help) usage ;;
esac

VERSION="$1"
shift

while (( $# > 0 )); do
  arg="$1"
  case "$arg" in
    -h|--help|help) usage ;;
    all|linux|windows) TARGET="$arg" ;;
    --draft) DRAFT=1 ;;
    --dry-run) DRY_RUN=1 ;;
    --skip-tag) SKIP_TAG=1 ;;
    --github) GITHUB=1 ;;
    --github-only) GITHUB_ONLY=1 ;;
    --github-wait=*) GITHUB_WAIT="${arg#--github-wait=}" ;;
    --github-wait)
      (( $# >= 2 )) || die "--github-wait requires seconds"
      GITHUB_WAIT="$2"
      shift
      ;;
    --env=*) ENV_FILE="${arg#--env=}" ;;
    --env)
      (( $# >= 2 )) || die "--env requires a path"
      ENV_FILE="$2"
      shift
      ;;
    *) die "unknown argument: $arg" ;;
  esac
  shift
done

[[ -n "$VERSION" ]] || die "version is required"
case "$TARGET" in
  all|linux|windows) ;;
  *) die "unknown target: $TARGET (expect all|linux|windows)" ;;
esac
if [[ "$GITHUB" -eq 1 && "$GITHUB_ONLY" -eq 1 ]]; then
  die "--github 与 --github-only 不能同时使用"
fi
[[ "$GITHUB_WAIT" =~ ^[1-9][0-9]*$ ]] || die "--github-wait 必须是正整数（秒），当前: $GITHUB_WAIT"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPT_DIR"
while [[ -n "$ROOT" && "$ROOT" != "/" && ! ( -f "$ROOT/go.mod" && -d "$ROOT/cmd/doc" ) ]]; do
  ROOT="$(cd "$ROOT/.." && pwd)"
done
[[ -f "$ROOT/go.mod" && -d "$ROOT/cmd/doc" ]] || die "cannot locate repo root (need go.mod and cmd/doc); script dir: $SCRIPT_DIR"
cd "$ROOT"

# ---------- load env ----------
load_dotenv() {
  local path="$1"
  [[ -f "$path" ]] || die "env file not found: $path"
  log "Load env: $path"
  # shellcheck disable=SC1090
  set -a
  # strip comments / blank; support optional export prefix and quotes
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ -z "$line" || "$line" == \#* ]] && continue
    [[ "$line" == export\ * ]] && line="${line#export }"
    [[ "$line" == *=* ]] || continue
    local name="${line%%=*}"
    local val="${line#*=}"
    name="${name%"${name##*[![:space:]]}"}"
    name="${name#"${name%%[![:space:]]*}"}"
    val="${val#"${val%%[![:space:]]*}"}"
    val="${val%"${val##*[![:space:]]}"}"
    if [[ ( "$val" == \"*\" && "$val" == *\" ) || ( "$val" == \'*\' && "$val" == *\' ) ]]; then
      val="${val:1:${#val}-2}"
    fi
    export "$name=$val"
  done < "$path"
  set +a
}

if [[ -z "$ENV_FILE" ]]; then
  if [[ -f "$SCRIPT_DIR/.env.release" ]]; then
    ENV_FILE="$SCRIPT_DIR/.env.release"
  fi
fi
if [[ -n "$ENV_FILE" ]]; then
  if [[ "$ENV_FILE" != /* ]]; then
    ENV_FILE="$ROOT/$ENV_FILE"
  fi
  load_dotenv "$ENV_FILE"
fi

TAG="v${VERSION}"
OWNER="${GITEA_OWNER:-}"
REPO="${GITEA_REPO:-}"
BASE="${GITEA_URL:-}"
TOKEN="${GITEA_TOKEN:-}"
BASE="${BASE%/}"

GH_OWNER="${GITHUB_OWNER:-$OWNER}"
GH_REPO="${GITHUB_REPO:-$REPO}"
GH_TOKEN="${GITHUB_TOKEN:-}"
GH_API="${GITHUB_API:-https://api.github.com}"
GH_UPLOAD="${GITHUB_UPLOAD:-https://uploads.github.com}"
GH_API="${GH_API%/}"
GH_UPLOAD="${GH_UPLOAD%/}"

NEED_GITEA=0
NEED_GITHUB=0
if [[ "$GITHUB_ONLY" -eq 1 ]]; then
  NEED_GITEA=1
  [[ "$DRY_RUN" -eq 0 ]] && NEED_GITHUB=1
elif [[ "$DRY_RUN" -eq 0 ]]; then
  NEED_GITEA=1
  [[ "$GITHUB" -eq 1 ]] && NEED_GITHUB=1
fi

if [[ "$NEED_GITEA" -eq 1 ]]; then
  if [[ -z "$OWNER" || -z "$REPO" || -z "$BASE" || -z "$TOKEN" ]]; then
    die "Missing Gitea env. Provide --env=deployments/scripts/.env.release or export:
  GITEA_URL / GITEA_TOKEN / GITEA_OWNER / GITEA_REPO
Or use --dry-run to build+package only（--github-only 除外，仍要能读 Gitea）。
See: deployments/scripts/.env.release.example"
  fi
  command -v curl >/dev/null 2>&1 || die "curl is required for Gitea/GitHub API"
fi
if [[ "$NEED_GITHUB" -eq 1 ]]; then
  if [[ -z "$GH_TOKEN" || -z "$GH_OWNER" || -z "$GH_REPO" ]]; then
    die "Missing GitHub env. 使用 --github / --github-only 时需要：
  GITHUB_TOKEN
  GITHUB_OWNER / GITHUB_REPO（可省略，默认与 GITEA_OWNER / GITEA_REPO 相同）"
  fi
  command -v curl >/dev/null 2>&1 || die "curl is required for GitHub API"
fi

# ---------- JSON parsing ----------
# 为减少外部依赖，release.sh 直接使用 deployments/scripts/lib/json.sh
# （无需 jq/python/go）
# 适合：Gitea / GitHub Release API 这种 JSON 响应（几十 KB 以内）
source "$SCRIPT_DIR/lib/json.sh"

norm_sha() { printf '%s' "$1" | tr 'A-F' 'a-f'; }

expected_asset_names() {
  local names=()
  if [[ "$TARGET" == "all" || "$TARGET" == "windows" ]]; then
    names+=("doc_${VERSION}_windows_amd64.zip")
  fi
  if [[ "$TARGET" == "all" || "$TARGET" == "linux" ]]; then
    names+=("doc_${VERSION}_linux_amd64.tar.gz")
  fi
  printf '%s\n' "${names[@]}"
}

curl_split() {
  # stdin unused; last line of $1 is HTTP code
  local blob="$1"
  CURL_CODE="$(printf '%s' "$blob" | tail -n1)"
  CURL_BODY="$(printf '%s' "$blob" | sed '$d')"
}

gitea_tag_commit() {
  local blob code body flat sha
  blob="$(curl -sS -w "\n%{http_code}" -H "Authorization: token ${TOKEN}" \
    "$BASE/api/v1/repos/$OWNER/$REPO/tags/$TAG" || true)"
  curl_split "$blob"
  code="$CURL_CODE"
  body="$CURL_BODY"
  [[ "$code" == "404" ]] && return 1
  [[ "$code" == "200" ]] || die "Gitea GET tag HTTP $code: $body"
  flat="$(json_flatten <<<"$body")"
  sha="$(json_get "$flat" '["commit","sha"]')"
  [[ -n "$sha" && "$sha" != "null" ]] || sha="$(json_get "$flat" '["id"]')"
  [[ -n "$sha" && "$sha" != "null" ]] || return 1
  printf '%s' "$sha"
}

github_tag_commit() {
  local blob code body flat typ sha blob2 body2 flat2
  blob="$(curl -sS -w "\n%{http_code}" \
    -H "Authorization: Bearer ${GH_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    -H "User-Agent: doc-release-script" \
    "$GH_API/repos/$GH_OWNER/$GH_REPO/git/ref/tags/$TAG" || true)"
  curl_split "$blob"
  code="$CURL_CODE"
  body="$CURL_BODY"
  [[ "$code" == "404" ]] && return 1
  [[ "$code" == "200" ]] || die "GitHub GET tag HTTP $code: $body"
  flat="$(json_flatten <<<"$body")"
  typ="$(json_get "$flat" '["object","type"]')"
  sha="$(json_get "$flat" '["object","sha"]')"
  if [[ "$typ" == "tag" && -n "$sha" && "$sha" != "null" ]]; then
    blob2="$(curl -sS -f \
      -H "Authorization: Bearer ${GH_TOKEN}" \
      -H "Accept: application/vnd.github+json" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      -H "User-Agent: doc-release-script" \
      "$GH_API/repos/$GH_OWNER/$GH_REPO/git/tags/$sha")"
    flat2="$(json_flatten <<<"$blob2")"
    sha="$(json_get "$flat2" '["object","sha"]')"
  fi
  [[ -n "$sha" && "$sha" != "null" ]] || return 1
  printf '%s' "$sha"
}

wait_github_tag() {
  local expected="$1"
  local exp_n got got_n
  exp_n="$(norm_sha "$expected")"
  log "[github] 等待 GitHub 出现 tag $TAG（commit $expected，最多 ${GITHUB_WAIT}s）..."
  local start="$SECONDS"
  while true; do
    if got="$(github_tag_commit)"; then
      got_n="$(norm_sha "$got")"
      if [[ "$got_n" == "$exp_n" ]]; then
        log "  GitHub tag $TAG 已同步，commit 一致"
        return 0
      fi
      die "GitHub 上已有 tag $TAG，但 commit 为 $got，与期望 $expected 不一致。请核对推送镜像，勿让 GitHub 按默认分支自动建 tag。"
    fi
    if (( SECONDS - start >= GITHUB_WAIT )); then
      die "等待 ${GITHUB_WAIT}s 后 GitHub 仍没有 tag $TAG（镜像未完成）。
Gitea 侧若已发布成功，请稍后执行：
  ./deployments/scripts/release.sh $VERSION $TARGET --github-only
不要移动或重打已有 tag。"
    fi
    log "  尚未看到 GitHub tag $TAG，5s 后重试..."
    sleep 5
  done
}

publish_github_release() {
  local body_text="$1"
  shift
  local files=("$@")
  (( ${#files[@]} > 0 )) || die "没有可上传到 GitHub 的附件"
  log "[github] 创建 GitHub Release $TAG ..."
  local draft_json=false
  [[ "$DRAFT" -eq 1 ]] && draft_json=true
  local escaped create_body create_resp create_code create_json create_flat release_id
  escaped="$(printf '%s' "$body_text" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' -e 's/'"$(printf '\t')"'/\\t/g' | tr '\n' ' ')"
  create_body=$(cat <<EOF
{"tag_name":"$TAG","name":"doc $TAG","body":"$escaped","draft":$draft_json,"prerelease":false}
EOF
)

  create_resp="$(curl -sS -w "\n%{http_code}" -X POST \
    -H "Authorization: Bearer ${GH_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    -H "User-Agent: doc-release-script" \
    -H "Content-Type: application/json" \
    -d "$create_body" \
    "$GH_API/repos/$GH_OWNER/$GH_REPO/releases" || true)"
  curl_split "$create_resp"
  create_code="$CURL_CODE"
  create_json="$CURL_BODY"
  release_id=""
  if [[ "$create_code" == "201" || "$create_code" == "200" ]]; then
    create_flat="$(json_flatten <<<"$create_json")"
    release_id="$(json_get "$create_flat" '["id"]')"
    log "  created GitHub release id=$release_id"
  else
    log "  创建失败 (HTTP $create_code)，尝试按 tag 读取已有 Release..."
    local get_resp get_flat
    get_resp="$(curl -sS -f \
      -H "Authorization: Bearer ${GH_TOKEN}" \
      -H "Accept: application/vnd.github+json" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      -H "User-Agent: doc-release-script" \
      "$GH_API/repos/$GH_OWNER/$GH_REPO/releases/tags/$TAG")"
    get_flat="$(json_flatten <<<"$get_resp")"
    release_id="$(json_get "$get_flat" '["id"]')"
    log "  reuse GitHub release id=$release_id"
  fi
  [[ -n "$release_id" && "$release_id" != "null" ]] || die "failed to resolve GitHub release id for $TAG"

  local assets_json assets_flat file name raw_name
  assets_json="$(curl -sS -f \
    -H "Authorization: Bearer ${GH_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    -H "User-Agent: doc-release-script" \
    "$GH_API/repos/$GH_OWNER/$GH_REPO/releases/$release_id/assets")"
  assets_flat="$(json_flatten <<<"$assets_json")"

  for file in "${files[@]}"; do
    name="$(basename "$file")"
    raw_name="$(printf '"%s"' "$name")"
    while IFS= read -r idx; do
      local old_id
      old_id="$(json_get "$assets_flat" "[$idx,\"id\"]")"
      [[ -z "$old_id" || "$old_id" == "null" ]] && continue
      log "  delete old GitHub asset: $name (id=$old_id)"
      curl -sS -f -X DELETE \
        -H "Authorization: Bearer ${GH_TOKEN}" \
        -H "Accept: application/vnd.github+json" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        -H "User-Agent: doc-release-script" \
        "$GH_API/repos/$GH_OWNER/$GH_REPO/releases/assets/$old_id" >/dev/null
    done < <(json_find_index "$assets_flat" '[]' name "$raw_name")

    log "  upload GitHub: $name"
    curl -sS -f -X POST \
      -H "Authorization: Bearer ${GH_TOKEN}" \
      -H "Accept: application/vnd.github+json" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      -H "User-Agent: doc-release-script" \
      -H "Content-Type: application/octet-stream" \
      --data-binary @"$file" \
      "$GH_UPLOAD/repos/$GH_OWNER/$GH_REPO/releases/$release_id/assets?name=$(printf '%s' "$name" | sed 's/ /%20/g')" >/dev/null
  done
  log "GitHub Release: https://github.com/$GH_OWNER/$GH_REPO/releases/tag/$TAG"
}

# ---------- GitHubOnly：不编译 ----------
if [[ "$GITHUB_ONLY" -eq 1 ]]; then
  log "[github-only] 核对 Gitea $BASE/$OWNER/$REPO 的 $TAG ..."
  GITEA_SHA=""
  GITEA_SHA="$(gitea_tag_commit)" || die "Gitea 上没有 tag $TAG，不能只发 GitHub。请先完整发版或检查版本号。"

  GITEA_REL_JSON="$(curl -sS -w "\n%{http_code}" -H "Authorization: token ${TOKEN}" \
    "$BASE/api/v1/repos/$OWNER/$REPO/releases/tags/$TAG" || true)"
  curl_split "$GITEA_REL_JSON"
  [[ "$CURL_CODE" == "200" ]] || die "Gitea 上有 tag $TAG，但没有对应 Release（HTTP $CURL_CODE）。不能只发 GitHub。"
  GITEA_REL_FLAT="$(json_flatten <<<"$CURL_BODY")"
  GITEA_REL_ID="$(json_get "$GITEA_REL_FLAT" '["id"]')"
  GITEA_REL_BODY="$(json_get "$GITEA_REL_FLAT" '["body"]')"
  [[ -n "$GITEA_REL_ID" && "$GITEA_REL_ID" != "null" ]] || die "无法解析 Gitea Release id"
  ASSETS_JSON="$(curl -sS -f -H "Authorization: token ${TOKEN}" \
    "$BASE/api/v1/repos/$OWNER/$REPO/releases/$GITEA_REL_ID/assets")"
  ASSETS_FLAT="$(json_flatten <<<"$ASSETS_JSON")"
  DOWNLOADS=()
  OUT_DIR="$ROOT/release"
  WANT_NAMES=()
  while IFS= read -r _n; do
    [[ -n "$_n" ]] && WANT_NAMES+=("$_n")
  done < <(expected_asset_names)
  (( ${#WANT_NAMES[@]} > 0 )) || die "Target=$TARGET 没有对应附件名"
  for want in "${WANT_NAMES[@]}"; do
    raw_want="$(printf '"%s"' "$want")"
    found_id=""
    found_idx=""
    while IFS= read -r idx; do
      found_idx="$idx"
      found_id="$(json_get "$ASSETS_FLAT" "[$idx,\"id\"]")"
      break
    done < <(json_find_index "$ASSETS_FLAT" '[]' name "$raw_want")
    [[ -n "$found_id" && "$found_id" != "null" ]] || die "Gitea Release $TAG 缺少附件: $want（当前 Target=$TARGET）"
    if [[ "$DRY_RUN" -eq 1 ]]; then
      continue
    fi
    mkdir -p "$OUT_DIR"
    dest="$OUT_DIR/$want"
    dl_url="$(json_get "$ASSETS_FLAT" "[$found_idx,\"browser_download_url\"]")"
    want_size="$(json_get "$ASSETS_FLAT" "[$found_idx,\"size\"]")"
    if [[ -z "$dl_url" || "$dl_url" == "null" ]]; then
      dl_url="$BASE/$OWNER/$REPO/releases/download/$TAG/$want"
    fi
    log "  从 Gitea 下载 $want"
    curl -sS -fL --retry 3 --retry-delay 2 \
      -H "Authorization: token ${TOKEN}" \
      -o "$dest" \
      "$dl_url"
    [[ -s "$dest" ]] || die "下载失败或空文件: $dest"
    got_size="$(wc -c < "$dest" | tr -d ' ')"
    if [[ -n "$want_size" && "$want_size" != "null" && "$got_size" != "$want_size" ]]; then
      die "下载大小不符: $want 期望 ${want_size} 字节，实际 ${got_size}。若约几百字节，多半下到了附件 JSON 而不是包。"
    fi
    if (( got_size < 1024 )); then
      die "下载文件过小 (${got_size} 字节): $want"
    fi
    DOWNLOADS+=("$dest")
  done

  if [[ "$DRY_RUN" -eq 1 ]]; then
    log "[DryRun] Gitea 已有 tag $TAG (commit $GITEA_SHA) 与附件 $(IFS=,; echo "${WANT_NAMES[*]}")。跳过下载与 GitHub。"
    log "Done."
    exit 0
  fi

  [[ -n "$GITEA_REL_BODY" && "$GITEA_REL_BODY" != "null" ]] || GITEA_REL_BODY="Auto release $TAG"
  wait_github_tag "$GITEA_SHA"
  publish_github_release "$GITEA_REL_BODY" "${DOWNLOADS[@]}"
  log "Done."
  exit 0
fi

# ---------- 1) Build ----------
log "[1/5] Build $TARGET release $VERSION ..."
BUILD_SH="$SCRIPT_DIR/build.sh"
[[ -x "$BUILD_SH" || -f "$BUILD_SH" ]] || die "build.sh not found: $BUILD_SH"
bash "$BUILD_SH" --target="$TARGET" --mode=release --version="$VERSION"

# ---------- 2) Package ----------
log "[2/5] Package ..."

rm -rf "$ROOT"/.release_stage_* 2>/dev/null || true
OUT_DIR="$ROOT/release"
mkdir -p "$OUT_DIR"
log "  output dir: $OUT_DIR"

ASSETS=()

copy_tree() {
  local src="$1" dst="$2"
  [[ -e "$src" ]] || die "source not found: $src"
  if [[ -f "$src" ]]; then
    mkdir -p "$(dirname "$dst")"
    cp -f "$src" "$dst"
    return
  fi
  mkdir -p "$dst"
  # trailing /. copies contents into dst
  cp -a "$src"/. "$dst"/
}

publish_shared_into_stage() {
  local stage="$1"

  [[ -d "$ROOT/web" ]] || die "missing required path: web/"
  copy_tree "$ROOT/web" "$stage/web"

  mkdir -p "$stage/conf"
  [[ -d "$ROOT/conf/lang" ]] || die "missing required path: conf/lang/"
  copy_tree "$ROOT/conf/lang" "$stage/conf/lang"
  [[ -f "$ROOT/conf/app.conf.example" ]] || die "missing required path: conf/app.conf.example"
  cp -f "$ROOT/conf/app.conf.example" "$stage/conf/app.conf.example"

  mkdir -p "$stage/uploads"

  mkdir -p "$stage/deployments"
  local sub
  for sub in spug systemd; do
    if [[ ! -d "$ROOT/deployments/$sub" ]]; then
      echo "[WARN] missing deployments/$sub (skip)" >&2
      continue
    fi
    copy_tree "$ROOT/deployments/$sub" "$stage/deployments/$sub"
  done

  if [[ -f "$ROOT/LICENSE.md" ]]; then
    cp -f "$ROOT/LICENSE.md" "$stage/LICENSE.md"
  fi
}

make_archive() {
  local source_dir="$1" archive_path="$2" format="$3"
  rm -f "$archive_path"
  (
    cd "$source_dir"
    case "$format" in
      zip)
        if command -v zip >/dev/null 2>&1; then
          zip -qr "$archive_path" .
        elif command -v tar >/dev/null 2>&1; then
          # bsdtar / GNU tar with auto-compress zip may not work; prefer zip
          die "zip command not found (required for windows .zip packages). Install zip."
        else
          die "zip not found"
        fi
        ;;
      tar.gz)
        command -v tar >/dev/null 2>&1 || die "tar not found; required for linux .tar.gz"
        tar -czf "$archive_path" .
        ;;
      *) die "unknown archive format: $format" ;;
    esac
  )
}

new_release_package() {
  local package_name="$1" format="$2" binary_path="$3" binary_name="$4"
  local archive_path="$OUT_DIR/$package_name"
  local stage
  stage="$(mktemp -d "$ROOT/.release_stage_XXXXXX")"

  cleanup_stage() { rm -rf "$stage"; }
  # shellcheck disable=SC2064
  trap cleanup_stage EXIT

  [[ -f "$binary_path" ]] || die "binary not found: $binary_path"
  cp -f "$binary_path" "$stage/$binary_name"
  if [[ "$binary_name" == "doc" ]]; then
    chmod 755 "$stage/$binary_name"
  fi
  publish_shared_into_stage "$stage"
  make_archive "$stage" "$archive_path" "$format"
  cleanup_stage
  trap - EXIT
  ASSETS+=("$archive_path")
}

if [[ "$TARGET" == "all" || "$TARGET" == "windows" ]]; then
  if [[ -f "$ROOT/dist/doc_windows_amd64.exe" ]]; then
    new_release_package \
      "doc_${VERSION}_windows_amd64.zip" \
      zip \
      "$ROOT/dist/doc_windows_amd64.exe" \
      "doc.exe"
  fi
fi

if [[ "$TARGET" == "all" || "$TARGET" == "linux" ]]; then
  if [[ -f "$ROOT/dist/doc_linux_amd64" ]]; then
    new_release_package \
      "doc_${VERSION}_linux_amd64.tar.gz" \
      tar.gz \
      "$ROOT/dist/doc_linux_amd64" \
      "doc"
  fi
fi

(( ${#ASSETS[@]} > 0 )) || die "no asset to upload, check dist/"

asset_names=""
for f in "${ASSETS[@]}"; do
  [[ -n "$asset_names" ]] && asset_names+=", "
  asset_names+="$(basename "$f")"
done
log "  packaged: $asset_names"

if [[ "$DRY_RUN" -eq 1 ]]; then
  echo
  log "[DryRun] skipped tag / Gitea / GitHub Release. packages are under release/"
  log "Done."
  exit 0
fi

# ---------- 3) Tag ----------
if [[ "$SKIP_TAG" -eq 0 ]]; then
  log "[3/5] Tag $TAG ..."
  if git tag --list "$TAG" | grep -qx "$TAG"; then
    point="$(git log -1 --format=%h "$TAG")"
    die "tag $TAG already exists (points to ${point}). Refusing to skip+push. Use a new version or delete the local tag after checking."
  fi
  git tag -a "$TAG" -m "Release $TAG"
  git push origin "refs/tags/$TAG"
else
  log "[3/5] Skip tag/push"
fi

# ---------- 4) Release ----------
log "[4/5] Create Release $TAG ..."
AUTH_HEADER="Authorization: token ${TOKEN}"
API="$BASE/api/v1/repos/$OWNER/$REPO"

DRAFT_JSON=false
[[ "$DRAFT" -eq 1 ]] && DRAFT_JSON=true

create_body=$(cat <<EOF
{"tag_name":"$TAG","name":"doc $TAG","body":"Auto release $TAG","draft":$DRAFT_JSON,"prerelease":false}
EOF
)

create_resp="$(curl -sS -w "\n%{http_code}" -X POST \
  -H "$AUTH_HEADER" -H "Content-Type: application/json" \
  -d "$create_body" \
  "$API/releases" || true)"
create_code="$(printf '%s' "$create_resp" | tail -n1)"
create_json="$(printf '%s' "$create_resp" | sed '$d')"

RELEASE_ID=""
if [[ "$create_code" == "201" || "$create_code" == "200" ]]; then
  create_flat="$(json_flatten <<<"$create_json")"
  RELEASE_ID="$(json_get "$create_flat" '["id"]')"
  log "  created release id=$RELEASE_ID"
else
  log "  create failed (HTTP $create_code), try fetch release by tag..."
  get_resp="$(curl -sS -f -H "$AUTH_HEADER" "$API/releases/tags/$TAG")"
  get_resp_flat="$(json_flatten <<<"$get_resp")"
  RELEASE_ID="$(json_get "$get_resp_flat" '["id"]')"
  log "  reuse release id=$RELEASE_ID"
fi
[[ -n "$RELEASE_ID" && "$RELEASE_ID" != "null" ]] || die "failed to resolve release id for $TAG"

# ---------- 5) Upload ----------
log "[5/5] Upload assets ..."
assets_json="$(curl -sS -f -H "$AUTH_HEADER" "$API/releases/$RELEASE_ID/assets")"
assets_flat="$(json_flatten <<<"$assets_json")"

for file in "${ASSETS[@]}"; do
  name="$(basename "$file")"
  raw_name="$(printf '"%s"' "$name")"
  # Gitea /releases/:id/assets 返回顶层数组，路径用 []
  while IFS= read -r idx; do
    old_id="$(json_get "$assets_flat" "[$idx,\"id\"]")"
    [[ -z "$old_id" || "$old_id" == "null" ]] && continue
    log "  delete old asset: $name (id=$old_id)"
    curl -sS -f -X DELETE -H "$AUTH_HEADER" \
      "$API/releases/$RELEASE_ID/assets/$old_id" >/dev/null
  done < <(json_find_index "$assets_flat" '[]' name "$raw_name")

  log "  upload: $name"
  curl -sS -f -X POST \
    -H "$AUTH_HEADER" \
    -F "attachment=@${file}" \
    "$API/releases/$RELEASE_ID/assets?name=$(printf '%s' "$name" | sed 's/ /%20/g')" >/dev/null
done

echo
log "Release published: $BASE/$OWNER/$REPO/releases/tag/$TAG"

if [[ "$GITHUB" -eq 1 ]]; then
  EXPECTED_SHA=""
  if EXPECTED_SHA="$(git rev-parse "$TAG^{commit}" 2>/dev/null)"; then
    :
  else
    EXPECTED_SHA="$(gitea_tag_commit)" || EXPECTED_SHA=""
  fi
  [[ -n "$EXPECTED_SHA" ]] || die "无法解析 $TAG 的 commit（本地与 Gitea 都没有）。GitHub 发布中止；Gitea 已成功，可用 --github-only 重试。"
  wait_github_tag "$EXPECTED_SHA"
  set +e
  publish_github_release "Auto release $TAG" "${ASSETS[@]}"
  gh_st=$?
  set -e
  if [[ "$gh_st" -ne 0 ]]; then
    log "[ERROR] Gitea 已发布成功，GitHub 上传失败。"
    log "补发： ./deployments/scripts/release.sh $VERSION $TARGET --github-only"
    exit 1
  fi
fi
