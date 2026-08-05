package router

import (
	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/controller"
	"git.itopcms.com/astrueus/doc/internal/mcp"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

func registerAPI() {
	web.Router("/api/template/get", &controller.TemplateController{}, "get:Get")
	web.Router("/api/template/add", &controller.TemplateController{}, "post:Add")
	web.Router("/api/template/remove", &controller.TemplateController{}, "post:Delete")

	web.Router("/api/attach/remove/", &controller.DocumentController{}, "post:RemoveAttachment")
	web.Router("/api/upload", &controller.DocumentController{}, "post:Upload")
	web.Router("/api/:key/create", &controller.DocumentController{}, "post:Create")
	web.Router("/api/:key/delete", &controller.DocumentController{}, "post:Delete")
	web.Router("/api/:key/content/?:id", &controller.DocumentController{}, "*:Content")
	web.Router("/api/search/user/:key", &controller.SearchController{}, "*:User")

	if config.MustGlobal().MCP.Enable {
		web.Handler("/mcp", mcp.NewHTTPHandler(), true)
		logs.Info("MCP HTTP handler mounted at /mcp (enable=true)")
	}
}
