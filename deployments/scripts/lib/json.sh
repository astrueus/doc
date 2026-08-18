#!/usr/bin/env bash
# ============================================================
# deployments/scripts/lib/json.sh —— 小型 JSON 解析器（bash + grep）
#
# 设计参考 JSON.sh (https://github.com/dominictarr/JSON.sh)：
#   1. 用 `grep -E -o` 做分词，尊重 JSON 字符串与转义；
#   2. 递归下降解析器把每个"叶子节点"展开为一行
#      "<path><TAB><value>"，其中 <path> 使用 JSON 数组表示，
#      例如 ["assets",0,"name"]。
#
# 目的：给发版脚本、Spug 脚本等提供**零第三方依赖**的
# JSON 解析能力（无需 jq/python3）。**性能一般**，仅适合几十 KB
# 以内的响应（Gitea Release 场景足够）。
#
# 用法：
#   source deployments/scripts/lib/json.sh
#
#   FLAT=$(json_flatten <<<"$json")           # 展平：path<TAB>value
#
#   json_get "$FLAT" '["id"]'                 # 取叶子（自动去掉字符串两端引号）
#   json_get_raw "$FLAT" '["id"]'             # 取叶子原始 JSON（字符串仍带引号）
#   json_paths "$FLAT" '["assets"]' | ...     # 列出某路径下的直接子路径
#   json_array_len "$FLAT" '["assets"]'       # 数组长度
#   json_find_index "$FLAT" '["assets"]' name '"doc.tar.gz"'
#         # 在数组元素里按 <key>==<raw-value> 匹配，输出下标
#
# 已覆盖：字符串（含 \" \\ \/ \n \r \t \b \f \uXXXX）、数字（含小数/科学计数）、
# 布尔、null、任意嵌套对象/数组、空对象/空数组。
# ============================================================

if [[ -n "${_LIB_JSON_LOADED:-}" ]]; then
  return 0 2>/dev/null || true
fi
_LIB_JSON_LOADED=1

# ---------- 分词 ----------
# 把 stdin 里的 JSON 分成 token，每行一个：
#   字符串（含首尾双引号）、数字、true/false/null、结构符 { } [ ] : ,
# 空白丢弃。字符串中的转义原样保留在 token 内。
json_tokenize() {
  local STRING='"(\\.|[^"\\])*"'
  local NUMBER='-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?'
  local KEYWORD='(true|false|null)'
  local WS='[[:space:]]+'
  # BSD/GNU grep 都支持 -E -o
  grep -E -o "$STRING|$KEYWORD|$NUMBER|[][{}:,]|$WS" \
    | grep -v -E "^[[:space:]]+$"
}

# ---------- 递归下降解析 ----------
# 内部实现，勿在外部调用；均通过全局 __J_CUR 做 lookahead。
_json_next() {
  IFS= read -r __J_CUR
}

_json_join_key() {
  # $1: 当前 path (形如 [] 或 ["a"] 或 ["a",0,"b"])
  # $2: 追加的 key/index 片段（key 已包含引号，index 为纯数字）
  local path="$1" seg="$2"
  if [[ "$path" == "[]" ]]; then
    printf '[%s]' "$seg"
  else
    printf '%s,%s]' "${path%]}" "$seg"
  fi
}

_json_parse_value() {
  local path="$1"
  case "$__J_CUR" in
    '{')  _json_parse_object "$path" ;;
    '[')  _json_parse_array  "$path" ;;
    '')   return 1 ;;
    *)
      printf '%s\t%s\n' "$path" "$__J_CUR"
      _json_next || __J_CUR=""
      ;;
  esac
}

_json_parse_object() {
  local path="$1" key sub
  _json_next || return 1
  if [[ "$__J_CUR" == '}' ]]; then
    printf '%s\t{}\n' "$path"
    _json_next || __J_CUR=""
    return 0
  fi
  while :; do
    # 键必须是 JSON 字符串（分词器会保留双引号）
    case "$__J_CUR" in
      '"'*'"') key="$__J_CUR" ;;
      *) return 1 ;;
    esac
    _json_next || return 1
    [[ "$__J_CUR" == ':' ]] || return 1
    _json_next || return 1
    sub="$(_json_join_key "$path" "$key")"
    _json_parse_value "$sub" || return 1
    case "$__J_CUR" in
      ',') _json_next || return 1 ;;
      '}') _json_next || __J_CUR=""; return 0 ;;
      *)   return 1 ;;
    esac
  done
}

_json_parse_array() {
  local path="$1" idx=0 sub
  _json_next || return 1
  if [[ "$__J_CUR" == ']' ]]; then
    printf '%s\t[]\n' "$path"
    _json_next || __J_CUR=""
    return 0
  fi
  while :; do
    sub="$(_json_join_key "$path" "$idx")"
    _json_parse_value "$sub" || return 1
    case "$__J_CUR" in
      ',') _json_next || return 1; idx=$((idx + 1)) ;;
      ']') _json_next || __J_CUR=""; return 0 ;;
      *)   return 1 ;;
    esac
  done
}

# ---------- 对外 API ----------

# json_flatten：读 stdin 的 JSON，输出 "path<TAB>value" 一行一叶子。
# 用法：FLAT=$(json_flatten <<<"$json") 或 cat file | json_flatten
json_flatten() {
  local __J_CUR
  {
    _json_next || return 0
    _json_parse_value "[]" || return 1
  } < <(json_tokenize)
}

# _json_unescape_string：把一段带外层引号的 JSON 字符串还原为普通字符串。
_json_unescape_string() {
  local s="$1"
  # 去外层引号
  s="${s#\"}"
  s="${s%\"}"
  # 常见转义（\uXXXX 交给 printf %b 处理不方便，此处用 python/perl 都算依赖，
  # 我们只处理最常用的几种；如需完整 unicode 恢复请配合 jq）。
  s="${s//\\\"/\"}"
  s="${s//\\\\/\\}"
  s="${s//\\\//\/}"
  s="${s//\\n/$'\n'}"
  s="${s//\\r/$'\r'}"
  s="${s//\\t/$'\t'}"
  s="${s//\\b/$'\b'}"
  s="${s//\\f/$'\f'}"
  printf '%s' "$s"
}

# json_get_raw <flat> <path>
#   打印指定 path 的**原始 JSON 值**（字符串仍带引号，数字/bool/null 原样）。
#   若无匹配则不输出、返回码 1。
json_get_raw() {
  local flat="$1" path="$2" val
  val="$(printf '%s\n' "$flat" | awk -F'\t' -v p="$path" '$1==p{print $2; exit}')"
  [[ -n "$val" ]] || return 1
  printf '%s\n' "$val"
}

# json_get <flat> <path>
#   同 json_get_raw，但若值为字符串则自动去掉双引号并做基础反转义。
json_get() {
  local flat="$1" path="$2" val
  val="$(json_get_raw "$flat" "$path")" || return 1
  case "$val" in
    '"'*'"') _json_unescape_string "$val"; echo ;;
    *)       printf '%s\n' "$val" ;;
  esac
}

# json_paths <flat> <parent_path>
#   列出 parent_path 之下的所有**直接子叶子/子容器**路径，去重。
#   例如 flat 里有 ["assets",0,"id"] / ["assets",0,"name"] / ["assets",1,"id"]，
#   json_paths $flat '["assets"]' → 输出 ["assets",0] ["assets",1]
json_paths() {
  local flat="$1" parent="$2"
  # 转成 awk 可用的 prefix：把 "[a]" 去尾后追加 ","；根路径 [] → "["
  local prefix
  if [[ "$parent" == "[]" ]]; then
    prefix="["
  else
    prefix="${parent%]},"
  fi
  printf '%s\n' "$flat" \
    | awk -F'\t' -v pre="$prefix" -v plen="${#prefix}" '
      substr($1,1,plen)==pre {
        rest = substr($1, plen+1)      # 例如 0,"name"] 或 "assets"]
        # 取到下一个 , 或 ] 之前
        n = length(rest)
        depth = 0; end = 0
        for (i=1; i<=n; i++) {
          c = substr(rest, i, 1)
          if (c == "[") depth++
          else if (c == "]") { if (depth==0) { end=i; break } else depth-- }
          else if (c == "," && depth==0) { end=i; break }
        }
        if (end==0) next
        seg = substr(rest, 1, end-1)
        print pre seg "]"
      }' \
    | awk '!seen[$0]++'
}

# json_array_len <flat> <array_path>
#   输出数组元素个数（0..N-1 的最大下标 + 1）。若不是数组或为空数组返回 0。
json_array_len() {
  local flat="$1" path="$2" prefix count=0
  if [[ "$path" == "[]" ]]; then
    prefix="["
  else
    prefix="${path%]},"
  fi
  count="$(printf '%s\n' "$flat" \
    | awk -F'\t' -v pre="$prefix" -v plen="${#prefix}" '
      substr($1,1,plen)==pre {
        rest = substr($1, plen+1)
        # 只统计以数字开头的直接子路径
        if (match(rest, /^[0-9]+/)) {
          idx = substr(rest, RSTART, RLENGTH)+0
          if (idx+1 > max) max = idx+1
        }
      }
      END { print (max?max:0) }')"
  printf '%s\n' "$count"
}

# json_find_index <flat> <array_path> <key> <raw_value>
#   在指定数组内查找元素：数组元素为对象，且对象里 key 的**原始 JSON 值**
#   与 raw_value 完全一致。输出所有匹配下标（每行一个）。
#
# 例：json_find_index "$FLAT" '["assets"]' name '"doc.tar.gz"'
json_find_index() {
  local flat="$1" path="$2" key="$3" raw="$4" prefix
  if [[ "$path" == "[]" ]]; then
    prefix="["
  else
    prefix="${path%]},"
  fi
  # 目标形如：<prefix><idx>,"<key>"]<TAB><raw>
  printf '%s\n' "$flat" \
    | awk -F'\t' -v pre="$prefix" -v plen="${#prefix}" -v k="\"$key\"" -v v="$raw" '
      substr($1,1,plen)==pre {
        rest = substr($1, plen+1)
        # rest 需形如 <digits>,<k>]
        if (match(rest, /^[0-9]+/)) {
          idx = substr(rest, RSTART, RLENGTH)
          tail = substr(rest, RSTART+RLENGTH)
          if (tail == "," k "]" && $2 == v) print idx
        }
      }'
}
