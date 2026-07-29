package router

import (
	"git.itopcms.com/jackliu/doc/internal/controller"
	"github.com/beego/beego/v2/server/web"
)

func registerPage() {
	web.Router("/", &controller.HomeController{}, "*:Index")

	web.Router("/search", &controller.SearchController{}, "get:Index")

	web.Router("/tag/:key", &controller.LabelController{}, "get:Index")
	web.Router("/tags", &controller.LabelController{}, "get:List")

	web.Router("/items", &controller.ItemsetsController{}, "get:Index")
	web.Router("/items/:key", &controller.ItemsetsController{}, "get:List")
}
