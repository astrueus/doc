# 本地编译 + 本地脚本发版

> 简单、可控、无需 Runner。开发者在本机用脚本一键完成「编译 → 打包 → 创建 Gitea Release → 上传附件」。

## 适用场景

- 团队规模小，发版频率不高
- 暂无 Gitea Actions Runner 资源
- 希望发版动作完全在本机可见、可调试

---

## 一、整体流程

```text
开发者本机
  scripts/release.{ps1|sh}  ──┐
    ├─ 调 scripts/build.{bat|sh} 编 release
    ├─ 打 zip（含 conf/static/views/lib/uploads 等）
    ├─ git tag vX.Y.Z && git push origin vX.Y.Z
    ├─ 调 Gitea API 创建 Release
    └─ 调 Gitea API 上传 zip 附件
                                ↓
                        Gitea Releases 页
                  https://git.itopcms.com/jackliu/doc/releases
```

> 编译复用现有 `scripts/build.sh` / `scripts/build.bat`，只是再在外层加一个 `release` 脚本负责打包和上传。

---

## 二、前置准备（一次性）

### 1. 创建 Gitea Personal Access Token

1. 登录 `https://git.itopcms.com`
2. 进入「用户设置 → 应用 → 生成新令牌（Token）」
3. 名称：`local-release`，作用域至少勾选：
   - `write:repository`
   - `write:release`（如未独立列出，则用 `repo` 全权限）
4. 妥善保存 Token，**不要写进仓库**

### 2. 本机设置环境变量（不进 git）

Windows（PowerShell）：

```powershell
setx GITEA_URL   "https://git.itopcms.com"
setx GITEA_TOKEN "你的PAT"
setx GITEA_OWNER "jackliu"
setx GITEA_REPO  "doc"
```

> `setx` 设置的是持久环境变量，重新开终端后才生效。

Linux / macOS：

```bash
# 追加到 ~/.bashrc 或 ~/.zshrc
export GITEA_URL=https://git.itopcms.com
export GITEA_TOKEN=你的PAT
export GITEA_OWNER=jackliu
export GITEA_REPO=doc
```

### 3. 构建工具链

参见 `scripts/README.md`：

- Go ≥ 1.25
- Windows 构建：MinGW-w64 或 Zig
- Linux 构建：gcc/clang 或 Zig 交叉编译

### 4. 私有模块拉取

若 Go 模块来自私有 Gitea：

```bash
go env -w GOPRIVATE=git.itopcms.com
go env -w GONOSUMDB=git.itopcms.com
```

---

## 三、版本号规范（端到端一致）

| 位置 | 形式 | 示例 | 来源 |
|------|------|------|------|
| Git tag | `vX.Y.Z` | `v1.0.0` | `git tag` |
| 构建参数 | `X.Y.Z`（无 `v`） | `1.0.0` | `release.{ps1\|sh}` 传给 `build.*` |
| 程序内 | `X.Y.Z` | `1.0.0` | `-ldflags -X conf.VERSION=...` |
| Release 名 | `Doc vX.Y.Z` | `Doc v1.0.0` | 脚本生成 |
| zip 文件名 | `doc_<os>_amd64.zip` | `doc_windows_amd64.zip` | 脚本生成 |

发版后用 `doc version` 验证程序内版本与 tag 一致。

---

## 四、推荐脚本：`scripts/release.ps1`（Windows / 跨平台）

将以下内容保存为 `scripts/release.ps1`，与现有 `build.bat` 同目录：

```powershell
<#
.SYNOPSIS
  Doc 一键发版脚本：本地编译 + 打 zip + 推 tag + 创建 Gitea Release + 上传附件。

.PARAMETER Version
  语义化版本号，不含 v 前缀。示例：1.0.0

.PARAMETER Target
  构建目标：all | linux | windows（默认 all）

.PARAMETER SkipTagPush
  跳过 git tag / push（已经手动打过 tag 时使用）

.PARAMETER Draft
  创建为草稿 Release（不公开）

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts\release.ps1 -Version 1.0.0
#>
param(
  [Parameter(Mandatory=$true)][string]$Version,
  [ValidateSet("all","linux","windows")][string]$Target = "all",
  [switch]$SkipTagPush,
  [switch]$Draft
)

$ErrorActionPreference = "Stop"

$Tag   = "v$Version"
$Owner = $env:GITEA_OWNER
$Repo  = $env:GITEA_REPO
$Base  = $env:GITEA_URL
$Token = $env:GITEA_TOKEN

if (-not $Owner -or -not $Repo -or -not $Base -or -not $Token) {
  throw "请先设置环境变量: GITEA_OWNER, GITEA_REPO, GITEA_URL, GITEA_TOKEN"
}

# ---------- 1) 编译 ----------
Write-Host "[1/5] Build $Target release $Version ..."
cmd /c "scripts\build.bat $Target release $Version"
if ($LASTEXITCODE -ne 0) { throw "build failed" }

# ---------- 2) 打 zip ----------
Write-Host "[2/5] Package zip ..."

$shared = @("conf","static","views","lib","uploads","favicon.ico","LICENSE.md")
foreach ($p in $shared) {
  if (-not (Test-Path $p)) { Write-Warning "missing shared path: $p" }
}

$assets = @()

if (($Target -eq "all" -or $Target -eq "windows") -and (Test-Path "dist\doc_windows_amd64.exe")) {
  $zip = "doc_windows_amd64.zip"
  if (Test-Path $zip) { Remove-Item $zip -Force }
  Compress-Archive -Force -Path (@("dist\doc_windows_amd64.exe") + $shared) -DestinationPath $zip
  $assets += $zip
}

if (($Target -eq "all" -or $Target -eq "linux") -and (Test-Path "dist\doc_linux_amd64")) {
  $zip = "doc_linux_amd64.zip"
  if (Test-Path $zip) { Remove-Item $zip -Force }
  Compress-Archive -Force -Path (@("dist\doc_linux_amd64") + $shared) -DestinationPath $zip
  $assets += $zip
}

if ($assets.Count -eq 0) { throw "no asset to upload, 请检查 dist/ 目录" }

# ---------- 3) 打 tag + push ----------
if (-not $SkipTagPush) {
  Write-Host "[3/5] Tag $Tag ..."
  $existing = git tag --list $Tag
  if (-not $existing) {
    git tag -a $Tag -m "Release $Tag"
  } else {
    Write-Host "  tag $Tag 已存在，跳过 git tag"
  }
  git push origin $Tag
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
} catch {
  Write-Host "  release 创建失败，尝试按 tag 读取已有 release..."
  $release = Invoke-RestMethod -Method Get `
    -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/tags/$Tag" `
    -Headers $headers
  Write-Host "  reuse release id=$($release.id)"
}

# ---------- 5) 上传附件（同名先删，再上传） ----------
Write-Host "[5/5] Upload assets ..."

$existingAssets = Invoke-RestMethod -Method Get `
  -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/$($release.id)/assets" `
  -Headers $headers

foreach ($file in $assets) {
  $name = Split-Path $file -Leaf
  $old = $existingAssets | Where-Object { $_.name -eq $name }
  if ($old) {
    Write-Host "  delete old asset: $name (id=$($old.id))"
    Invoke-RestMethod -Method Delete `
      -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/$($release.id)/assets/$($old.id)" `
      -Headers $headers | Out-Null
  }
  Write-Host "  upload: $name"
  Invoke-RestMethod -Method Post `
    -Uri "$Base/api/v1/repos/$Owner/$Repo/releases/$($release.id)/assets?name=$name" `
    -Headers $headers -InFile $file -ContentType "application/octet-stream" | Out-Null
}

Write-Host ""
Write-Host "Release published: $Base/$Owner/$Repo/releases/tag/$Tag"
```

### 配套 `scripts/release.bat`（一行包装）

```bat
@echo off
if "%~1"=="" (
  echo Usage: release.bat 1.0.0 [all^|linux^|windows]
  exit /b 1
)
set TARGET=%~2
if "%TARGET%"=="" set TARGET=all
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0release.ps1" -Version %~1 -Target %TARGET%
```

---

## 五、推荐脚本：`scripts/release.sh`（Linux / macOS）

```bash
#!/usr/bin/env bash
# Doc 一键发版脚本（Linux/macOS）
# Usage:
#   scripts/release.sh 1.0.0 [all|linux|windows]
set -euo pipefail

VERSION="${1:?Usage: release.sh <version> [target]}"
TARGET="${2:-all}"
TAG="v$VERSION"

: "${GITEA_URL:?GITEA_URL not set}"
: "${GITEA_TOKEN:?GITEA_TOKEN not set}"
: "${GITEA_OWNER:?GITEA_OWNER not set}"
: "${GITEA_REPO:?GITEA_REPO not set}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

API="$GITEA_URL/api/v1/repos/$GITEA_OWNER/$GITEA_REPO"
AUTH=(-H "Authorization: token $GITEA_TOKEN")

# 1) build
echo "[1/5] build $TARGET release $VERSION"
./scripts/build.sh "$TARGET" release "$VERSION"

# 2) zip
echo "[2/5] zip"
SHARED=(conf static views lib uploads favicon.ico LICENSE.md)
ASSETS=()

if { [ "$TARGET" = "all" ] || [ "$TARGET" = "linux" ]; } && [ -f dist/doc_linux_amd64 ]; then
  rm -f doc_linux_amd64.zip
  zip -r doc_linux_amd64.zip dist/doc_linux_amd64 "${SHARED[@]}"
  ASSETS+=(doc_linux_amd64.zip)
fi
if { [ "$TARGET" = "all" ] || [ "$TARGET" = "windows" ]; } && [ -f dist/doc_windows_amd64.exe ]; then
  rm -f doc_windows_amd64.zip
  zip -r doc_windows_amd64.zip dist/doc_windows_amd64.exe "${SHARED[@]}"
  ASSETS+=(doc_windows_amd64.zip)
fi

[ ${#ASSETS[@]} -gt 0 ] || { echo "no asset to upload"; exit 1; }

# 3) tag
echo "[3/5] tag $TAG"
if ! git tag --list "$TAG" | grep -q .; then
  git tag -a "$TAG" -m "Release $TAG"
fi
git push origin "$TAG"

# 4) create release (or reuse)
echo "[4/5] create release"
RID=$(curl -fsS "${AUTH[@]}" -H "Content-Type: application/json" \
  -X POST "$API/releases" \
  -d "{\"tag_name\":\"$TAG\",\"name\":\"Doc $TAG\",\"body\":\"Auto release $TAG\",\"draft\":false,\"prerelease\":false}" \
  2>/dev/null | sed -n 's/.*"id":\s*\([0-9]\+\).*/\1/p' | head -1 || true)

if [ -z "$RID" ]; then
  RID=$(curl -fsS "${AUTH[@]}" "$API/releases/tags/$TAG" | sed -n 's/.*"id":\s*\([0-9]\+\).*/\1/p' | head -1)
fi
echo "  release id = $RID"

# 5) upload assets (同名先删)
echo "[5/5] upload"
EXIST_JSON=$(curl -fsS "${AUTH[@]}" "$API/releases/$RID/assets")
for f in "${ASSETS[@]}"; do
  NAME="$(basename "$f")"
  OLD_ID=$(echo "$EXIST_JSON" | sed -n "/\"name\":\"$NAME\"/,/}/p" | sed -n 's/.*"id":\s*\([0-9]\+\).*/\1/p' | head -1 || true)
  if [ -n "$OLD_ID" ]; then
    curl -fsS "${AUTH[@]}" -X DELETE "$API/releases/$RID/assets/$OLD_ID" >/dev/null
  fi
  curl -fsS "${AUTH[@]}" -X POST "$API/releases/$RID/assets?name=$NAME" \
    -H "Content-Type: application/octet-stream" --data-binary "@$f" >/dev/null
  echo "  uploaded $NAME"
done

echo
echo "Release published: $GITEA_URL/$GITEA_OWNER/$GITEA_REPO/releases/tag/$TAG"
```

执行：

```bash
chmod +x scripts/release.sh
./scripts/release.sh 1.0.0
```

---

## 六、日常发版操作

```text
1. 更新代码到主分支并通过自测
2. 在项目根目录执行：
   Windows:  scripts\release.bat 1.0.0
   Linux:    ./scripts/release.sh 1.0.0
3. 打开 Gitea Releases 页面验证：
   https://git.itopcms.com/jackliu/doc/releases/tag/v1.0.0
4. 下载附件解压，在测试机执行 doc(.exe) version 验证版本号
```

---

## 七、发布包目录约定

每个 zip 解压后必须可独立运行，目录结构：

```text
doc_linux_amd64.zip
├── dist/doc_linux_amd64        # 可执行文件
├── conf/                       # 含 app.conf.example
├── static/
├── views/
├── lib/                        # 含 time/zoneinfo.zip
├── uploads/
├── favicon.ico
└── LICENSE.md
```

> Linux 部署侧需要把 `dist/doc_linux_amd64` 重命名/复制为 `doc`（详见 `deploy-spug-local.md`）。

---

## 八、常见问题

### Q1：`tag 已存在` 错误
`release.ps1` 已做兼容：tag 存在时跳过创建；release 存在时按 tag 复用并删旧附件。如需强制重发，删除远端 tag 后重来：

```bash
git push origin :refs/tags/v1.0.0
git tag -d v1.0.0
```

### Q2：附件上传 413 / 慢
- Gitea 默认对附件大小有限制（站点管理 → 配置）
- 大附件建议先 `7z` 压缩或拆分

### Q3：Token 泄漏处理
- 立刻去 Gitea 用户设置吊销该 Token
- 重新生成，并更新本机 `GITEA_TOKEN`

### Q4：私有依赖拉不到
- 确认 `GOPRIVATE` 已设
- 在 `~/.netrc` 或 `git config --global url.<base>.insteadOf` 配置凭据

### Q5：发版失败后清理
```powershell
# 删除本地未推 tag
git tag -d v1.0.0
# 删除远端 tag
git push origin :refs/tags/v1.0.0
# 删除 Gitea 上的 Release（API）
curl -X DELETE -H "Authorization: token $env:GITEA_TOKEN" `
  "$env:GITEA_URL/api/v1/repos/$env:GITEA_OWNER/$env:GITEA_REPO/releases/tags/v1.0.0"
```

---

## 九、安全清单

- [ ] PAT 仅给到必要权限
- [ ] PAT 仅放环境变量，不进 git
- [ ] 公网下载若涉及私有仓库，发布附件可被未授权访问吗？（按 Gitea 仓库可见性决定）
- [ ] 关键发版打 GPG 签名（可选）：`git tag -s v1.0.0`

---

## 十、与 Spug 协同

`release.{ps1|sh}` 只负责把发布包上传到 Gitea Release。把它部署到服务器请参考 [`deploy-spug-local.md`](./deploy-spug-local.md)。
