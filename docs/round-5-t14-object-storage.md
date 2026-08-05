# Round 5 · T14 · 对象存储完全重构（S3 兼容 + 全量迁移）

> 对应 [round-5-execution-plan.md §十三丙 T14](./round-5-execution-plan.md#十三丙t14--对象存储上传--旧数据迁移1~2-周)。  
> **决策前提（2026-08-05）：**  
> 1. **不兼容**旧「直写本地磁盘」业务代码路径，**完全重构**上传 / 下载 / 删除 / URL 生成；  
> 2. **旧 `uploads/` 必须可全量迁移**到对象存储（含校验、断点、回写引用）；  
> 3. **必须**走 **S3 API**，尽量覆盖市面主流对象存储；  
> 4. 配置仅 `DOC_STORAGE_*`（见 [env 硬切](./round-5-env-mindoc-to-doc.md)）。  
> **状态：** ⏳ 待实施（建议独立 sprint）。

---

## 一、现状债（为何推倒重来）

| 问题 | 现状 | 后果 |
|---|---|---|
| 无存储抽象 | Controller 内 `filepath.Join(WorkingDirectory, "uploads", …)` + `os.Create` | 多实例落盘不一致；难测；难换后端 |
| URL 与物理路径耦合 | 库表 / Markdown 存 `/uploads/...` 或相对 WorkingDir 路径 | 迁云后链接易断 |
| 下载走本机文件 | `Output.Download(localPath)` | s3 模式必须改 |
| 无统一校验 | 大小/扩展名散落各 Upload | 策略难统一 |
| 无迁移工具 | 历史文件只在单机磁盘 | 无法平滑上云 |

**上传点清单（全部重接）：**

| 入口 | 现状路径习惯 |
|---|---|
| `DocumentController.Upload` | `uploads/{bookIdentify}/images|files/...` |
| `SettingController.Upload` | `uploads/{YYYYMM}/`（头像） |
| `BlogController.Upload` | `uploads/blog/{YYYYMM}/` |
| `BookController.UploadCover` | `uploads/{identify}/images/...` |
| Manager 附件清理 / 下载 | 直读本地 `FilePath` |

---

## 二、目标与原则（最佳实践）

| # | 原则 | 实践 |
|---|---|---|
| P1 | **单一抽象** | 业务只依赖 `internal/storage.BlobStore`；禁止再 `os.Create` 到 uploads |
| P2 | **S3 为唯一远程协议** | 不接各云专有 SDK 作为主路径；专有能力仅作可选增强 |
| P3 | **Key 即契约** | 对象 key 全局唯一、与 URL 可映射；迁移前后 key 稳定 |
| P4 | **读写分离策略** | 上传经应用；读优先 **预签名 / CDN**，反代仅 fallback |
| P5 | **全量可迁移** | 扫描 → 上传 → 校验 → 回写 DB/正文 → 报告；可重跑、可 dry-run |
| P6 | **本地仅开发默认** | `driver=local` 实现同一接口，生产推荐 `s3` |
| P7 | **安全默认** | 私有桶 + 签名 URL；密钥只走 env；服务端强制 ContentType / 大小 / 扩展名 |

---

## 三、市场主流对象存储与 S3 兼容性

下列产品均提供（或兼容）**S3 API**；本项目以「自定义 endpoint + path-style/virtual-host + AK/SK」接入即可：

| 产品 | S3 兼容 | 备注 |
|---|---|---|
| **Amazon S3** | 原生 | 金标准；regional endpoint |
| **MinIO** | 原生 / 极强 | 本地与私有化首选；常需 `force_path_style=true` |
| **阿里云 OSS** | S3 兼容模式 | 需开兼容或使用正确 endpoint；部分 ACL/头有差异 |
| **腾讯云 COS** | S3 兼容 | 兼容 endpoint 文档明确；签名细节偶有坑 |
| **华为云 OBS** | S3 兼容 | 兼容模式可用 |
| **七牛 Kodo** | S3 兼容 | 需 S3 网关 endpoint |
| **又拍云 / 其它** | 部分提供 S3 网关 | 以厂商文档为准做联调矩阵 |
| **Cloudflare R2** | S3 兼容 | 无 egress 费场景友好；endpoint 特殊 |
| **Backblaze B2** | S3 兼容 | 常用第三方备份 |
| **Ceph RGW / SeaweedFS / Garage** | S3 | 自建常见 |

**策略：** 核心只用 S3 子集（Put/Get/Head/Delete/List/Presign）；厂商差异收敛到配置（endpoint、region、path-style、checksum），用 **兼容性测试矩阵** 锁住，而不是为每家写驱动。

---

## 四、主流 Go SDK 深度对比

### 4.1 对照表

| SDK | 定位 | S3 兼容面 | 优点 | 缺点 | 推荐场景 |
|---|---|---|---|---|---|
| **[aws-sdk-go-v2/service/s3](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3)** | 官方 AWS S3 客户端 | 极广（标准 S3 + 兼容 endpoint） | 功能全、社区大、预签名/分片/校验成熟；长期维护 | 依赖体积大；MinIO/国产云需正确设 `BaseEndpoint` + `UsePathStyle`；API 偏「AWS 风格」 | **默认远程引擎（推荐）** |
| **[minio-go v7](https://github.com/minio/minio-go)** | MinIO 官方、S3 友好客户端 | 对 MinIO/S3 兼容存储很强 | 轻量、分片/并发传输体验好、预签名简单、文档亲民 | 个别「超 AWS 扩展」API 在真 AWS 上勿用；多云抽象弱于 gocloud | 强依赖 MinIO 私有化、要极简依赖时 |
| **[gocloud.dev/blob](https://gocloud.dev/howto/blob/)** | 可移植 Blob 抽象（S3/GCS/Azure…） | 经 s3blob 走 AWS SDK | 换云成本低；URL 打开桶；接口干净 | 多一层抽象；S3 高级特性暴露不全；排障多跳一层 | 明确要多云（含非 S3）时 |
| **厂商专有 SDK**（阿里 oss-go-sdk、腾讯 cos-go-sdk-v5 等） | 各云完整能力 | 非主路径 | 生命周期/图片处理/回调等强 | **破坏「只认 S3」**；多 SDK 维护成本爆炸 | **本轮禁止作主驱动**；仅可选插件 |
| **自行 HTTP 签 V4** | 裸协议 | 理论全 | 无依赖 | 安全与分片自研成本极高 | 否决 |

### 4.2 终裁（SDK）

| 项 | 结论 |
|---|---|
| **远程默认** | **`aws-sdk-go-v2` S3 client**（`BaseEndpoint` + `UsePathStyle` + static creds） |
| **local** | 自研文件系统驱动，实现同一 `BlobStore` |
| **不选** | 业务层直依赖 minio-go / 厂商 SDK |
| **可选** | 若团队更熟 MinIO 且生产全是 MinIO，允许 `s3` 驱动内部用 minio-go，但接口与配置面仍按 S3 语义暴露；**不推荐**双引擎并存 |
| **gocloud** | 不作为默认；若未来要 Azure/GCS 原生再评估「第二驱动」 |

### 4.3 S3 兼容接入最佳实践（配置层）

```text
必配：endpoint, region, bucket, access_key, secret_key
常配：force_path_style=true（MinIO / 部分国产）
      use_ssl=true
      prefix=（桶内环境隔离，如 prod/）
可选：cdn_base（公网/CDN 读）
      presign_expire=15m
      part_size / concurrency（大文件分片）
```

联调清单（每个目标云勾一次）：

- [ ] PutObject <5MB  
- [ ] 分片 Put > part_size  
- [ ] HeadObject（etag/size）  
- [ ] GetObject / 预签名 GET  
- [ ] DeleteObject  
- [ ] ListObjectsV2（迁移扫描桶侧）  
- [ ] 中文 / 空格 key（应避免；key 规范见下）  

---

## 五、目标架构（完全重构）

### 5.1 包结构

```text
internal/storage/
├── blob.go           # BlobStore 接口
├── key.go            # Key 生成 / 校验（禁止 Controllers 拼路径）
├── url.go            # PublicURL / Presign 策略
├── validate.go       # 扩展名、MIME、大小（承接原 [upload]）
├── local/
│   └── store.go
├── s3/
│   └── store.go      # aws-sdk-go-v2
├── migrate/
│   ├── scanner.go    # 扫本地 uploads
│   ├── uploader.go   # 并发上传 + 校验
│   ├── rewrite.go    # DB / Markdown 引用回写
│   └── report.go     # ok/fail 清单
└── storagetest/      # 内存 fake

internal/cli/storage_migrate.go   # doc storage migrate
```

### 5.2 接口（比旧稿更完整）

```go
type BlobStore interface {
    Put(ctx context.Context, key string, r io.Reader, size int64, opts PutOptions) (*ObjectInfo, error)
    Get(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)
    Head(ctx context.Context, key string) (*ObjectInfo, error)
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string, fn func(ObjectInfo) error) error

    // 读路径：优先预签名；CDN/公网可读时可用 PublicURL
    PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
    PresignPut(ctx context.Context, key string, expiry time.Duration, opts PutOptions) (string, error) // 可选：直传
    PublicURL(key string) string // cdn_base + key；未配 CDN 可返回空
}

type ObjectInfo struct {
    Key         string
    Size        int64
    ETag        string
    ContentType string
}
```

业务上传统一走应用内 `Put`（权限已在 Controller/service 校验）。**浏览器直传 PresignPut** 可作为二期，本轮不做也可。

### 5.3 Key 规范（新契约，迁移时对齐）

**逻辑 key（存库与 Markdown）：**

```text
uploads/{scope}/...
```

示例：

```text
uploads/books/{bookIdentify}/images/{uuid}{ext}
uploads/books/{bookIdentify}/files/{uuid}{ext}
uploads/avatars/{yyyyMM}/{uuid}{ext}
uploads/blog/{yyyyMM}/{uuid}{ext}
uploads/covers/{bookIdentify}/{uuid}{ext}
```

规则：

1. **禁止**用用户原始文件名做唯一段（只用 uuid + 安全 ext），避免覆盖与路径穿越。  
2. 库表 `attachment.file_path`、头像字段、封面字段：**统一存 key**（可带前导 `/` 或统一不带，二选一；推荐 **无前导斜杠的 key**，展示时再格式化）。  
3. Markdown 内链统一成 `/uploads/...` 或可配置的 `cdn_base + key`；迁移工具负责 rewrite。  
4. 桶内实际对象：`{prefix}{key}`（`DOC_STORAGE_PREFIX`）。

> 与「旧相对 WorkingDirectory 路径」在语义上兼容（仍以 `uploads/` 开头），但**生成逻辑全新**（uuid、分层），旧文件迁移时 **保持旧相对路径作为 key**，保证链接不断。

### 5.4 读路径策略

| 模式 | 行为 |
|---|---|
| `access=presign`（默认生产） | 下载/预览 302 → `PresignGet` |
| `access=cdn` | `PublicURL`（桶或 CDN 已公共读 / 鉴权在 CDN） |
| `access=proxy` | 应用 `Get` 流式转发（内网、调试） |
| `driver=local` | 静态 `/uploads` 或同样走 Presign 语义的本地 URL |

---

## 六、全量迁移设计（必须可完全迁移）

### 6.1 命令

```text
doc storage migrate \
  [--dry-run] \
  [--source-dir=$DOC_HOME/uploads] \
  [--concurrency=8] \
  [--rewrite=none|db|content|all] \
  [--resume=runtime/storage-migrate-state.json] \
  [--verify=size,etag]
```

### 6.2 流水线

```text
1. Inventory
   - 递归扫描 source-dir
   - 规范化为 key（相对 uploads/…；跳过非法/隐藏文件）
   - 输出 inventory.csv：key, size, mtime, sha256(可选)

2. Upload
   - 并发 worker；Head 已存在且 size/etag 一致 → skip
   - Put；失败写入 fail 队列
   - 状态机 resume：已成功 key 不重传

3. Verify
   - 抽检或全量 Head；可选下载校验 sha256
   - 生成 verify-report

4. Rewrite（可分步、可回滚）
   - db：attachment / member.avatar / book.cover 等字段 → 统一 key
   - content：文档 markdown/html 中 `/uploads/...`、旧绝对路径 → 新 URL 形态
   - 先备份 SQL / 导出受影响文档 id 列表

5. Cutover
   - 配置 driver=s3；停写本地
   - 观察期保留本地目录 N 天（≥7，建议 30）
   - 确认后归档或删除本地（需二次确认标志）
```

### 6.3 「完全迁移」验收标准

| 项 | 标准 |
|---|---|
| 覆盖率 | inventory 中 100% 成功或进入「永久跳过」白名单（损坏文件）并有报告 |
| 一致性 | 抽样 Head size 一致；关键附件可下载 |
| 引用 | rewrite=all 后：随机抽文档内图片/附件可打开 |
| 可重跑 | 二次 migrate 全部 skip、无重复对象膨胀 |
| 回滚 | 未删本地前：`driver=local` + 恢复 DB 备份可回退 |

### 6.4 风险与对策

| 风险 | 对策 |
|---|---|
| Markdown 非常规 URL | rewrite 支持多种正则；dry-run 打出「未匹配但仍像 uploads」清单人工处理 |
| 超大文件 | 分片上传；单独提高超时 |
| 迁移中双写 | cutover 窗口短维护；或迁移期双写 local+s3（可选增强，非必须） |
| 密钥泄露 | 仅 env；CI 不用真实生产密钥 |

---

## 七、业务重构切片（不兼容旧写盘）

### T14-a · 内核

- `BlobStore` + local + s3(aws-sdk-go-v2)  
- key / validate / URL  
- 单测：local roundtrip；s3 对 MinIO testcontainer 或 build tag 集成测  

### T14-b · 业务全切

- 所有 Upload / Download / Remove 走 storage  
- **删除** Controller 内 `os.Create` uploads、本地 `Output.Download(path)`（改为 Presign/proxy）  
- 统一返回给前端的 URL 生成函数  

### T14-c · 全量迁移工具 + 运维文档

- `doc storage migrate` 全流程  
- 升级手册：兼容性矩阵（MinIO / OSS / COS / S3 勾选表）  

### T14-d（可选）· 直传与图片处理

- PresignPut 浏览器直传  
- 缩略图：应用内生成后 Put，或后续接云处理（仍经 key）  

---

## 八、配置（全新）

```ini
[storage]
driver = local                 # local | s3
access_mode = presign          # presign | cdn | proxy

# s3
endpoint = "${DOC_STORAGE_ENDPOINT}"
region = "${DOC_STORAGE_REGION||us-east-1}"
bucket = "${DOC_STORAGE_BUCKET}"
access_key = "${DOC_STORAGE_ACCESS_KEY}"
secret_key = "${DOC_STORAGE_SECRET_KEY}"
force_path_style = "${DOC_STORAGE_FORCE_PATH_STYLE||true}"
use_ssl = "${DOC_STORAGE_USE_SSL||true}"
prefix = "${DOC_STORAGE_PREFIX}"
cdn_base = "${DOC_STORAGE_CDN_BASE}"
presign_expire = 15m

# 上传策略（原 [upload] 可并入或引用）
max_size = 10M
allowed_ext = .jpg,.png,.gif,.webp,.pdf,.zip,...
```

环境变量：**仅** `DOC_STORAGE_*`。

---

## 九、兼容性测试矩阵（实施必填）

| 后端 | endpoint 形态 | path-style | Put/Get/Head/Del | Presign | 分片 | 结论 |
|---|---|---|---|---|---|---|
| MinIO | `http://127.0.0.1:9000` | true | | | | 必过（CI） |
| AWS S3 | 区域 endpoint | false | | | | 发布前过 |
| 阿里云 OSS S3 | 文档兼容地址 | 按文档 | | | | 发布前过 |
| 腾讯云 COS S3 | 文档兼容地址 | 按文档 | | | | 发布前过 |
| 其它 | … | … | | | | 按需 |

---

## 十、验收

- [ ] 业务代码无直写 `WorkingDirectory/uploads`（除 local 驱动内部）  
- [ ] MinIO：新上传、预签名读、删除通过  
- [ ] 至少再验证 1 家公有云 S3 兼容（OSS 或 COS 或 AWS）  
- [ ] `migrate --dry-run` + 全量 migrate + verify 覆盖率达标  
- [ ] `--rewrite=all` 后旧站内容链接可访问  
- [ ] 多实例：无本地磁盘依赖  
- [ ] 运维文档含矩阵与回滚步骤  

---

## 十一、明确不做（本 sprint）

- MCP 二进制附件  
- 厂商图片处理 / 持久化事件通知（可后挂）  
- 以专有 SDK 为主驱动  
- 迁移期无备份强删本地  

---

## 十二、工作量粗估

| 切片 | 人天 |
|---|---|
| T14-a 内核 + MinIO 测 | 3~4 |
| T14-b 业务全切 | 3~4 |
| T14-c 全量迁移 + 文档 + 矩阵 | 3~5 |
| 公有云抽测 | 1 |
| **合计** | **约 10~14 天**（原「1~2 周」偏紧；按完全迁移+多云建议按 **2~3 周** 排） |

---

## 十三、参考

- [AWS SDK for Go v2 · S3](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3)  
- [minio-go](https://github.com/minio/minio-go)  
- [Go CDK blob](https://gocloud.dev/howto/blob/)  
- 阿里云 OSS / 腾讯云 COS / 华为 OBS「S3 兼容」官方文档  
- [round-5-env-mindoc-to-doc.md](./round-5-env-mindoc-to-doc.md) · [round-5-scripts-layout.md](./round-5-scripts-layout.md)  
- 现状上传：`DocumentController.Upload` 等  
