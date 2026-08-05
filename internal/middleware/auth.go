package middleware

import (
	"encoding/json"
	"net/url"

	"git.itopcms.com/astrueus/doc/internal/auth"
	"git.itopcms.com/astrueus/doc/internal/config"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

// FilterUser 要求已登录；未登录时带 ?url= 回跳登录页。
func FilterUser(ctx *context.Context) {
	if id, ok := auth.MemberIDFromSession(ctx.Input.Session(config.LoginSessionName)); ok && id > 0 {
		return
	}

	if ctx.Input.IsAjax() {
		jsonData := map[string]any{
			"errcode": 403,
			"message": "请登录后再操作",
		}
		returnJSON, _ := json.Marshal(jsonData)
		ctx.ResponseWriter.Write(returnJSON)
		return
	}

	ctx.Redirect(302, config.URLFor("AccountController.Login")+"?url="+url.PathEscape(config.BaseUrl+ctx.Request.URL.RequestURI()))
}

func registerAuth() {
	web.InsertFilter("/manager", web.BeforeRouter, FilterUser)
	web.InsertFilter("/manager/*", web.BeforeRouter, FilterUser)
	web.InsertFilter("/setting", web.BeforeRouter, FilterUser)
	web.InsertFilter("/setting/*", web.BeforeRouter, FilterUser)
	web.InsertFilter("/member/api-tokens", web.BeforeRouter, FilterUser)
	web.InsertFilter("/member/api-tokens/*", web.BeforeRouter, FilterUser)
	web.InsertFilter("/book", web.BeforeRouter, FilterUser)
	web.InsertFilter("/book/*", web.BeforeRouter, FilterUser)
	web.InsertFilter("/api/*", web.BeforeRouter, FilterUser)
	web.InsertFilter("/manage/*", web.BeforeRouter, FilterUser)
}
