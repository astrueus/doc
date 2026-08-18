#!/usr/bin/env bash
# ============================================================
# 白名单包测试 + 覆盖率门槛（Round 5 T7）
#
# 用法：
#   ./deployments/scripts/test.sh
#   make test
#   TEST_PKGS='./pkg/...' COVER_MIN=0 ./deployments/scripts/test.sh
#
# 环境变量：
#   TEST_PKGS            空格分隔的包列表（默认见下）
#   COVER_PROFILE        默认 coverage.out
#   COVER_REPORT         默认 coverage.txt
#   COVER_MIN            百分比；未设则读 docs/round-5/coverage-baseline.txt；都无则为 0
#   COVER_BASELINE_FILE  基线文件路径
#   RACE                 1/0；默认非 Windows 为 1
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

DEFAULT_PKGS="./pkg/... ./internal/errs/... ./internal/auth/... ./internal/logging/... ./internal/i18n/... ./internal/repository/..."
# shellcheck disable=SC2206
PKGS=( ${TEST_PKGS:-$DEFAULT_PKGS} )

COVER_PROFILE="${COVER_PROFILE:-coverage.out}"
COVER_REPORT="${COVER_REPORT:-coverage.txt}"
BASELINE_FILE="${COVER_BASELINE_FILE:-docs/round-5/coverage-baseline.txt}"

if [[ -z "${RACE:-}" ]]; then
  case "$(uname -s 2>/dev/null || echo unknown)" in
    MINGW*|MSYS*|CYGWIN*) RACE=0 ;;
    *) RACE=1 ;;
  esac
fi

RACE_ARGS=()
if [[ "$RACE" == "1" ]]; then
  RACE_ARGS=(-race)
fi

if [[ -z "${GOTEST_P:-}" ]]; then
  case "$(uname -s 2>/dev/null || echo unknown)" in
    MINGW*|MSYS*|CYGWIN*) GOTEST_P=1 ;;
    *) GOTEST_P="" ;;
  esac
fi
P_ARGS=()
if [[ -n "${GOTEST_P}" ]]; then
  P_ARGS=(-p "$GOTEST_P")
fi

echo "[test] packages: ${PKGS[*]}"
echo "[test] race=$RACE coverprofile=$COVER_PROFILE"

go test "${RACE_ARGS[@]}" "${P_ARGS[@]}" -count=1 -coverprofile="$COVER_PROFILE" -covermode=atomic "${PKGS[@]}"

go tool cover -func="$COVER_PROFILE" | tee "$COVER_REPORT"

TOTAL="$(awk '/^total:/ { gsub(/%/, "", $NF); print $NF }' "$COVER_REPORT")"
[[ -n "$TOTAL" ]] || die "cannot parse cover total from $COVER_REPORT"

if [[ -z "${COVER_MIN:-}" && -f "$BASELINE_FILE" ]]; then
  COVER_MIN="$(tr -d '[:space:]%' < "$BASELINE_FILE")"
fi
COVER_MIN="${COVER_MIN:-0}"

echo "[test] cover total=${TOTAL}% min=${COVER_MIN}%"

awk -v t="$TOTAL" -v m="$COVER_MIN" 'BEGIN { if ((t + 0) < (m + 0)) exit 1 }' \
  || die "coverage ${TOTAL}% is below gate ${COVER_MIN}%"

echo "[test] ok"
