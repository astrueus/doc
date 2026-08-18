#!/usr/bin/env bash
# json.sh 单测。运行：bash deployments/scripts/lib/json_test.sh
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck disable=SC1091
source "$HERE/json.sh"

FAIL=0
PASS=0

_ok() { PASS=$((PASS+1)); printf '  \033[32mok\033[0m  %s\n' "$1"; }
_ng() { FAIL=$((FAIL+1)); printf '  \033[31mNG\033[0m  %s\n    expected: %s\n    actual:   %s\n' "$1" "$2" "$3"; }

expect_eq() {
  local desc="$1" want="$2" got="$3"
  if [[ "$want" == "$got" ]]; then _ok "$desc"; else _ng "$desc" "$want" "$got"; fi
}

# ---------- 基本类型 ----------
JSON='{"id":42,"tag_name":"v1.0.0","draft":false,"body":null}'
FLAT=$(json_flatten <<<"$JSON")

expect_eq "顶层 number" "42"      "$(json_get "$FLAT" '["id"]')"
expect_eq "顶层 string" "v1.0.0"  "$(json_get "$FLAT" '["tag_name"]')"
expect_eq "顶层 bool"   "false"   "$(json_get "$FLAT" '["draft"]')"
expect_eq "顶层 null"   "null"    "$(json_get "$FLAT" '["body"]')"

# raw 版本对字符串保留引号
expect_eq "raw string"  '"v1.0.0"' "$(json_get_raw "$FLAT" '["tag_name"]')"

# ---------- 嵌套对象 ----------
JSON='{"author":{"id":7,"login":"jack"}}'
FLAT=$(json_flatten <<<"$JSON")
expect_eq "嵌套 id"    "7"    "$(json_get "$FLAT" '["author","id"]')"
expect_eq "嵌套 login" "jack" "$(json_get "$FLAT" '["author","login"]')"

# ---------- 嵌套数组 ----------
JSON='[{"id":1,"name":"a"},{"id":2,"name":"b"},{"id":3,"name":"a"}]'
FLAT=$(json_flatten <<<"$JSON")
expect_eq "数组[0].id"   "1" "$(json_get "$FLAT" '[0,"id"]')"
expect_eq "数组[2].name" "a" "$(json_get "$FLAT" '[2,"name"]')"
expect_eq "数组长度"     "3" "$(json_array_len "$FLAT" '[]')"

FOUND=$(json_find_index "$FLAT" '[]' name '"a"' | tr '\n' ' ')
expect_eq "按 name=a 找下标" "0 2 " "$FOUND"

# ---------- 字符串转义 ----------
JSON='{"s":"a\"b\\c\/d\ne"}'
FLAT=$(json_flatten <<<"$JSON")
# 反转义后：a"b\c/d<LF>e
WANT=$(printf 'a"b\\c/d\ne')
expect_eq "字符串转义 (\" \\ / \\n)" "$WANT" "$(json_get "$FLAT" '["s"]')"

# ---------- 空对象 / 空数组 ----------
JSON='{"a":{},"b":[]}'
FLAT=$(json_flatten <<<"$JSON")
expect_eq "空对象"   "{}" "$(json_get_raw "$FLAT" '["a"]')"
expect_eq "空数组"   "[]" "$(json_get_raw "$FLAT" '["b"]')"

# ---------- 数字/科学计数 ----------
JSON='{"n1":0,"n2":-1,"n3":1.5,"n4":1e3,"n5":-2.5E-2}'
FLAT=$(json_flatten <<<"$JSON")
expect_eq "n1=0"     "0"      "$(json_get "$FLAT" '["n1"]')"
expect_eq "n2=-1"    "-1"     "$(json_get "$FLAT" '["n2"]')"
expect_eq "n3=1.5"   "1.5"    "$(json_get "$FLAT" '["n3"]')"
expect_eq "n4=1e3"   "1e3"    "$(json_get "$FLAT" '["n4"]')"
expect_eq "n5"       "-2.5E-2" "$(json_get "$FLAT" '["n5"]')"

# ---------- 实战：Gitea Release 响应形状 ----------
JSON='{"id":42,"tag_name":"v1.0.0","author":{"id":7,"login":"jack"},"assets":[{"id":100,"name":"doc_1.0.0_linux_amd64.tar.gz"},{"id":101,"name":"other.zip"},{"id":102,"name":"doc_1.0.0_linux_amd64.tar.gz"}]}'
FLAT=$(json_flatten <<<"$JSON")
expect_eq "release.id"           "42" "$(json_get "$FLAT" '["id"]')"
expect_eq "author.id"            "7"  "$(json_get "$FLAT" '["author","id"]')"
expect_eq "assets.length"        "3"  "$(json_array_len "$FLAT" '["assets"]')"

IDX=$(json_find_index "$FLAT" '["assets"]' name '"doc_1.0.0_linux_amd64.tar.gz"' | tr '\n' ' ')
expect_eq "按 name 找 asset 下标" "0 2 " "$IDX"

# 逐个下标查 id
IDS=""
for i in $(json_find_index "$FLAT" '["assets"]' name '"doc_1.0.0_linux_amd64.tar.gz"'); do
  IDS="$IDS$(json_get "$FLAT" "[\"assets\",$i,\"id\"]") "
done
expect_eq "对应 asset id"        "100 102 " "$IDS"

# ---------- 汇总 ----------
echo
printf '%d passed, %d failed\n' "$PASS" "$FAIL"
[[ "$FAIL" -eq 0 ]] || exit 1
