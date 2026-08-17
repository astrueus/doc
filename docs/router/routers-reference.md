# 路由分类参考

本文档基于 `routers/router.go` 中的路由定义，并对照各 Controller 实现整理：**哪些路由渲染 HTML 页面，哪些是纯接口（JSON / 文件 / 图片 / 重定向）**。

路由注册文件：`routers/router.go`  
过滤器（登录校验等）：`routers/filter.go`（不含路由定义）

## 判定标准

| 类型 | 判定依据 |
|------|----------|
| **页面渲染** | 设置 `TplName`，返回 HTML 模板 |
| **纯接口** | 返回 JSON（`JsonResult`）、图片、文件下载，或仅做重定向 |
| **混合** | 同一路径：GET 渲染页面，POST 或 Ajax 请求返回 JSON |

## 需登录的路由

以下路径在 `filter.go` 中配置了登录过滤；未登录时 Ajax 请求返回 JSON 403，普通请求重定向到登录页：

- `/manager`、`/manager/*`
- `/setting`、`/setting/*`
- `/book`、`/book/*`
- `/api/*`
- `/manage/*`

---

## 一、页面渲染路由

GET 为主，返回 HTML 模板。

### 前台 / 公共

| 路由 | 方法 | 模板 / 说明 |
|------|------|-------------|
| `/` | * | `home/index.tpl` 首页 |
| `/login` | GET | `account/login.tpl` 登录页 |
| `/register` | GET | `account/register.tpl` 注册页 |
| `/find_password` | GET | 找回密码页（step1 / step2） |
| `/search` | GET | `search/index.tpl` 搜索页 |
| `/blogs` | * | `blog/list.tpl` 博客列表 |
| `/blog-:id.html` | GET | `blog/index.tpl` 博客详情（加密博客为 `blog/index_password.tpl`） |
| `/docs/:key` | * | `document/*_read.tpl` 文档项目首页 |
| `/docs/:key/:id` | GET（非 Ajax） | 文档阅读页 |
| `/tag/:key` | GET | `label/index.tpl` 标签页 |
| `/tags` | GET | `label/list.tpl` 标签列表 |
| `/items` | GET | `items/index.tpl` 项目空间首页 |
| `/items/:key` | GET | `items/list.tpl` 项目空间列表 |
| `/comment/index` | * | `comment/index.tpl` 评论页 |

### 用户设置

| 路由 | 方法 | 模板 |
|------|------|------|
| `/setting` | GET | `setting/index.tpl` |
| `/setting/password` | GET | `setting/password.tpl` |

### 项目管理

| 路由 | 方法 | 模板 |
|------|------|------|
| `/book` | * | `book/index.tpl` 我的项目 |
| `/book/:key/dashboard` | * | `book/dashboard.tpl` 项目仪表盘 |
| `/book/:key/setting` | * | `book/setting.tpl` 项目设置 |
| `/book/:key/users` | * | `book/users.tpl` 项目成员 |
| `/book/:key/teams` | * | `book/team.tpl` 项目团队 |

### 文档编辑 / 历史

| 路由 | 方法 | 模板 / 说明 |
|------|------|-------------|
| `/api/:key/edit/?:id` | * | 文档编辑器（markdown / html 等） |
| `/api/:key/compare/:id` | * | `document/compare.tpl` 版本对比 |
| `/history/get` | GET | `document/history.tpl` 文档历史 |

### 博客管理

| 路由 | 方法 | 模板 |
|------|------|------|
| `/manage/blogs` | * | `blog/manage_list.tpl` |
| `/manage/blogs/setting/?:id` | GET | `blog/manage_setting.tpl` |
| `/manage/blogs/edit/?:id` | GET | `blog/manage_edit.tpl` |

### 后台管理 `/manager/*`

| 路由 | 模板 / 说明 |
|------|-------------|
| `/manager` | `manager/index.tpl` 后台首页 |
| `/manager/users` | 用户列表 |
| `/manager/users/edit/:id` | GET：编辑用户页 |
| `/manager/books` | 项目列表 |
| `/manager/books/edit/:key` | GET：编辑项目页 |
| `/manager/comments` | 评论管理 |
| `/manager/setting` | GET：系统设置页 |
| `/manager/attach/list` | 附件列表 |
| `/manager/attach/detailed/:id` | 附件详情 |
| `/manager/label/list` | 标签列表 |
| `/manager/team` | 团队列表 |
| `/manager/team/member/list/:id` | 团队成员列表 |
| `/manager/team/book/list/:id` | 团队项目列表 |
| `/manager/itemsets` | 项目空间管理 |

### 特殊说明

| 路由 | 说明 |
|------|------|
| `/api/template/list` | 路径在 `/api/` 下，但 `TemplateController.List()` 实际渲染 `template/list.tpl`（POST 也返回 HTML，不是 JSON） |

---

## 二、纯接口路由

返回 JSON、二进制资源或重定向，不渲染业务页面。

### 账户相关

| 路由 | 方法 | 返回类型 |
|------|------|----------|
| `/login` | POST | JSON |
| `/register` | POST | JSON |
| `/find_password` | POST | JSON |
| `/valid_email` | POST | JSON |
| `/dingtalk_login` | * | JSON |
| `/captcha` | * | JPEG 验证码图片 |
| `/logout` | * | 302 重定向 |
| `/qrlogin/:app` | * | 302 重定向（钉钉扫码登录） |

### 用户设置

| 路由 | 方法 | 返回类型 |
|------|------|----------|
| `/setting` | POST | JSON |
| `/setting/password` | POST | JSON |
| `/setting/upload` | * | JSON（头像上传） |

### 项目管理 `/book/*`

| 路由 | 方法 |
|------|------|
| `/book/create` | POST（非 POST 返回错误 JSON） |
| `/book/itemsets/search` | * |
| `/book/:key/release` | POST |
| `/book/:key/sort` | POST |
| `/book/users/create` | POST |
| `/book/users/change` | POST |
| `/book/users/delete` | POST |
| `/book/users/import` | POST |
| `/book/users/copy` | POST |
| `/book/setting/save` | POST |
| `/book/setting/open` | POST |
| `/book/setting/transfer` | POST |
| `/book/setting/upload` | POST |
| `/book/setting/delete` | POST |
| `/book/team/add` | POST |
| `/book/team/delete` | POST |
| `/book/team/search` | * |

### 文档 API `/api/*`

| 路由 | 方法 | 说明 |
|------|------|------|
| `/api/template/get` | GET | JSON |
| `/api/template/add` | POST | JSON |
| `/api/template/remove` | POST | JSON |
| `/api/attach/remove/` | POST | JSON |
| `/api/upload` | POST | JSON（文件上传） |
| `/api/:key/create` | POST | JSON |
| `/api/:key/delete` | POST | JSON |
| `/api/:key/content/?:id` | GET / POST | JSON（读 / 写文档内容） |
| `/api/search/user/:key` | * | JSON（搜索用户） |

### 文档阅读 / 导出

| 路由 | 方法 | 返回类型 |
|------|------|----------|
| `/docs/:key/:id` | GET（Ajax） | JSON |
| `/docs/:key/search` | POST | JSON |
| `/history/delete` | * | JSON |
| `/history/restore` | * | JSON |
| `/export/:key` | * | 文件下载（zip / pdf 等）或错误 HTML 页 |
| `/qrcode/:key.png` | GET | PNG 二维码图片 |
| `/attach_files/:key/:attach_id` | GET | 附件文件下载 |

### 博客

| 路由 | 方法 | 返回类型 |
|------|------|----------|
| `/blog-:id.html` | POST | JSON（密码校验） |
| `/blog-attach/:id/:attach_id` | GET | 文件下载 |
| `/manage/blogs/setting/?:id` | POST | JSON |
| `/manage/blogs/edit/?:id` | POST | JSON |
| `/manage/blogs/delete` | POST | JSON |
| `/manage/blogs/upload` | POST | JSON |
| `/manage/blogs/attach/:id` | POST | JSON |

### 评论

| 路由 | 方法 | 说明 |
|------|------|------|
| `/comment/create` | POST | JSON（当前实现较简单） |
| `/comment/lists` | GET | 空实现，无实际响应逻辑 |

### 后台管理 `/manager/*`（POST 接口）

| 路由 | 方法 |
|------|------|
| `/manager/member/create` | POST |
| `/manager/member/delete` | POST |
| `/manager/member/update-member-status` | POST |
| `/manager/member/change-member-role` | POST |
| `/manager/users/edit/:id` | POST |
| `/manager/books/edit/:key` | POST |
| `/manager/books/delete` | * |
| `/manager/books/token` | POST |
| `/manager/books/transfer` | POST |
| `/manager/books/open` | POST |
| `/manager/setting` | POST |
| `/manager/attach/delete` | POST |
| `/manager/label/delete/:id` | POST |
| `/manager/team/create` | POST |
| `/manager/team/edit` | POST |
| `/manager/team/delete` | POST |
| `/manager/team/member/add` | POST |
| `/manager/team/member/delete` | POST |
| `/manager/team/member/change_role` | POST |
| `/manager/team/member/search` | * |
| `/manager/team/book/add` | POST |
| `/manager/team/book/delete` | POST |
| `/manager/team/book/search` | * |
| `/manager/itemsets/edit` | POST |
| `/manager/itemsets/delete` | POST |

---

## 三、混合路由

同一路径按 HTTP 方法或请求类型（如 Ajax）区分响应形式。

| 路由 | GET | POST / Ajax |
|------|-----|-------------|
| `/login` | 登录页 | JSON |
| `/register` | 注册页 | JSON |
| `/find_password` | 找回密码页 | JSON |
| `/setting` | 设置页 | JSON |
| `/setting/password` | 密码页 | JSON |
| `/docs/:key/:id` | 阅读页 | Ajax → JSON |
| `/blog-:id.html` | 博客页 | 密码校验 → JSON |
| `/manage/blogs/setting/?:id` | 设置页 | JSON |
| `/manage/blogs/edit/?:id` | 编辑页 | JSON |
| `/manager/users/edit/:id` | 编辑页 | JSON |
| `/manager/books/edit/:key` | 编辑页 | JSON |
| `/manager/setting` | 设置页 | JSON |

---

## 四、统计与注意点

| 类型 | 数量（约） |
|------|-----------|
| 纯页面渲染 | ~35 条 |
| 纯接口 | ~65 条 |
| 混合 | 11 条 |

1. **路径名不等于接口类型**：`/api/template/list` 虽在 `/api/` 下，实际渲染 HTML 页面，不是 JSON API。
2. **`filter.go` 不含路由**：仅注册登录过滤器和响应头，路由定义全部在 `router.go`。
3. **评论接口不完整**：`/comment/lists` 与 `/comment/create` 实现较简陋，归类为接口路由但功能不完整。
4. **导出接口**：`/export/:key` 成功时返回文件下载，失败或转换中时可能返回 HTML 错误页。

## 相关文件

| 文件 | 职责 |
|------|------|
| `routers/router.go` | 路由注册 |
| `routers/filter.go` | 全局过滤器、登录校验 |
| `controllers/*.go` | 各路由对应的业务逻辑 |
