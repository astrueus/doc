#!/usr/bin/env bash
# ============================================================
# T12 缓存压测入口：跑击穿 / 负缓存 / Soft-TTL 单测 + 并行 bench。
#
# 用法：
#   ./deployments/scripts/cache_load_test.sh
#   powershell -File deployments/scripts/cache_load_test.ps1
#
# 不要求本机 Redis。对真实 HTTP / MCP 的手工压测见
# docs/round-5/round-5-t12-ops.md。
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
export CGO_ENABLED=1

echo "==> Aside 击穿 / 负缓存 / Soft-TTL / Token 接入"
go test ./internal/cache/ ./internal/repository/ -count=1 \
  -run 'TestAsideStampede|TestAsideNegative|TestAsideSoft|TestMemberRepo_ResolveAPIToken|TestDocumentRepo_FindUsesAside|TestBlogRepo_FindUsesAside'

echo "==> Aside 并行 GetOrLoad bench"
go test ./internal/cache/ -bench BenchmarkAsideGetOrLoadParallel -benchtime=2s -count=1

echo "ok"
