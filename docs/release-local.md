# 本地编译 + 本地脚本发版

> 简单、可控、无需 Runner。开发者在本机用脚本一键完成「编译 → 打包 → 创建 Gitea Release → 上传附件」。  
> **脚本已落盘：** `scripts/release.ps1`、`scripts/release.bat`；密钥放 `scripts/.env.release`（gitignore，见 `.env.release.example`）。

## 适用场景

- 团队规模小，发版频率不高
- 暂无 Gitea Actions Runner 资源
- 希望发版动作完全在本机可见、可调试

---

## 一、整体流程

```text
开发者本机
  scripts/release.bat|ps1  ──┐
    ├─ 读取 scripts/.env.release（或 --env=）
    ├─ 调 scripts/build.bat 编 release
    ├─ 打 zip（含 conf/、web/、uploads/ 等）
    ├─ git tag vX.Y.Z && git push origin vX.Y.Z   （可用 --dry-run 跳过）
    ├─ 调 Gitea API 创建 Release
    └─ 调 Gitea API 上传 zip 附件
                                ↓
                        Gitea Releases 页
                  https://git.itopcms.com/jackliu/doc/releases
```

---

## 二、前置准备（一次性）

### 1. 创建 Gitea Personal Access Token

1. 登录 `https://git.itopcms.com`
2. 进入「用户设置 → 应用 → 管理 Access Token → 生成新的令牌」
3. 名称例如：`local-release`
4. 仓库访问：私有仓选 **全部（公开、私有和受限）**
5. 权限：仅将 **`repository` 设为读写**；其余保持无访问
6. 生成后复制 Token，**不要写进仓库**

### 2. 本机环境文件（推荐，不进 git）

```bat
copy scripts\.env.release.example scripts\.env.release
```

编辑 `scripts/.env.release`：

```ini
GITEA_URL=https://git.itopcms.com
GITEA_TOKEN=你的PAT
GITEA_OWNER=jackliu
GITEA_REPO=doc
```

> `scripts/.env.release` 已在 `.gitignore` 中。也可用系统环境变量；脚本会加载 `--env` 文件。

### 3. 构建工具链

参见 `scripts/README.md`：Go ≥ 1.25；Windows 构建需 Zig 或 MinGW。

### 4. 私有模块拉取

```bash
go env -w GOPRIVATE=git.itopcms.com
go env -w GONOSUMDB=git.itopcms.com
```

---

## 三、版本号规范（端到端一致）

| 位置 | 形式 | 示例 | 来源 |
|------|------|------|------|
| Git tag | `vX.Y.Z` | `v1.0.0` | `git tag` |
| 构建参数 | `X.Y.Z`（无 `v`） | `1.0.0` | `release` 传给 `build.*` |
| 程序内 | `X.Y.Z` | `1.0.0` | `-ldflags` → `internal/config.VERSION` |
| Release 名 | `Doc vX.Y.Z` | `Doc v1.0.0` | 脚本生成 |
| zip 文件名 | `doc_<os>_amd64.zip` | `doc_windows_amd64.zip` | 脚本生成 |

发版后用 `doc version` 验证程序内版本与 tag 一致。

---

## 四、日常操作

```bat
REM 只编译+打包（无需 Token）
scripts\release.bat 0.0.1-test windows --dry-run

REM 草稿发版（默认读 scripts\.env.release）
scripts\release.bat 0.0.1-test windows --draft

REM 显式指定 env 文件
scripts\release.bat 0.0.1-test windows --env=scripts\.env.release --draft

REM 正式发版（勿用已存在的生产 tag 做试验）
scripts\release.bat 1.0.0 all
```

PowerShell 等价：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\release.ps1 `
  -Version 0.0.1-test -Target windows -EnvFile scripts\.env.release -Draft
```

验证：

1. 打开 `https://git.itopcms.com/jackliu/doc/releases`
2. 下载 zip 解压，执行 `dist\doc_windows_amd64.exe version`

常用开关：`--dry-run` / `--draft` / `--skip-tag` / `--env=PATH`

---

## 五、发布包目录约定

每个 zip 解压后结构（Round 2 定型）：

```text
doc_windows_amd64.zip
├── dist/doc_windows_amd64.exe
├── conf/
├── web/          # static + views
├── uploads/
├── favicon.ico
└── LICENSE.md
```

> Linux 部署侧通常把 `dist/doc_linux_amd64` 复制/重命名为 `doc`（详见 `deploy-spug-local.md`）。

---

## 六、常见问题

### Q1：tag / Release 已存在

脚本会跳过重复 `git tag`；Release 创建失败时按 tag 复用并覆盖同名附件。强制重来：

```bash
git push origin :refs/tags/v0.0.1-test
git tag -d v0.0.1-test
```

### Q2：附件上传 413 / 慢

检查 Gitea 站点附件大小限制；可先 `--dry-run` 看本地 zip 体积。

### Q3：Token 泄漏

立刻在 Gitea 吊销该 Token，更新 `scripts/.env.release`。

### Q4：发版失败后清理

```powershell
git tag -d v0.0.1-test
git push origin :refs/tags/v0.0.1-test
# 删除 Release（需已加载 Token）
curl -X DELETE -H "Authorization: token $env:GITEA_TOKEN" `
  "$env:GITEA_URL/api/v1/repos/$env:GITEA_OWNER/$env:GITEA_REPO/releases/tags/v0.0.1-test"
```

---

## 七、安全清单

- [ ] PAT 仅 `repository` 读写
- [ ] `scripts/.env.release` 不进 git
- [ ] 试验用 `0.0.x-test` + `--draft`，测完删 tag/Release
- [ ] 公开仓注意附件可见性

---

## 八、与 Spug 协同

`release` 脚本只负责把发布包上传到 Gitea Release。部署到服务器请参考 [`deploy-spug-local.md`](./deploy-spug-local.md)。
