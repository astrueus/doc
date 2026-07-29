package router

import (
	"git.itopcms.com/jackliu/doc/internal/controller"
	"github.com/beego/beego/v2/server/web"
)

func registerDocument() {
	web.Router("/history/get", &controller.DocumentController{}, "get:History")
	web.Router("/history/delete", &controller.DocumentController{}, "*:DeleteHistory")
	web.Router("/history/restore", &controller.DocumentController{}, "*:RestoreHistory")

	web.Router("/docs/:key", &controller.DocumentController{}, "*:Index")
	web.Router("/docs/:key/:id", &controller.DocumentController{}, "*:Read")
	web.Router("/docs/:key/search", &controller.DocumentController{}, "post:Search")
	web.Router("/export/:key", &controller.DocumentController{}, "*:Export")
	web.Router("/qrcode/:key.png", &controller.DocumentController{}, "get:QrCode")

	web.Router("/attach_files/:key/:attach_id", &controller.DocumentController{}, "get:DownloadAttachment")

	web.Router("/comment/create", &controller.CommentController{}, "post:Create")
	web.Router("/comment/lists", &controller.CommentController{}, "get:Lists")
	web.Router("/comment/index", &controller.CommentController{}, "*:Index")
}
