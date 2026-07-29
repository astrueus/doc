package middleware

import (
	"regexp"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

// StartRouter validates session cookie shape before static handlers.
func StartRouter(ctx *context.Context) {
	sessname, _ := web.AppConfig.String("session::sessionname")
	sessionId := ctx.Input.Cookie(sessname)
	if sessionId == "" {
		return
	}
	// sessionId must be alphanumeric, length 32..512
	if ok, err := regexp.MatchString(`^[a-zA-Z0-9]{32,512}$`, sessionId); !ok || err != nil {
		panic("401")
	}
}

func registerSession() {
	web.InsertFilter("/*", web.BeforeStatic, StartRouter, web.WithReturnOnOutput(false))
}
