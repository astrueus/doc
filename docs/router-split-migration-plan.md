# 路由拆分与 API 前缀调整计划

## 背景

当前项目路由集中注册在 `routers/router.go` 的 `init()` 中。根据 `docs/routers-reference.md` 的分类，现有路由同时包含页面渲染、JSON API、文件下载、图片响应、重定向以及混合路由。

目标是：

1. 将 API 接口路由和模板渲染路由拆分到不同路由文件。
2. API 接口统一使用 `/api/` 前缀。
3. 模板渲染路由不再使用 `/api/` 前缀。

## 总体策略

先做低风险拆分，再做路径迁移。

1. 先把 `routers/router.go` 中的路由按职责拆成多个注册函数。
2. 保留一个统一入口 `init()`，显式控制注册顺序。
3. 将当前位于 `/api/` 下但实际渲染模板的路由迁出 `/api/`。
4. 同步修改模板或前端脚本中的硬编码路径。
5. 最后再考虑是否把历史混合路由的 POST JSON 行为迁移到 `/api/`。

不建议给每个路由文件都写独立 `init()`，因为参数路由和静态路由存在匹配顺序要求，统一入口更容易维护。

## 建议文件结构

```text
routers/
  router.go          # 统一入口，只调用各类注册函数
  page.go            # 页面渲染路由
  api.go             # JSON / 上传 / 删除等 API 路由
  admin.go           # 后台管理路由，可选
  filter.go          # 登录过滤器和全局过滤器
```

第一阶段可以只拆成 3 个文件：

```text
routers/
  router.go
  page.go
  api.go
```

## `routers/router.go` 调整

将当前 `init()` 改成统一入口：

```go
package routers

func init() {
	registerPageRoutes()
	registerAPIRoutes()
}
```

如果后续拆出后台管理路由，可以调整为：

```go
func init() {
	registerPageRoutes()
	registerAdminRoutes()
	registerAPIRoutes()
}
```

注册顺序建议保持“静态明确路由优先，参数路由靠后”的原则。

## 页面渲染路由文件

新增 `routers/page.go`，放置返回 HTML 模板的路由。

示例：

```go
package routers

import (
	"git.itopcms.com/jackliu/doc/controllers"
	"github.com/beego/beego/v2/server/web"
)

func registerPageRoutes() {
	web.Router("/", &controllers.HomeController{}, "*:Index")

	web.Router("/login", &controllers.AccountController{}, "get:Login")
	web.Router("/register", &controllers.AccountController{}, "get:Register")
	web.Router("/find_password", &controllers.AccountController{}, "get:FindPassword")

	web.Router("/book", &controllers.BookController{}, "*:Index")
	web.Router("/book/:key/dashboard", &controllers.BookController{}, "*:Dashboard")
	web.Router("/book/:key/setting", &controllers.BookController{}, "*:Setting")
	web.Router("/book/:key/users", &controllers.BookController{}, "*:Users")
	web.Router("/book/:key/teams", &controllers.BookController{}, "*:Team")

	web.Router("/book/:key/edit/?:id", &controllers.DocumentController{}, "*:Edit")
	web.Router("/book/:key/compare/:id", &controllers.DocumentController{}, "*:Compare")
	web.Router("/book/template/list", &controllers.TemplateController{}, "post:List")

	web.Router("/docs/:key", &controllers.DocumentController{}, "*:Index")
	web.Router("/docs/:key/:id", &controllers.DocumentController{}, "*:Read")

	web.Router("/search", &controllers.SearchController{}, "get:Index")
	web.Router("/tag/:key", &controllers.LabelController{}, "get:Index")
	web.Router("/tags", &controllers.LabelController{}, "get:List")
	web.Router("/items", &controllers.ItemsetsController{}, "get:Index")
	web.Router("/items/:key", &controllers.ItemsetsController{}, "get:List")
}
```

注意：`/login`、`/register`、`/find_password` 当前是 `*:xxx` 混合路由。如果本次只做文件拆分，可以暂时保留原写法；如果要严格区分 API 和页面，需要拆成 GET 页面和 `/api/...` POST 接口。

## API 路由文件

新增 `routers/api.go`，放置 JSON、上传、删除、内容读写等接口。

示例：

```go
package routers

import (
	"git.itopcms.com/jackliu/doc/controllers"
	"github.com/beego/beego/v2/server/web"
)

func registerAPIRoutes() {
	web.Router("/api/template/get", &controllers.TemplateController{}, "get:Get")
	web.Router("/api/template/add", &controllers.TemplateController{}, "post:Add")
	web.Router("/api/template/remove", &controllers.TemplateController{}, "post:Delete")

	web.Router("/api/attach/remove/", &controllers.DocumentController{}, "post:RemoveAttachment")
	web.Router("/api/upload", &controllers.DocumentController{}, "post:Upload")
	web.Router("/api/:key/create", &controllers.DocumentController{}, "post:Create")
	web.Router("/api/:key/delete", &controllers.DocumentController{}, "post:Delete")
	web.Router("/api/:key/content/?:id", &controllers.DocumentController{}, "*:Content")
	web.Router("/api/search/user/:key", &controllers.SearchController{}, "*:User")
}
```

需要注意 `TemplateController.List()` 不应放在这里，因为它渲染 `template/list.tpl`，不是 JSON API。

## 需要迁移的 `/api` 模板路由

当前有 3 个重点路由需要迁出 `/api/`。

| 当前路由 | 控制器方法 | 当前行为 | 建议新路由 |
| --- | --- | --- | --- |
| `/api/:key/edit/?:id` | `DocumentController.Edit` | 渲染文档编辑器模板 | `/book/:key/edit/?:id` |
| `/api/:key/compare/:id` | `DocumentController.Compare` | 渲染版本对比模板 | `/book/:key/compare/:id` |
| `/api/template/list` | `TemplateController.List` | 渲染 `template/list.tpl` | `/book/template/list` |

迁移后删除旧路由，避免同一页面同时存在 `/api` 和非 `/api` 两套入口。

## 需要同步修改的调用点

### `views/book/index.tpl`

当前项目列表编辑入口硬编码了 `/api/`：

```tpl
:href="'{{.BaseUrl}}/api/' + item.identify + '/edit'"
```

需要改成：

```tpl
:href="'{{.BaseUrl}}/book/' + item.identify + '/edit'"
```

### `views/document/markdown_edit_template.tpl`

当前模板使用 `urlfor`：

```tpl
{{urlfor "TemplateController.List"}}
```

如果只保留新的 `/book/template/list` 路由，`urlfor` 通常会自动生成新地址，不需要手动改模板。

迁移后仍需验证 `window.template.listUrl` 是否生成了 `/book/template/list`。

### `views/document/history.tpl`

当前版本对比入口使用：

```tpl
{{urlfor "DocumentController.Compare" ":key" .Model.Identify ":id" ""}}
```

如果只保留新的 `/book/:key/compare/:id` 路由，`urlfor` 通常会自动生成新地址。

迁移后需验证历史版本弹窗打开地址不再是 `/api/:key/compare/:id`。

## 登录过滤器调整

当前 `routers/filter.go` 已经覆盖：

```go
web.InsertFilter("/book", web.BeforeRouter, middleware.FilterUser)
web.InsertFilter("/book/*", web.BeforeRouter, middleware.FilterUser)
web.InsertFilter("/api/*", web.BeforeRouter, middleware.FilterUser)
```

如果新路由使用 `/book/:key/edit/?:id`、`/book/:key/compare/:id`、`/book/template/list`，不需要新增过滤器，因为 `/book/*` 已覆盖。

如果改用其他前缀，例如 `/editor/*` 或 `/template/*`，必须同步增加登录过滤：

```go
web.InsertFilter("/editor/*", web.BeforeRouter, middleware.FilterUser)
web.InsertFilter("/template/*", web.BeforeRouter, middleware.FilterUser)
```

另外，`routers/filter.go` 当前存在两套类似的登录过滤注册，一套使用 `middleware.FilterUser`，一套使用本地 `FilterUser`。后续建议统一成一套，避免新增路由前缀时漏改。

## 混合路由的后续处理

如果目标只是“模板渲染路由不使用 `/api/`”，迁移上面 3 个路由即可。

如果目标升级为“所有 JSON 接口都必须使用 `/api/`”，则还需要处理大量历史混合路由，例如：

| 当前路由 | GET 行为 | POST 行为 | 建议 |
| --- | --- | --- | --- |
| `/login` | 登录页 | JSON 登录接口 | GET 保留 `/login`，POST 迁到 `/api/login` |
| `/register` | 注册页 | JSON 注册接口 | GET 保留 `/register`，POST 迁到 `/api/register` |
| `/find_password` | 找回密码页 | JSON 接口 | GET 保留 `/find_password`，POST 迁到 `/api/find_password` |
| `/setting` | 设置页 | JSON 保存接口 | GET 保留 `/setting`，POST 迁到 `/api/setting` |
| `/setting/password` | 密码页 | JSON 保存接口 | GET 保留 `/setting/password`，POST 迁到 `/api/setting/password` |
| `/manage/blogs/edit/?:id` | 编辑页 | JSON 保存接口 | GET 保留原路径，POST 迁到 `/api/manage/blogs/edit/?:id` |
| `/manager/setting` | 后台设置页 | JSON 保存接口 | GET 保留原路径，POST 迁到 `/api/manager/setting` |

这类改动会影响前端表单、Ajax 请求、反向路由和登录过滤，建议单独作为第二阶段处理。

## 推荐实施阶段

### 第一阶段：仅拆文件，不改路径

1. 新增 `page.go` 和 `api.go`。
2. 将现有路由按页面和接口移动到对应注册函数。
3. 保持所有 URL 不变。
4. 编译并验证现有行为不变。

这一阶段风险最低，主要解决路由文件过大的问题。

### 第二阶段：迁出 `/api` 下的模板路由

1. 将 `/api/:key/edit/?:id` 改为 `/book/:key/edit/?:id`。
2. 将 `/api/:key/compare/:id` 改为 `/book/:key/compare/:id`。
3. 将 `/api/template/list` 改为 `/book/template/list`。
4. 修改 `views/book/index.tpl` 中硬编码的编辑地址。
5. 验证 `urlfor` 生成的编辑、对比、模板列表地址。

这一阶段解决“模板渲染路由不应该带 `/api/`”的问题。

### 第三阶段：规范历史 JSON 接口

1. 梳理所有混合路由。
2. 将 GET 页面和 POST JSON 拆成不同 URL。
3. 将 POST JSON 路由迁到 `/api/...`。
4. 修改前端表单和 Ajax 请求。
5. 对旧 URL 决定是否保留兼容或直接删除。

这一阶段改动面较大，建议配合接口回归测试。

## 验证清单

完成第一、二阶段后至少验证：

1. 项目列表点击“编辑”进入 `/book/{identify}/edit`。
2. 文档编辑器可以正常加载文档树。
3. 文档内容保存接口仍请求 `/api/{identify}/content/{id}`。
4. 新建、删除文档接口仍请求 `/api/{identify}/create` 和 `/api/{identify}/delete`。
5. 模板列表请求地址为 `/book/template/list`，并能返回模板 HTML。
6. 模板新增、删除、读取仍请求 `/api/template/add`、`/api/template/remove`、`/api/template/get`。
7. 历史版本对比弹窗打开 `/book/{identify}/compare/{historyId}`。
8. 未登录访问 `/book/{identify}/edit` 会被登录过滤拦截。
9. 未登录 Ajax 请求 `/api/*` 仍返回 JSON 403。
10. 全项目编译通过。

## 风险点

1. `urlfor` 会根据控制器方法反向生成路由。如果同一个控制器方法同时注册了旧路由和新路由，生成结果可能不符合预期。
2. 参数路由如 `/api/:key/...` 可能覆盖后续静态路由，注册顺序必须谨慎。
3. 当前混合路由较多，一次性全部迁移到 `/api` 会影响大量前端请求。
4. `filter.go` 中登录过滤重复注册，后续维护时容易遗漏新前缀。

## 建议最终效果

短期目标：

```text
/api/*                  只保留 JSON / 上传 / 删除 / 内容读写接口
/book/:key/edit         文档编辑页面
/book/:key/compare/:id  文档版本对比页面
/book/template/list     模板列表 HTML 片段
```

长期目标：

```text
GET  页面路由       不使用 /api 前缀
POST JSON 接口     统一使用 /api 前缀
下载、图片、导出    按资源语义保留独立路径，必要时单独归类
```
