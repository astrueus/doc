package router

import (
	"git.itopcms.com/astrueus/doc/internal/controller"
	"github.com/beego/beego/v2/server/web"
)

func registerBlog() {
	web.Router("/manage/blogs", &controller.BlogController{}, "*:ManageList")
	web.Router("/manage/blogs/setting/?:id", &controller.BlogController{}, "*:ManageSetting")
	web.Router("/manage/blogs/edit/?:id", &controller.BlogController{}, "*:ManageEdit")
	web.Router("/manage/blogs/delete", &controller.BlogController{}, "post:ManageDelete")
	web.Router("/manage/blogs/upload", &controller.BlogController{}, "post:Upload")
	web.Router("/manage/blogs/attach/:id", &controller.BlogController{}, "post:RemoveAttachment")

	web.Router("/blogs", &controller.BlogController{}, "*:List")
	web.Router("/blog-attach/:id:int/:attach_id:int", &controller.BlogController{}, "get:Download")
	web.Router("/blog-:id([0-9]+).html", &controller.BlogController{}, "*:Index")
}
