# 本地编译 + 本地脚本发版



> 简单、可控、无需 Runner。开发者在本机用脚本一键完成「编译 → 打包 → 创建 Gitea Release → 上传附件」。  

> **脚本已落盘：**  

> - Windows：`deployments/scripts/release.ps1`、`deployments/scripts/release.bat`  

> - Linux / macOS：`deployments/scripts/release.sh`  

> - JSON 解析：`deployments/scripts/lib/json.sh`（`release.sh` 已引用）  

> 密钥放 `deployments/scripts/.env.release`（gitignore，见 `.env.release.example`）。  
> 日常也可 `just release <version>` / `make release VERSION=<version>`。



## 适用场景



- 团队规模小，发版频率不高

- 暂无 Gitea Actions Runner 资源

- 希望发版动作完全在本机可见、可调试



---



## 一、整体流程



```text

开发者本机

  deployments/scripts/release.bat|ps1|sh  ──┐

    ├─ 读取 deployments/scripts/.env.release（或 --env=）

    ├─ 调 build.bat / build.sh 编 release

    ├─ 打包到 release/：Windows → .zip，Linux → .tar.gz（文件名 doc_<version>_<os>_amd64.*）

    ├─ git tag vX.Y.Z && git push origin refs/tags/vX.Y.Z   （可用 --dry-run 跳过）

    ├─ 调 Gitea API 创建 Release

    └─ 调 Gitea API 上传附件

                                ↓

                        Gitea Releases 页

                  https://git.itopcms.com/astrueus/doc/releases

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



Windows：



```bat

copy scripts\.env.release.example scripts\.env.release

```



Linux / macOS：



```bash

cp deployments/scripts/.env.release.example deployments/scripts/.env.release

```



编辑 `deployments/scripts/.env.release`：



```ini

GITEA_URL=https://git.itopcms.com

GITEA_TOKEN=你的PAT

GITEA_OWNER=astrueus

GITEA_REPO=doc

```



> `deployments/scripts/.env.release` 已在 `.gitignore` 中。也可用系统环境变量；脚本会加载 `--env` 文件。



### 3. 构建工具链



参见 `deployments/scripts/README.md`：Go ≥ 1.25；Windows 构建需 Zig 或 MinGW。  

`release.sh` 额外需要：`curl`、`tar`；打 Windows zip 时需 `zip`。  
发版脚本的 JSON 解析使用仓库内 `deployments/scripts/lib/json.sh`（纯 bash，无需 jq/python/go）。

需要在其它脚本里通用解析 JSON，可 `source deployments/scripts/lib/json.sh`（零第三方依赖，参考 JSON.sh 思路）。



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

| Release 名 | `doc vX.Y.Z` | `doc v1.0.0` | 脚本生成 |

| 发布包文件名 | `doc_<version>_windows_amd64.zip` / `doc_<version>_linux_amd64.tar.gz` | `doc_1.0.0_windows_amd64.zip` | 输出到 `release/`（gitignore） |



发版后用 `doc version` 验证程序内版本与 tag 一致。



---



## 四、日常操作



### Windows



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



### Linux / macOS



```bash

# 只编译+打包（默认 target=linux）

./deployments/scripts/release.sh 0.0.1-test --dry-run



# 草稿发版

./deployments/scripts/release.sh 0.0.1-test linux --draft



# 正式双平台

./deployments/scripts/release.sh 1.0.0 all

```



验证：



1. 打开 `https://git.itopcms.com/astrueus/doc/releases`

2. Windows：下载 zip，执行 `doc.exe version`

3. Linux：下载 tar.gz，执行 `./doc version`



常用开关：`--dry-run` / `--draft` / `--skip-tag` / `--env=PATH`



---



## 五、发布包目录约定



每个包解压后结构（Round 2 定型）：



```text

release/                                          # 本地产物目录（不进 git）

├── doc_1.0.0_windows_amd64.zip

└── doc_1.0.0_linux_amd64.tar.gz

# 解压后内容：

#   doc.exe（Windows）或 doc（Linux）   （包根目录，无平台后缀）

#   conf/app.conf.example + conf/lang/

#   web/  uploads/(空)  deployments/{spug,systemd}/  LICENSE.md

```



> 部署时请将 `conf/app.conf.example` 复制为 `conf/app.conf` 再改配置。



---



## 六、常见问题



### Q1：tag / Release 已存在



脚本会跳过重复 `git tag`；Release 创建失败时按 tag 复用并覆盖同名附件。强制重来：



```bash

git push origin :refs/tags/v0.0.1-test

git tag -d v0.0.1-test

```



### Q2：附件上传 413 / 慢



检查 Gitea 站点附件大小限制；可先 `--dry-run` 看本地产物体积。



### Q3：Token 泄漏



立刻在 Gitea 吊销该 Token，更新 `deployments/scripts/.env.release`。



### Q4：发版失败后清理



```bash

git tag -d v0.0.1-test

git push origin :refs/tags/v0.0.1-test

# 删除 Release（需已加载 Token）

curl -X DELETE -H "Authorization: token $GITEA_TOKEN" \

  "$GITEA_URL/api/v1/repos/$GITEA_OWNER/$GITEA_REPO/releases/tags/v0.0.1-test"

```



### Q5：Linux 打 Windows zip 报 zip not found



安装 `zip`（如 `apt install zip` / `yum install zip`），或只发 Linux：`./deployments/scripts/release.sh 1.0.0 linux`。



---



## 七、安全清单



- [ ] PAT 仅 `repository` 读写

- [ ] `deployments/scripts/.env.release` 不进 git

- [ ] 试验用 `0.0.x-test` + `--draft`，测完删 tag/Release

- [ ] 公开仓注意附件可见性



---



## 八、与 Spug 协同



`release` 脚本只负责把发布包上传到 Gitea Release。部署到服务器请参考 [`deploy-spug-local.md`](../deploy-spug/deploy-spug-local.md)。

