# Doc 文档管理系统

Doc 是一款针对 IT 团队开发的简单好用的文档管理系统，基于 [MinDoc](https://github.com/mindoc-org/mindoc) 二次开发，由 [git.itopcms.com/astrueus/doc](https://git.itopcms.com/astrueus/doc) 自主维护。

MinDoc 的前身是 [SmartWiki](https://github.com/lifei6671/SmartWiki) 文档系统。SmartWiki 是基于 PHP 框架 Laravel 开发的一款文档管理系统。因 PHP 的部署对普通用户来说太复杂，原作者改用 Golang 重写了 MinDoc，方便部署和使用。

开发缘起是公司 IT 部门需要一款简单实用的项目接口文档管理和分享的系统。其功能和界面源于 kancloud。

可以用来储存日常接口文档、数据库字典、手册说明等文档。内置项目管理、用户管理、权限管理等功能，能够满足大部分中小团队的文档管理需求。

### 开发 & 维护说明

- 上游项目：[mindoc-org/mindoc](https://github.com/mindoc-org/mindoc)
- 本仓库：`git.itopcms.com/astrueus/doc`（Go 模块路径与仓库地址一致）
- 可执行文件统一为 **`doc`**

感谢原作者 [lifei6671](https://github.com/lifei6671) 创造了 MinDoc 并持续维护。

---

# Round 2 升级须知（现网必看）

从旧布局 / Round 1 升到本仓库 Round 2 结构时，**必须清 session 与 gob 文件缓存**，否则登录态反序列化会失败。完整步骤见：

- [docs/round-1-4/upgrade-round-2.md](docs/round-1-4/upgrade-round-2.md)
- [CHANGELOG.md](CHANGELOG.md)

摘要：停服 → 清 `runtime/session/*`（或 Redis/MySQL session）→ 清 `runtime/cache/*` → 部署启动 → 用户重新登录。

---

# 安装与使用

时区数据已通过 `import _ "time/tzdata"` 内嵌进二进制，部署时无需额外时区文件。

工作目录可通过环境变量 `DOC_HOME` 或启动参数 `-dir` 指定（优先级：`-dir` > `DOC_HOME` > 可执行文件所在目录）。上传相关：`upload_file_size` 为业务单文件限制；`upload_max_size` / `upload_max_memory` 为框架 HTTP 层限制（值支持 `KB`/`MB`/`GB`）。

更多信息可参考上游 [MinDoc 使用手册](https://www.iminho.me/wiki/docs/mindoc/mindoc-summary.md)。

对于没有 Golang 使用经验的用户，可以从内部发布渠道下载编译好的程序。

如果有 Golang 开发经验，建议通过编译安装，要求 Go 版本不小于 **1.25.0**（需支持 `CGO` 和 `go mod`）。

### 使用构建脚本（推荐）

项目提供了跨平台构建脚本（权威路径在 `deployments/scripts/`；根目录 `make` / `just` 为快捷入口）：

| 入口 | 平台 | 说明 |
|------|------|------|
| `make build` / `just build` | 各平台 | 调用 `deployments/scripts/build.sh` 或 `build.bat` |
| `make test` / `just test` | 各平台 | 白名单包测试 + 覆盖率门槛 |
| `make run` / `just run` | 各平台 | 开发启动：`go run`（不落盘二进制；固定 `--dir` 为仓库根） |
| `deployments/scripts/build.sh` | Linux / macOS | Linux 本机构建使用系统 gcc/clang |
| `deployments/scripts/build.bat` | Windows | Linux 交叉编译使用 **Zig** |

编译 Windows 时传入 `mingw` 可改用 **MinGW-w64**。默认**调试模式**（产物输出到项目根目录），也可切换为**发布模式**（产物输出到 `dist/`）。

```bash
# Linux：本机构建 + 交叉编译 Windows（交叉编译需 Zig）
make build
# 等价：bash deployments/scripts/build.sh

# 只构建 Linux（仅需 gcc/clang）
bash deployments/scripts/build.sh --target=linux

# 发布构建
bash deployments/scripts/build.sh --mode=release --version=1.0.0
```

```bat
REM Windows：Zig 构建 Linux + Windows（需安装 Zig 并加入 PATH）
just build
REM 或：deployments\scripts\build.bat

REM 只构建 Windows（Zig）
deployments\scripts\build.bat --target=windows

REM 使用 MinGW-w64 构建 Windows
deployments\scripts\build.bat --target=windows --toolchain=mingw

REM 发布构建
deployments\scripts\build.bat --mode=release --version=1.0.0
```

更多参数说明见 [deployments/scripts/README.md](deployments/scripts/README.md)。

### 跑测试

```bash
make test          # 或 just test
# 权威脚本：
bash deployments/scripts/test.sh
```

Windows：`just test` 或 `powershell -File deployments\scripts\test.ps1`。  
默认闸 `pkg/`、`internal/{errs,auth,logging,i18n,repository}`；覆盖率不低于 [docs/round-5/coverage-baseline.txt](docs/round-5/coverage-baseline.txt)。CI 见 `.github/workflows/test.yml`。

### 开发运行

日常改代码、打断点用 `go run`，不要先 `just build`。脚本会打开 CGO，并把 `--dir` 钉在仓库根（`go run` 的临时二进制不能当工作目录）。**不加热重载**，改完停掉再跑一次。

```bash
# 首次：复制配置（没有 app.conf 时程序也会从 example 拷一份）
cp conf/app.conf.example conf/app.conf

just run install    # 初始化数据库
just run            # 启动 Web（默认 127.0.0.1:8181）
just run mcp        # 其它子命令原样后传
```

Windows：`just run`（或 `powershell -File deployments\scripts\run.ps1`）。  
Linux / macOS：`just run` 或 `make run`；传子命令用 `make run ARGS=install`。  
just 自身的 flag：`just run -- --help`。

需要发布用二进制时再 `just build` 后执行 `./doc`（Windows 为 `.\doc.exe`）。

### 手动编译

```bash
# 克隆源码
git clone https://git.itopcms.com/astrueus/doc.git
cd doc

# 安装依赖
go mod tidy

# 编译（SQLite 需要 CGO 支持）
go build -ldflags "-w" -o doc

# 数据库初始化（执行前需配置 conf/app.conf）
./doc install

# 启动
./doc
```

Windows 下编译产物为 `doc.exe`，命令同理：

```powershell
go build -ldflags "-w" -o doc.exe
.\doc.exe install
.\doc.exe
```

### 私有仓库 Go 环境（如需要）

```bash
go env -w GOPRIVATE=git.itopcms.com
go env -w GONOSUMDB=git.itopcms.com
```

### 数据库配置

如果使用 **MySQL** 存储数据，编码必须是 `utf8mb4_general_ci`。请在安装前，把数据库配置填充到项目目录下的 `conf/app.conf` 中。

如果使用 **SQLite** 数据库，则直接在配置文件中配置数据库路径即可。

如果 `conf` 目录下不存在 `app.conf`，请将 `app.conf.example` 重命名为 `app.conf`。

example 里的密钥（Session、数据库、LDAP、HTTP 自定义登录、SMTP）只有 `${DOC_*}` 占位、**没有明文默认值**。用 MySQL 时必须设置 `DOC_DB_PASSWORD`；生产必须设置 `DOC_SESSION_KEY`。启用 LDAP / HTTP 自定义登录 / 邮件时再设对应变量。

**默认程序会自动初始化一个超级管理员用户：`admin`，密码：`123456`。请登录后重新设置密码。**

### 常用命令

```bash
./doc                 # 启动 Web 服务（等价 ./doc web）
./doc web             # 显式启动 Web 服务
./doc install         # 初始化数据库
./doc version         # 查看当前版本
./doc password --account admin --password newpass   # 修改用户密码
./doc migrate         # 执行数据库迁移
./doc mcp             # MCP stdio（默认；身份见 mcp_stdio_member）
./doc mcp --http      # MCP Streamable HTTP（监听 mcp_listen，需 Bearer doc_…）
./doc service install # 安装系统服务（服务名 docd，绕过 cobra）
./doc --help          # 查看全部子命令
```

Web 主进程在 `mcp_enable=true` 时也会挂载 `/mcp`（与主站同端口）。HTTP MCP **必须走 HTTPS**（反代）；`0.0.0.0` 监听会打印告警。Token 在登录后 `/member/api-tokens` 生成。

完整接入说明（Claude Desktop / Cursor / 工具速查）：[docs/mcp-integration.md](docs/mcp-integration.md)。

### 邮件配置示例

```ini
# 是否启用邮件
enable_mail=true
# smtp 服务器的账号
smtp_user_name=admin@example.com
# smtp 服务器的地址
smtp_host=smtp.example.com
# 密码
smtp_password=your_password
# 端口号
smtp_port=25
# 邮件发送人的地址
form_user_name=admin@example.com
# 邮件有效期 30 分钟
mail_expired=30
```

---

# 使用 Docker 部署

可参考项目内置的 `Dockerfile` 自行构建镜像。构建完成后，容器内程序目录为 `/doc`，数据同步目录为 `/doc-sync-host`。

```bash
# 构建镜像
docker build --progress plain --rm --build-arg TAG=0.0.1 --tag doc:latest .

# 运行（Linux / macOS）
export DOC=/home/ubuntu/doc-docker
docker run -it --name=doc --restart=always \
  -v "${DOC}:/doc-sync-host" \
  -p 8181:8181 \
  -e DOC_ENABLE_EXPORT=true \
  -d doc:latest
```

Windows：

```powershell
set DOC=//d/doc
docker run -it --name=doc --restart=always -v "%DOC%":"/doc-sync-host" -p 8181:8181 -e DOC_ENABLE_EXPORT=true -d doc:latest
```

启动镜像时常用的环境变量（完整列表见 [`conf/app.conf.example`](conf/app.conf.example)；对照表见 [docs/round-5/round-5-env-mindoc-to-doc.md](docs/round-5/round-5-env-mindoc-to-doc.md)）：

```ini
DOC_ADDR                    监听地址（容器内常用 0.0.0.0）
DOC_PORT                    监听端口（默认 8181）
DOC_DB_ADAPTER              数据库类型（mysql / sqlite3）
DOC_DB_HOST                 MySQL 地址
DOC_DB_PORT                 MySQL 端口
DOC_DB_DATABASE             数据库名或 sqlite 文件路径
DOC_DB_USERNAME             数据库账号
DOC_DB_PASSWORD             数据库密码（MySQL 必填，example 无默认）
DOC_SESSION_KEY             Session 签名密钥（生产必填，example 无默认）
DOC_HTTP_LOGIN_SECRET       HTTP 自定义登录签名（启用时必填）
DOC_LDAP_PASSWORD           LDAP 绑定密码（启用 LDAP 时必填）
DOC_SMTP_PASSWORD           SMTP 密码（启用邮件时必填）
DOC_ENABLE_EXPORT           开启导出（默认 false）
DOC_HOME                    工作目录（也可用启动参数 -dir）
```

> 说明：环境变量前缀为 **`DOC_*`**；旧 `MINDOC_*` **不再读取**（Breaking）。

### docker-compose 一键安装

1. 修改 `docker-compose.yml` 中的配置信息，主要修改 `image` 为本地构建的镜像，以及 `volumes` 节点，将宿主机的目录映射到容器内。
2. 一键完成所有环境搭建：

   ```bash
   docker-compose up -d
   ```

3. 浏览器访问：<http://localhost:8181/>

4. 常用命令：

   ```bash
   docker-compose up -d      # 启动
   docker-compose stop       # 停止
   docker-compose restart    # 重启
   docker-compose down       # 停止并删除容器
   ```

---

# 使用的技术

- [Beego](https://github.com/beego/beego) v2
- MySQL 5.6+ / SQLite
- [editor.md](https://github.com/pandao/editor.md) Markdown 编辑器
- [Bootstrap](https://github.com/twbs/bootstrap) 3.2
- [jQuery](https://github.com/jquery/jquery)
- [WebUploader](https://github.com/fex-team/webuploader) 文件上传框架
- [NProgress](https://github.com/rstacruz/nprogress)
- [jsTree](https://github.com/vakata/jstree) 树状结构库
- [Font Awesome](https://github.com/FortAwesome/Font-Awesome) 字体库
- [Cropper](https://github.com/fengyuanchen/cropper) 图片剪裁库
- [layer](https://github.com/sentsin/layer) 弹出层框架
- [highlight.js](https://github.com/highlightjs/highlight.js) 代码高亮库
- [Turndown](https://github.com/domchristie/turndown) HTML 转 Markdown 库
- [wangEditor](https://github.com/wangeditor-team/wangEditor) 富文本编辑器
- [Vue.js](https://github.com/vuejs/vue)

---

# 主要功能

- 项目管理：可以对项目进行编辑更改、成员添加等。
- 文档管理：添加和删除文档等。
- 评论管理：可以管理文档评论和自己发布的评论。
- 用户管理：添加和禁用用户，个人资料更改等。
- 用户权限管理：实现用户角色的变更。
- 项目加密：可以设置项目公开状态，私有项目需要通过 Token 访问。
- 站点配置：可开启匿名访问、验证码等。
