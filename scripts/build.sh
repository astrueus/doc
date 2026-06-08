#!/usr/bin/env bash
# ============================================================
#  Doc multi-platform build script (Linux)
#
#  Usage: build.sh [target] [mode|version|mingw] [version]
#
#  target       : all | linux | windows  (default: all)
#  mode         : debug | release        (default: debug)
#  version      : optional, auto-detected from git
#  win toolchain: mingw | mingw-w64    (default: zig for Windows cross-compile)
#
#  Examples:
#    build.sh
#    build.sh windows
#    build.sh windows mingw
#    build.sh windows mingw 1.2.0
#    build.sh all release
#    build.sh linux release 2.0.0
# ============================================================

TARGET="all"
MODE="debug"
VERSION=""
WIN_TOOLCHAIN="zig"
BUILD_OK=1

ARG1="${1:-}"
ARG2="${2:-}"
ARG3="${3:-}"
ARG4="${4:-}"

usage() {
    echo
    echo "Usage: build.sh [target] [mode|version|mingw] [version]"
    echo
    echo " target         : all | linux | windows   (default: all)"
    echo " mode           : debug | release          (default: debug)"
    echo " version        : optional, auto-detected from git"
    echo " win toolchain  : mingw | mingw-w64       (default: zig)"
    echo
    echo " Toolchain:"
    echo "   Linux         - native gcc/clang (host toolchain)"
    echo "   Windows       - Zig by default (zig cc -target x86_64-windows-gnu)"
    echo "                   pass mingw / mingw-w64 to use x86_64-w64-mingw32-gcc"
    echo
    echo " Build modes:"
    echo "   debug   - output to project root: doc / doc.exe"
    echo "   release - output to dist/ with -s strip"
    echo
    echo " Examples:"
    echo "   build.sh"
    echo "   build.sh windows"
    echo "   build.sh windows mingw"
    echo "   build.sh windows mingw 1.2.0"
    echo "   build.sh mingw windows release"
    echo "   build.sh all release"
    echo "   build.sh linux release 2.0.0"
    echo
    exit 0
}

tolower() {
    echo "$1" | tr '[:upper:]' '[:lower:]'
}

is_version() {
    [[ "$1" =~ ^[vV][0-9] ]] || [[ "$1" =~ ^[0-9]+\.[0-9] ]]
}

is_keyword() {
    local arg
    arg="$(tolower "$1")"
    case "$arg" in
        mingw|mingw-w64|debug|release|all|linux|windows|win) return 0 ;;
        *) return 1 ;;
    esac
}

detect_mingw() {
    local arg
    arg="$(tolower "$1")"
    if [[ "$arg" == "mingw" || "$arg" == "mingw-w64" ]]; then
        WIN_TOOLCHAIN="mingw"
    fi
}

try_set_version() {
    local arg="$1"
    [[ -z "$arg" ]] && return 0
    is_keyword "$arg" && return 0
    if is_version "$arg"; then
        VERSION="$arg"
    fi
}

parse_optional_args() {
    local a1="$1" a2="$2" a3="$3"
    local mode_arg
    mode_arg="$(tolower "$a1")"
    if [[ "$mode_arg" == "debug" ]]; then
        MODE="debug"
        try_set_version "$a2"
        try_set_version "$a3"
        return 0
    fi
    if [[ "$mode_arg" == "release" ]]; then
        MODE="release"
        try_set_version "$a2"
        try_set_version "$a3"
        return 0
    fi
    try_set_version "$a1"
    try_set_version "$a2"
    try_set_version "$a3"
}

parse_args() {
    for arg in "$@"; do
        detect_mingw "$arg"
    done

    if [[ -z "$ARG1" ]]; then
        validate_target
        return $?
    fi

    local arg1_lower
    arg1_lower="$(tolower "$ARG1")"

    if [[ "$arg1_lower" == "debug" ]]; then
        MODE="debug"
        TARGET="all"
        parse_optional_args "$ARG2" "$ARG3" "$ARG4"
        validate_target
        return $?
    fi
    if [[ "$arg1_lower" == "release" ]]; then
        MODE="release"
        TARGET="all"
        parse_optional_args "$ARG2" "$ARG3" "$ARG4"
        validate_target
        return $?
    fi
    if [[ "$arg1_lower" == "mingw" || "$arg1_lower" == "mingw-w64" ]]; then
        TARGET="$(tolower "$ARG2")"
        [[ "$TARGET" == "win" ]] && TARGET="windows"
        parse_optional_args "$ARG3" "$ARG4" ""
        validate_target
        return $?
    fi

    TARGET="$(tolower "$ARG1")"
    [[ "$TARGET" == "win" ]] && TARGET="windows"
    parse_optional_args "$ARG2" "$ARG3" "$ARG4"
    validate_target
    return $?
}

validate_target() {
    case "$TARGET" in
        all|linux|windows) ;;
        *)
            echo "[ERROR] unknown target: $TARGET"
            return 1
            ;;
    esac
    case "$MODE" in
        debug|release) ;;
        *)
            echo "[ERROR] unknown mode: $MODE"
            return 1
            ;;
    esac
    case "$WIN_TOOLCHAIN" in
        zig|mingw) ;;
        *)
            echo "[ERROR] unknown win toolchain: $WIN_TOOLCHAIN"
            return 1
            ;;
    esac
    return 0
}

resolve_version() {
    if [[ -n "$1" ]]; then
        VERSION="$1"
        return 0
    fi

    VERSION="$(git describe --tags --always --dirty 2>/dev/null || true)"
    if [[ -n "$VERSION" ]]; then
        return 0
    fi

    VERSION="$(git rev-parse --short HEAD 2>/dev/null || true)"
    if [[ -n "$VERSION" ]]; then
        return 0
    fi

    VERSION="dev"
}

need_zig() {
    if [[ "$TARGET" == "linux" ]]; then
        return 1
    fi
    if [[ "$TARGET" == "windows" && "$WIN_TOOLCHAIN" == "mingw" ]]; then
        return 1
    fi
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
        go build -ldflags "$LDFLAGS" -o "$OUT_LINUX" .
    )
    if [[ $? -ne 0 ]]; then
        echo "[ERROR] Linux build failed"
        BUILD_OK=0
    else
        echo "[OK]    $OUT_LINUX"
    fi
}

do_windows() {
    if [[ "$WIN_TOOLCHAIN" == "mingw" ]]; then
        echo "[BUILD] Windows amd64 -> $OUT_WINDOWS (MinGW-w64)"
        if ! command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
            echo "[ERROR] x86_64-w64-mingw32-gcc not found. Install mingw-w64 (e.g. apt install mingw-w64)"
            BUILD_OK=0
            return 0
        fi
        CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
            go build -ldflags "$LDFLAGS" -o "$OUT_WINDOWS" .
    else
        echo "[BUILD] Windows amd64 -> $OUT_WINDOWS (Zig)"
        CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="zig cc -target x86_64-windows-gnu" \
            go build -ldflags "$LDFLAGS" -o "$OUT_WINDOWS" .
    fi

    if [[ $? -ne 0 ]]; then
        if [[ "$WIN_TOOLCHAIN" == "mingw" ]]; then
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

case "$(tolower "${ARG1:-}")" in
    help|-h|--help) usage ;;
esac

parse_args || exit 1

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

resolve_version "$VERSION"

GO_VER="$(go version 2>/dev/null | awk '{print $3}')"
[[ -z "$GO_VER" ]] && GO_VER="unknown"

BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%S 2>/dev/null || echo unknown)"

LDFLAGS_COMMON="-X 'git.itopcms.com/jackliu/doc/conf.VERSION=${VERSION}' -X 'git.itopcms.com/jackliu/doc/conf.BUILD_TIME=${BUILD_TIME}' -X 'git.itopcms.com/jackliu/doc/conf.GO_VERSION=${GO_VER}'"

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
echo " Win toolchain  : $WIN_TOOLCHAIN"
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
    all)
        do_linux
        do_windows
        ;;
    linux)
        do_linux
        ;;
    windows)
        do_windows
        ;;
    *)
        echo "[ERROR] unknown target: $TARGET"
        exit 1
        ;;
esac

echo
if [[ "$BUILD_OK" == "1" ]]; then
    echo "[SUCCESS] build completed"
    exit 0
else
    echo "[FAILED] one or more targets failed to build"
    exit 1
fi
