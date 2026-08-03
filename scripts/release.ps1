<#
.SYNOPSIS
  Doc 一键发版：编译 → 打 zip →（可选）打 tag / 创建 Gitea Release / 上传附件。

.PARAMETER Version
  语义化版本号，不含 v 前缀。示例：1.0.0

.PARAMETER Target
  构建目标：all | linux | windows（默认 windows，本机联调更省事）

.PARAMETER EnvFile
  环境变量文件路径（KEY=VALUE）。默认：仓库根下 scripts/.env.release（若存在）。
  需要：GITEA_URL、GITEA_TOKEN、GITEA_OWNER、GITEA_REPO（DryRun 可不设）。

.PARAMETER SkipTagPush
  跳过 git tag / push。

.PARAMETER Draft
  创建为草稿 Release。

.PARAMETER DryRun
  只编译 + 打 zip，不打 tag、不调 Gitea API（无需 Token）。

.EXAMPLE
  # 使用默认 scripts/.env.release
  powershell -ExecutionPolicy Bypass -File scripts\release.ps1 -Version 0.0.1-test -Target windows -Draft

.EXAMPLE
  # 指定环境文件 + 只打包不上传
  powershell -ExecutionPolicy Bypass -File scripts\release.ps1 -Version 0.0.1-test -EnvFile .\scripts\.env.release -DryRun
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

# 控制台中文：尽量 UTF-8，避免 cmd 下乱码
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

# ---------- 定位仓库根 ----------
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = (Resolve-Path (Join-Path $ScriptDir "..")).Path
Set-Location $Root

# ---------- 加载 .env ----------
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
Missing Gitea env. Provide EnvFile (scripts/.env.release) or export:
  GITEA_URL / GITEA_TOKEN / GITEA_OWNER / GITEA_REPO
Or use -DryRun to build+zip only.
See: scripts/.env.release.example
"@
  }
}

# ---------- 1) 编译 ----------
Write-Host "[1/5] Build $Target release $Version ..."
$buildBat = Join-Path $Root "scripts\build.bat"
cmd /c "`"$buildBat`" --target=$Target --mode=release --version=$Version"
if ($LASTEXITCODE -ne 0) { throw "build failed (exit $LASTEXITCODE)" }

# ---------- 2) 打 zip（Round 2 后资源在 conf/ + web/） ----------
Write-Host "[2/5] Package zip ..."

# 清理上次失败残留的 staging 目录
Get-ChildItem -LiteralPath $Root -Directory -Filter ".release_stage_*" -ErrorAction SilentlyContinue |
  ForEach-Object { Remove-Item -LiteralPath $_.FullName -Recurse -Force -ErrorAction SilentlyContinue }

$shared = @("conf", "web", "uploads", "favicon.ico", "LICENSE.md")
foreach ($p in $shared) {
  if (-not (Test-Path -LiteralPath (Join-Path $Root $p))) {
    Write-Warning "missing shared path: $p (optional, skip)"
  }
}

$assets = @()

function Copy-TreeRobust {
  param([string]$Src, [string]$DstParent)
  $name = Split-Path -Leaf $Src
  $dst = Join-Path $DstParent $name
  if (Test-Path -LiteralPath $Src -PathType Leaf) {
    Copy-Item -LiteralPath $Src -Destination $DstParent -Force
    return
  }
  # robocopy 对占用/长路径更稳；退出码 0-7 视为成功
  $null = & robocopy $Src $dst /E /NFL /NDL /NJH /NJS /NC /NS /NP /R:2 /W:1
  if ($LASTEXITCODE -ge 8) {
    throw "robocopy failed ($LASTEXITCODE): $Src -> $dst"
  }
}

function New-ZipFromDirectory {
  param([string]$SourceDir, [string]$ZipPath)
  if (Test-Path -LiteralPath $ZipPath) { Remove-Item -LiteralPath $ZipPath -Force }

  # 优先 tar（Win10+ 自带），避开 Compress-Archive 锁文件问题
  $tar = Get-Command tar.exe -ErrorAction SilentlyContinue
  if ($tar) {
    Push-Location $SourceDir
    try {
      & tar.exe -a -c -f $ZipPath *
      if ($LASTEXITCODE -ne 0) { throw "tar zip failed: $LASTEXITCODE" }
    }
    finally { Pop-Location }
    return
  }

  Add-Type -AssemblyName System.IO.Compression.FileSystem
  [System.IO.Compression.ZipFile]::CreateFromDirectory($SourceDir, $ZipPath)
}

function New-ReleaseZip {
  param(
    [string]$ZipName,
    [string[]]$BinaryPaths
  )
  $zipPath = Join-Path $Root $ZipName
  $stage = Join-Path $Root (".release_stage_" + [guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Path $stage | Out-Null
  try {
    $distStage = Join-Path $stage "dist"
    New-Item -ItemType Directory -Path $distStage | Out-Null
    foreach ($bin in $BinaryPaths) {
      if (-not (Test-Path -LiteralPath $bin)) { throw "binary not found: $bin" }
      Copy-Item -LiteralPath $bin -Destination $distStage -Force
    }
    foreach ($p in $shared) {
      $src = Join-Path $Root $p
      if (Test-Path -LiteralPath $src) {
        Copy-TreeRobust -Src $src -DstParent $stage
      }
    }
    New-ZipFromDirectory -SourceDir $stage -ZipPath $zipPath
  }
  finally {
    if (Test-Path -LiteralPath $stage) {
      Remove-Item -LiteralPath $stage -Recurse -Force -ErrorAction SilentlyContinue
    }
  }
  return $zipPath
}

if (($Target -eq "all" -or $Target -eq "windows") -and (Test-Path "dist\doc_windows_amd64.exe")) {
  $assets += (New-ReleaseZip -ZipName "doc_windows_amd64.zip" -BinaryPaths @((Join-Path $Root "dist\doc_windows_amd64.exe")))
}

if (($Target -eq "all" -or $Target -eq "linux") -and (Test-Path "dist\doc_linux_amd64")) {
  $assets += (New-ReleaseZip -ZipName "doc_linux_amd64.zip" -BinaryPaths @((Join-Path $Root "dist\doc_linux_amd64")))
}

if ($assets.Count -eq 0) { throw "no asset to upload, check dist/" }

$assetNames = ($assets | ForEach-Object { Split-Path $_ -Leaf }) -join ", "
Write-Host "  packaged: $assetNames"

if ($DryRun) {
  Write-Host ""
  Write-Host "[DryRun] skipped tag / Gitea Release. zip files are in repo root."
  Write-Host "Done."
  exit 0
}

# ---------- 3) 打 tag + push ----------
if (-not $SkipTagPush) {
  Write-Host "[3/5] Tag $Tag ..."
  $existing = git tag --list $Tag
  if (-not $existing) {
    git tag -a $Tag -m "Release $Tag"
  }
  else {
    Write-Host "  tag $Tag 已存在，跳过 git tag"
  }
  git push origin $Tag
}
else {
  Write-Host "[3/5] Skip tag/push"
}

# ---------- 4) 创建（或获取）Release ----------
Write-Host "[4/5] Create Release $Tag ..."
$headers = @{ Authorization = "token $Token" }

$body = @{
  tag_name   = $Tag
  name       = "Doc $Tag"
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
  Write-Host "  release 创建失败，尝试按 tag 读取已有 release..."
  $release = Invoke-RestMethod -Method Get `
    -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/tags/$Tag" `
    -Headers $headers
  Write-Host "  reuse release id=$($release.id)"
}

# ---------- 5) 上传附件 ----------
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
