package router

import (
	"git.itopcms.com/jackliu/doc/internal/controller"
	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/", &controller.HomeController{}, "*:Index")

	web.Router("/login", &controller.AccountController{}, "*:Login")
	web.Router("/dingtalk_login", &controller.AccountController{}, "*:DingTalkLogin")
	web.Router("/qrlogin/:app", &controller.AccountController{}, "*:QRLogin")
	web.Router("/logout", &controller.AccountController{}, "*:Logout")
	web.Router("/register", &controller.AccountController{}, "*:Register")
	web.Router("/find_password", &controller.AccountController{}, "*:FindPassword")
	web.Router("/valid_email", &controller.AccountController{}, "post:ValidEmail")
	web.Router("/captcha", &controller.AccountController{}, "*:Captcha")

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

	//web.Router("/manager/config",  &controller.ManagerController{}, "*:Config")

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

	web.Router("/setting", &controller.SettingController{}, "*:Index")
	web.Router("/setting/password", &controller.SettingController{}, "*:Password")
	web.Router("/setting/upload", &controller.SettingController{}, "*:Upload")

	web.Router("/book", &controller.BookController{}, "*:Index")
	web.Router("/book/:key/dashboard", &controller.BookController{}, "*:Dashboard")
	web.Router("/book/:key/setting", &controller.BookController{}, "*:Setting")
	web.Router("/book/:key/users", &controller.BookController{}, "*:Users")
	web.Router("/book/:key/release", &controller.BookController{}, "post:Release")
	web.Router("/book/:key/sort", &controller.BookController{}, "post:SaveSort")
	web.Router("/book/:key/teams", &controller.BookController{}, "*:Team")

	web.Router("/book/create", &controller.BookController{}, "*:Create")
	web.Router("/book/itemsets/search", &controller.BookController{}, "*:ItemsetsSearch")

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

	//管理文章的路由
	web.Router("/manage/blogs", &controller.BlogController{}, "*:ManageList")
	web.Router("/manage/blogs/setting/?:id", &controller.BlogController{}, "*:ManageSetting")
	web.Router("/manage/blogs/edit/?:id", &controller.BlogController{}, "*:ManageEdit")
	web.Router("/manage/blogs/delete", &controller.BlogController{}, "post:ManageDelete")
	web.Router("/manage/blogs/upload", &controller.BlogController{}, "post:Upload")
	web.Router("/manage/blogs/attach/:id", &controller.BlogController{}, "post:RemoveAttachment")

	//读文章的路由
	web.Router("/blogs", &controller.BlogController{}, "*:List")
	web.Router("/blog-attach/:id:int/:attach_id:int", &controller.BlogController{}, "get:Download")
	web.Router("/blog-:id([0-9]+).html", &controller.BlogController{}, "*:Index")

	//模板相关接口
	web.Router("/api/template/get", &controller.TemplateController{}, "get:Get")
	web.Router("/api/template/list", &controller.TemplateController{}, "post:List")
	web.Router("/api/template/add", &controller.TemplateController{}, "post:Add")
	web.Router("/api/template/remove", &controller.TemplateController{}, "post:Delete")

	web.Router("/api/attach/remove/", &controller.DocumentController{}, "post:RemoveAttachment")
	web.Router("/api/:key/edit/?:id", &controller.DocumentController{}, "*:Edit")
	web.Router("/api/upload", &controller.DocumentController{}, "post:Upload")
	web.Router("/api/:key/create", &controller.DocumentController{}, "post:Create")
	web.Router("/api/:key/delete", &controller.DocumentController{}, "post:Delete")
	web.Router("/api/:key/content/?:id", &controller.DocumentController{}, "*:Content")
	web.Router("/api/:key/compare/:id", &controller.DocumentController{}, "*:Compare")
	web.Router("/api/search/user/:key", &controller.SearchController{}, "*:User")

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

	web.Router("/search", &controller.SearchController{}, "get:Index")

	web.Router("/tag/:key", &controller.LabelController{}, "get:Index")
	web.Router("/tags", &controller.LabelController{}, "get:List")

	web.Router("/items", &controller.ItemsetsController{}, "get:Index")
	web.Router("/items/:key", &controller.ItemsetsController{}, "get:List")

}
