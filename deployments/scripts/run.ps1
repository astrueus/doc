<#
.SYNOPSIS
  开发启动：go run（不落盘二进制，不加热重载）。

.NOTES
  go run 的二进制在临时目录；必须传 --dir 指向仓库根。
  sqlite 需要 CGO。CC 已设置则保留；否则优先 gcc，其次 Zig。
  启动前清除 GOOS/GOARCH，避免残留交叉编译变量导致无法本机执行。
  不用 param()，以便 --help / --http 等原样后传。
  用法：just run / just run install
#>
$ErrorActionPreference = "Stop"
$DocArgs = @($args)

function Find-RepoRoot {
  param([string]$Start)
  $dir = (Resolve-Path -LiteralPath $Start).Path
  while ($dir) {
    if ((Test-Path -LiteralPath (Join-Path $dir "go.mod")) -and (Test-Path -LiteralPath (Join-Path $dir "cmd\doc"))) {
      return $dir
    }
    $parent = Split-Path -Parent $dir
    if ($parent -eq $dir) { break }
    $dir = $parent
  }
  throw "cannot locate repo root (need go.mod and cmd\doc); start: $Start"
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = Find-RepoRoot -Start $ScriptDir
Set-Location $Root

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
$env:CGO_ENABLED = "1"

if (-not $env:CC) {
  if (Get-Command gcc -ErrorAction SilentlyContinue) {
    # 使用 PATH 中的 gcc，不设 CC
  } elseif (Get-Command zig -ErrorAction SilentlyContinue) {
    $env:CC = "zig cc -target x86_64-windows-gnu"
  } else {
    throw "未找到 C 编译器。请安装 MinGW-w64（gcc）或 Zig，并加入 PATH"
  }
}

$ccDisplay = if ($env:CC) { $env:CC } else { "(default)" }
# MCP stdio 占用 stdout；Write-Host 走控制台，不进成功流
Write-Host "[run] dir=$Root cgo=1 cc=$ccDisplay"

# just 的 `just run -- --help` 可能把 `--` 一并后传；丢掉以免 cobra 当成默认 web 启动
if ($DocArgs.Count -ge 1 -and $DocArgs[0] -eq "--") {
  if ($DocArgs.Count -eq 1) {
    $DocArgs = @()
  } else {
    $DocArgs = $DocArgs[1..($DocArgs.Count - 1)]
  }
}

$goArgs = @("run", "./cmd/doc", "--dir", $Root) + $DocArgs
& go @goArgs
exit $LASTEXITCODE
