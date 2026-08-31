<#
.SYNOPSIS
  白名单包测试 + 覆盖率门槛（Round 5 T7）。

.NOTES
  Windows 默认不开 -race（需 MinGW gcc）；CI / Git Bash 用 test.sh。
  覆盖率门槛：COVER_MIN，或 docs/round-5/coverage-baseline.txt。
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

$defaultPkgs = "./pkg/... ./internal/errs/... ./internal/auth/... ./internal/logging/... ./internal/i18n/... ./internal/repository/... ./internal/cache/..."
$pkgText = if ($env:TEST_PKGS) { $env:TEST_PKGS } else { $defaultPkgs }
$pkgs = $pkgText -split "\s+" | Where-Object { $_ }

$coverProfile = if ($env:COVER_PROFILE) { $env:COVER_PROFILE } else { "coverage.out" }
$coverReport = if ($env:COVER_REPORT) { $env:COVER_REPORT } else { "coverage.txt" }
$baselineFile = if ($env:COVER_BASELINE_FILE) { $env:COVER_BASELINE_FILE } else { "docs/round-5/coverage-baseline.txt" }

$race = $env:RACE
if (-not $race) { $race = "0" }

$parallel = $env:GOTEST_P
if (-not $parallel) { $parallel = "1" }

$goArgs = @("test")
if ($race -eq "1") { $goArgs += "-race" }
$goArgs += @("-count=1", "-p", $parallel, "-coverprofile=$coverProfile", "-covermode=atomic") + $pkgs

Write-Host "[test] packages: $($pkgs -join ' ')"
Write-Host "[test] race=$race coverprofile=$coverProfile"

& go @goArgs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$funcOut = & go tool cover "-func=$coverProfile"
$funcOut | Tee-Object -FilePath $coverReport | Out-Host

$totalLine = $funcOut | Where-Object { $_ -match "^total:" } | Select-Object -Last 1
if (-not $totalLine) { throw "cannot parse cover total from $coverReport" }
if ($totalLine -notmatch "([\d.]+)%\s*$") { throw "cannot parse cover percent: $totalLine" }
$total = [double]$Matches[1]

$coverMinText = $env:COVER_MIN
if (-not $coverMinText -and (Test-Path -LiteralPath $baselineFile)) {
  $coverMinText = (Get-Content -LiteralPath $baselineFile -Raw).Trim().TrimEnd("%")
}
if (-not $coverMinText) { $coverMinText = "0" }
$coverMin = [double]$coverMinText

Write-Host "[test] cover total=${total}% min=${coverMin}%"
if ($total -lt $coverMin) {
  throw "coverage ${total}% is below gate ${coverMin}%"
}

Write-Host "[test] ok"
