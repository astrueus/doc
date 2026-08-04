# Round 5 · T14 · 对象存储 + 旧数据迁移 — 细化方案

> 对应 [round-5-execution-plan.md §十三丙 T14](./round-5-execution-plan.md#十三丙t14--对象存储上传--旧数据迁移1~2-周)。  
> 建议独立 sprint；与 [T15](./round-5-scripts-layout.md) 约定：迁移工具优先 **Go 子命令**，不新增 bash。  
> 配置键与 [MINDOC→DOC](./round-5-env-mindoc-to-doc.md) 对齐，使用 `DOC_STORAGE_*`。  
> **状态：** ⏳ 待实施。

---

## 一、现状

| 上传点 | 路径约定 |
|---|---|
| `DocumentController.Upload` | `uploads/{bookIdentify}/...` |
| `SettingController.Upload` | `uploads/{YYYYMM}/`（头像等） |
| `BlogController.Upload` | `uploads/blog/{YYYYMM}/` |
| `BookController.UploadCover` | `uploads/...` |

- 物理根：`filepath.Join(config.WorkingDirectory, "uploads", ...)`  
- HTTP：bootstrap 挂载 `/uploads`  
- 配置：仅有 `[upload]`（大小/扩展名）；**无** `[storage]`  

---

## 二、接口设计

建议包：`internal/storage`

```go
type Storage interface {
    Put(ctx context.Context, key string, r io.Reader, size int64, opts PutOptions) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    // PublicURL：可公开读时返回 CDN/公网 URL；否则返回站内反代路径或空
    PublicURL(key string) string
    // SignedURL：私有桶临时链接；local 驱动可返回 /uploads/... 等价路径
    SignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type PutOptions struct {
    ContentType string
    ACL         string // 可选；S3 兼容
}
```

### Key 约定

- **逻辑 key** 与现网相对路径一致：`uploads/bookIdent/xxx.png`（含 `uploads/` 前缀或统一去掉二选一，**全仓统一**）。  
- 推荐：**key = 相对 WorkingDirectory 的路径**（与库内/Markdown 引用一致），便于迁移与回滚。

### 驱动

| 驱动 | 说明 |
|---|---|
| `local` | 写 `WorkingDirectory`；行为与现网一致；`PublicURL` → `/` + key |
| `s3` | S3 API 兼容（MinIO / 阿里云 OSS S3 兼容 / 其他）；SDK 推荐 `aws-sdk-go-v2` |

---

## 三、配置

```ini
[storage]
driver = local                 # local | s3
# s3
endpoint = 
region = us-east-1
bucket = 
access_key = 
secret_key = 
force_path_style = true        # MinIO 常要
cdn_base =                     # 可选；PublicURL 前缀
prefix =                       # 可选；桶内额外前缀
```

环境变量（新品牌）：

```text
DOC_STORAGE_DRIVER
DOC_STORAGE_ENDPOINT
DOC_STORAGE_REGION
DOC_STORAGE_BUCKET
DOC_STORAGE_ACCESS_KEY
DOC_STORAGE_SECRET_KEY
DOC_STORAGE_FORCE_PATH_STYLE
DOC_STORAGE_CDN_BASE
DOC_STORAGE_PREFIX
```

仅 `DOC_STORAGE_*`（见 [env 方案](./round-5-env-mindoc-to-doc.md)，**不**兼容 `MINDOC_*`）。强类型进 `config.Config`。

---

## 四、业务切换（T14-b）

所有上传点改为：

1. 生成与现网相同的相对 key  
2. `storage.Put(...)`  
3. 返回给前端的 URL：优先 `PublicURL` / `SignedURL`，保证编辑器插入链接仍可用  

下载 / 删附件：经 `Get` / `Delete`；local 驱动下可继续用 HTTP 静态挂载；s3 模式：

- **方案 1**：应用反代流式 `Get`（实现简单，带宽走应用）  
- **方案 2**：重定向到签名 URL（推荐生产）

本轮默认：**上传 + 签名/公网 URL 读取**；反代作 fallback。

---

## 五、迁移命令（T14-c）

```text
doc storage migrate [--dry-run] [--prefix uploads/] [--rewrite-db] [--rewrite-content]
```

落点：`cmd` / `internal/cli` 子命令（与 T15 一致），**不**放 bash；若必须有外壳脚本则进 `deployments/scripts/`。

### 流程

```text
1. 扫描本地 WorkingDirectory/uploads/**
2. 对每个文件：计算 key → Head 桶内是否已存在 → Put
3. 校验：size / etag（可选）
4. 成功清单写入 runtime/storage-migrate-ok.log
5. 失败清单 runtime/storage-migrate-fail.log（可重跑）
6. --rewrite-db：附件表路径若存绝对/本地路径则改为统一 key
7. --rewrite-content：扫描文档 Markdown 中 ](/uploads/...) 形式（谨慎；默认关，先 dry-run）
```

### 运维约定

- 先 `--dry-run`  
- 小样本实迁 → 全量  
- 保留本地副本 **N 天**（文档写清，建议 ≥7）  
- 回滚：`driver=local` + 本地文件仍在即可；已 rewrite 的需备份 DB  

升级文档：`docs/` 下短文或并入部署文档（迁移顺序、权限、IAM）。

---

## 六、PR 拆分

| PR | 内容 |
|---|---|
| T14-a | `internal/storage` 接口 + local + s3 + `[storage]` 配置 + 单测（可用 miniredis？不，用 mock / local） |
| T14-b | Document / Setting / Blog / Book 上传点切换 |
| T14-c | `doc storage migrate` + 运维文档 + dry-run 说明 |

---

## 七、验收

- [ ] `driver=local` 与现网上传/访问一致  
- [ ] `driver=s3`（MinIO 本地即可）新上传可访问  
- [ ] migrate dry-run + 小样本成功  
- [ ] 多实例：s3 模式下新文件不依赖单机磁盘  
- [ ] 环境变量 `DOC_STORAGE_*` 生效  

---

## 八、明确不做

- MCP 二进制附件上传  
- 一次改完历史 CDN / 外链边角  
- 非 S3 协议的厂商专有 API（可后续加驱动）  

---

## 九、参考

- [round-5-scripts-layout.md](./round-5-scripts-layout.md)  
- [round-5-env-mindoc-to-doc.md](./round-5-env-mindoc-to-doc.md)  
- [`internal/config/working_dir.go`](../internal/config/) · S3 / MinIO 文档  
