# Round 1 · 执行文档（低风险重构 + 后续轮次前置）

> 本文是 [refactor-roadmap.md §五 Round 1](./refactor-roadmap.md#🥇-round-1低风险重构--后续轮次前置准备1-周) 的**可执行分解**。
> 目标：在**不动包路径、不动目录结构**的前提下，一周内落地一批"内部重构 + 后续轮次前置"改动，为 Round 2/3 铺路。
> **约束：** 所有改动**对用户零感知**，不上任何新功能。
>
> **状态（2026-07-29）：✅ 已完成**（分支 `v2.1.0`，相对 `origin/v2.1.0` ahead 7；实施记录见 [§十](#十实施记录2026-07-29)）。

---

## 一、范围与不做清单

### 本轮做

| 序号 | 任务 | 目录/文件影响面 | 上线感知 |
|---|---|---|---|
| T1 | `configs/` 目录独立 + smtp 双引号 bug | 新增 `configs/`；`conf/` 保留 Go 源码 | 部署脚本需同步（见 §六） |
| T2 | `ioutil` → `os.ReadFile` / `os.WriteFile` | 12 个 Go 文件 | 无 |
| T3 | `interface{}` → `any` | 全仓 | 无 |
| T4 | `main.go` 引入 `spf13/cobra` | `main.go`、`commands/` | 命令行**兼容**旧写法（见 §四 T4） |
| T5 | `cache.Cache` 抽接口 + `context` + **msgpack** | `cache/`、`models/DocumentModel.go`、`models/Blog.go` | ⚠️ **需清 `cache/`**（序列化格式变了） |
| T6 | `BaseController.Prepare` options 缓存 | `controllers/BaseController.go` | 后台切换开关**最长 5 分钟**生效（见 §四 T6） |
| T7 | `internal/errs/` + `BizError` + `JsonError(err)` helper | 新增 `internal/errs/`；`controllers/BaseController.go` | SearchController.User 的 errcode 改为 600x 统一码 |

### 本轮**不做**（明确排除）

- ❌ **不搬 Go 源码目录**：`conf/enumerate.go` / `conf/mail.go` 保留在 `conf/` 包，Round 2 再迁 `internal/config/`。
- ❌ **不动 `import` 路径**：`git.itopcms.com/jackliu/doc/conf` 等 30+ import 保持不变。
- ❌ **不拆 `app.conf` 为多文件**：`[section]` 分组也留到 Round 2（配合强类型 Config struct 一起做）。
- ~~❌ **不换 gob 序列化**~~ → **实施时已变更决策**：T5 一并换成 `msgpack/v5`（见 [§十 偏差说明](#102-相对原文档的偏差与决策)）。`utils/gob.go`（cookie remember）与 `gob.Register(models.Member{})`（Beego session）**仍保留**。
- ❌ **不改 Controller/Model 内部逻辑**（除 T5 caller 迁 ctx、T6/T7 明确点位）。
- ❌ **前端 P0（editor.md / katex / mermaid）**：已在历史 PR 完成，不再列入。见 [refactor-roadmap.md §四](./refactor-roadmap.md#四支线前端与静态资源)。

---

## 二、前置条件

- Go 1.25.0（已满足，`go.mod` line 3）
- 分支：~~建议 `feature/round-1-refactor`~~ **实际在 `v2.1.0` 上直接完成**（7 个独立 commit，未新建 feature 分支）
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

| # | PR 名（计划） | 内容 | 大小估计 | 上线感知 |
|---|---|---|---|---|
| 1 | `refactor(round1): move configs and fix smtp quotes` | T1 | 小（rename + 6 处文本改） | 部署脚本同步 |
| 2 | `refactor(round1): replace ioutil with os/io` | T2 | 中（12 文件） | 无 |
| 3 | `refactor(round1): interface{} → any` | T3 | 中（触碰面广、内容浅） | 无 |
| 4 | `feat(round1): introduce cobra CLI (with legacy shim)` | T4 | 中 | 兼容 |
| 5 | `refactor(round1): cache interface with context` | T5 | 小→中（含 msgpack） | ⚠️ 清 `cache/` |
| 6 | `perf(round1): cache options in BaseController.Prepare` | T6 | 小 | 后台开关最长 5 分钟生效 |
| 7 | `feat(round1): internal/errs + BizError + JsonError helper` | T7 | 小 | SearchController errcode 变更 |

**建议合入顺序：** 1 → 2 → 3 → 5 → 7 → 6 → 4。
理由：把风险最低的先合，T4（cobra）触碰启动流程放最后，出问题好定位。

**实际执行：** 按上述顺序做了 7 个任务 commit，另追加 1 个 T1 session 回归补丁（共 8；未单拆 PR）；commit 说明见 [§七](#七追踪表合入后勾选对应-refactor-roadmapmd-七) / [§十](#十实施记录2026-07-29)。英文说明曾统一改写为中文（备份分支已删除）。

---

## 六、上线检查清单

### 部署脚本必改（配合 T1）

- [x] `Dockerfile`：构建阶段 `cp configs/app.conf.example configs/app.conf`（最终镜像仍靠工作目录内 configs）
- [ ] `docker-compose.yml`：本仓库示例**未挂载** conf 目录（仅 env）；若现网 compose 有 `./conf` 挂载需自行改为 `./configs`
- [x] `start.sh`：`SYNC_LIST` 中 `conf` → `configs`
- [x] `scripts/spug_run.sh`：`$WWW/configs/app.conf(.example)`
- [x] `README.md`：配置路径说明已更新
- [x] `.travis.yml` / `docs/release-gitea-actions.md` / `docs/deploy-spug-local.md`：路径同步
- [x] `.gitignore`：`/conf/app.conf` → `/configs/app.conf`
- [ ] 现网升级 note：**先把 `conf/app.conf` 备份并搬到 `configs/`，再上线新版本**（运维侧待执行）

### 回归清单

- [x] `go build ./...` 通过（实施过程中每步验证）
- [ ] `go vet ./...` 无告警（建议合入前再跑）
- [ ] `./doc install` 走一遍数据库初始化
- [ ] `./doc` / `./doc web` 启动 web 服务，首页 200
- [x] `./doc version` / `./doc mcp` / `./doc --help` 冒烟通过
- [ ] `./doc service install` + Windows/Linux 系统服务能启停
- [ ] 登录、创建 book、创建 document、上传附件走一遍冒烟
- [ ] 后台修改"启用匿名访问"开关，保存后**立即**生效（有 Invalidate）；未保存的其他路径最长 5 分钟
- [ ] Docker 镜像构建成功、容器能读到配置
- [x] 无 `io/ioutil` import（注释里残留一处 `ziptil` 可忽略）
- [ ] **升级后清空 `cache/`**（T5 msgpack 不兼容旧 gob 数据）

### 灰度建议

- T1 配置路径、T5 缓存序列化属于**有上线动作**的变更：先备份 `conf/` 与 `cache/`，升级后清缓存、用户侧无需重登（session 的 gob 未换）
- T6 最长 5 分钟滞后仅在**未走 Manager Setting 保存**时出现；走 Setting 会主动 invalidate

---

## 七、追踪表（合入后勾选，对应 refactor-roadmap.md §七）

| 编号 | 任务 | PR | Commit | 上线日期 |
|---|---|---|---|---|
| T1 | `configs/` 独立 + smtp 引号修复 | （直接 commit） | `5638121` | 待推送 / 待发布 |
| T2 | `ioutil` → `os` | （直接 commit） | `e17aed9` | 同上 |
| T3 | `interface{}` → `any` | （直接 commit） | `32fe27a` | 同上 |
| T4 | `main.go` cobra | （直接 commit） | `801441b` | 同上 |
| T5 | `cache.Cache` + msgpack + context | （直接 commit） | `ba785eb` | 同上 |
| T6 | `BaseController.Prepare` 缓存 | （直接 commit） | `e52564c` | 同上 |
| T7 | `internal/errs/` + `BizError` + `JsonError` | （直接 commit） | `dd65d39` | 同上 |
| T1-fix | 预加载 `configs/app.conf`，修复 session 未启用 | （直接 commit） | （见 §10.3.1 / 最近一次 fix） | 同上 |

**实际合入顺序（与建议一致）：** T1 → T2 → T3 → T5 → T7 → T6 → T4 → **T1-fix**。

---

## 八、Round 2 前置产物核对（本轮结束时应满足）

- ✅ `configs/` 目录存在，`conf/` 只剩 `enumerate.go` + `mail.go`（Go 源码）
- ✅ `internal/errs/` 已存在（后续 Round 3 MCP 直接 import）
- ✅ `cache.Cache` 接口已存在（Round 3 MCP HTTP 模式的 token 缓存直接用）；**序列化已是 msgpack**，Round 2 包路径变更后缓存在清过库的前提下更稳
- ✅ `spf13/cobra` 已引入（Round 3 `doc mcp` / `doc mcp --http` 子命令直接注册；当前 `doc mcp` 为 stub）
- ✅ `main.go` 已经是「cobra 派发」+ 保留 `service` 兼容分支（**未**把 service 收进 cobra）
- ✅ 无 `io/ioutil` import（正式代码）
- ✅ 无裸 `interface{}`（除 `type ... interface{...}` 声明）

如以上任一项未满足，Round 2 起手会踩坑，**请务必回补**。

---

## 九、参考

- [refactor-roadmap.md §五 Round 1](./refactor-roadmap.md#🥇-round-1低风险重构--后续轮次前置准备1-周)
- [refactor-roadmap.md §六 关键风险清单](./refactor-roadmap.md#六关键风险清单)（#13：session 仍用 gob + `gob.Register(Member)`；**缓存侧已在 Round 1 换 msgpack**）
- [frontend-backend-split-migration-plan.md](./frontend-backend-split-migration-plan.md) — 附录 A 有 `ViewsPath`/`StaticDir`/字体路径的详细定位（Round 2 主用，本轮只 T1 的配置路径已够）

---

## 十、实施记录（2026-07-29）

### 10.1 完成情况总览

| 项 | 结果 |
|---|---|
| 分支 | `v2.1.0`（未另开 `feature/round-1-refactor`） |
| 提交数 | 8（7 项任务 + 1 个 T1 session 回归补丁；说明已改写为中文） |
| 英文原稿备份 | ~~`backup/round1-english-msgs`~~ **已删除**（2026-07-29） |
| 构建 | 各任务完成后 `go build` 通过；`doc version` / `doc mcp` / `doc --help` 冒烟通过；补丁后首页 `GET /` → 302（登录）无 panic |
| 远端 | 相对 `origin/v2.1.0` **ahead 8**，尚未 push |

### 10.2 相对原文档的偏差与决策

| 点 | 原计划 | 实际 | 原因 / 影响 |
|---|---|---|---|
| 工作分支 | 新建 `feature/round-1-refactor` | 在 `v2.1.0` 直接提交 | 按实施时确认；注意 push 前与线上发布节奏对齐 |
| T5 序列化 | 保留 gob，只抽接口 | **换成 `github.com/vmihailenco/msgpack/v5`** | 提前消化 roadmap 风险 13 中「缓存绑定包路径」；**升级必须清空 `cache/`** |
| T5 caller | 包级函数可当 shim 少改 | 包级 API **签名加 `context.Context`**，改 `DocumentModel` / `Blog` 调用处 | 仍保留 `cache.Get/Put/Delete...` 包级函数，走 `cache.Default`；未做真正 DI 注入 Model |
| T5 `gob.Register` | （原计划 Round 1 不动） | 删 `Blog`/`Document`/`Template` 注册；**保留** `gob.Register(models.Member{})` | Member 仍给 Beego **session** 用；cookie remember 继续用 `utils/gob.go` |
| T4 service | 文档写 main.go pre-check | **按文档**：`main.go` 预检 `service install\|remove\|restart`，**不进 cobra** | 曾一度讨论收进 cobra，后纠正回文档方案 |
| T4 深度 | 薄封装即可 | **深度重构**：拆出 `root/web/version/install_cmd/password/migrate_cmd/mcp.go`；`password`/`migrate` 正式成子命令 | `commands` 不能 import `daemon`（循环依赖）→ 由 `main` 注入 `commands.StartWebServer` |
| T7 errcode | 示例改用 JsonError | `SearchController.User` 从 404/403/500 改为 **6002/6004/6005/6001** | 已确认前端不硬编码旧码；其余 controller 仍用 `JsonResult` |
| T1 硬编码范围 | 文档写 4 处 | 额外改了 `command.go` 的 `WorkingDir("conf",…)`、`spug_run.sh`、`.travis.yml`、README、运维文档 | 漏改会启动失败；规划类 md（roadmap 等）按确认**未改** |
| T6 TTL | 5 分钟 + Setting invalidate | 同计划；文件为 `controllers/options_cache.go` | Setting POST 成功后调 `InvalidateOptions()`，一般立即生效 |

### 10.3 各任务关键改动面（便于排查）

#### T1 · `5638121`

- 迁入：`configs/app.conf.example`、`configs/lang/*`（本地 `configs/app.conf` 被 gitignore，需现网手搬）
- Go：`conf/enumerate.go` 默认路径、`commands/command.go` / `install.go` 的 i18n 与 WorkingDir
- 脚本/CI：`Dockerfile`、`start.sh`、`scripts/spug_run.sh`、`.travis.yml`、`.gitignore`
- **副作用：** 见 [§10.3.1](#1031-t1-已知副作用与补丁session--nil-pointer)（已用 T1-fix 止血）

#### T2 · `e17aed9`

- 触及 11 个正式文件（`ziptil` 内仅注释残留 `ioutil`）
- `ManagerController.Config`：`ioutil.TempFile` → `os.CreateTemp`
- `BookModel`：`ReadDir` → `os.ReadDir`（`DirEntry` 仍有 `IsDir`/`Name`，调用处兼容）

#### T3 · `32fe27a`

- 21 个 Go 文件，约 48 处 `interface{}` → `any`
- 含 `cache_null.go` 等；随后 T5 又改了一轮 cache 签名

#### T5 · `ba785eb` ⚠️ 上线敏感

- 新增：`cache/iface.go`、`cache/beego_adapter.go`；重写 `cache/cache.go`
- 包级 API：`Get/Put/Delete/Incr/Decr/IsExist/ClearAll(ctx, …)`
- 依赖：`go.mod` 增加 `msgpack/v5`
- **升级必做：** `rm -rf cache/*`（或等价清 Redis/file 缓存）；不清会出现反序列化失败日志

#### T7 · `dd65d39`

- 新增：`internal/errs/biz.go`、`internal/errs/codes.go`（含 Round 3 占位码 `VERSION_CONFLICT` / `RATE_LIMITED` / `CONFIRM_REQUIRED`）
- `BaseController.JsonError`
- 示例：`SearchController.User`（并补上原缺失的 `return`，避免继续往下执行）

#### T6 · `e52564c`

- 新增：`controllers/options_cache.go`（`atomic.Pointer` + 5min TTL）
- `ManagerController.Setting` POST 后 `InvalidateOptions()`
- CHANGELOG/运维说明：若后台改 option 但未走 Setting，最长 5 分钟生效

#### T4 · `801441b`

- 新增 cobra 文件：`commands/root.go`、`web.go`、`version.go`、`install_cmd.go`、`password.go`、`migrate_cmd.go`、`mcp.go`
- 删除：`commands/update.go`（逻辑并入 `version`）
- `main.go`：`service` 预检 → `commands.Execute()`；`StartWebServer` 注入避免 import cycle
- 子命令：`web`（默认）、`install`、`version`、`password`、`migrate`、`mcp`（stub 文案固定）
- **不**在 `--help` 里出现 `service`（设计如此）

#### 10.3.1 T1 已知副作用与补丁（session → nil panic）

**现象（2026-07-29）：** 服务能起来，访问 `GET /` 时崩溃：

```
Handler crashed with error runtime error: invalid memory address or nil pointer dereference
.../beego/.../controller.go:727
.../controllers/BaseController.go:50   // c.GetSession(...)
.../controllers/HomeController.go:17
```

**根因时间线（Beego v2 路由级 session 早绑定）：**

1. beego 包 `init` 默认尝试加载 `./conf/app.conf`；T1 已迁到 `configs/app.conf`，加载失败 → `BConfig.WebConfig.Session.SessionOn` 保持默认 **`false`**
2. `_ "routers"` 的 `init` 里大量 `web.Router(...)`：每条路由创建时把**当时**的 `SessionOn` 固化到 `route.sessionOn`（beego `router.go` ≈525）→ 全部为 `false`
3. 之后 `ResolveCommand()` → `web.LoadAppConfig("ini", configs/app.conf)` 把全局 `SessionOn` 改回 `true`，但**已注册路由不会回填** `route.sessionOn`
4. 请求进入时按路由自己的 `sessionOn` 决定是否 `GlobalSessions.SessionStart`；为 `false` 则 `ctx.Input.CruSession` 不赋值
5. `Controller.GetSession` 里 `CruSession == nil` 时仍对 `CruSession.Get` 解引用 → **nil pointer**

**补丁：** 在 `conf/enumerate.go` 的 `init()` 末尾（依赖链 `routers → controllers → conf`，必定早于 `routers.init`）：

1. 若 `ConfigurationFile`（`configs/app.conf`）存在 → `web.LoadAppConfig("ini", ...)` 预热 Beego 配置
2. **保底** `web.BConfig.WebConfig.Session.SessionOn = true`（覆盖 `-dir` / 预加载失败场景，避免再次固化成 false；其余键仍由后续 `ResolveCommand` 正式 `LoadAppConfig` 覆盖）

**验证：** 补丁后 `GET /` → `302`（未登录跳登录页），无 panic。

**仍未消除的噪音：** stderr  
`init global config instance failed. ... open conf/app.conf`  
来自 beego `core/config/ini.go` 包级 `init` 硬编码 `InitGlobalInstance("ini", "conf/app.conf")`，应用侧无法关掉。  
**不要**靠「交换 `conf/` ↔ `configs/`」消除：`conf/` 是 Go 包（`enumerate.go`/`mail.go`），对换会撞包名并搞反语义。可选路径：Windows junction/`conf/app.conf` 兼容占位、或 Round 2 配置重构时再评估；正式配置根仍是 `configs/`。

**给 Round 2+ 的教训：**

- Beego v2：`web.Router` 注册前必须保证 `SessionOn`（及依赖其 init 次序的相关开关）已是终态；**仅在 `ResolveCommand` / `web.Run` 前加载配置不够**
- 目录迁移若离开 Beego 默认 `conf/app.conf`，要么在更早的 `init` 预加载，要么保留兼容文件/链接，否则会静默“全局已开 session、路由级没开”
- 路由拆分（Round 2）时勿在包 `init` 过早 `Router`，除非配置/session 已经就位；可考虑显式 `WithRouterSessionOn(true)` 或延后到 `main`/启动函数里注册

### 10.4 已知遗留 / Round 2 前建议

1. 合入前补跑：`go vet`、完整冒烟（install / 登录 / 上传 / service）
2. 发布说明中写明两处运维动作：**迁 configs**、**清 cache/**
3. session 仍 gob：Round 2 改包路径后仍要清 session 或做降级（roadmap 风险 13），与本次 cache msgpack 无关
4. `SearchController.User` errcode 变更若有外部脚本依赖旧数字，需联调
5. ~~删除备份分支（可选）：`git branch -D backup/round1-english-msgs`~~ **已删除**
6. beego stderr `open conf/app.conf`：见 §10.3.1，Round 2 配置重构前可忽略；勿交换 `conf`/`configs` 目录名
