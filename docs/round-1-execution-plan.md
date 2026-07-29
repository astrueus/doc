# Round 1 · 执行文档（低风险重构 + 后续轮次前置）

> 本文是 [refactor-roadmap.md §五 Round 1](./refactor-roadmap.md#🥇-round-1低风险重构--后续轮次前置准备1-周) 的**可执行分解**。
> 目标：在**不动包路径、不动目录结构**的前提下，一周内落地一批"内部重构 + 后续轮次前置"改动，为 Round 2/3 铺路。
> **约束：** 所有改动**对用户零感知**，不上任何新功能。

---

## 一、范围与不做清单

### 本轮做

| 序号 | 任务 | 目录/文件影响面 | 上线感知 |
|---|---|---|---|
| T1 | `configs/` 目录独立 + smtp 双引号 bug | 新增 `configs/`；`conf/` 保留 Go 源码 | 部署脚本需同步（见 §六） |
| T2 | `ioutil` → `os.ReadFile` / `os.WriteFile` | 12 个 Go 文件 | 无 |
| T3 | `interface{}` → `any` | 全仓 | 无 |
| T4 | `main.go` 引入 `spf13/cobra` | `main.go`、`commands/` | 命令行**兼容**旧写法（见 §四 T4） |
| T5 | `cache.Cache` 抽接口 + `context` 传递 | `cache/` | 无（旧函数保留 shim） |
| T6 | `BaseController.Prepare` options 缓存 | `controllers/BaseController.go` | 后台切换开关**最长 5 分钟**生效（见 §四 T6） |
| T7 | `internal/errs/` + `BizError` + `JsonError(err)` helper | 新增 `internal/errs/`；`controllers/BaseController.go` | 无 |

### 本轮**不做**（明确排除）

- ❌ **不搬 Go 源码目录**：`conf/enumerate.go` / `conf/mail.go` 保留在 `conf/` 包，Round 2 再迁 `internal/config/`。
- ❌ **不动 `import` 路径**：`git.itopcms.com/jackliu/doc/conf` 等 30+ import 保持不变。
- ❌ **不拆 `app.conf` 为多文件**：`[section]` 分组也留到 Round 2（配合强类型 Config struct 一起做）。
- ❌ **不换 gob 序列化**：换成 msgpack/json 涉及缓存清理策略，放 Round 2 一起处理（虽然 §六风险 13 提到"有条件的话 Round 1 就换"，本轮先不做，减小 blast radius）。
- ❌ **不改 Controller/Model 内部逻辑**（除 T6/T7 明确点位）。
- ❌ **前端 P0（editor.md / katex / mermaid）**：已在历史 PR 完成，不再列入。见 [refactor-roadmap.md §四](./refactor-roadmap.md#四支线前端与静态资源)。

---

## 二、前置条件

- Go 1.25.0（已满足，`go.mod` line 3）
- 分支：`feature/round-1-refactor`（建议单独一条长分支，内部按 T1~T7 拆 7 个小 PR 顺序合入）
- 本地能跑 `go build ./...` + `go test ./...`（当前项目无 `_test.go`，Round 4 再补测试）

---

## 三、任务依赖关系

```
T1（configs/ 独立）─────────┐
                            ├─► T4（cobra）─┐
T2（ioutil→os）─────────────┤              ├─► 集成回归
T3（interface{}→any）───────┤              │
T5（cache 抽接口）──────────┤              │
T6（Prepare 缓存）──────────┤              │
T7（BizError/JsonError）────┴──────────────┘
```

T1~T3、T5~T7 相互独立，可并行开工；T4 依赖 T1（configs 路径已就位），最后集成。

---

## 四、逐任务执行细节

### T1 · `configs/` 目录独立 + smtp 双引号 bug（0.5~1 天）

**目标：** 把 `conf/` 里**非 Go 源码**的文件搬到 `configs/`，为 Round 2 目录激进迁移把"配置文件解耦"这一小步先走掉。**Go 源码保留原位**。

**具体步骤：**

1. `git mv conf/app.conf configs/app.conf`
2. `git mv conf/app.conf.example configs/app.conf.example`
3. `git mv conf/lang configs/lang`（整个子目录）
4. 修 smtp 双引号 bug：
   - `configs/app.conf:106`：`smtp_host="${MINDOC_SMTP_HOST||smtp.163.com}""` → 去掉尾部多余引号
   - `configs/app.conf:110`：`smtp_port="${MINDOC_SMTP_PORT||25}""` → 同上
   - 同步修 `configs/app.conf.example` 对应两行（如果也有）
5. 改硬编码 4 处：
   - `conf/enumerate.go:75` `ConfigurationFile = "./conf/app.conf"` → `"./configs/app.conf"`
   - `conf/enumerate.go:391` `filepath.Abs("./conf/app.conf")` → `filepath.Abs("./configs/app.conf")`
   - `commands/install.go:106` `i18n.SetMessage(lang, "conf/lang/"+lang+".ini")` → `"configs/lang/"+lang+".ini"`
   - `commands/command.go:280`（同上 i18n 硬编码）
6. 部署脚本同步（**上线前必做**，见 §六）：
   - `Dockerfile`、`start.sh`、`sync_host.sh`、`docker-compose.yml`：所有 `./conf/` → `./configs/`
   - `commands/daemon/daemon.go` 若引用配置路径，同步

**验收：**

- `go build ./...` 通过
- 本地 `./doc` 启动能读到配置，能收到"发送测试邮件"（不再因引号 bug 报解析错误）
- Docker 构建 + 容器启动能读到 `/app/configs/app.conf`

**回滚：** 单 revert 即可（`git mv` 是 rename，diff 干净）。

---

### T2 · `ioutil` 全局替换 `os.ReadFile` / `os.WriteFile`（0.5 天）

**受影响 12 文件：**（`Grep ioutil\. --type go` 已定位）

- `controllers/BaseController.go`
- `controllers/ManagerController.go`
- `converter/converter.go`
- `converter/util.go`
- `mail/smtp.go`
- `models/BookModel.go`
- `models/BookResult.go`
- `models/Member.go`
- `utils/dingtalk/dingtalk.go`
- `utils/filetil/filetil.go`
- `utils/wkhtmltopdf/wkhtmltopdf.go`
- `utils/ziptil/ziptil.go`

**替换规则：**

| 旧 | 新 |
|---|---|
| `ioutil.ReadFile(x)` | `os.ReadFile(x)` |
| `ioutil.WriteFile(x, b, perm)` | `os.WriteFile(x, b, perm)` |
| `ioutil.ReadDir(x)` | `os.ReadDir(x)`（**注意返回类型从 `[]os.FileInfo` 变成 `[]os.DirEntry`**） |
| `ioutil.ReadAll(r)` | `io.ReadAll(r)` |
| `ioutil.NopCloser(r)` | `io.NopCloser(r)` |
| `ioutil.Discard` | `io.Discard` |
| `ioutil.TempFile(dir, pat)` | `os.CreateTemp(dir, pat)` |
| `ioutil.TempDir(dir, pat)` | `os.MkdirTemp(dir, pat)` |

**注意：** `ioutil.ReadDir` 用了返回 `os.FileInfo` 的地方，改 `os.ReadDir` 后要 `entry.Info()` 才能拿到 `FileInfo`。必须逐处 review。

**批量脚本（PowerShell 参考，实际用 StrReplace 精确改）：**

```powershell
# 只做扫描确认，实际改动走 IDE 逐文件替换
rg 'ioutil\.' --type go -l
```

**验收：** `go build ./...` + `grep -R 'io/ioutil'` 无残留 import。

---

### T3 · `interface{}` → `any`（0.5 天）

**范围：** 全仓 Go 文件。**排除**：`vendor/`（无）、`docs/`。

**做法（推荐 IDE 全局查找替换）：** 只替换 `interface{}` 这个字符序列，不动 `type XXX interface { ... }` 声明。

**注意：** 有些老的 `type M map[string]interface{}` 别名要保留兼容性，可以先都换成 `any`，只要 `go build` 过就行；`beego/orm` 的 API 返回类型是 `interface{}`，调用侧改成 `any` 不影响。

**验收：** `go build ./...` 通过；`rg 'interface\{\}' --type go` 只剩 `type ... interface{...}` 声明形式。

---

### T4 · `main.go` 引入 `spf13/cobra`（1~1.5 天）

**当前问题：** `main.go:21-29` 手写 `os.Args[1] == "service"` 分派，且 `commands.RegisterCommand()` 里塞了 `install` / `update` 等（beego CLI 风格），命令行体验割裂；Round 3 的 `doc mcp` 子命令必须靠 cobra 才能优雅落地。

**新命令表：**

| 命令 | 保留/新增 | 说明 |
|---|---|---|
| `doc`（无参数） | 保留 | 启动 web 服务（等价 `doc web`） |
| `doc web` | 新增 | 显式启动 web 服务 |
| `doc install` | 保留 | 数据库/初始账号初始化（对齐现有 `commands/install.go`） |
| `doc version` | 新增 | 打印版本号 |
| `doc service install / remove / restart` | 保留（兼容） | 系统服务安装（`kardianos/service`）— **旧脚本继续可用** |
| `doc mcp`（占位） | 新增（Round 1 只加空 stub） | Round 3 实现，本轮只注册命令、返回 "not implemented yet" |

**步骤：**

1. `go get github.com/spf13/cobra@latest`
2. 新增 `commands/root.go`（cobra root），把现有 `RegisterCommand()` 逻辑迁进各子命令：
   ```go
   // commands/root.go（示意）
   var rootCmd = &cobra.Command{
       Use:   "doc",
       Short: "doc — Documentation & knowledge base server",
       Run:   runWeb,  // 无子命令时默认跑 web
   }
   ```
3. 新增 `commands/web.go`、`commands/version.go`、`commands/mcp_stub.go`
4. `commands/install.go` 用 cobra 包一层
5. `main.go` 简化为：
   ```go
   func main() {
       if err := commands.Execute(); err != nil {
           os.Exit(1)
       }
   }
   ```
6. **兼容层**：`main.go` 里保留对 `os.Args[1] == "service"` 的 pre-check，直接调 `daemon.Install/Uninstall/Restart`，绕过 cobra（这套子命令属于系统 service 管理，走 cobra 反而绕）。
7. `README.md` 更新命令使用说明。

**验收：**

- `./doc --help` 能打印子命令列表
- `./doc install` 行为等价旧版
- `./doc service install` / `remove` / `restart` 行为等价旧版
- `./doc mcp` 打印 `mcp command is planned for Round 3, not implemented yet`

**回滚：** 单 revert；`kardianos/service` 未动。

---

### T5 · `cache.Cache` 抽接口 + `context` 传递（1~1.5 天）

**目标：** 让 `cache` 包对外暴露 `Cache` 接口，`context.Context` 全流程传递，为 Round 4 上 singleflight/metrics/分层缓存铺路。**本轮不换 gob、不换 redis 库**。

**新增/改动：**

```
cache/
├─ cache.go          # 现有：保留旧函数签名做 shim（Round 2/3 内逐步废弃）
├─ cache_null.go     # 现有
├─ iface.go          # 【新增】Cache 接口定义
├─ beego_adapter.go  # 【新增】BeegoCache（包 beego/v2/client/cache）
└─ null.go           # 【新增】NullCache（新接口实现）
```

**接口定义：**

```go
// cache/iface.go
package cache

import (
    "context"
    "time"
)

type Cache interface {
    Get(ctx context.Context, key string, dst any) error
    Set(ctx context.Context, key string, val any, ttl time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    IsExist(ctx context.Context, key string) (bool, error)
    Incr(ctx context.Context, key string) (int64, error)
    Clear(ctx context.Context) error
}
```

**兼容策略：**

- 保留 `cache.Get(key, e)` / `cache.Put(key, v, t)` 等**包级函数**做 shim，内部调 `defaultCache.Get(context.Background(), ...)`。
- 新代码（T6 的 BaseController options 缓存、Round 3 的 MCP）直接用 `Cache` 接口 + 显式 ctx。
- Round 2/3/4 逐步把老 callers 从 shim 迁到接口。

**注意：** 现有 `cache.Get` 依赖 `gob` 反序列化 + `nilctx = context.TODO()`，本轮保持不变，只是**新增**一层接口 + 新实现，不删旧行为。

**验收：** `go build ./...` 通过；旧 caller 行为不变；新接口能被 T6 使用。

---

### T6 · `BaseController.Prepare` options 缓存（0.5 天）

**当前问题（`controllers/BaseController.go:68`）：** 每次请求都执行 `models.NewOption().All()` 全表读，DB 压力大。

**方案：**

- 首次启动时装载 + 5 分钟内存缓存（`sync.Map` 或 `atomic.Value` 存 `map[string]string`）。
- 后台 `ManagerController` 修改 option 时**主动 invalidate**（在 option save 完成后调 `options.Reload()`）。

**实现位置：** 新增 `controllers/options_cache.go`（或 `internal/cache/options.go` 也可，但为减小改动放 controllers 层）。

```go
// controllers/options_cache.go（示意）
var optionsCache atomic.Pointer[map[string]string]
var optionsExpiresAt atomic.Int64

func loadOptions() map[string]string {
    if p := optionsCache.Load(); p != nil && time.Now().Unix() < optionsExpiresAt.Load() {
        return *p
    }
    // reload from DB, atomic.Store...
}

func InvalidateOptions() { optionsExpiresAt.Store(0) }
```

**Prepare 里改用 `loadOptions()`**，ManagerController 里保存后调 `InvalidateOptions()`。

**验收：**

- 单元冒烟：连续访问 10 次首页，DB 日志里 `SELECT * FROM md_options` 只出现 1 次
- 后台改 `ENABLE_ANONYMOUS` 开关后，最长 5 分钟内自动生效；主动 invalidate 后**立即**生效
- 顶部 §一 已注明"上线感知：后台切换开关最长 5 分钟生效"— 请在 CHANGELOG 明说

---

### T7 · `internal/errs/` + `BizError` + `JsonError(err)` helper（0.5 天）

**目标：** 给 Round 3 的 MCP 工具统一错误返回、给 Round 2 目录迁移时"顺带收敛" controller 错误返回打基础。

**新增文件：**

- `internal/errs/biz.go`
- `internal/errs/codes.go`
- （不动 controller 现有 `JsonResult`，只**新增** `JsonError` 供新代码使用）

**代码骨架：**

```go
// internal/errs/biz.go
package errs

import "errors"

type BizError struct {
    Code int
    Msg  string
    // 用于日志的原始 err，不返回给客户端
    Cause error
}

func (e *BizError) Error() string { return e.Msg }
func (e *BizError) Unwrap() error { return e.Cause }

func New(code int, msg string) *BizError                { return &BizError{Code: code, Msg: msg} }
func Wrap(code int, msg string, cause error) *BizError  { return &BizError{Code: code, Msg: msg, Cause: cause} }

func AsBiz(err error) (*BizError, bool) {
    var b *BizError
    if errors.As(err, &b) { return b, true }
    return nil, false
}
```

```go
// internal/errs/codes.go —— Round 3 MCP 使用；Round 1 先占位
const (
    CodeUnknown        = 6000
    CodeInternal       = 6001
    CodeInvalidParam   = 6002
    CodeUnauthorized   = 6003
    CodeForbidden      = 6004
    CodeNotFound       = 6005
    CodeVersionConflict = 6100 // Round 3 MCP 乐观锁
    CodeRateLimited    = 6200 // Round 3 MCP 限流
    CodeConfirmRequired = 6300 // Round 3 MCP delete 保护
)
```

**controllers/BaseController.go 加 JsonError：**

```go
// controllers/BaseController.go
func (c *BaseController) JsonError(err error) {
    if b, ok := errs.AsBiz(err); ok {
        c.JsonResult(b.Code, b.Msg)
        return
    }
    logs.Error("unhandled error:", err)
    c.JsonResult(errs.CodeInternal, "系统内部错误")
}
```

**验收：** `go build ./...` 通过；至少在 1 个现有 controller（推荐 `SearchController`，代码短）**示例性**改用 `c.JsonError(errs.New(...))`，其余保持原 `JsonResult` 不动（Round 2 目录迁移时批量收敛）。

---

## 五、PR 拆分与合入顺序

| # | PR 名 | 内容 | 大小估计 | 上线感知 |
|---|---|---|---|---|
| 1 | `refactor(round1): move configs and fix smtp quotes` | T1 | 小（rename + 6 处文本改） | 部署脚本同步 |
| 2 | `refactor(round1): replace ioutil with os/io` | T2 | 中（12 文件） | 无 |
| 3 | `refactor(round1): interface{} → any` | T3 | 中（触碰面广、内容浅） | 无 |
| 4 | `feat(round1): introduce cobra CLI (with legacy shim)` | T4 | 中 | 兼容 |
| 5 | `refactor(round1): cache interface with context` | T5 | 小 | 无 |
| 6 | `perf(round1): cache options in BaseController.Prepare` | T6 | 小 | 后台开关最长 5 分钟生效 |
| 7 | `feat(round1): internal/errs + BizError + JsonError helper` | T7 | 小 | 无 |

**建议合入顺序：** 1 → 2 → 3 → 5 → 7 → 6 → 4。
理由：把风险最低的先合，T4（cobra）触碰启动流程放最后，出问题好定位。

---

## 六、上线检查清单

### 部署脚本必改（配合 T1）

- [ ] `Dockerfile`：`COPY conf /app/conf` → `COPY configs /app/configs`
- [ ] `docker-compose.yml`：volume 挂载 `./conf:/app/conf` → `./configs:/app/configs`
- [ ] `start.sh` / `sync_host.sh`：所有 `./conf/` → `./configs/`
- [ ] `doc.service` / spug 部署脚本（若引用配置路径）
- [ ] `README.md`：更新配置文件位置说明
- [ ] 现网升级 note：**先把 `conf/app.conf` 备份并搬到 `configs/`，再上线新版本**；老版本仍从 `./conf/app.conf` 读，回滚安全

### 回归清单

- [ ] `go build ./...` + `go vet ./...` 无告警
- [ ] `./doc install` 走一遍数据库初始化
- [ ] `./doc`（无参）启动 web 服务，首页 200
- [ ] `./doc service install` + Windows/Linux 系统服务能启停
- [ ] 登录、创建 book、创建 document、上传附件走一遍冒烟
- [ ] 后台修改"启用匿名访问"开关，5 分钟内首页效果生效
- [ ] Docker 镜像构建成功、容器启动读到 `/app/configs/app.conf`
- [ ] Log 无 ioutil deprecated 警告（`go vet` + `staticcheck` 若装了）

### 灰度建议

- 无 breaking change，可直接全量升级
- 部署时**先备份 `conf/` 目录**，若 T1 部署脚本漏改导致启动失败可秒回滚

---

## 七、追踪表（合入后勾选，对应 refactor-roadmap.md §七）

| 编号 | 任务 | PR | Commit | 上线日期 |
|---|---|---|---|---|
| T1 | `configs/` 独立 + smtp 引号修复 | | | |
| T2 | `ioutil` → `os` | | | |
| T3 | `interface{}` → `any` | | | |
| T4 | `main.go` cobra | | | |
| T5 | `cache.Cache` 接口 | | | |
| T6 | `BaseController.Prepare` 缓存 | | | |
| T7 | `internal/errs/` + `BizError` + `JsonError` | | | |

---

## 八、Round 2 前置产物核对（本轮结束时应满足）

- ✅ `configs/` 目录存在，`conf/` 只剩 `enumerate.go` + `mail.go`（Go 源码）
- ✅ `internal/errs/` 已存在（后续 Round 3 MCP 直接 import）
- ✅ `cache.Cache` 接口已存在（Round 3 MCP HTTP 模式的 token 缓存直接用）
- ✅ `spf13/cobra` 已引入（Round 3 `doc mcp` / `doc mcp --http` 子命令直接注册）
- ✅ `main.go` 已经是"cobra 派发" + 保留 `service` 兼容分支
- ✅ 无 `io/ioutil` import
- ✅ 无裸 `interface{}`（除 interface 声明）

如以上任一项未满足，Round 2 起手会踩坑，**请务必回补**。

---

## 九、参考

- [refactor-roadmap.md §五 Round 1](./refactor-roadmap.md#🥇-round-1低风险重构--后续轮次前置准备1-周)
- [refactor-roadmap.md §六 关键风险清单](./refactor-roadmap.md#六关键风险清单)（尤其 #13 gob/session 兼容性 — Round 2 才处理，本轮不涉及）
- [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) — 附录 A 有 `ViewsPath`/`StaticDir`/字体路径的详细定位（Round 2 主用，本轮只 T1 的配置路径已够）
