# Round 5 · T15 · `scripts/` 是否迁入 `deployments/` — 细化方案

> 对应 [round-5-execution-plan.md §十三丁 T15](./round-5-execution-plan.md#十三丁t15--scripts-是否迁入-deployments0.5~1-天)。  
> **定位：** 评估 + 搬迁；牵动 T7（测试脚本）与文档路径。  
> **状态：** ✅ 已拍板并落地：方案 A（全迁）；根快捷入口用 **Makefile / justfile**。

---

## 一、现状盘点

### 1.1 迁前 `scripts/` 内容（现已在 `deployments/scripts/`）

| 文件 | 用途 | 谁在调用 |
|---|---|---|
| [`build.sh`](../../deployments/scripts/build.sh) / [`build.bat`](../../deployments/scripts/build.bat) | 多平台编译（Zig / MinGW） | 开发者本地；`release.*` 内部调用 |
| [`release.sh`](../../deployments/scripts/release.sh) / [`release.bat`](../../deployments/scripts/release.bat) / [`release.ps1`](../../deployments/scripts/release.ps1) | 一键发版（编译 → 打包 → tag → Gitea Release） | 开发者本地 |
| [`.env.release.example`](../../deployments/scripts/.env.release.example) | 发版环境变量模板 | `release.*` |
| [`lib/json.sh`](../../deployments/scripts/lib/json.sh) / [`lib/json_test.sh`](../../deployments/scripts/lib/json_test.sh) | 纯 bash JSON 解析 + 单测 | `release.sh` |
| [`README.md`](../../deployments/scripts/README.md) | 使用说明 | 人 |
| `test.sh` / `test.ps1` | 白名单测试 + 覆盖率门槛（T7，迁后新增） | `make test` / CI |

### 1.2 `deployments/` 现有内容

| 文件 | 用途 |
|---|---|
| [`Dockerfile`](../../deployments/Dockerfile) / [`docker-compose.yml`](../../deployments/docker-compose.yml) | 镜像与本地栈 |
| [`start.sh`](../../deployments/start.sh) | 容器入口 |
| [`sync_host.sh`](../../deployments/sync_host.sh) | docker ↔ host 同步 |
| [`systemd/doc.service`](../../deployments/systemd/doc.service) | systemd unit |
| [`spug/spug_run.sh`](../../deployments/spug/spug_run.sh) | Spug 后置脚本 |

### 1.3 关键硬编码路径

- [x] `deployments/spug/spug_run.sh:11` — `REPO/scripts/` **仅作服务器侧运维习惯目录**，与仓库内 `scripts/` 无耦合  
- [x] `deployments/spug/spug_run.sh:55` — 从发布包 `deployments/spug/` 拷贝，与仓库 `scripts/` 无耦合  
- [x] 迁后**删除**根 `scripts/`（不留目录、不留 README）  
- [x] `docs/release/release-local.md` / `docs/deploy-spug-*.md` / `README.md` — 批量替换为 `make`/`just` 或权威路径  
- [x] `release.*` 内部相对路径（`lib/`、调用 `build.*`）— 随迁改  
- [x] Gitea Actions：`.gitea/workflows/test.yml` 调用 `deployments/scripts/test.sh`；发版文档示例已改路径  

**结论：** 仓库内没有生产运行时依赖根 `scripts/` 的硬编码；搬迁主要是**文档 + 脚本自引用**。

---

## 二、评估问题（回顾）

1. 构建脚本与部署脚本是否同属「运维/发布」树？→ **方案 A：是，统一进 `deployments/`**  
2. 测试脚本落点？→ **`deployments/scripts/test.*`**  
3. 路径变长 / 跨平台日常命令？→ **根目录铺 `Makefile` / `justfile` 做快捷入口**（见 §4.1）；**不做**根 `scripts/` 转发封装  
4. 开发者 vs 服务器？→ 权威实现在 `deployments/scripts/`；人机入口在仓库根的 make/just  

---

## 三、候选方案（归档）

### 方案 A · 全迁：`scripts/` → `deployments/scripts/` ✅ 已选

目标树：

```
/                           # 仓库根
├── Makefile                # Linux/macOS（及装了 make 的环境）快捷入口
├── justfile                # 跨平台首选（含 Windows）；可与 Makefile 并存
└── deployments/
    ├── scripts/            # 权威实现（原根 scripts/ 全部迁入）
    │   ├── README.md
    │   ├── build.sh
    │   ├── build.bat
    │   ├── release.sh
    │   ├── release.bat
    │   ├── release.ps1
    │   ├── .env.release.example
    │   ├── test.sh
    │   ├── test.ps1
    │   └── lib/
    ├── Dockerfile
    ├── docker-compose.yml
    ├── start.sh
    ├── sync_host.sh
    ├── systemd/
    └── spug/
```

- 根 **`scripts/` 迁完即删**，不留过渡目录 / README  
- **优点：** 单一运维树；日常用 `make test` / `just release 1.0.0`；Windows/Linux 可分开或统一用 just  
- **缺点：** 需安装 `make` 和/或 `just`；文档与脚本内路径需批量改；搬迁 PR 有路径噪音  

### 方案 B · 维持 / 方案 C · 折中

（未选）

---

## 四、实施要点（方案 A）

1. `git mv scripts/* deployments/scripts/`（保留 `lib/`）；修正 `release.sh` 等内部相对路径。  
2. **仓库根新增 `Makefile` 与/或 `justfile`**（§4.1）。  
3. **删除根 `scripts/` 目录**（不留过渡 README）。用法说明写在 `deployments/scripts/README.md` + 根 README / AGENTS 的 make/just 示例。  
4. 更新文档：[`docs/release/release-local.md`](../release/release-local.md)、[`docs/deploy-spug/deploy-spug-local.md`](../deploy-spug/deploy-spug-local.md)、根 [`README.md`](../../README.md)、[`AGENTS.md`](../../AGENTS.md) — 日常命令改为 `make`/`just`。  
5. T7：权威 `deployments/scripts/test.sh`；根侧 `make test` / `just test`。  
6. T14：`doc storage migrate` 仍为 Go 子命令；外壳若有则进 `deployments/scripts/`，并可挂 `make storage-migrate`。  
7. 冒烟：权威脚本 + `make`/`just` 入口均可跑 build / release dry-run / test。

### 4.1 根目录快捷入口：`Makefile` / `justfile`（建议：要）

**结论：用根目录任务文件替代根 `scripts/` 薄封装。**

| 原则 | 说明 |
|---|---|
| 权威实现只在一处 | 逻辑、`lib/`、`.env.release.example` 只在 `deployments/scripts/` |
| 根入口零业务 | Make/Just 只负责调用对应 `.sh` / `.bat` / `.ps1`，不复制逻辑、不改默认参数 |
| 覆盖高频命令 | 至少：`build`、`release`、`test`（T7）；已补 `run`（开发 `go run`）；按需加 `help` |
| CI | **直接调** `deployments/scripts/...`；也可 `make test`（需保证 runner 有 make） |
| 根 `scripts/` | **迁完即删**；仓库中不再保留该目录 |

#### 平台策略（可分开）

| 文件 | 主要面向 | 说明 |
|---|---|---|
| **`justfile`** | **跨平台首选**（Windows / Linux / macOS） | [`just`](https://github.com/casey/just) 对 Windows 友好；配方里按 OS 选 `release.ps1` / `release.sh` |
| **`Makefile`** | Linux / macOS / CI（GNU Make） | Windows 原生 cmd 不友好；可用 Git Bash / WSL / 自行安装 make，**不强制** Windows 用户依赖 Make |
| 两者并存 | 推荐 | 配方保持同名同义（`build` / `release` / `test` / `run`）；Just 与 Make 都只调权威脚本 |

#### 目标命令（示例）

```text
make help | just --list
make build | just build
make test  | just test
make run   | just run
make release VERSION=1.0.0
just release 1.0.0
```

`justfile` 示意（按 OS 分支，可分开维护 Windows 配方）：

```just
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

build:
    {{ if os() == "windows" { "deployments/scripts/build.bat" } else { "bash deployments/scripts/build.sh" } }}

test:
    {{ if os() == "windows" { "deployments/scripts/test.ps1" } else { "bash deployments/scripts/test.sh" } }}

run *args:
    {{ if os() == "windows" { "deployments/scripts/run.ps1" } else { "bash deployments/scripts/run.sh" } }} {{ args }}

release version:
    {{ if os() == "windows" { "deployments/scripts/release.ps1" } else { "bash deployments/scripts/release.sh" } }} {{version}}
```

`Makefile` 示意（偏 Unix）：

```makefile
.PHONY: build test run release help
build:
	bash deployments/scripts/build.sh
test:
	bash deployments/scripts/test.sh
run:
	bash deployments/scripts/run.sh $(ARGS)
release:
	bash deployments/scripts/release.sh "$(VERSION)"
help:
	@echo "build | test | run | release VERSION=x.y.z"
```

**明确不做：** 根 `scripts/` 任何残留（含 README / 转发脚本）；避免 Make/Just 与第二套脚本入口并存。

### 与其他任务的联动

| 任务 | 影响 |
|---|---|
| **T7** 测试工程化 | 权威：`deployments/scripts/test.sh`；本地：`make test` / `just test`；CI 调权威路径 |
| **开发启动** | 权威：`deployments/scripts/run.sh` / `run.ps1`；本地：`make run` / `just run`（`go run` + `--dir` 仓库根；不加热重载） |
| **T14** 对象存储 | `doc storage migrate`；可选 `make`/`just` 挂名 |
| 未来 CI | `bash deployments/scripts/test.sh`（或 `make test`） |

---

## 五、验收

- [x] 结论写入本文件「决策」段  
- [x] `git mv` 完成；`release.*` / `build.*` 内部路径已修  
- [x] 根目录 `Makefile` 与 `justfile` 可驱动 build/release/test，且与权威脚本行为一致  
- [x] Linux（及 Windows）入口已写清；文档写清依赖（`make` / `just`）  
- [x] 根 **`scripts/` 目录已删除**（仓库中不存在）  
- [x] 路径引用（README、release/spug 文档、AGENTS）已更新为 make/just 或 `deployments/scripts/`  
- [x] [round-5-execution-plan.md §十四追踪表](./round-5-execution-plan.md#十四追踪表) T15 状态更新  

---

## 六、决策

| 项 | 结论 |
|---|---|
| 方案 | ✅ **A · 全迁**（`scripts/` → `deployments/scripts/`） |
| T7 测试脚本落点 | 权威：`deployments/scripts/test.sh` / `test.ps1` |
| T14 迁移工具落点 | `doc storage migrate`（Go 子命令）；外壳若有则 `deployments/scripts/` |
| 根快捷入口 | ✅ **`Makefile` + `justfile`**（可分开服务 Unix / 跨平台；Just 覆盖 Windows） |
| 根 `scripts/` | ❌ **迁完即删**；无过渡期目录 / README |
| CI | 直接调用 `deployments/scripts/...` |
| 决策日期 | 2026-08-04 |
