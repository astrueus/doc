#!/usr/bin/env bash
# ============================================================
# 开发启动：go run（不落盘二进制，不加热重载）
#
# 用法：
#   ./deployments/scripts/run.sh
#   make run
#   make run ARGS=install
#   just run
#   just run install
#
# go run 的二进制在临时目录；必须传 --dir 指向仓库根，
# 否则会去临时目录找 conf / web/static / web/views。
#
# 环境：
#   CGO_ENABLED  强制为 1（sqlite）
#   CC           已设置则保留；Windows 未设置时优先 gcc，否则 zig
#   GOOS/GOARCH  启动前清除，避免残留交叉编译变量导致无法本机执行
# ============================================================

set -euo pipefail

die() { echo "[ERROR] $*" >&2; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPT_DIR"
while [[ -n "$ROOT" && "$ROOT" != "/" && ! ( -f "$ROOT/go.mod" && -d "$ROOT/cmd/doc" ) ]]; do
  ROOT="$(cd "$ROOT/.." && pwd)"
done
[[ -f "$ROOT/go.mod" && -d "$ROOT/cmd/doc" ]] || die "cannot locate repo root; script dir: $SCRIPT_DIR"
cd "$ROOT"

# 本机 go run，忽略残留的交叉编译 GOOS/GOARCH
unset GOOS GOARCH
export CGO_ENABLED=1

os="$(uname -s 2>/dev/null || echo unknown)"
case "$os" in
  MINGW*|MSYS*|CYGWIN*)
    if [[ -z "${CC:-}" ]]; then
      if command -v gcc >/dev/null 2>&1; then
        :
      elif command -v zig >/dev/null 2>&1; then
        export CC="zig cc -target x86_64-windows-gnu"
      else
        die "未找到 C 编译器。请安装 MinGW-w64（gcc）或 Zig，并加入 PATH"
      fi
    fi
    ;;
  *)
    if [[ -z "${CC:-}" ]]; then
      if ! command -v gcc >/dev/null 2>&1 && ! command -v clang >/dev/null 2>&1; then
        die "未找到 gcc/clang。请安装系统编译器（如 build-essential）"
      fi
    fi
    ;;
esac

cc_display="${CC:-(default)}"
# MCP stdio 占用 stdout，提示只走 stderr
echo "[run] dir=$ROOT cgo=1 cc=$cc_display" >&2

# just 的 `just run -- --help` 可能把 `--` 一并后传；丢掉以免 cobra 当成默认 web 启动
if [[ "${1:-}" == "--" ]]; then
  shift
fi

exec go run ./cmd/doc --dir "$ROOT" "$@"
