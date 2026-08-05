package middleware

import (
	"git.itopcms.com/astrueus/doc/internal/config"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

// FinishRouter attaches response security / product headers.
func FinishRouter(ctx *context.Context) {
	ctx.ResponseWriter.Header().Add("TopDoc-Version", config.VERSION)
	ctx.ResponseWriter.Header().Add("TopDoc-Site", "https://doc.itopcms.com")
	ctx.ResponseWriter.Header().Add("X-XSS-Protection", "1; mode=block")
}

func registerHeaders() {
	web.InsertFilter("/*", web.BeforeRouter, FinishRouter, web.WithReturnOnOutput(false))
}
