package middleware

import (
	"encoding/json"
	"regexp"

	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/model"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

func FilterUser(ctx *context.Context) {
	_, ok := ctx.Input.Session(config.LoginSessionName).(model.Member)

	if !ok {
		if ctx.Input.IsAjax() {
			jsonData := make(map[string]any, 3)

			jsonData["errcode"] = 403
			jsonData["message"] = "请登录后再操作"

			returnJSON, _ := json.Marshal(jsonData)

			ctx.ResponseWriter.Write(returnJSON)
		} else {
			ctx.Redirect(302, config.URLFor("AccountController.Login"))
		}
	}
}

func StartRouter(ctx *context.Context) {
	sessname, _ := web.AppConfig.String("session::sessionname")
	sessionId := ctx.Input.Cookie(sessname)
	if sessionId != "" {
		//sessionId必须是数字字母组成，且最小32个字符，最大1024字符
		if ok, err := regexp.MatchString(`^[a-zA-Z0-9]{32,512}$`, sessionId); !ok || err != nil {
			panic("401")
		}
	}
}
