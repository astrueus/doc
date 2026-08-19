<#
.SYNOPSIS
  Doc 一键发版：编译 → 打 zip →（可选）打 tag / 创建 Gitea Release / 上传附件。

.PARAMETER Version
  语义化版本号，不含 v 前缀。示例：1.0.0

.PARAMETER Target
  构建目标：all | linux | windows（默认 windows，本机联调更省事）

.PARAMETER EnvFile
  环境变量文件路径（KEY=VALUE）。默认：本脚本目录 deployments/scripts/.env.release（若存在）。
  需要：GITEA_URL、GITEA_TOKEN、GITEA_OWNER、GITEA_REPO（DryRun 可不设）。

.PARAMETER SkipTagPush
  跳过 git tag / push。

.PARAMETER Draft
  创建为草稿 Release。

.PARAMETER DryRun
  只编译 + 打 zip，不打 tag、不调 Gitea API（无需 Token）。

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File deployments\scripts\release.ps1 -Version 0.0.1-test -Target windows -Draft

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File deployments\scripts\release.ps1 -Version 0.0.1-test -EnvFile .\deployments\scripts\.env.release -DryRun
#>
param(
  [Parameter(Mandatory = $true)][string]$Version,
  [ValidateSet("all", "linux", "windows")][string]$Target = "windows",
  [string]$EnvFile = "",
  [switch]$SkipTagPush,
  [switch]$Draft,
  [switch]$DryRun
)

$ErrorActionPreference = "Stop"

try {
  if ($Host.Name -eq "ConsoleHost") {
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    $OutputEncoding = [System.Text.Encoding]::UTF8
  }
}
catch { }

function Import-DotEnv {
  param([Parameter(Mandatory = $true)][string]$Path)
  if (-not (Test-Path -LiteralPath $Path)) {
    throw "env file not found: $Path"
  }
  Write-Host "Load env: $Path"
  Get-Content -LiteralPath $Path -Encoding UTF8 | ForEach-Object {
    $line = $_.Trim()
    if ($line -eq "" -or $line.StartsWith("#")) { return }
    if ($line.StartsWith("export ")) { $line = $line.Substring(7).Trim() }
    $eq = $line.IndexOf("=")
    if ($eq -lt 1) { return }
    $name = $line.Substring(0, $eq).Trim()
    $val = $line.Substring($eq + 1).Trim()
    if (($val.StartsWith('"') -and $val.EndsWith('"')) -or ($val.StartsWith("'") -and $val.EndsWith("'"))) {
      $val = $val.Substring(1, $val.Length - 2)
    }
    Set-Item -Path "Env:$name" -Value $val
  }
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
function Find-RepoRoot {
  param([string]$Start)
  $dir = (Resolve-Path -LiteralPath $Start).Path
  while ($dir) {
    $mod = Join-Path $dir "go.mod"
    $cmd = Join-Path $dir "cmd\doc"
    if ((Test-Path -LiteralPath $mod) -and (Test-Path -LiteralPath $cmd)) {
      return $dir
    }
    $parent = Split-Path -Parent $dir
    if ($parent -eq $dir) { break }
    $dir = $parent
  }
  throw "cannot locate repo root (need go.mod and cmd\doc); start: $Start"
}
$Root = Find-RepoRoot -Start $ScriptDir
Set-Location $Root

if (-not $EnvFile) {
  $defaultEnv = Join-Path $ScriptDir ".env.release"
  if (Test-Path -LiteralPath $defaultEnv) { $EnvFile = $defaultEnv }
}
if ($EnvFile) {
  if (-not [System.IO.Path]::IsPathRooted($EnvFile)) {
    $EnvFile = Join-Path $Root $EnvFile
  }
  Import-DotEnv -Path $EnvFile
}

$Tag = "v$Version"
$Owner = $env:GITEA_OWNER
$Repo = $env:GITEA_REPO
$Base = $env:GITEA_URL
$Token = $env:GITEA_TOKEN

if (-not $DryRun) {
  if (-not $Owner -or -not $Repo -or -not $Base -or -not $Token) {
    throw @"
Missing Gitea env. Provide EnvFile (deployments/scripts/.env.release) or export:
  GITEA_URL / GITEA_TOKEN / GITEA_OWNER / GITEA_REPO
Or use -DryRun to build+zip only.
See: deployments/scripts/.env.release.example
"@
  }
}

# ---------- 1) Build ----------
Write-Host "[1/5] Build $Target release $Version ..."
$buildBat = Join-Path $ScriptDir "build.bat"
cmd /c "`"$buildBat`" --target=$Target --mode=release --version=$Version"
if ($LASTEXITCODE -ne 0) { throw "build failed (exit $LASTEXITCODE)" }

# ---------- 2) Package ----------
Write-Host "[2/5] Package zip ..."

Get-ChildItem -LiteralPath $Root -Directory -Filter ".release_stage_*" -ErrorAction SilentlyContinue |
  ForEach-Object { Remove-Item -LiteralPath $_.FullName -Recurse -Force -ErrorAction SilentlyContinue }

$OutDir = Join-Path $Root "release"
if (-not (Test-Path -LiteralPath $OutDir)) {
  New-Item -ItemType Directory -Path $OutDir | Out-Null
}
Write-Host "  output dir: $OutDir"

$assets = @()

function Copy-TreeRobust {
  param([string]$Src, [string]$Dst)
  if (-not (Test-Path -LiteralPath $Src)) {
    throw "source not found: $Src"
  }
  if (Test-Path -LiteralPath $Src -PathType Leaf) {
    $parent = Split-Path -Parent $Dst
    if (-not (Test-Path -LiteralPath $parent)) {
      New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    Copy-Item -LiteralPath $Src -Destination $Dst -Force
    return
  }
  if (-not (Test-Path -LiteralPath $Dst)) {
    New-Item -ItemType Directory -Path $Dst -Force | Out-Null
  }
  $null = & robocopy $Src $Dst /E /NFL /NDL /NJH /NJS /NC /NS /NP /R:2 /W:1
  if ($LASTEXITCODE -ge 8) {
    throw "robocopy failed ($LASTEXITCODE): $Src -> $Dst"
  }
}

function Publish-SharedIntoStage {
  param([string]$Stage)

  # web/ entire tree (favicon lives under web/static if present)
  $webSrc = Join-Path $Root "web"
  if (-not (Test-Path -LiteralPath $webSrc)) { throw "missing required path: web/" }
  Copy-TreeRobust -Src $webSrc -Dst (Join-Path $Stage "web")

  # conf/: only lang/ + app.conf.example (never local app.conf)
  $confStage = Join-Path $Stage "conf"
  New-Item -ItemType Directory -Path $confStage -Force | Out-Null
  $langSrc = Join-Path $Root "conf\lang"
  if (-not (Test-Path -LiteralPath $langSrc)) { throw "missing required path: conf/lang/" }
  Copy-TreeRobust -Src $langSrc -Dst (Join-Path $confStage "lang")
  $exampleSrc = Join-Path $Root "conf\app.conf.example"
  if (-not (Test-Path -LiteralPath $exampleSrc)) { throw "missing required path: conf/app.conf.example" }
  Copy-Item -LiteralPath $exampleSrc -Destination (Join-Path $confStage "app.conf.example") -Force

  # uploads/: empty placeholder only
  New-Item -ItemType Directory -Path (Join-Path $Stage "uploads") -Force | Out-Null

  # deployments/: only spug/ + systemd/
  $depStage = Join-Path $Stage "deployments"
  New-Item -ItemType Directory -Path $depStage -Force | Out-Null
  foreach ($sub in @("spug", "systemd")) {
    $src = Join-Path $Root "deployments\$sub"
    if (-not (Test-Path -LiteralPath $src)) {
      Write-Warning "missing deployments/$sub (skip)"
      continue
    }
    Copy-TreeRobust -Src $src -Dst (Join-Path $depStage $sub)
  }

  $license = Join-Path $Root "LICENSE.md"
  if (Test-Path -LiteralPath $license) {
    Copy-Item -LiteralPath $license -Destination (Join-Path $Stage "LICENSE.md") -Force
  }
}

function New-ArchiveFromDirectory {
  param(
    [string]$SourceDir,
    [string]$ArchivePath,
    [ValidateSet("zip", "tar.gz")][string]$Format
  )
  if (Test-Path -LiteralPath $ArchivePath) { Remove-Item -LiteralPath $ArchivePath -Force }

  $tar = Get-Command tar.exe -ErrorAction SilentlyContinue

  if ($Format -eq "zip") {
    if ($tar) {
      Push-Location $SourceDir
      try {
        & tar.exe -a -c -f $ArchivePath *
        if ($LASTEXITCODE -ne 0) { throw "tar zip failed: $LASTEXITCODE" }
      }
      finally { Pop-Location }
      return
    }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::CreateFromDirectory($SourceDir, $ArchivePath)
    return
  }

  # tar.gz — require tar
  if (-not $tar) {
    throw "tar.exe not found; required to build linux .tar.gz packages"
  }
  Push-Location $SourceDir
  try {
    & tar.exe -czf $ArchivePath *
    if ($LASTEXITCODE -ne 0) { throw "tar.gz failed: $LASTEXITCODE" }
  }
  finally { Pop-Location }
}

function New-ReleasePackage {
  param(
    [string]$PackageName,
    [ValidateSet("zip", "tar.gz")][string]$Format,
    [string]$BinaryPath,
    [string]$BinaryNameInPackage
  )
  $archivePath = Join-Path $OutDir $PackageName
  $stage = Join-Path $Root (".release_stage_" + [guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Path $stage | Out-Null
  try {
    if (-not (Test-Path -LiteralPath $BinaryPath)) { throw "binary not found: $BinaryPath" }
    # 包内可执行文件统一为 doc / doc.exe，不带 _windows_amd64 / _linux_amd64
    Copy-Item -LiteralPath $BinaryPath -Destination (Join-Path $stage $BinaryNameInPackage) -Force
    Publish-SharedIntoStage -Stage $stage
    New-ArchiveFromDirectory -SourceDir $stage -ArchivePath $archivePath -Format $Format
  }
  finally {
    if (Test-Path -LiteralPath $stage) {
      Remove-Item -LiteralPath $stage -Recurse -Force -ErrorAction SilentlyContinue
    }
  }
  return $archivePath
}

# 压缩包名带平台；包内二进制统一为 doc.exe / doc
if (($Target -eq "all" -or $Target -eq "windows") -and (Test-Path "dist\doc_windows_amd64.exe")) {
  $assets += (New-ReleasePackage `
    -PackageName "doc_${Version}_windows_amd64.zip" `
    -Format zip `
    -BinaryPath (Join-Path $Root "dist\doc_windows_amd64.exe") `
    -BinaryNameInPackage "doc.exe")
}

if (($Target -eq "all" -or $Target -eq "linux") -and (Test-Path "dist\doc_linux_amd64")) {
  $assets += (New-ReleasePackage `
    -PackageName "doc_${Version}_linux_amd64.tar.gz" `
    -Format tar.gz `
    -BinaryPath (Join-Path $Root "dist\doc_linux_amd64") `
    -BinaryNameInPackage "doc")
}

if ($assets.Count -eq 0) { throw "no asset to upload, check dist/" }

$assetNames = ($assets | ForEach-Object { Split-Path $_ -Leaf }) -join ", "
Write-Host "  packaged: $assetNames"

if ($DryRun) {
  Write-Host ""
  Write-Host "[DryRun] skipped tag / Gitea Release. packages are under release/"
  Write-Host "Done."
  exit 0
}

# ---------- 3) Tag ----------
if (-not $SkipTagPush) {
  Write-Host "[3/5] Tag $Tag ..."
  $existing = git tag --list $Tag
  if ($existing) {
    $point = (git log -1 --format=%h $Tag).Trim()
    throw "tag $Tag already exists (points to $point). Refusing to skip+push. Use a new version or delete the local tag after checking."
  }
  git tag -a $Tag -m "Release $Tag"
  git push origin "refs/tags/$Tag"
}
else {
  Write-Host "[3/5] Skip tag/push"
}

# ---------- 4) Release ----------
Write-Host "[4/5] Create Release $Tag ..."
$headers = @{ Authorization = "token $Token" }

$body = @{
  tag_name   = $Tag
  name       = "doc $Tag"
  body       = "Auto release $Tag"
  draft      = [bool]$Draft
  prerelease = $false
} | ConvertTo-Json

$release = $null
try {
  $release = Invoke-RestMethod -Method Post `
    -Uri "$Base/api/v1/repos/$Owner/$Repo/releases" `
    -Headers $headers -ContentType "application/json" -Body $body
  Write-Host "  created release id=$($release.id)"
}
catch {
  Write-Host "  create failed, try fetch release by tag..."
  $release = Invoke-RestMethod -Method Get `
    -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/tags/$Tag" `
    -Headers $headers
  Write-Host "  reuse release id=$($release.id)"
}

# ---------- 5) Upload ----------
Write-Host "[5/5] Upload assets ..."

$existingAssets = Invoke-RestMethod -Method Get `
  -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/$($release.id)/assets" `
  -Headers $headers

foreach ($file in $assets) {
  $name = Split-Path $file -Leaf
  $old = @($existingAssets | Where-Object { $_.name -eq $name })
  if ($old.Count -gt 0) {
    foreach ($o in $old) {
      Write-Host "  delete old asset: $name (id=$($o.id))"
      Invoke-RestMethod -Method Delete `
        -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/$($release.id)/assets/$($o.id)" `
        -Headers $headers | Out-Null
    }
  }
  Write-Host "  upload: $name"
  Invoke-RestMethod -Method Post `
    -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/$($release.id)/assets?name=$name" `
    -Headers $headers -InFile $file -ContentType "application/octet-stream" | Out-Null
}

Write-Host ""
Write-Host "Release published: $Base/$Owner/$Repo/releases/tag/$Tag"
