<#
.SYNOPSIS
  Doc 一键发版：编译 → 打包 →（可选）打 tag / 创建 Gitea Release / 上传附件。
  默认只发 Gitea。加 -GitHub 则在 Gitea 成功后再发 GitHub（等镜像 tag）；
  -GitHubOnly 不编包、不打 tag，从 Gitea 核对并下载附件后再发 GitHub。

.PARAMETER Version
  语义化版本号，不含 v 前缀。示例：1.0.0

.PARAMETER Target
  构建目标：all | linux | windows（默认 windows，本机联调更省事）。
  -GitHubOnly 时表示 Gitea 上必须已有对应附件，不再编译。

.PARAMETER EnvFile
  环境变量文件路径（KEY=VALUE）。默认：本脚本目录 deployments/scripts/.env.release（若存在）。
  Gitea：GITEA_URL、GITEA_TOKEN、GITEA_OWNER、GITEA_REPO（DryRun 可不设）。
  GitHub：GITHUB_TOKEN；GITHUB_OWNER / GITHUB_REPO 可缺省为 Gitea 的 owner/repo。

.PARAMETER SkipTagPush
  跳过 git tag / push。

.PARAMETER Draft
  创建为草稿 Release。

.PARAMETER DryRun
  只编译 + 打 zip，不打 tag、不调发布 API（无需 Token）。
  与 -GitHubOnly 合用时：只核对 Gitea 的 tag/包，不下载、不发 GitHub。

.PARAMETER GitHub
  Gitea 发完后，等待 GitHub 镜像出现同一 tag（commit 一致），再上传同一批附件。

.PARAMETER GitHubOnly
  只发 GitHub：先确认 Gitea 已有该 tag 与附件，再下载并上传。不与 -GitHub 同时使用。

.PARAMETER GitHubWait
  等待 GitHub 出现 tag 的超时秒数，默认 90；每隔 5 秒检查一次，超时则失败。

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File deployments\scripts\release.ps1 -Version 0.0.1-test -Target windows -Draft

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File deployments\scripts\release.ps1 -Version 2.3.2 -Target linux -GitHub
#>
param(
  [Parameter(Mandatory = $true)][string]$Version,
  [ValidateSet("all", "linux", "windows")][string]$Target = "windows",
  [string]$EnvFile = "",
  [switch]$SkipTagPush,
  [switch]$Draft,
  [switch]$DryRun,
  [switch]$GitHub,
  [switch]$GitHubOnly,
  [int]$GitHubWait = 90
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
try {
  [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
}
catch { }

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
  throw "cannot locate repo root (need go.mod and cmd/doc); start: $Start"
}
$Root = Find-RepoRoot -Start $ScriptDir
Set-Location $Root

if ($GitHub -and $GitHubOnly) {
  throw "-GitHub 与 -GitHubOnly 不能同时使用"
}
if ($GitHubWait -lt 1) {
  throw "-GitHubWait 必须 >= 1（秒）"
}

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
if ($Base) { $Base = $Base.TrimEnd("/") }

$GhOwner = if ($env:GITHUB_OWNER) { $env:GITHUB_OWNER } else { $Owner }
$GhRepo = if ($env:GITHUB_REPO) { $env:GITHUB_REPO } else { $Repo }
$GhToken = $env:GITHUB_TOKEN
$GhApi = if ($env:GITHUB_API) { $env:GITHUB_API.TrimEnd("/") } else { "https://api.github.com" }
$GhUpload = if ($env:GITHUB_UPLOAD) { $env:GITHUB_UPLOAD.TrimEnd("/") } else { "https://uploads.github.com" }

$needGitea = (-not $DryRun) -or $GitHubOnly
$needGithub = ($GitHub -or $GitHubOnly) -and -not $DryRun

if ($needGitea) {
  if (-not $Owner -or -not $Repo -or -not $Base -or -not $Token) {
    throw @"
Missing Gitea env. Provide EnvFile (deployments/scripts/.env.release) or export:
  GITEA_URL / GITEA_TOKEN / GITEA_OWNER / GITEA_REPO
Or use -DryRun to build+zip only（-GitHubOnly 除外，仍要能读 Gitea）。
See: deployments/scripts/.env.release.example
"@
  }
}
if ($needGithub) {
  if (-not $GhToken -or -not $GhOwner -or -not $GhRepo) {
    throw @"
Missing GitHub env. 使用 -GitHub / -GitHubOnly 时需要：
  GITHUB_TOKEN
  GITHUB_OWNER / GITHUB_REPO（可省略，默认与 GITEA_OWNER / GITEA_REPO 相同）
"@
  }
}

function Get-ExpectedAssetNames {
  param([string]$Ver, [string]$Tgt)
  $names = @()
  if ($Tgt -eq "all" -or $Tgt -eq "windows") {
    $names += "doc_${Ver}_windows_amd64.zip"
  }
  if ($Tgt -eq "all" -or $Tgt -eq "linux") {
    $names += "doc_${Ver}_linux_amd64.tar.gz"
  }
  return $names
}

function Get-HttpStatus {
  param($ErrorRecord)
  $ex = $ErrorRecord.Exception
  $resp = $null
  if ($ex.PSObject.Properties["Response"]) { $resp = $ex.Response }
  if (-not $resp -and $ex.InnerException -and $ex.InnerException.PSObject.Properties["Response"]) {
    $resp = $ex.InnerException.Response
  }
  if ($resp -and $resp.StatusCode) {
    return [int]$resp.StatusCode
  }
  return 0
}

function Invoke-GiteaJson {
  param([string]$Method = "GET", [string]$Uri, [object]$Body = $null, [string]$InFile = "", [string]$ContentType = "application/json")
  $headers = @{ Authorization = "token $Token" }
  $p = @{ Method = $Method; Uri = $Uri; Headers = $headers }
  if ($InFile) {
    $p.InFile = $InFile
    $p.ContentType = $ContentType
  }
  elseif ($null -ne $Body) {
    $p.ContentType = $ContentType
    $p.Body = $Body
  }
  return Invoke-RestMethod @p
}

function Get-GiteaHeaders {
  return @{ Authorization = "token $Token" }
}

function Get-GithubHeaders {
  return @{
    Authorization            = "Bearer $GhToken"
    Accept                   = "application/vnd.github+json"
    "X-GitHub-Api-Version"   = "2022-11-28"
    "User-Agent"             = "doc-release-script"
  }
}

function Get-GiteaTagCommit {
  $uri = "$Base/api/v1/repos/$Owner/$Repo/tags/$Tag"
  try {
    $t = Invoke-GiteaJson -Uri $uri
  }
  catch {
    if ((Get-HttpStatus $_) -eq 404) { return $null }
    throw
  }
  if ($t.commit -and $t.commit.sha) { return [string]$t.commit.sha }
  if ($t.id) { return [string]$t.id }
  return $null
}

function Get-GithubTagCommit {
  $refUri = "$GhApi/repos/$GhOwner/$GhRepo/git/ref/tags/$Tag"
  try {
    $ref = Invoke-RestMethod -Method Get -Uri $refUri -Headers (Get-GithubHeaders)
  }
  catch {
    if ((Get-HttpStatus $_) -eq 404) { return $null }
    throw
  }
  $obj = $ref.object
  if (-not $obj) { return $null }
  if ($obj.type -eq "commit") { return [string]$obj.sha }
  if ($obj.type -eq "tag") {
    $tagObj = Invoke-RestMethod -Method Get -Uri "$GhApi/repos/$GhOwner/$GhRepo/git/tags/$($obj.sha)" -Headers (Get-GithubHeaders)
    if ($tagObj.object -and $tagObj.object.sha) { return [string]$tagObj.object.sha }
  }
  return [string]$obj.sha
}

function Wait-GithubMirroredTag {
  param([Parameter(Mandatory = $true)][string]$ExpectedSha)
  $expected = $ExpectedSha.ToLowerInvariant()
  $deadline = [datetime]::UtcNow.AddSeconds($GitHubWait)
  Write-Host "[github] 等待 GitHub 出现 tag $Tag（commit $ExpectedSha，最多 ${GitHubWait}s）..."
  while ($true) {
    $sha = Get-GithubTagCommit
    if ($sha) {
      if ($sha.ToLowerInvariant() -eq $expected) {
        Write-Host "  GitHub tag $Tag 已同步，commit 一致"
        return
      }
      throw "GitHub 上已有 tag $Tag，但 commit 为 $sha，与期望 $ExpectedSha 不一致。请核对推送镜像，勿让 GitHub 按默认分支自动建 tag。"
    }
    if ([datetime]::UtcNow -ge $deadline) {
      throw @"
等待 ${GitHubWait}s 后 GitHub 仍没有 tag $Tag（镜像未完成）。
Gitea 侧若已发布成功，请稍后执行：
  deployments\scripts\release.bat $Version $Target --github-only
不要移动或重打已有 tag。
"@
    }
    Write-Host "  尚未看到 GitHub tag $Tag，5s 后重试..."
    Start-Sleep -Seconds 5
  }
}

function Get-GiteaReleaseByTag {
  try {
    return Invoke-GiteaJson -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/tags/$Tag"
  }
  catch {
    if ((Get-HttpStatus $_) -eq 404) { return $null }
    throw
  }
}

function Save-GiteaAssetFile {
  param($Att, [string]$Dest, [string]$Name)
  # GET /releases/{id}/assets/{id} 在部分 Gitea 上仍返回 JSON 元数据（约几百字节），
  # 真正文件在 browser_download_url（/attachments/{uuid}）。
  $url = [string]$Att.browser_download_url
  if (-not $url) {
    $url = "$Base/$Owner/$Repo/releases/download/$Tag/$Name"
  }
  Write-Host "  从 Gitea 下载 $Name"
  $curl = Get-Command curl.exe -ErrorAction SilentlyContinue
  if ($curl) {
    & curl.exe -fsSL --retry 3 --retry-delay 2 -H "Authorization: token $Token" -o $Dest $url
    if ($LASTEXITCODE -ne 0) { throw "curl 下载失败: $Name" }
  }
  else {
    Invoke-WebRequest -Uri $url -Headers (Get-GiteaHeaders) -OutFile $Dest -UseBasicParsing
  }
  if (-not (Test-Path -LiteralPath $Dest)) {
    throw "下载后找不到文件: $Dest"
  }
  $got = [int64](Get-Item -LiteralPath $Dest).Length
  $want = 0
  if ($Att.size) { $want = [int64]$Att.size }
  if ($want -gt 0 -and $got -ne $want) {
    throw "下载大小不符: $Name 期望 $want 字节，实际 $got。若约几百字节，多半下到了附件 JSON 而不是包。"
  }
  if ($got -lt 1024) {
    throw "下载文件过小 ($got 字节): $Name"
  }
}

function Publish-GithubReleaseFromFiles {
  param([string[]]$Files, [string]$BodyText)
  if (-not $Files -or $Files.Count -eq 0) {
    throw "没有可上传到 GitHub 的附件"
  }
  Write-Host "[github] 创建 GitHub Release $Tag ..."
  $ghHeaders = Get-GithubHeaders
  $createBody = @{
    tag_name    = $Tag
    name        = "doc $Tag"
    body        = $BodyText
    draft       = [bool]$Draft
    prerelease  = $false
  } | ConvertTo-Json
  $release = $null
  try {
    $release = Invoke-RestMethod -Method Post -Uri "$GhApi/repos/$GhOwner/$GhRepo/releases" `
      -Headers $ghHeaders -ContentType "application/json; charset=utf-8" -Body $createBody
    Write-Host "  created GitHub release id=$($release.id)"
  }
  catch {
    Write-Host "  创建失败，尝试按 tag 读取已有 Release..."
    $release = Invoke-RestMethod -Method Get -Uri "$GhApi/repos/$GhOwner/$GhRepo/releases/tags/$Tag" -Headers $ghHeaders
    Write-Host "  reuse GitHub release id=$($release.id)"
  }

  $existingAssets = @()
  try {
    $existingAssets = @(Invoke-RestMethod -Method Get -Uri "$GhApi/repos/$GhOwner/$GhRepo/releases/$($release.id)/assets" -Headers $ghHeaders)
  }
  catch { $existingAssets = @() }

  foreach ($file in $Files) {
    $name = Split-Path $file -Leaf
    $old = @($existingAssets | Where-Object { $_.name -eq $name })
    foreach ($o in $old) {
      Write-Host "  delete old GitHub asset: $name (id=$($o.id))"
      Invoke-RestMethod -Method Delete -Uri "$GhApi/repos/$GhOwner/$GhRepo/releases/assets/$($o.id)" -Headers $ghHeaders | Out-Null
    }
    Write-Host "  upload GitHub: $name"
    $uploadUri = "$GhUpload/repos/$GhOwner/$GhRepo/releases/$($release.id)/assets?name=$([uri]::EscapeDataString($name))"
    $uploadHeaders = Get-GithubHeaders
    Invoke-RestMethod -Method Post -Uri $uploadUri -Headers $uploadHeaders `
      -InFile $file -ContentType "application/octet-stream" | Out-Null
  }
  Write-Host "GitHub Release: https://github.com/$GhOwner/$GhRepo/releases/tag/$Tag"
}

function Get-LocalTagCommit {
  $parse = & git rev-parse "$Tag^{commit}" 2>$null
  if ($LASTEXITCODE -eq 0 -and $parse) { return $parse.Trim() }
  return $null
}

$OutDir = Join-Path $Root "release"
$expectedNames = @(Get-ExpectedAssetNames -Ver $Version -Tgt $Target)

# ---------- GitHubOnly：不编译，核 Gitea 后下包再发 GitHub ----------
if ($GitHubOnly) {
  Write-Host "[github-only] 核对 Gitea $Base/$Owner/$Repo 的 $Tag ..."
  $giteaSha = Get-GiteaTagCommit
  if (-not $giteaSha) {
    throw "Gitea 上没有 tag $Tag，不能只发 GitHub。请先完整发版或检查版本号。"
  }
  $giteaRel = Get-GiteaReleaseByTag
  if (-not $giteaRel) {
    throw "Gitea 上有 tag $Tag，但没有对应 Release。不能只发 GitHub。"
  }
  $giteaAssets = @($giteaRel.assets)
  if ($giteaAssets.Count -eq 0) {
    throw "Gitea Release $Tag 没有附件。"
  }
  $byName = @{}
  foreach ($a in $giteaAssets) { $byName[$a.name] = $a }
  $missing = @($expectedNames | Where-Object { -not $byName.ContainsKey($_) })
  if ($missing.Count -gt 0) {
    throw "Gitea Release $Tag 缺少附件: $($missing -join ', ')（当前 Target=$Target）"
  }

  if ($DryRun) {
    Write-Host "[DryRun] Gitea 已有 tag $Tag (commit $giteaSha) 与附件 $($expectedNames -join ', ')。跳过下载与 GitHub。"
    Write-Host "Done."
    exit 0
  }

  if (-not (Test-Path -LiteralPath $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir | Out-Null
  }
  $downloaded = @()
  foreach ($name in $expectedNames) {
    $att = $byName[$name]
    $dest = Join-Path $OutDir $name
    Save-GiteaAssetFile -Att $att -Dest $dest -Name $name
    $downloaded += $dest
  }

  $bodyText = if ($giteaRel.body) { [string]$giteaRel.body } else { "Auto release $Tag" }
  Wait-GithubMirroredTag -ExpectedSha $giteaSha
  Publish-GithubReleaseFromFiles -Files $downloaded -BodyText $bodyText
  Write-Host "Done."
  exit 0
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

  $webSrc = Join-Path $Root "web"
  if (-not (Test-Path -LiteralPath $webSrc)) { throw "missing required path: web/" }
  Copy-TreeRobust -Src $webSrc -Dst (Join-Path $Stage "web")

  $confStage = Join-Path $Stage "conf"
  New-Item -ItemType Directory -Path $confStage -Force | Out-Null
  $langSrc = Join-Path $Root "conf\lang"
  if (-not (Test-Path -LiteralPath $langSrc)) { throw "missing required path: conf/lang/" }
  Copy-TreeRobust -Src $langSrc -Dst (Join-Path $confStage "lang")
  $exampleSrc = Join-Path $Root "conf\app.conf.example"
  if (-not (Test-Path -LiteralPath $exampleSrc)) { throw "missing required path: conf/app.conf.example" }
  Copy-Item -LiteralPath $exampleSrc -Destination (Join-Path $confStage "app.conf.example") -Force

  New-Item -ItemType Directory -Path (Join-Path $Stage "uploads") -Force | Out-Null

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
  Write-Host "[DryRun] skipped tag / Gitea / GitHub Release. packages are under release/"
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

if ($GitHub) {
  $expectedSha = Get-LocalTagCommit
  if (-not $expectedSha) {
    $expectedSha = Get-GiteaTagCommit
  }
  if (-not $expectedSha) {
    throw "无法解析 $Tag 的 commit（本地与 Gitea 都没有）。GitHub 发布中止；Gitea 已成功，可用 --github-only 重试。"
  }
  try {
    Wait-GithubMirroredTag -ExpectedSha $expectedSha
    Publish-GithubReleaseFromFiles -Files $assets -BodyText "Auto release $Tag"
  }
  catch {
    Write-Host "[ERROR] Gitea 已发布成功，GitHub 失败：$($_.Exception.Message)"
    Write-Host "补发： deployments\scripts\release.bat $Version $Target --github-only"
    throw
  }
}
