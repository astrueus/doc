package router

import (
	"git.itopcms.com/jackliu/doc/internal/controller"
	"github.com/beego/beego/v2/server/web"
)

func registerManager() {
	web.Router("/manager", &controller.ManagerController{}, "*:Index")
	web.Router("/manager/users", &controller.ManagerController{}, "*:Users")
	web.Router("/manager/users/edit/:id", &controller.ManagerController{}, "*:EditMember")
	web.Router("/manager/member/create", &controller.ManagerController{}, "post:CreateMember")
	web.Router("/manager/member/delete", &controller.ManagerController{}, "post:DeleteMember")
	web.Router("/manager/member/update-member-status", &controller.ManagerController{}, "post:UpdateMemberStatus")
	web.Router("/manager/member/change-member-role", &controller.ManagerController{}, "post:ChangeMemberRole")
	web.Router("/manager/books", &controller.ManagerController{}, "*:Books")
	web.Router("/manager/books/edit/:key", &controller.ManagerController{}, "*:EditBook")
	web.Router("/manager/books/delete", &controller.ManagerController{}, "*:DeleteBook")

	web.Router("/manager/comments", &controller.ManagerController{}, "*:Comments")
	web.Router("/manager/setting", &controller.ManagerController{}, "*:Setting")
	web.Router("/manager/books/token", &controller.ManagerController{}, "post:CreateToken")
	web.Router("/manager/books/transfer", &controller.ManagerController{}, "post:Transfer")
	web.Router("/manager/books/open", &controller.ManagerController{}, "post:PrivatelyOwned")

	web.Router("/manager/attach/list", &controller.ManagerController{}, "*:AttachList")
	web.Router("/manager/attach/detailed/:id", &controller.ManagerController{}, "*:AttachDetailed")
	web.Router("/manager/attach/delete", &controller.ManagerController{}, "post:AttachDelete")
	web.Router("/manager/label/list", &controller.ManagerController{}, "get:LabelList")
	web.Router("/manager/label/delete/:id", &controller.ManagerController{}, "post:LabelDelete")

	web.Router("/manager/team", &controller.ManagerController{}, "*:Team")
	web.Router("/manager/team/create", &controller.ManagerController{}, "POST:TeamCreate")
	web.Router("/manager/team/edit", &controller.ManagerController{}, "POST:TeamEdit")
	web.Router("/manager/team/delete", &controller.ManagerController{}, "POST:TeamDelete")

	web.Router("/manager/team/member/list/:id", &controller.ManagerController{}, "*:TeamMemberList")
	web.Router("/manager/team/member/add", &controller.ManagerController{}, "POST:TeamMemberAdd")
	web.Router("/manager/team/member/delete", &controller.ManagerController{}, "POST:TeamMemberDelete")
	web.Router("/manager/team/member/change_role", &controller.ManagerController{}, "POST:TeamChangeMemberRole")
	web.Router("/manager/team/member/search", &controller.ManagerController{}, "*:TeamSearchMember")

	web.Router("/manager/team/book/list/:id", &controller.ManagerController{}, "*:TeamBookList")
	web.Router("/manager/team/book/add", &controller.ManagerController{}, "POST:TeamBookAdd")
	web.Router("/manager/team/book/delete", &controller.ManagerController{}, "POST:TeamBookDelete")
	web.Router("/manager/team/book/search", &controller.ManagerController{}, "*:TeamSearchBook")

	web.Router("/manager/itemsets", &controller.ManagerController{}, "*:Itemsets")
	web.Router("/manager/itemsets/edit", &controller.ManagerController{}, "post:ItemsetsEdit")
	web.Router("/manager/itemsets/delete", &controller.ManagerController{}, "post:ItemsetsDelete")
}
