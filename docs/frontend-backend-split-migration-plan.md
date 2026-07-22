# 前后端目录拆分迁移执行清单

> 在**同一个仓库内**通过目录结构把"服务端 Go 代码"和"前端视图/静态资源"分离，便于阅读、维护和后续演进。
> 不拆仓、不引入构建工具、不改变现有功能与 URL。

## 一、迁移目标

- 顶层目录按"职责"分组：后端代码 / 前端资源 / 部署运维
- 不改变任何 HTTP URL（`/static/*`、`/uploads/*`、页面路由保持不变）
- 不改变运行时数据目录（`runtime/`、`cache/`、`uploads/` 保留在根目录）
- Beego 框架的 `views`/`static` 默认路径通过 **代码配置** 改写，模板内容不动
- 每一步可独立 commit，单独验证，可回滚

---

## 二、目标目录结构

```text
doc/
├── server/                          # 后端 Go 代码（第 3 步迁入，可选）
│   ├── controllers/
│   ├── models/
│   ├── middleware/
│   ├── routers/
│   ├── converter/
│   ├── commands/
│   ├── utils/
│   └── main.go
│
├── conf/                            # ⚠️ 保留在根目录！它既是配置目录又是 Go 包
│   ├── enumerate.go                 # Go 源码（package conf）
│   ├── mail.go                      # Go 源码（package conf）
│   ├── app.conf
│   ├── app.conf.example
│   └── lang/
│       ├── zh-cn.ini
│       └── en-us.ini
│
├── web/                             # 前端代码（第 1 步迁入）
│   ├── views/                       # 原 views/  (.tpl 模板)
│   ├── static/                      # 原 static/ (第三方库 + 自家 js/css)
│   └── README.md                    # 前端目录说明
│
├── deploy/                          # 部署/运维（第 2 步迁入）
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── start.sh
│   ├── sync_host.sh
│   └── scripts/
│       ├── build.sh / build.bat
│       ├── spug_run.sh
│       ├── doc.service
│       └── README.md
│
├── docs/                            # 项目文档（保留）
├── runtime/                         # 运行时（保留，含日志/session）
├── cache/                           # 运行时缓存（保留）
├── uploads/                         # 用户上传（保留）
├── graphics/                        # 验证码图片素材（保留）
├── mail/                            # 邮件模板（保留）
├── go.mod
├── go.sum
├── README.md
├── LICENSE.md
├── simsun.ttc
└── .gitignore
```

### 关于 `conf/` 的特殊说明

`conf/` 目录**不能整体移动**，原因：
- `conf/enumerate.go`、`conf/mail.go` 声明的是 `package conf`
- 全仓库 30+ 文件都在 `import "git.itopcms.com/jackliu/doc/conf"`
- `commands/install.go` 与 `commands/command.go` 中 `i18n.SetMessage(lang, "conf/lang/"+lang+".ini")` 硬编码了 `conf/lang/` 路径
- `conf/enumerate.go` 中 `ConfigurationFile = "./conf/app.conf"` 硬编码了配置路径

所以 **第一阶段 `conf/` 整体保留原位**。如果将来想把它纯净化（Go 代码挪到 `server/config/`、配置文件挪到 `deploy/conf/`），需要另起一个独立的重构 PR，参见本文 **第八节"可选进阶"**。

---

## 三、迁移阶段总览

| 阶段 | 改动范围 | 风险 | 收益 |
|------|---------|------|------|
| 阶段 1 | 创建 `web/`，挪 `views/` + `static/` | 低 | 前端资源集中可见 |
| 阶段 2 | 创建 `deploy/`，挪 Dockerfile/脚本 | 低 | 根目录清爽 |
| 阶段 3 | 创建 `server/`，挪 Go 业务包（除 `conf`） | 中 | 后端代码独立 |
| 阶段 4 | 文档与 CI 同步更新 | 低 | 文档与代码一致 |

> 推荐：先做阶段 1、2、4，跑稳后再决定是否做阶段 3。

---

## 四、阶段 1：迁移前端资源到 `web/`

### 4.1 文件移动

```bash
# 使用 git mv 保留历史
git mv views web/views
git mv static web/static
```

### 4.2 代码改动

#### (1) `commands/command.go`

文件位置：`commands/command.go` 第 311、332、334、337、342 行附近。

修改前：

```go
if err := gocaptcha.SetFontPath(conf.WorkingDir("static", "fonts")); err != nil {
    log.Fatal("读取字体文件时出错 -> ", err)
}
// ...
web.BConfig.WebConfig.StaticDir["/static"] = filepath.Join(conf.WorkingDirectory, "static")
web.BConfig.WebConfig.StaticDir["/uploads"] = uploads
web.BConfig.WebConfig.ViewsPath = conf.WorkingDir("views")
// ...
fonts := conf.WorkingDir("static", "fonts")
// ...
if err := gocaptcha.SetFontPath(filepath.Join(conf.WorkingDirectory, "static", "fonts")); err != nil {
```

修改后：

```go
if err := gocaptcha.SetFontPath(conf.WorkingDir("web", "static", "fonts")); err != nil {
    log.Fatal("读取字体文件时出错 -> ", err)
}
// ...
web.BConfig.WebConfig.StaticDir["/static"] = filepath.Join(conf.WorkingDirectory, "web", "static")
web.BConfig.WebConfig.StaticDir["/uploads"] = uploads
web.BConfig.WebConfig.ViewsPath = conf.WorkingDir("web", "views")
// ...
fonts := conf.WorkingDir("web", "static", "fonts")
// ...
if err := gocaptcha.SetFontPath(filepath.Join(conf.WorkingDirectory, "web", "static", "fonts")); err != nil {
```

> URL 前缀 `/static`、`/uploads` **不要改**，模板里 `{{.}}` 引用的资源 URL 都不会变。

#### (2) `app.conf` 与 `app.conf.example`

若不在 Go 代码里配置而走 ini 配置，也可以改这两个文件（与上面的代码改动**二选一**，推荐用代码改动方式，避免发版时配置文件未同步出现 404）：

```ini
StaticDir = static:web/static
ViewsPath = web/views
```

> 当前项目走的是代码配置方式（`commands/command.go`），所以**优先改代码**，配置文件不动。

#### (3) `Dockerfile`

第 22 行：

修改前：

```dockerfile
RUN rm -rf cache commands controllers converter .git .github graphics mail models routers utils
```

修改后（保留 `web` 目录不删除）：

```dockerfile
RUN rm -rf cache commands controllers converter .git .github graphics mail models routers utils
# web 目录已包含 views 和 static，无需特殊处理
```

> 注意原 Dockerfile 第 21 行 `RUN rm appveyor.yml docker-compose.yml Dockerfile ... start.sh conf/*.go` 仍需保留，但若执行阶段 2（脚本搬到 `deploy/`），需要同步更新。

#### (4) `start.sh` 注释

第 18-21 行的注释中提到的 `static`、`views`：

修改前：

```bash
# static: 静态文件
# uploads: 上传文件
# views: 页面视图
# echo "export SYNC_LIST='conf;database;runtime;static;uploads;views'" >> /doc-sync-host/sync.sh
```

修改后：

```bash
# web: 前端资源(views + static)
# uploads: 上传文件
# echo "export SYNC_LIST='conf;database;runtime;web;uploads'" >> /doc-sync-host/sync.sh
```

> 默认 `SYNC_LIST='conf;database;uploads'` 不包含 `static/views`，所以默认行为不受影响；只是注释和示例需要更新。

#### (5) `docs/release-local.md` 第 19 行

修改前：

```text
├─ 打 zip（含 conf/static/views/uploads 等）
```

修改后：

```text
├─ 打 zip（含 conf/web/uploads 等）
```

> `docs/release-local.md` 中其他 `static`、`views` 字样也需要同步排查替换。

### 4.3 验证步骤

```bash
go build -o doc.exe .
./doc.exe
# 浏览器访问首页，确认：
# 1. 页面能正常渲染（说明 views 路径生效）
# 2. /static/css/main.css 等资源 200（说明 static 路径生效）
# 3. /static/fonts/*.ttf 验证码字体能加载
# 4. 登录/注册的验证码图能显示（gocaptcha 字体路径）
```

### 4.4 提交粒度

建议拆成 2 个 commit：

1. `refactor(web): move views/static into web/` —— 仅文件移动
2. `refactor(web): update beego paths to web/views and web/static` —— 代码与配置更新

---

## 五、阶段 2：迁移部署文件到 `deploy/`

### 5.1 文件移动

```bash
mkdir deploy
git mv Dockerfile         deploy/Dockerfile
git mv docker-compose.yml deploy/docker-compose.yml
git mv start.sh           deploy/start.sh
git mv sync_host.sh       deploy/sync_host.sh
git mv scripts            deploy/scripts
```

> `scripts/` 整体下沉到 `deploy/scripts/`。

### 5.2 代码与脚本改动

#### (1) `deploy/Dockerfile`

由于 `Dockerfile` 现在位于 `deploy/`，但 `docker build` 一般在仓库根目录执行（`context = .`），需要：

**方式 A**：保持 build context 在根目录，构建命令改为 `-f deploy/Dockerfile`：

```bash
docker build -f deploy/Dockerfile --build-arg TAG=0.0.1 -t doc:latest .
```

`Dockerfile` 内容基本不动，但有 2 处需要调整：

第 28 行：

```dockerfile
# 修改前
ADD simsun.ttc /usr/share/fonts/win/
# 修改后（路径不变，simsun.ttc 仍在根目录）
ADD simsun.ttc /usr/share/fonts/win/
```

第 29 行：

```dockerfile
# 修改前
ADD start.sh /app
# 修改后
ADD deploy/start.sh /app
```

第 125 行：

```dockerfile
# 修改前
RUN chmod +x /doc/start.sh
# 后端运行时 start.sh 已经通过 COPY --from=build /app /doc 进入容器，路径仍是 /doc/start.sh，无需改
```

第 21 行 `RUN rm ...` 中的 `start.sh` 路径在容器内仍是 `start.sh`（已在 `/app/start.sh`），不需要改名。

#### (2) `deploy/start.sh` 中对 `sync_host.sh` 的引用

第 30 行：

```bash
# 修改前
echo "source /doc/sync_host.sh" >> /doc-sync-host/sync.sh
# 修改后（sync_host.sh 仍随 Dockerfile 一起 COPY 到 /doc/）
echo "source /doc/sync_host.sh" >> /doc-sync-host/sync.sh
```

> 容器内路径不变，因为 Dockerfile 是 `COPY --from=build /app /doc`，且 `sync_host.sh` 通过 `COPY .` 进入 `/app`。
> **但**：由于 `sync_host.sh` 已移到 `deploy/sync_host.sh`，需要在 Dockerfile 里显式 ADD：

新增到 Dockerfile build 阶段：

```dockerfile
ADD deploy/sync_host.sh /app
```

并在第 21 行的 `RUN rm ...` 中确保不会误删 `sync_host.sh`（实际不会，因为它已 ADD 到 /app 根）。

#### (3) `scripts/spug_run.sh` 路径更新

`scripts/spug_run.sh` 自身随 `scripts/` 整体迁到 `deploy/scripts/`，但脚本里写死了 spug 服务器上的运行路径：

```bash
WWW=/data/wwwroot/doc.itopcms.com
REPO=/data/repos/doc.itopcms.com/resource
SERVICE_SRC="$REPO/scripts/$SERVICE_NAME"
```

第 12 行 `SERVICE_SRC` 需要改为：

```bash
SERVICE_SRC="$REPO/deploy/scripts/$SERVICE_NAME"
```

> 同时第 43 行 `cp -rf "$WWW/scripts/." "$REPO/scripts/"` 也要改为：
>
> ```bash
> cp -rf "$WWW/deploy/scripts/." "$REPO/deploy/scripts/"
> ```
>
> 并预先 `mkdir -p "$REPO/deploy/scripts"`。

#### (4) `docs/deploy-spug-local.md` 和 `docs/deploy-spug-actions.md`

里面所有 `scripts/spug_run.sh`、`scripts/doc.service`、`scripts/build.sh` 的引用都要更新为 `deploy/scripts/...`。

#### (5) `docker-compose.yml`

`docker-compose.yml` 移到 `deploy/` 后，运行命令改为：

```bash
docker compose -f deploy/docker-compose.yml up -d
```

或在 `deploy/docker-compose.yml` 中如果有 `build:` 指令，需要把 context 改为 `..`。

#### (6) `.gitignore`

第 31 行：

```text
# 修改前
/conf/app.conf
# 修改后（路径未变，保持原样）
/conf/app.conf
```

> `conf/` 仍在根目录，所以不动。

### 5.3 验证步骤

```bash
# 本地构建测试
docker build -f deploy/Dockerfile --build-arg TAG=test -t doc:test .
docker run --rm -p 8181:8181 doc:test
# 浏览器验证首页和静态资源
```

### 5.4 提交粒度

1. `chore(deploy): move Dockerfile/scripts into deploy/` —— 文件移动
2. `chore(deploy): update build context and script paths` —— 路径调整
3. `docs: update deploy docs for deploy/ directory` —— 文档同步

---

## 六、阶段 3（可选）：迁移后端代码到 `server/`

### 6.1 风险评估

- 改动大：所有 `import "git.itopcms.com/jackliu/doc/<pkg>"` 都要改为 `.../doc/server/<pkg>`
- 影响 30+ 个 Go 文件
- 影响 Dockerfile 中 `-ldflags` 的 `-X` 参数（仍是 `conf.VERSION`，因为 `conf/` 不迁）

### 6.2 操作步骤

```bash
mkdir server
git mv controllers server/controllers
git mv models      server/models
git mv middleware  server/middleware
git mv routers     server/routers
git mv converter   server/converter
git mv commands    server/commands
git mv utils       server/utils
git mv main.go     server/main.go
```

### 6.3 `go.mod` 不动

`module git.itopcms.com/jackliu/doc` **保持不变**，因为子目录路径会自动拼接。

### 6.4 全局 import 替换

使用 IDE 全局替换功能：

| 原 import | 新 import |
|-----------|-----------|
| `git.itopcms.com/jackliu/doc/controllers` | `git.itopcms.com/jackliu/doc/server/controllers` |
| `git.itopcms.com/jackliu/doc/models`      | `git.itopcms.com/jackliu/doc/server/models` |
| `git.itopcms.com/jackliu/doc/middleware`  | `git.itopcms.com/jackliu/doc/server/middleware` |
| `git.itopcms.com/jackliu/doc/routers`     | `git.itopcms.com/jackliu/doc/server/routers` |
| `git.itopcms.com/jackliu/doc/converter`   | `git.itopcms.com/jackliu/doc/server/converter` |
| `git.itopcms.com/jackliu/doc/commands`    | `git.itopcms.com/jackliu/doc/server/commands` |
| `git.itopcms.com/jackliu/doc/utils`       | `git.itopcms.com/jackliu/doc/server/utils` |

> `git.itopcms.com/jackliu/doc/conf` **不变**。

### 6.5 `main.go` 同步

`main.go` 移动到 `server/main.go` 后，`go build` 命令需要改：

```bash
# 修改前
go build -o doc .
# 修改后
go build -o doc ./server
```

需要同步更新：
- `scripts/build.sh` 与 `scripts/build.bat`（已移到 `deploy/scripts/`）
- `Dockerfile` 第 18 行的 `go build -o doc_linux_amd64`：

  ```dockerfile
  # 修改前
  RUN go build -o doc_linux_amd64 -ldflags "..."
  # 修改后
  RUN go build -o doc_linux_amd64 -ldflags "..." ./server
  ```

- `Dockerfile` 第 22 行的清理列表：

  ```dockerfile
  # 修改前
  RUN rm -rf cache commands controllers converter .git .github graphics mail models routers utils
  # 修改后
  RUN rm -rf cache .git .github graphics mail server
  ```

  > 注意：`server/` 整体被删除是因为编译产物已生成，但**保留 `server/main.go`** 之外的代码也无意义。

- `.travis.yml`、`appveyor.yml` 中的 `go build` 命令

### 6.6 `WorkingDirectory` 影响

`conf/enumerate.go` 中：

```go
func init() {
    if p, err := filepath.Abs("./conf/app.conf"); err == nil {
        ConfigurationFile = p
    }
    if p, err := filepath.Abs("./"); err == nil {
        WorkingDirectory = p
    }
    // ...
}
```

`./` 是相对于**可执行文件运行目录**的，不是相对源码目录。所以**不受 `main.go` 位置变化影响**，只要运行 doc 时当前目录是项目根（包含 `conf/`、`web/`、`uploads/` 的目录），就能正常工作。

### 6.7 验证步骤

```bash
go build -o doc.exe ./server
./doc.exe
# 浏览器访问，全流程冒烟
```

### 6.8 提交粒度

1. `refactor(server): move backend Go packages into server/` —— `git mv` 操作
2. `refactor(server): update import paths to server/*` —— 全局 import 替换
3. `chore(build): point go build to ./server` —— 构建脚本与 Dockerfile

---

## 七、阶段 4：文档与 CI 同步

### 7.1 文档清单

需要更新的文档（按优先级）：

- [ ] `README.md` —— 目录结构说明
- [ ] `docs/README.md` —— 加入本迁移文档索引
- [ ] `docs/release-local.md` —— zip 打包内容、`scripts/build.sh` 路径
- [ ] `docs/release-gitea-actions.md` —— CI 构建路径
- [ ] `docs/deploy-spug-local.md` —— spug 部署路径
- [ ] `docs/deploy-spug-actions.md` —— spug actions 路径
- [ ] `docs/router-split-migration-plan.md` —— 若已做阶段 3，更新 `routers/` 为 `server/routers/`
- [ ] `docs/routers-reference.md` —— 同上
- [ ] `scripts/README.md` —— 移到 `deploy/scripts/README.md`，更新内容
- [ ] 新增 `web/README.md` —— 前端目录约定（如：哪些是第三方库、哪些是自家代码）

### 7.2 CI 配置

- `.travis.yml`、`appveyor.yml`：若 `main.go` 已迁到 `server/`，构建命令需同步
- `.github/`：检查 GitHub Actions workflow

### 7.3 README 顶层目录结构示例

在 `README.md` 中加一节"项目目录"：

```markdown
## 项目目录

- `server/` — Go 后端代码（controllers/models/routers 等）
- `web/`    — 前端模板与静态资源（views + static）
- `deploy/` — 部署与运维相关（Dockerfile、scripts 等）
- `conf/`   — 应用配置（含配置文件与 Go 配置包）
- `docs/`   — 项目文档
- `runtime/`、`cache/`、`uploads/` — 运行时产生的数据，不进版本控制
```

---

## 八、可选进阶：彻底净化 `conf/`

如果将来想让 `conf/` 真正变成**纯配置目录**，需要做：

### 8.1 拆分 Go 代码与配置文件

```text
server/config/         # 新位置，原 conf/*.go
  config.go            # 由 enumerate.go 改名
  mail.go
deploy/conf/           # 新位置，原 conf/app.conf*
  app.conf
  app.conf.example
  lang/
    zh-cn.ini
    en-us.ini
```

### 8.2 修改硬编码路径

- `server/config/config.go` 第 75 行：

  ```go
  // 修改前
  ConfigurationFile = "./conf/app.conf"
  // 修改后
  ConfigurationFile = "./deploy/conf/app.conf"
  ```

  以及第 348 行 `filepath.Abs("./conf/app.conf")`。

- `commands/command.go` 第 280 行：

  ```go
  // 修改前
  if err := i18n.SetMessage(lang, "conf/lang/"+lang+".ini"); err != nil {
  // 修改后
  if err := i18n.SetMessage(lang, "deploy/conf/lang/"+lang+".ini"); err != nil {
  ```

- `commands/install.go` 第 106 行同上。

- `commands/command.go` 第 305-308 行 `conf.WorkingDir("conf", "app.conf")` → `conf.WorkingDir("deploy", "conf", "app.conf")`

### 8.3 全局 import 替换

```text
git.itopcms.com/jackliu/doc/conf  →  git.itopcms.com/jackliu/doc/server/config
```

### 8.4 包名变更

把 `package conf` 改为 `package config`，所有调用方 `conf.Xxx()` 改为 `config.Xxx()`，影响面**非常大**。

> 这一步建议**单独成 PR**，并且只在确认前三个阶段稳定后再做。

---

## 九、回滚预案

每个阶段都是独立 commit，回滚方式：

| 阶段 | 回滚操作 |
|------|---------|
| 阶段 1 | `git revert` 对应 commit，`views`/`static` 自动回到根目录 |
| 阶段 2 | 同上，Dockerfile/脚本回到根目录，构建命令去掉 `-f deploy/Dockerfile` |
| 阶段 3 | 同上，但 import 改动较多，回滚后需重新跑 `go build` 验证 |

> 不要把多个阶段揉成一个 commit，否则回滚成本会显著上升。

---

## 十、迁移完成后的目录效果（阶段 1+2 完成后）

```text
doc/
├── conf/                  # 配置（保留 Go 包）
├── deploy/                # 部署相关
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── start.sh
│   ├── sync_host.sh
│   └── scripts/
├── web/                   # 前端资源
│   ├── views/
│   ├── static/
│   └── README.md
├── controllers/           # 后端（阶段 3 才会迁）
├── models/
├── middleware/
├── routers/
├── converter/
├── commands/
├── utils/
├── docs/
├── runtime/  cache/  uploads/  graphics/  mail/
├── main.go
├── go.mod  go.sum
├── README.md  LICENSE.md
├── simsun.ttc
└── .gitignore
```

仅完成阶段 1+2，根目录就已经从 **22 个一级目录** 减到 **15 个**，并且"哪些是前端、哪些是部署、哪些是运行时"一眼可辨。

---

## 十一、推荐执行顺序与节奏

1. **本周**：阶段 1（前端迁移） + 阶段 4 文档部分 —— 风险最低、收益直观
2. **下周**：阶段 2（部署迁移） + spug 部署脚本同步验证 —— 涉及部署测试
3. **稳定一段时间后**：阶段 3（后端迁移） —— 大改动，单独排期
4. **未来某天**：阶段 8（`conf/` 净化） —— 可选，做不做都行

每个阶段做完都要：
- 本地跑通 `go build` + 启动
- Docker 构建跑通
- 部署到测试环境验证一遍

---

## 附录 A：本次梳理涉及到的关键代码引用清单

| 引用类型 | 文件位置 | 现状 | 阶段 1 是否需要改 |
|---------|---------|------|-----------------|
| 静态资源路径 | `commands/command.go:332` | `filepath.Join(conf.WorkingDirectory, "static")` | ✅ 改为 `"web", "static"` |
| 模板路径 | `commands/command.go:334` | `conf.WorkingDir("views")` | ✅ 改为 `"web", "views"` |
| 验证码字体 | `commands/command.go:311,337,342` | `conf.WorkingDir("static", "fonts")` | ✅ 改为 `"web", "static", "fonts"` |
| 视图路径读取 | `controllers/BaseController.go:79,163,194` | `web.BConfig.WebConfig.ViewsPath` | ❌ 间接引用，无需改 |
| 视图路径读取 | `models/BookResult.go:272` | `web.BConfig.WebConfig.ViewsPath` | ❌ 间接引用，无需改 |
| 视图路径读取 | `commands/command.go:510,523` | `web.BConfig.WebConfig.ViewsPath` | ❌ 间接引用，无需改 |
| 配置文件路径 | `conf/enumerate.go:75,348` | `"./conf/app.conf"` | ❌ `conf/` 不动 |
| 语言文件路径 | `commands/command.go:280` | `"conf/lang/"+lang+".ini"` | ❌ `conf/` 不动 |
| 语言文件路径 | `commands/install.go:106` | `"conf/lang/"+lang+".ini"` | ❌ `conf/` 不动 |
| URL 资源路径 | `conf/enumerate.go:93,104` 等 | `/static/images/...`（URL） | ❌ URL 不变 |

## 附录 B：URL 与磁盘路径对照（重要）

| URL 路径（不变） | 磁盘路径（阶段 1 后） |
|----------------|---------------------|
| `/static/css/main.css` | `web/static/css/main.css` |
| `/static/js/main.js`   | `web/static/js/main.js` |
| `/static/fonts/*.ttf`  | `web/static/fonts/*.ttf` |
| `/uploads/avatar/...`  | `uploads/avatar/...` （**不变**） |
| 模板 `errors/error.tpl` | `web/views/errors/error.tpl` |

> 用户看到的 URL、模板里写的 `{{.}}/static/...`、上传文件的访问 URL，**完全不变**。这是阶段 1 风险低的根本原因。
