package router

import (
	"git.itopcms.com/astrueus/doc/internal/controller"
	"github.com/beego/beego/v2/server/web"
)

func registerBook() {
	web.Router("/book", &controller.BookController{}, "*:Index")

	// Static /book/* paths before :key so they are not captured as identify.
	web.Router("/book/create", &controller.BookController{}, "*:Create")
	web.Router("/book/itemsets/search", &controller.BookController{}, "*:ItemsetsSearch")
	web.Router("/book/template/list", &controller.TemplateController{}, "post:List")

	web.Router("/book/users/create", &controller.BookMemberController{}, "post:AddMember")
	web.Router("/book/users/change", &controller.BookMemberController{}, "post:ChangeRole")
	web.Router("/book/users/delete", &controller.BookMemberController{}, "post:RemoveMember")
	web.Router("/book/users/import", &controller.BookController{}, "post:Import")
	web.Router("/book/users/copy", &controller.BookController{}, "post:Copy")

	web.Router("/book/setting/save", &controller.BookController{}, "post:SaveBook")
	web.Router("/book/setting/open", &controller.BookController{}, "post:PrivatelyOwned")
	web.Router("/book/setting/transfer", &controller.BookController{}, "post:Transfer")
	web.Router("/book/setting/upload", &controller.BookController{}, "post:UploadCover")
	web.Router("/book/setting/delete", &controller.BookController{}, "post:Delete")

	web.Router("/book/team/add", &controller.BookController{}, "POST:TeamAdd")
	web.Router("/book/team/delete", &controller.BookController{}, "POST:TeamDelete")
	web.Router("/book/team/search", &controller.BookController{}, "*:TeamSearch")

	web.Router("/book/:key/dashboard", &controller.BookController{}, "*:Dashboard")
	web.Router("/book/:key/setting", &controller.BookController{}, "*:Setting")
	web.Router("/book/:key/users", &controller.BookController{}, "*:Users")
	web.Router("/book/:key/release", &controller.BookController{}, "post:Release")
	web.Router("/book/:key/sort", &controller.BookController{}, "post:SaveSort")
	web.Router("/book/:key/teams", &controller.BookController{}, "*:Team")

	// Moved out of /api/* (page routes must not use /api prefix).
	web.Router("/book/:key/edit/?:id", &controller.DocumentController{}, "*:Edit")
	web.Router("/book/:key/compare/:id", &controller.DocumentController{}, "*:Compare")
}
