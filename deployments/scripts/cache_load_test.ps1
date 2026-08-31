<#
.SYNOPSIS
  T12 缓存压测入口：击穿 / 负缓存 / Soft-TTL 单测 + 并行 bench。

.NOTES
  不要求本机 Redis。手工 HTTP / MCP 压测见 docs/round-5/round-5-t12-ops.md。
#>
$ErrorActionPreference = "Stop"

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

$env:CGO_ENABLED = "1"

Write-Host "==> Aside 击穿 / 负缓存 / Soft-TTL / Token 接入"
go test ./internal/cache/ ./internal/repository/ -count=1 `
  -run "TestAsideStampede|TestAsideNegative|TestAsideSoft|TestMemberRepo_ResolveAPIToken|TestDocumentRepo_FindUsesAside|TestBlogRepo_FindUsesAside"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "==> Aside 并行 GetOrLoad bench"
go test ./internal/cache/ -bench BenchmarkAsideGetOrLoadParallel -benchtime=2s -count=1
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "ok"
