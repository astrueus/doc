# Git 分支、版本与多人协作约定（草案）

> **状态：** 📝 草案（2026-08-17）。发版脚本已改为 `refs/tags/`；一次性切换 A～C 已完成，待保护 `master`（D）与电脑 B 跟上（E），见 [git-workflow-cutover.md](./git-workflow-cutover.md)。  
> **适用范围：** 仓库 `git.itopcms.com/astrueus/doc`，Gitea + 本机 `deployments/scripts/release.*` + Spug 部署。  
> **相关：** [release-local.md](./release-local.md)（怎么跑脚本）、[release-gitea-actions.md](./release-gitea-actions.md)（CI 发版）、[../deploy-spug/](../deploy-spug/)（怎么部署）、[git-workflow-cutover.md](./git-workflow-cutover.md)（一次性切换）、根目录 [AGENTS.md](../../AGENTS.md)（中文与决策确认）。

日常以本文为准。切换进度与残留分支见 [git-workflow-cutover.md](./git-workflow-cutover.md)。「与现状差异」只保留尚未落地的项。

---

## 目录

1. [为什么要写这份](#一为什么要写这份)
2. [原则](#二原则)
3. [角色与 Gitea 权限](#三角色与-gitea-权限)
4. [版本号](#四版本号)
5. [Tag](#五tag)
6. [分支命名](#六分支命名)
7. [推荐的分支模型](#七推荐的分支模型)
8. [日常开发（从拉分支到合入）](#八日常开发从拉分支到合入)
9. [提交说明](#九提交说明)
10. [Pull Request 与评审](#十pull-request-与评审)
11. [发版](#十一发版)
12. [热修](#十二热修)
13. [CHANGELOG](#十三changelog)
14. [环境、密钥与生产](#十四环境密钥与生产)
15. [上游 MinDoc](#十五上游-mindoc)
16. [明确不要做的](#十六明确不要做的)
17. [与现状差异（过渡清单）](#十七与现状差异过渡清单)
18. [命令速查](#十八命令速查)

---

## 一、为什么要写这份

发版脚本执行 `git push origin v2.2.1` 时，Git 无法区分 **分支** `refs/heads/v2.2.1` 和 **tag** `refs/tags/v2.2.1`，报 `src refspec v2.2.1 matches more than one`。编译、打包、Gitea Release、上传附件可以成功，只有 **推 tag** 失败。

根因不是 Gitea，而是：**版本号既当分支名、又当 tag 名**。本仓库历史上还有 `v1.0.0`、`v2.0.0`、`v2.1.0`、`v2.2.0`、`v2.2.1` 等同名开发分支，以后每次按版本号发版都会再撞一次。

这份文档给出多人协作下的全量约定：分支、版本、tag、PR、发版、热修、密钥。目标是小团队能执行，不上完整 GitFlow。

---

## 二、原则

1. **`master` 保持可发版**（随时能打 patch）。未完成、不能编译的代码不进 `master`。
2. **版本号只出现在 tag（以及包名、CHANGELOG、Release 标题）上**，禁止再用 `v2.2.1` 当分支名。
3. **进 `master` 走 Pull Request**。禁止日常直接 push `master`（紧急热修见第十二节，事后补 PR）。
4. **一个版本只打一次 tag，默认不移动、不 `--force`。** 发错就升 PATCH 再打。
5. **密钥不进 Git。** `deployments/scripts/.env.release`、Gitea Token、生产 `conf/app.conf` 不提交。
6. **文档、注释、提交说明用简体中文**（见 [AGENTS.md](../../AGENTS.md)）。
7. **生产只部署「打了 tag 的发布包」**，不在服务器上 `git pull` + 现场编译。

触及 HTTP / MCP 契约、鉴权、Session、数据迁移、生产配置时，先按 [AGENTS.md「决策确认」](../../AGENTS.md#决策确认) 确认再改。

---

## 三、角色与 Gitea 权限

| 角色 | 人数建议 | 职责 |
|------|----------|------|
| 维护者 | 1～2 人 | 保护 `master`、合 PR、打 tag、跑发版脚本、管 Token、生产发布 |
| 开发 | 其余写代码的人 | 从 `master` 拉功能分支、开 PR、互相看代码 |
| 只读 | 只需看代码 / 下包的人 | clone、下 Release，无写权限 |

Gitea 仓库建议（可按习惯改严或改松，但建议至少做到前两条）：

| 设置 | 建议 | 说明 |
|------|------|------|
| `master` 禁止直接 push | **要** | 只能通过 PR 合入 |
| 合入前至少 1 个评审 | 人少时维护者自审也可以 | 流程在，避免「顺手 push」 |
| 推 tag / 建 Release | 仅维护者 | 避免两个人同一天打两个 `v2.3.0` |
| PR 必须 CI 绿 | 有 Runner 再开 | 目前以本机测试为主也可 |

发版用的 Gitea Token：

- 权限尽量只覆盖 **写 Release / 写仓库**（按 Gitea 实际勾选）。
- 优先用「发版机器人」账号，避免绑定某个人、人走了就不能发。
- 放在每人（或维护者）本机 `deployments/scripts/.env.release`，已 gitignore。

---

## 四、版本号

采用 [SemVer](https://semver.org/lang/zh-CN/)：`MAJOR.MINOR.PATCH`。

**口头、CHANGELOG、发版脚本参数、安装包文件名：都不带 `v`。**  
例：脚本 `scripts\release.bat 2.3.0`，包名 `doc_2.3.0_linux_amd64.tar.gz`。

| 怎么升 | 何时 | 本仓库例子 |
|--------|------|------------|
| **MAJOR** `2.x` → `3.0.0` | 无法平滑升级的不兼容 | 环境变量 `MINDOC_*` → `DOC_*`；Session 序列化格式变化 |
| **MINOR** `2.2` → `2.3.0` | 兼容的新能力 | 对象存储、新 MCP 工具、Vite |
| **PATCH** `2.2.1` → `2.2.2` | 兼容修复、文档、脚本 | 发版脚本修推送、文档目录整理 |

预发布（可选）：`2.3.0-rc.1`、`2.3.0-beta.1`。对应 tag：`v2.3.0-rc.1`。  
RC 可以打多次（`rc.1`、`rc.2`）；正式 `2.3.0` 只打一次。

**谁决定版本号：** 发版的维护者在发版当次确定，不要每人在功能分支里改「当前版本」打架。二进制里的版本由发版脚本 `--version=` / 构建参数注入。

内部排期（Round 5、T14）**不是**版本号，只写在 `docs/round-5/` 和提交说明的范围里，例如 `docs(round5):`。

---

## 五、Tag

Tag 表示：**这一版发出去的源码钉在哪一次 commit**。

| 规则 | 正确 | 错误 |
|------|------|------|
| 正式版 | `v2.2.1` | `2.2.1`（缺 v）、`V2.2.1`、`release-2.2.1` |
| 预发布 | `v2.3.0-rc.1` | `v2.3.0_rc1` |
| 对准哪次提交 | 打包装用的那次 | 事后又提交的文档整理 |
| 生命周期 | 打一次，一直留着 | 当常规手段 `git tag -f` |

推送 **必须写全名**（这是避免与分支撞车的硬规则）：

```bat
git push origin refs/tags/v2.2.1
```

等价：`git push origin tag v2.2.1`。

**禁止：** `git push origin v2.2.1`（名字可能同时匹配分支和 tag）。

含义：

- `git checkout v2.2.1` → 当时发版的源码。
- Gitea Release 附件 → 用这次源码编出来的包。
- 二者应对齐。对不齐时，优先相信 **包**（用户装的是包）；再决定要不要补救 tag（改历史，需团队知情）。

发版脚本若发现 **本地已有同名 tag**：应 **失败退出**，由人判断是「已经发过」还是「本机脏 tag」，不要静默 skip 再去 push。

---

## 六、分支命名

### 6.1 主干

| 名字 | 用途 |
|------|------|
| `master` | **唯一长期分支**。可发版的代码都往这里合。发版在 `master`（或从其拉出的短命 `release/X.Y`）上打 tag |

是否改名为 `main`：可以，但是一次性迁移（默认分支、文档、Spug、脚本）。不改也完全可以，本仓库继续 `master` 即可。

### 6.2 功能 / 修复（短命）

前缀表示 **这次分支主要改什么**，和提交类型对齐。选一个最能概括「合进去之后别人最先感知到的变化」的前缀即可，不要为每个文件类型再开一条分支。

```text
feat/<简述>        feat/oauth2-provider
fix/<简述>         fix/session-login
docs/<简述>        docs/git-workflow
chore/<简述>       chore/release-push-tag
refactor/<简述>    refactor/cache-facade
test/<简述>        test/mcp-ratelimit
ci/<简述>          ci/gitea-release
```

| 前缀 | 使用范围（做这些事用它） | 不要用它的情况 | 例子 |
|------|--------------------------|----------------|------|
| **feat/** | **用户或调用方能感知的新能力**：新页面/接口/MCP 工具、新配置项、新子命令、新存储后端等。Round 任务里「新增能力」也走这条。 | 只是修已有功能的 bug；只改文档；只搬文件不改行为。 | `feat/oauth2-provider`、`feat/r5-object-storage` |
| **fix/** | **修缺陷**：行为不对、报错、安全漏洞、兼容性回归。含「按现有需求本该如此但写错了」。 | 新功能顺带修无关 bug（应拆开，或仍以新功能为主用 feat）；「其实是重写一坨」更像 refactor。 | `fix/session-login`、`fix/mcp-append-version` |
| **docs/** | **只动说明**：`docs/**`、根 README、CHANGELOG 润色、注释补全且无行为变化。 | 文档是新功能的一部分（跟着 feat 走）；改发版脚本本身用 chore/ci。 | `docs/git-workflow`、`docs/round5-index` |
| **chore/** | **工程杂务、不改产品行为**：gitignore、依赖小升级、发版脚本路径、目录搬家、示例配置校对、与功能无关的格式化。 | 改 CI 工作流（用 ci）；大范围重构（用 refactor）；用户能感到的变化（用 feat/fix）。 | `chore/release-push-tag`、`chore/docs-layout` |
| **refactor/** | **重构：对外行为保持不变**（或仅内部结构变化）：拆文件、分层、重命名、抽函数。可附带测试，但仍以结构为主。 | 重构时改了接口语义或修了用户可见 bug——按「别人最先感知」改用 feat 或 fix，或拆成两个 PR。 | `refactor/cache-facade`、`refactor/book-model-split` |
| **test/** | **只加/改测试**：单测、集成测、测试脚本夹具；生产代码基本不动。 | 为新功能写测试（跟着 feat）；修测试是因为修了 bug（跟着 fix）。 | `test/mcp-ratelimit`、`test/pkg-gob` |
| **ci/** | **持续集成 / 流水线**：Gitea Actions、workflow、PR 门禁、CI 用的构建矩阵。 | 本机 `deployments/scripts/release.*` 小改（chore 即可）；只有 CI 里顺带改了一行文档仍用 ci。 | `ci/gitea-release`、`ci/pr-go-test` |

拿不准时：

1. **先看用户/调用方有没有新能力或少了一个坑** → `feat` / `fix`。  
2. 都没有，再看是不是 **纯文档 / 纯测试 / 纯 CI**。  
3. 剩下的工程事 → `chore`；大段挪代码且行为不变 → `refactor`。  
4. 一条分支里又有 feat 又有 docs：前缀跟 **主目的**（一般是 feat），文档当附属，不必再开 `docs/` 分支。

其它规则：

- 小写、短横线、尽量短。
- **不要带版本号**（不要 `feat/v2.3.0-vite`）。
- 轮次可写在简述里：`feat/r5-object-storage`。Round 只是排期，不是前缀。
- 功能类统一 `feat/`，不要混用 `feature/`。
- 合进 `master` 后 **不要立刻删**功能分支；按 §8.4 大约保留最近 3 条。
- 个人临时草稿可以用任意本地名，但 **推远程、开 PR 前** 改成上表格式。
- 热修生产用 `hotfix/`（见 6.4），不要用 `fix/` 从旧 tag 上直接发版。`fix/` 是合进 `master` 的常规修复。

### 6.3 维护线（可选，默认不用）

仅当「`master` 已经在开发 2.3，但线上 2.2 还要补丁」时：

```text
release/2.2
```

从要冻结的 `master`（或 `v2.2.1` 那个 tag）拉出，只合 bugfix，打 `v2.2.2`、`v2.2.3`。  
**分支叫 `release/2.2`，tag 叫 `v2.2.2`，不要同名。**

### 6.4 热修

```text
hotfix/session-clear
```

从 `master` 或 `release/X.Y` 拉出，合回去之后打新的 PATCH tag。不要在旧 tag 上直接 commit。

### 6.5 禁止

| 禁止 | 原因 |
|------|------|
| 分支名 `v2.2.1`、`v2.3.0` | 与 tag 撞名，`git push origin v2.2.1` 歧义 |
| 长期个人名分支当集成分支（如一直往 `joker` 合） | 别人不知道该跟哪条线；合入必须走 `feat/*` + PR |
| 一个功能分支活几个月 | 反复冲突；拆小 PR |

---

## 七、推荐的分支模型

小团队默认：**GitHub Flow 变体**（没有长期 `develop`）。

```text
feat/a ──PR──┐
feat/b ──PR──┼──► master ──(发版)── tag vX.Y.Z ──► Gitea Release ──► Spug
fix/c  ──PR──┘
```

只有「主线已超前、旧线还要补丁」才加 `release/X.Y`。

不推荐一上来就上完整 GitFlow（`develop` + 一堆长期 `release/*` + `hotfix/*` 仪式），和当前「发版频率不高、本机脚本发版」不匹配。

---

## 八、日常开发（从拉分支到合入）

### 8.1 开工

```bat
git checkout master
git pull origin master
git checkout -b feat/r5-vite
```

每天开工先 `git pull` `master`，减少后面冲突。

### 8.2 开发

- 小步提交，推 **自己的功能分支**（`git push -u origin HEAD`）。
- 不要直接往 `master` 推。
- 本地未要求时，Agent 也不应擅自 `commit` / `push`（见决策确认）。

### 8.3 合入前

- 再 `git fetch` 并把 `master` rebase 或 merge 进自己的分支，解决冲突。
- 跑通你改动相关的测试（至少 `go test` 能跑的包）。
- 用户可见变化写进 `CHANGELOG.md` 的 `Unreleased`（或在 PR 里写，由维护者代填）。

### 8.4 合入后（保留约 3 条功能分支）

用户确认合入 `master` 之后：

1. `git checkout master && git pull`（或快进合入后拉最新）。
2. **不要立刻删除**刚合入的功能分支（本地、远程都先留着，方便对照与回查）。
3. 按合入时间大约保留最近 **3** 条**已合入** `master` 的功能分支（`feat/` / `fix/` / `chore/` / `ci/` 等；不含 `master`、`release/*`、`hotfix/*`）。
4. 将出现第 4 条时：列出最旧的一条（本地 + `origin`），**确认后再删**；未合入的、或用户点名要留的，不要当可删项。

```bat
:: 合入后先切回 master，不删刚合入的分支
git checkout master
git pull origin master

:: 查看已合入 master、仍留着的功能分支（按提交时间，新的在上）
git branch --merged master --sort=-committerdate
git branch -r --merged origin/master --sort=-committerdate
```

超过 3 条、用户确认删除最旧的一条时（示例名请换成实际分支）：

```bat
git branch -d feat/最旧短名
git push origin --delete feat/最旧短名
```

Agent 未获「合入 master」的明确指示时，不得自行合主线或删分支。权威摘要见仓库根 [AGENTS.md](../../AGENTS.md)「功能分支保留」。

---

## 九、提交说明

与 [AGENTS.md](../../AGENTS.md) 一致，并带范围，方便以后摘 CHANGELOG。

```text
<类型>(<范围>): <一句话说明为什么 / 做什么>
```

类型：`feat` | `fix` | `docs` | `chore` | `refactor` | `test` | `ci`。

```text
# 好
fix(session): 修复只存 member_id 后编辑页误跳登录
docs(round5): 按熟悉前缀整理文档目录
chore(release): 推 tag 改为 refs/tags，避免与分支同名

# 不好
update
fix
fix(round4): polish zap console UX
```

一个提交只做一类事。发版脚本、业务功能、无关重构不要捆在一起。

功能分支上可以有多个 commit。合进 `master` 时：

- **默认 squash**（Gitea「压缩合并」）：`master` 上一条清晰说明。
- PR 本身已经拆成可读的多个 commit、希望保留：用 rebase 合并。
- 人少、PR 极小：merge commit 也可以，团队选一种当默认，不要混用三种。

---

## 十、Pull Request 与评审

**每个进 `master` 的改动都要有 PR**（含文档、脚本、AGENTS）。

### 10.1 标题与正文

标题：与提交说明同风格，中文。

建议正文：

```markdown
## 做了什么
- …

## 为什么
- …

## 怎么验
- [ ] 本地已跑：…
- [ ] 相关页面 / MCP 工具：…

## 风险
- 是否改对外契约 / 鉴权 / 迁移 / 生产配置（是则 @维护者）
```

### 10.2 谁看

| 改动 | 评审 |
|------|------|
| 普通 bug、文档、测试 | 任意一人（可交叉） |
| HTTP / MCP 契约、鉴权、Session、迁移、`conf` 语义 | **维护者必须看** |
| 发版脚本、Spug、Dockerfile | 维护者看 |

人少时「自己开 PR、自己点通过」可以，但必须过一遍「怎么验」和「风险」，不能当没开 PR。

### 10.3 冲突

在功能分支上解决，再推。禁止为了合入而对 `master` `push --force`。

---

## 十一、发版

操作人：维护者。工作区干净，当前分支为 **`master`**（或 `release/X.Y`），不要在名为 `vX.Y.Z` 的分支上发版。

### 11.1 步骤

1. 把 `CHANGELOG.md` 里对应内容从 `Unreleased` 收成该版本一节（可单独 PR 先合）。
2. `git checkout master && git pull`。
3. 确认 `git log -1` 就是要发出去的代码。
4. 跑测试（至少约定范围内的 `go test`）。
5. 本机发版（详见 [release-local.md](./release-local.md)）：

   ```bat
   scripts\release.bat 2.3.0
   ```

   脚本应做到：编译打包 → `git tag -a v2.3.0` → `git push origin refs/tags/v2.3.0` → Gitea 建 Release → 上传 `doc_2.3.0_*`。

6. Release 说明粘贴 CHANGELOG 该节；有破坏性操作（清 session、改环境变量）放最上面。
7. Spug 先吃预发（若有），再生产。只拉 **该 tag 对应 Release 的附件**。

`--dry-run`：只编包，不打 tag、不调 API，用来核对产物。

### 11.2 脚本必须遵守

`deployments/scripts/release.ps1` / `release.sh`：

1. `git push origin refs/tags/v$Version`（禁止裸 `git push origin vX.Y.Z`）
2. 本地已有同名 tag → **退出码非 0**，打印「已存在，指向 xxx」，不要 skip 再 push
3. 文档 [release-local.md](./release-local.md) 流程图与 [release-gitea-actions.md](./release-gitea-actions.md) 使用同一写法

### 11.3 tag 和事后文档提交

发版后若又合了「只改文档」的 commit（例如整理 `docs/`）：

- **不要**把已发布的 `v2.2.1` 挪到新 commit。
- 文档跟下一个 PATCH（如 `v2.2.2`）走，或接受「tag 比最新 master 少一两个文档 commit」。

---

## 十二、热修

生产出问题、必须出包：

```text
master ──► hotfix/session-clear ──PR──► master ──► 发版 v2.2.2
```

若 `master` 已经有不该进生产的 2.3 功能：

1. 从 tag `v2.2.1` 拉 `release/2.2`（若还没有）。
2. 在其上开 `hotfix/...`，合入 `release/2.2`。
3. 打 `v2.2.2` 并发包。
4. **把同一修复 cherry-pick 回 `master`**，避免下次正式版把洞带回去。

不要：`git checkout v2.2.1` 后直接改并 force tag。

---

## 十三、CHANGELOG

- 文件：仓库根 `CHANGELOG.md`。
- 顶部保持 **Unreleased**（可按 Round 分子节，与现在类似）。
- 发版时把该版本用户可见项收成 `## 2.3.0`（或 `## v2.3.0`，团队选一种，建议 **不带 v 与包名一致**，标题里写版本号即可）。
- **Breaking** 用警告块，写清操作（清 session、改 `DOC_*`、不能平滑升）。现有 Round 2/4/5 写法保持。
- 纯内部重构可以写短，但运维必须看的步骤不能埋在列表中间。

合 PR 的人最好自己改 Unreleased；漏了由维护者发版前补。

---

## 十四、环境、密钥与生产

| 环境 | 代码从哪来 | 配置 |
|------|------------|------|
| 开发 | 功能分支 / 本地 `master` | 本机 `conf/app.conf`，不提交 |
| 预发（建议有） | 待发的 `master` 或 `vX.Y.Z-rc.1` 包 | 独立库、独立 `DOC_*` |
| 生产 | **只部署已发布 tag 的 zip/tar.gz** | 服务器 / Spug 上的配置与密钥 |

禁止：生产机 `git pull master && go build`。

密钥：

| 文件 | 规则 |
|------|------|
| `deployments/scripts/.env.release` | gitignore；不要复制给无关的人 |
| `deployments/scripts/.env.release.example` | 无真实 Token |
| `conf/app.conf` | gitignore；example 进库 |
| `conf/app.conf.example` | 密钥只用 `${DOC_*}`，**不写明文默认值** |
| Gitea PAT | 到期轮换；离职撤销 |

工作目录、上传限制等仍以 `README.md` / `conf/app.conf.example` 为准。

---

## 十五、上游 MinDoc

跟进清单见 [../upstream-mindoc-checklist.md](../upstream-mindoc-checklist.md)。

约定：

- 用 `git remote add upstream https://github.com/mindoc-org/mindoc.git`（只需一次）。
- **按功能 cherry-pick**，不要整库 merge。
- 每次移植后改 import 路径与 CLI 文案（`mindoc` → `doc`）。
- 上游跟进也走 `feat/upstream-xxx` + PR，不要直接推进 `master`。
- 是否跟进行为变化（搜索、OAuth、编辑器）属于决策确认里的兼容性选择，先问再做。

---

## 十六、明确不要做的

- 用 `v2.2.1` 当开发分支，再在上面 `git push origin v2.2.1` 发版。
- 对 `master` 或已共享的 tag `push --force`（移动 tag 属于例外，须全员知情）。
- 生产 `git pull` + 现场编译。
- 一个 PR 混「业务功能 + 发版脚本重构 + 无关格式化」。
- tag 已存在时脚本静默 skip，再裸推同名引用。
- 把个人长期分支当「准 master」。
- 提交 `conf/app.conf`、`.env.release`、Token、本机 `doc.exe`。

---

## 十七、与现状差异（过渡清单）

电脑 A 的一次性切换（合入 `master`、改发版脚本、`v*` → `archive/*`）**已完成**。下面只列还没落地的差异。完整步骤与核对见 [git-workflow-cutover.md](./git-workflow-cutover.md)。

| 现状 | 本文约定 | 建议 |
|------|----------|------|
| `master` 是否禁止直推 | 应禁止，只能 PR | **待做**：Gitea 开保护（切换文档步骤 D） |
| 电脑 B 可能仍停在旧 `v2.2.1` | 两台都在 `master` 上拉 `feat/*` | **待做**：电脑 B `fetch` / 切 `master` / `prune`（步骤 E） |
| tag `v2.2.1` 仍指向打 tag 时的提交 | 一个版本只打一次 tag，默认不移动 | **不** `tag -f`；下一正式版打新 tag |
| CHANGELOG 用 Unreleased / Round N | 可保留 Round 作内部对照 | 发正式版时加 `## 2.x.x` 节更清晰 |
| 本机脚本发版、无强制 CI | 允许 | 有 Runner 再加 PR 检查 |

已落地（不必再做）：

- 开发不再用 `v2.2.1` 当集成分支；远程已无 `refs/heads/v1.0.0`…`v2.2.1`，改为只读 `archive/1.0.0`…`archive/2.2.1`。
- 发版脚本推 tag 使用 `refs/tags/`；本地已有同名 tag 则失败退出，不再 skip 再裸推。
- 电脑 A 当前在 `master`（`6563803`）。
- 已删远程 `chore/release-push-tag`、`docs/cutover-progress`、`joker`；不要给 `archive/*` 开 PR。个人草稿不要当第二主干。
- `feature/round-3-mcp` 已改名为 `feat/round3-mcp`。新分支统一 `feat/`，不要再用 `feature/`。

推荐落地顺序：

1. ~~改 `release.ps1` / `release.sh` / `release-local.md` 的 tag 推送写法~~（已做）。
2. ~~旧 `v*` 开发分支改为 `archive/X.Y.Z` 后删除旧名~~（已做）。
3. `master` 开保护（步骤 D，若还没有）。
4. 电脑 B 跟上 `master`（步骤 E）。
5. 下一功能用 `feat/...`，不再开 `v2.2.2` 分支。
6. 下一正式版在 `master` 上打 `v2.2.2` 或 `v2.3.0`。

---

## 十八、命令速查

```bat
:: 开工
git checkout master
git pull
git checkout -b feat/短名
git push -u origin HEAD

:: 开 PR：Gitea 上从 feat/短名 合入 master

:: 合入后（先留着刚合入的分支；约保留 3 条，见 §8.4）
git checkout master
git pull
:: 不要马上 git branch -d / push --delete
:: 超过 3 条时，确认后再删最旧的一条

:: 推当前功能分支（不要写裸版本号）
git push origin HEAD

:: 只推 tag（发版脚本应使用这一条）
git push origin refs/tags/v2.3.0

:: 只推分支（若必须写名字）
git push origin refs/heads/master

:: 预演发版（不打 tag、不调 API）
scripts\release.bat 2.3.0 --dry-run

:: 正式发版（改脚本之后）
scripts\release.bat 2.3.0
```

---

## 十九、修订记录

| 日期 | 说明 |
|------|------|
| 2026-08-17 | 初稿。由 `v2.2.1` 分支与 tag 同名导致推送失败引出。待对照习惯后修订；发版脚本尚未按本文修改。 |
| 2026-08-17 | 补充一次性切换步骤：[git-workflow-cutover.md](./git-workflow-cutover.md)；旧版本开发分支改为 `archive/X.Y.Z`。 |
| 2026-08-17 | 发版脚本推 tag 改为 `refs/tags/`；本地已有同名 tag 则失败退出。 |
| 2026-08-17 | 「与现状差异」改为只列未完成项：A～C 已切完；待保护 `master`、电脑 B 跟上。不挪 tag `v2.2.1`。已删合完的短命分支。 |
| 2026-08-17 | 已删个人分支 `joker`（无独有提交）。 |
| 2026-08-17 | `feature/round-3-mcp` 改名为 `feat/round3-mcp`。 |
