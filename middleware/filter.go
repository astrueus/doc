package middleware

import (
	"encoding/json"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/models"
	"regexp"
)

func FilterUser(ctx *context.Context) {
	_, ok := ctx.Input.Session(conf.LoginSessionName).(models.Member)

	if !ok {
		if ctx.Input.IsAjax() {
			jsonData := make(map[string]interface{}, 3)

			jsonData["errcode"] = 403
			jsonData["message"] = "请登录后再操作"

			returnJSON, _ := json.Marshal(jsonData)

			ctx.ResponseWriter.Write(returnJSON)
		} else {
			ctx.Redirect(302, conf.URLFor("AccountController.Login"))
		}
	}
}

func StartRouter(ctx *context.Context) {
	sessname, _ := web.AppConfig.String("sessionname")
	sessionId := ctx.Input.Cookie(sessname)
	if sessionId != "" {
		//sessionId必须是数字字母组成，且最小32个字符，最大1024字符
		if ok, err := regexp.MatchString(`^[a-zA-Z0-9]{32,512}$`, sessionId); !ok || err != nil {
			panic("401")
		}
	}
}