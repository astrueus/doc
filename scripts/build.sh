#!/usr/bin/env bash
# ============================================================
#  Doc multi-platform build script (Linux / macOS)
#
#  Usage:
#    build.sh [--target=all|linux|windows]      (-t)
#             [--mode=debug|release]            (-m)
#             [--version=X.Y.Z]                 (-v)
#             [--toolchain=zig|mingw]           (-x)
#             [-h|--help]
#
#  Defaults:
#    --target=all
#    --mode=debug
#    --toolchain=zig            (only affects Windows target)
#    --version                  auto-detected via `git describe`
#
#  Toolchain:
#    Linux    native gcc/clang (host toolchain; --toolchain is ignored)
#    Windows  Zig by default (zig cc -target x86_64-windows-gnu)
#             --toolchain=mingw uses x86_64-w64-mingw32-gcc instead
#             (alias: mingw-w64)
#
#  Build modes:
#    debug    output to project root: doc / doc.exe
#    release  output to dist/ with -s strip
#
#  Examples:
#    build.sh
#    build.sh --target=windows
#    build.sh --target=windows --toolchain=mingw
#    build.sh --target=windows --toolchain=mingw --version=1.2.0
#    build.sh --mode=release
#    build.sh -t linux -m release -v 2.0.0
#    build.sh -t windows -x mingw -v 1.2.0
# ============================================================

set -u

TARGET="all"
MODE="debug"
VERSION=""
TOOLCHAIN="zig"
BUILD_OK=1

die() { echo "[ERROR] $*" >&2; exit 1; }
tolower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

usage() {
    cat <<'EOF'

Usage: build.sh [--target=all|linux|windows]      (-t)
                [--mode=debug|release]            (-m)
                [--version=X.Y.Z]                 (-v)
                [--toolchain=zig|mingw]           (-x)
                [-h|--help]

Defaults:
  --target=all
  --mode=debug
  --toolchain=zig            (only affects Windows target)
  --version                  auto-detected via `git describe`

Toolchain:
  Linux    native gcc/clang (host toolchain; --toolchain is ignored)
  Windows  Zig by default (zig cc -target x86_64-windows-gnu)
           --toolchain=mingw uses x86_64-w64-mingw32-gcc instead
           (alias: mingw-w64)

Build modes:
  debug    output to project root: doc / doc.exe
  release  output to dist/ with -s strip

All flags support four forms: --key=value, --key value, -k=value, -k value.

Examples:
  build.sh
  build.sh --target=windows
  build.sh --target=windows --toolchain=mingw
  build.sh --target=windows --toolchain=mingw --version=1.2.0
  build.sh --mode=release
  build.sh -t linux -m release -v 2.0.0
  build.sh -t windows -x mingw -v 1.2.0

EOF
    exit 0
}

# 支持 --key=value / --key value / -k=value / -k value
parse_args() {
    while (( $# > 0 )); do
        local key="$1" val="" has_val=0
        case "$key" in
            -h|--help|help) usage ;;
            --*=*|-[!-]=*)
                val="${key#*=}"
                key="${key%%=*}"
                has_val=1
                ;;
            --*|-[!-])
                if (( $# >= 2 )) && [[ "$2" != -* ]]; then
                    val="$2"
                    has_val=1
                    shift
                fi
                ;;
            *) die "unexpected argument: $key (use --key=value or -k value)" ;;
        esac

        # 短标签归一化到长标签
        case "$key" in
            -t) key="--target" ;;
            -m) key="--mode" ;;
            -v) key="--version" ;;
            -x) key="--toolchain" ;;
        esac

        case "$key" in
            --target)
                (( has_val )) || die "--target/-t requires a value"
                TARGET="$(tolower "$val")"
                [[ "$TARGET" == "win" ]] && TARGET="windows"
                ;;
            --mode)
                (( has_val )) || die "--mode/-m requires a value"
                MODE="$(tolower "$val")"
                ;;
            --version)
                (( has_val )) || die "--version/-v requires a value"
                VERSION="$val"
                ;;
            --toolchain)
                (( has_val )) || die "--toolchain/-x requires a value"
                TOOLCHAIN="$(tolower "$val")"
                [[ "$TOOLCHAIN" == "mingw-w64" ]] && TOOLCHAIN="mingw"
                ;;
            *) die "unknown option: $key" ;;
        esac
        shift
    done
}

validate() {
    case "$TARGET"    in all|linux|windows) ;; *) die "unknown target: $TARGET (expect all|linux|windows)" ;; esac
    case "$MODE"      in debug|release)     ;; *) die "unknown mode: $MODE (expect debug|release)" ;; esac
    case "$TOOLCHAIN" in zig|mingw)         ;; *) die "unknown toolchain: $TOOLCHAIN (expect zig|mingw)" ;; esac
}

resolve_version() {
    [[ -n "$VERSION" ]] && return 0
    VERSION="$(git describe --tags --always --dirty 2>/dev/null || true)"
    [[ -n "$VERSION" ]] && return 0
    VERSION="$(git rev-parse --short HEAD 2>/dev/null || true)"
    [[ -n "$VERSION" ]] && return 0
    VERSION="dev"
}

need_zig() {
    [[ "$TARGET" == "linux" ]] && return 1
    [[ "$TARGET" == "windows" && "$TOOLCHAIN" == "mingw" ]] && return 1
    return 0
}

do_linux() {
    echo "[BUILD] Linux amd64 -> $OUT_LINUX (native)"
    if ! command -v gcc >/dev/null 2>&1 && ! command -v clang >/dev/null 2>&1; then
        echo "[ERROR] gcc/clang not found. Install build-essential or equivalent"
        BUILD_OK=0
        return 0
    fi
    (
        unset GOOS GOARCH CC
        export CGO_ENABLED=1
        go build -ldflags "$LDFLAGS" -o "$OUT_LINUX" ./cmd/doc
    )
    if [[ $? -ne 0 ]]; then
        echo "[ERROR] Linux build failed"
        BUILD_OK=0
    else
        echo "[OK]    $OUT_LINUX"
    fi
}

do_windows() {
    if [[ "$TOOLCHAIN" == "mingw" ]]; then
        echo "[BUILD] Windows amd64 -> $OUT_WINDOWS (MinGW-w64)"
        if ! command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
            echo "[ERROR] x86_64-w64-mingw32-gcc not found. Install mingw-w64 (e.g. apt install mingw-w64)"
            BUILD_OK=0
            return 0
        fi
        CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
            go build -ldflags "$LDFLAGS" -o "$OUT_WINDOWS" ./cmd/doc
    else
        echo "[BUILD] Windows amd64 -> $OUT_WINDOWS (Zig)"
        CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="zig cc -target x86_64-windows-gnu" \
            go build -ldflags "$LDFLAGS" -o "$OUT_WINDOWS" ./cmd/doc
    fi

    if [[ $? -ne 0 ]]; then
        if [[ "$TOOLCHAIN" == "mingw" ]]; then
            echo "[ERROR] Windows build failed. Check mingw-w64 / x86_64-w64-mingw32-gcc"
        else
            echo "[ERROR] Windows build failed. Check Zig installation"
        fi
        BUILD_OK=0
    else
        echo "[OK]    $OUT_WINDOWS"
    fi
}

# ---------- main ----------
parse_args "$@"
validate

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

resolve_version

GO_VER="$(go version 2>/dev/null | awk '{print $3}')"
[[ -z "$GO_VER" ]] && GO_VER="unknown"

BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%S 2>/dev/null || echo unknown)"

LDFLAGS_COMMON="-X 'git.itopcms.com/jackliu/doc/internal/config.VERSION=${VERSION}' -X 'git.itopcms.com/jackliu/doc/internal/config.BUILD_TIME=${BUILD_TIME}' -X 'git.itopcms.com/jackliu/doc/internal/config.GO_VERSION=${GO_VER}'"

if [[ "$MODE" == "release" ]]; then
    mkdir -p dist
    OUT_LINUX="dist/doc_linux_amd64"
    OUT_WINDOWS="dist/doc_windows_amd64.exe"
    LDFLAGS="-w -s ${LDFLAGS_COMMON}"
else
    OUT_LINUX="doc"
    OUT_WINDOWS="doc.exe"
    LDFLAGS="-w ${LDFLAGS_COMMON}"
fi

echo
echo "========================================"
echo " Doc Build"
echo " Mode           : $MODE"
echo " Target         : $TARGET"
echo " Version        : $VERSION"
echo " Build Time     : $BUILD_TIME"
echo " Linux toolchain: native (gcc/clang)"
echo " Win toolchain  : $TOOLCHAIN"
echo "========================================"
echo

if ! command -v go >/dev/null 2>&1; then
    echo "[ERROR] go not found. Please install Go 1.25+"
    exit 1
fi

if need_zig && ! command -v zig >/dev/null 2>&1; then
    echo "[ERROR] zig not found. Install Zig and add it to PATH"
    echo "        https://ziglang.org/download/"
    exit 1
fi

echo "[INFO] go mod tidy ..."
if ! go mod tidy; then
    echo "[ERROR] go mod tidy failed"
    exit 1
fi

case "$TARGET" in
    all)     do_linux; do_windows ;;
    linux)   do_linux ;;
    windows) do_windows ;;
esac

echo
if [[ "$BUILD_OK" == "1" ]]; then
    echo "[SUCCESS] build completed"
    exit 0
else
    echo "[FAILED] one or more targets failed to build"
    exit 1
fi
