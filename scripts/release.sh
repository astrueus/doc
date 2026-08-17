#!/usr/bin/env bash
# ============================================================
#  Doc 一键发版（Linux / macOS）：编译 → 打包 →（可选）tag / Gitea Release
#
#  Usage:
#    release.sh <version> [all|linux|windows] [options...]
#
#  Options:
#    --env=PATH     env 文件（默认：存在则用 scripts/.env.release）
#    --draft        创建草稿 Release
#    --dry-run      只编译+打包，不打 tag、不调 Gitea API
#    --skip-tag     跳过 git tag / push
#    -h|--help
#
#  Examples:
#    ./scripts/release.sh 0.0.1-test linux --dry-run
#    ./scripts/release.sh 0.0.1-test all --env=scripts/.env.release --draft
#    ./scripts/release.sh 1.0.0 linux
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

die() { echo "[ERROR] $*" >&2; exit 1; }
log() { echo "$*"; }

usage() {
  cat <<'EOF'
Usage: release.sh <version> [all|linux|windows] [options...]

Options:
  --env=PATH     env file (default: scripts/.env.release if present)
  --draft        create draft release
  --dry-run      build+package only
  --skip-tag     skip git tag/push
  -h|--help

Examples:
  ./scripts/release.sh 0.0.1-test linux --dry-run
  ./scripts/release.sh 0.0.1-test all --env=scripts/.env.release --draft
  ./scripts/release.sh 1.0.0 linux
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

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
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

if [[ "$DRY_RUN" -eq 0 ]]; then
  if [[ -z "$OWNER" || -z "$REPO" || -z "$BASE" || -z "$TOKEN" ]]; then
    die "Missing Gitea env. Provide --env=scripts/.env.release or export:
  GITEA_URL / GITEA_TOKEN / GITEA_OWNER / GITEA_REPO
Or use --dry-run to build+package only.
See: scripts/.env.release.example"
  fi
  command -v curl >/dev/null 2>&1 || die "curl is required for Gitea API"
fi

# ---------- JSON parsing ----------
# 为减少外部依赖，release.sh 直接使用 scripts/lib/json.sh
# （无需 jq/python/go）
# 适合：Gitea Release API 这种 JSON 响应（几十 KB 以内）
source "$ROOT/scripts/lib/json.sh"

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
  log "[DryRun] skipped tag / Gitea Release. packages are under release/"
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
{"tag_name":"$TAG","name":"Doc $TAG","body":"Auto release $TAG","draft":$DRAFT_JSON,"prerelease":false}
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
