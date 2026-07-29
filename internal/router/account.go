package router

import (
	"git.itopcms.com/jackliu/doc/internal/controller"
	"github.com/beego/beego/v2/server/web"
)

func registerAccount() {
	web.Router("/login", &controller.AccountController{}, "*:Login")
	web.Router("/dingtalk_login", &controller.AccountController{}, "*:DingTalkLogin")
	web.Router("/qrlogin/:app", &controller.AccountController{}, "*:QRLogin")
	web.Router("/logout", &controller.AccountController{}, "*:Logout")
	web.Router("/register", &controller.AccountController{}, "*:Register")
	web.Router("/find_password", &controller.AccountController{}, "*:FindPassword")
	web.Router("/valid_email", &controller.AccountController{}, "post:ValidEmail")
	web.Router("/captcha", &controller.AccountController{}, "*:Captcha")

	web.Router("/setting", &controller.SettingController{}, "*:Index")
	web.Router("/setting/password", &controller.SettingController{}, "*:Password")
	web.Router("/setting/upload", &controller.SettingController{}, "*:Upload")
}
