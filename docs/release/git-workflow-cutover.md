# Git 协作约定 · 切换执行文档

> **状态：** 🚧 执行中（2026-08-17）  
> **长期规则：** [git-workflow.md](./git-workflow.md)（切完后日常以那篇为准）  
> **适用范围：** 仓库 `git.itopcms.com/astrueus/doc`。一人、两台电脑、两个 Gitea 账号。  
> **本文职责：** 按已拍板的选择，把现状一次切到「主干 `master` + 短命 `feat/*` + tag 发版」。切完可当历史记录留着，不必当日常手册。

进度：步骤 A 已完成（PR [#1](https://git.itopcms.com/astrueus/doc/pulls/1) squash 合入 `master` @ `7317f1e`）。步骤 B 随 `chore/release-push-tag` 合入。步骤 C 待做。

---

## 目录

1. [已拍板的选择](#一已拍板的选择)
2. [切之前的仓库快照](#二切之前的仓库快照)
3. [谁做什么](#三谁做什么)
4. [执行顺序](#四执行顺序)
5. [步骤 A：合入 `v2.2.1` → `master`](#五步骤-a合入-v221--master)
6. [步骤 B：改发版脚本与文档](#六步骤-b改发版脚本与文档)
7. [步骤 C：版本号分支改名为 `archive/*`](#七步骤-c版本号分支改名为-archive)
8. [步骤 D：Gitea 保护 `master`](#八步骤-dgitea-保护-master)
9. [步骤 E：另一台电脑跟上](#九步骤-e另一台电脑跟上)
10. [切完后的日常（摘要）](#十切完后的日常摘要)
11. [明确不做](#十一明确不做)
12. [完成标准](#十二完成标准)

---

## 一、已拍板的选择

| 事项 | 决定 |
|------|------|
| 主干 | 继续 `master`，不改名 `main` |
| 旧 `v1.0.0`～`v2.2.1` **开发分支** | **先复制成 `archive/2.2.1` 这种名字，确认远程已有后再删旧名**。不直接删。 |
| 不用 `release/2.2.1` | `release/X.Y` 留给「主线已超前、旧线还要补丁」。现在这些是冻结快照，用 `archive/`。 |
| 分支名 | `archive/2.2.1`（**不带**前缀 `v`），避免和 tag `v2.2.1` 搅在一起 |
| tag `v2.2.1` | **不** `tag -f`、不挪到 `b7e7317`。下一正式版打新 tag。 |
| 推 tag | 脚本改为 `git push origin refs/tags/vX.Y.Z`；本地已有同名 tag 则失败退出 |
| 日常推分支 | `git push origin HEAD` 即可，不必写 `refs/heads/` |
| 评审 | 自己开 PR、自己合；Gitea **不要**开「至少 1 个评审」 |
| 强制 CI 绿 | 先不开 |
| 本次是否发 `2.2.2` 包 | **否**。切流程与发新包分开，要发再说 |
| `joker`、旧 `feat/*` | **不在本次必做**。合入仍走 PR；不要当第二主干 |

`archive/*` 只读考古：不要再往上面推功能。需要某次提交时 `cherry-pick` 到 `master`。

---

## 二、切之前的仓库快照

记录于 **2026-08-17**，执行前用第 4 节命令再核对一次（电脑 A 可能又多了提交）。

| 引用 | 指向（短哈希） | 含义 |
|------|----------------|------|
| 分支 `master` | `b7e7317` | 可发版主干；更接近当时打包装的代码 |
| 分支 `v2.2.1` | `32e17da` | **当前开发线**；比 `master` 多文档整理与协作约定 |
| tag `v2.2.1` | `fc5a716` | 更早的提交；与分支同名，导致 `git push origin v2.2.1` 歧义 |
| 分支 `v2.2.0` | `4cae5cf` | 历史开发分支（相对 `master` 无独有提交） |
| 分支 `v2.1.0` | `dd91e44` | 同上 |
| 分支 `v2.0.1` | `c0eda5d` | 同上 |
| 分支 `v2.0.0` | `d76b0f3` | 同上 |
| 分支 `v1.0.0` | `e55dd27` | 与 `joker` 同一提交 |
| `joker` | `e55dd27` | 个人分支，本次不改名、不删 |

`v2.2.1` 相对 `master` 多出的提交（必须先合进 `master`，再改名）：

```text
7716432 chore(release): example 中不再预填 GITEA_OWNER
84e8213 docs: 按熟悉前缀整理文档目录并补充协作约定
32e17da docs(release): 补充小团队与大团队 Git 协作约定草案
```

远程版本号分支 → 目标 archive 名：

| 现名（删除前） | 改名为 | 备注 |
|----------------|--------|------|
| `v2.2.1` | `archive/2.2.1` | 先合 `master` 再改名 |
| `v2.2.0` | `archive/2.2.0` | |
| `v2.1.0` | `archive/2.1.0` | |
| `v2.0.1` | `archive/2.0.1` | |
| `v2.0.0` | `archive/2.0.0` | |
| `v1.0.0` | `archive/1.0.0` | 不要叫 `archive/v1.0.0` |

Gitea：`https://git.itopcms.com/astrueus/doc`  
开 PR 快捷入口：`https://git.itopcms.com/astrueus/doc/pulls/new/v2.2.1`（目标选 `master`）

---

## 三、谁做什么

先定死哪台是 **电脑 A（发版 / 切流程）**、哪台是 **电脑 B（另一账号）**。下文「本机」默认电脑 A。

| 角色 | 做什么 |
|------|--------|
| 电脑 A | 合 `master`、改脚本、推 `archive/*`、删远程旧 `v*` 分支名 |
| 电脑 B | 切的过程中 **不要提交、不要往 `v2.2.1` push**。等 A 说 `master` 已更新，再 `fetch` + 切 `master`（步骤 E） |
| Gitea 网页 | 合 PR（若没用 API）、保护 `master`（步骤 D）。Agent **改不了**保护规则 |

两账号都需要能推功能分支。发版 Token 仍只放各机 `scripts/.env.release`，不进 Git。

---

## 四、执行顺序

不要跳步。尤其：**先合 `master`，再 `archive/`，最后才删 `v2.2.1` 这个名字。**

```text
A 合入 v2.2.1 → master
  → B 跟上 master（可在改脚本前先跟上一次）
  → A 改发版脚本并合进 master
  → 两台再 pull master
  → A 把 v* 复制为 archive/* ，确认远端存在
  → A 删除远程 v* 开发分支（不删 tag）
  → 两台 git fetch --prune
  → Gitea 保护 master（可与删分支对调：若怕合 PR 被拦住，保护放最后）
```

建议在电脑 A 用清单往下勾。每步完成后把本文「状态」或下方勾选改掉（执行当时改即可）。

---

## 五、步骤 A：合入 `v2.2.1` → `master`

**目的：** 文档和协作约定进入主干；之后不再把 `v2.2.1` 当开发线。

### A.1 电脑 A 核对

```bat
git fetch origin
git status
git log --oneline origin/master..origin/v2.2.1
```

工作区应干净。`master..v2.2.1` 应仍是上面那几次（若又多了文档提交，一并合进去即可）。

引用务必写全，避免和 tag 歧义：

```bat
git rev-parse origin/v2.2.1
git rev-parse refs/tags/v2.2.1
```

### A.2 开 PR 并合入

优先 Gitea PR：`v2.2.1` → `master`。

- 标题示例：`docs: 文档目录整理与 Git 协作约定`
- 人少可自审后 Merge。建议 squash（`master` 上一条中文说明）或普通 merge，**不要**对已推送的 `master` rebase。

若网页不便、且 `master` **尚未**禁止直推，等价于本机：

```bat
git checkout master
git pull origin master
git merge origin/v2.2.1
git push origin master
```

有保护、直推失败时，必须走 PR。

### A.3 电脑 A 切到主干

```bat
git fetch origin
git checkout master
git pull origin master
git branch -d v2.2.1
```

本地删不掉（提示未合并）时：先看 `git log master..v2.2.1`，有独有提交再处理，不要 `-D`。

此后电脑 A **不要**再 `checkout v2.2.1` 写新代码。远程 `v2.2.1` 先留着，等步骤 C。

---

## 六、步骤 B：改发版脚本与文档

**目的：** 推 tag 不再用裸 `v2.2.1`，避免再和分支撞名。从 **已更新的 `master`** 拉分支，不要在旧版本分支上改。

```bat
git checkout master
git pull origin master
git checkout -b chore/release-push-tag
```

| 文件 | 改法 |
|------|------|
| `scripts/release.ps1` | `git push origin $Tag` → `git push origin "refs/tags/$Tag"` |
| `scripts/release.sh` | `git push origin "$TAG"` → `git push origin "refs/tags/$TAG"` |
| 上述两文件「tag 已存在则 skip」 | **退出码非 0**，打印已有 tag 指向哪次 commit；不要 skip 再 push |
| `docs/release/release-local.md` | 流程图里 `git push origin vX.Y.Z` 改为 `refs/tags/vX.Y.Z` |
| `docs/release/release-gitea-actions.md` | 同样的裸 push 改为全名 |
| `docs/README.md` | 「分支 `v2.2.1`」改为跟 `master` / 最新 tag，避免以后又按版本分支干活 |
| [git-workflow.md](./git-workflow.md) | 修订记录补一行：脚本已按本文改；「尚未改发版脚本」划掉 |

推送并 PR 合进 `master`。合完两台都 `git checkout master && git pull origin master`。

脚本改完前：不要跑会 push 裸版本号的正式发版；核对打包可用 `scripts\release.bat <版本> --dry-run`。

手工补推 tag（仅当需要）只允许：

```bat
git push origin refs/tags/v2.2.1
```

---

## 七、步骤 C：版本号分支改名为 `archive/*`

**目的：** 保留指针，去掉与 tag 同名的 `refs/heads/vX.Y.Z`。

### C.1 改名前再确认没有「只在旧分支上的提交」

对每一个即将删掉的旧名：

```bat
git fetch origin
git log --oneline origin/master..origin/v2.2.0
```

把 `v2.2.0` 换成表里每一个。输出为空：只是旧指针，可以改名。  
有输出：先看要不要 cherry-pick 进 `master`（`v2.2.1` 应已在步骤 A 合完，这里应为空）。

### C.2 先推新名（旧名还在）

每个版本执行一对「复制」即可（不是改 commit、不是 force tag）：

```bat
git push origin refs/heads/v2.2.1:refs/heads/archive/2.2.1
git push origin refs/heads/v2.2.0:refs/heads/archive/2.2.0
git push origin refs/heads/v2.1.0:refs/heads/archive/2.1.0
git push origin refs/heads/v2.0.1:refs/heads/archive/2.0.1
git push origin refs/heads/v2.0.0:refs/heads/archive/2.0.0
git push origin refs/heads/v1.0.0:refs/heads/archive/1.0.0
```

### C.3 确认远程已有 `archive/*`

```bat
git ls-remote --heads origin "refs/heads/archive/*"
```

应看到 `archive/1.0.0` … `archive/2.2.1`。没有齐就不要删旧名。

电脑 B 也可先 `git fetch origin`，能看到 `origin/archive/2.2.1` 再往下。

### C.4 再删远程旧名（不删 tag）

```bat
git push origin --delete v2.2.1
git push origin --delete v2.2.0
git push origin --delete v2.1.0
git push origin --delete v2.0.1
git push origin --delete v2.0.0
git push origin --delete v1.0.0
```

**禁止：** `git push origin --delete refs/tags/v2.2.1` 或任何删 tag。

### C.5 两台电脑清理跟踪引用

```bat
git fetch --prune
git remote prune origin
```

本地还占着 `v2.2.1` 时：

```bat
git checkout master
git branch -d v2.2.1
```

若仍想留本地别名（一般不必）：

```bat
git branch -m v2.2.1 archive/2.2.1
```

### C.6 改名后 `checkout v2.2.1` 的含义

- `git checkout v2.2.1` → 进入 **tag**（当前是 `fc5a716`，detached HEAD），不再是原来的开发分支。
- 要看当时开发线上的文档提交 → `git checkout archive/2.2.1`。
- 日常开发 → `master` 上拉 `feat/...`。

---

## 八、步骤 D：Gitea 保护 `master`

用有 Admin 的账号改一次，两台共享同一套规则。

| 设置 | 选择 |
|------|------|
| 默认分支 | `master` |
| 保护 `master` | **禁止直接 push**，只能 PR |
| 合入所需评审数 | **0**（不要「至少 1 个」） |
| 允许自己合自己的 PR | **要** |
| PR 必须 CI 绿 | **先关** |
| 推 tag / 建 Release | 两账号都能即可；习惯上只在电脑 A 发版 |

若步骤 A 还没合完、又怕保护拦住直推：把本步骤放到 A、B、C 之后。

---

## 九、步骤 E：另一台电脑跟上

在电脑 A 完成步骤 A（至少 `master` 已包含 `v2.2.1` 上的文档）之后执行。步骤 C 删旧名之后再执行一次 `fetch --prune`。

```bat
git fetch origin
git checkout master
git pull origin master
git branch -d v2.2.1
git fetch --prune
```

B 上若有未推送改动：先 `git stash` 或提交到 `feat/...` / `docs/...` 再 push，**不要**在过时的 `v2.2.1` 上继续写。

`git log master..v2.2.1` 若还有独有提交：cherry-pick 到新功能分支开 PR，不要把旧分支硬推回去。

---

## 十、切完后的日常（摘要）

完整说明见 [git-workflow.md](./git-workflow.md)。两台电脑：

```bat
git checkout master
git pull origin master
git checkout -b feat/短名
git push -u origin HEAD
```

换电脑接着干：先 `git fetch`，再 `checkout` 同一条 `feat/短名` 并 `pull`。同一分支两台轮流写时先 pull 再改。

合入：Gitea 上 `feat/短名` → `master`。然后两台 `checkout master && pull`，删本地/远程功能分支。

下一正式版：只在电脑 A、工作区干净、当前分支 `master`、步骤 B 已完成后：

```bat
scripts\release.bat 2.2.2
```

版本号：仅文档/脚本用 PATCH `2.2.2`；Round 5 用户可见能力再用 MINOR `2.3.0`。发版前把 `CHANGELOG.md` 收成 `## 2.2.2`（标题不带 `v`）。

---

## 十一、明确不做

- 不 `git tag -f v2.2.1`，不把 tag 挪到文档提交上。
- 不把冻结分支命名为 `release/2.2.1`。
- 不开长期 `develop` / GitFlow。
- 不为「像两个人」强制另一账号 Approve。
- 不在服务器 `git pull` + 现场编译。
- 不提交 `scripts/.env.release`、`conf/app.conf`、Token。
- 本次不跑正式发版脚本出新包（除非另说）。
- 不删 `refs/tags/v2.2.1` 或 `v0.0.1-test`。

---

## 十二、完成标准

- [ ] 两台电脑当前分支都是 `master`，且包含原 `v2.2.1` 上应保留的提交
- [ ] 发版脚本推 tag 使用 `refs/tags/...`；已存在同名 tag 会失败退出
- [ ] 远程存在 `archive/1.0.0` … `archive/2.2.1`
- [ ] 远程 **没有** `refs/heads/v1.0.0` … `v2.2.1`（tag 仍在）
- [ ] `git push origin v2.2.1` 不再报 `matches more than one`（只可能碰到 tag；脚本仍写全名）
- [ ] Gitea：`master` 禁止直推；不强制评审
- [ ] 新工作从 `master` 拉 `feat/*`（或 `fix/` `docs/` `chore/`），不再开 `v2.2.2`

---

## 十三、修订记录

| 日期 | 说明 |
|------|------|
| 2026-08-17 | 初稿。旧版本开发分支改为 `archive/X.Y.Z` 后再删旧名；不挪 tag；一人两机两账号。 |
