package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"html/template"

	"git.itopcms.com/astrueus/doc/internal/auth"
	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/errs"
	"git.itopcms.com/astrueus/doc/internal/i18n"
	"git.itopcms.com/astrueus/doc/internal/model"
	"git.itopcms.com/astrueus/doc/pkg/gob"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

type BaseController struct {
	web.Controller
	Member                *model.Member
	Option                map[string]string
	EnableAnonymous       bool
	EnableDocumentHistory bool
	Lang                  string
}

type CookieRemember struct {
	MemberId int
	Account  string
	Time     time.Time
}

// Prepare 预处理.
func (c *BaseController) Prepare() {
	c.Data["SiteName"] = "Doc"
	c.Data["Member"] = model.NewMember()
	controller, action := c.GetControllerAndAction()

	c.Data["ActionName"] = action
	c.Data["ControllerName"] = controller

	c.EnableAnonymous = false
	c.EnableDocumentHistory = false

	if memberID, ok := auth.MemberIDFromSession(c.GetSession(config.LoginSessionName)); ok && memberID > 0 {
		if member, err := model.NewMember().Find(memberID); err == nil && member != nil && member.MemberId > 0 {
			c.Member = member
			c.Data["Member"] = c.Member
		}
	} else {
		var remember CookieRemember
		//如果Cookie中存在登录信息，从cookie中获取用户信息
		if cookie, ok := c.GetSecureCookie(config.GetAppKey(), "login"); ok {
			if err := gob.Decode(cookie, &remember); err == nil {
				if member, err := model.NewMember().Find(remember.MemberId); err == nil {
					c.Member = member
					c.Data["Member"] = member
					c.SetMember(*member)
				}
			}
		}
	}
	config.BaseUrl = c.BaseUrl()
	c.Data["BaseUrl"] = c.BaseUrl()

	c.Option = loadOptions()
	for k, v := range c.Option {
		c.Data[k] = v
	}
	c.EnableAnonymous = strings.EqualFold(c.Option["ENABLE_ANONYMOUS"], "true")
	c.EnableDocumentHistory = strings.EqualFold(c.Option["ENABLE_DOCUMENT_HISTORY"], "true")
	c.Data["HighlightStyle"] = config.MustGlobal().App.HighlightStyle

	if b, err := os.ReadFile(filepath.Join(web.BConfig.WebConfig.ViewsPath, "widgets", "scripts.tpl")); err == nil {
		c.Data["Scripts"] = template.HTML(string(b))
	}

	c.SetLang()
}

// 判断用户是否登录.
func (c *BaseController) isUserLoggedIn() bool {
	return c.Member != nil && c.Member.MemberId > 0
}

// SetMember 获取或设置当前登录用户信息,如果 MemberId 小于 0 则标识删除 Session。
// Session 只存 member_id（int），避免 gob 序列化结构体随包路径失效。
func (c *BaseController) SetMember(member model.Member) {

	if member.MemberId <= 0 {
		c.DelSession(config.LoginSessionName)
		c.DelSession("uid")
		c.DestroySession()
	} else {
		c.SetSession(config.LoginSessionName, member.MemberId)
		c.SetSession("uid", member.MemberId)
	}
}

// JsonResult 响应 json 结果
func (c *BaseController) JsonResult(errCode int, errMsg string, data ...any) {
	jsonData := make(map[string]any, 3)

	jsonData["errcode"] = errCode
	jsonData["message"] = errMsg

	if len(data) > 0 && data[0] != nil {
		jsonData["data"] = data[0]
	}

	returnJSON, err := json.Marshal(jsonData)
	if err != nil {
		logs.Error(err)
	}

	c.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Ctx.ResponseWriter.Header().Set("Cache-Control", "no-cache, no-store")
	_, err = io.WriteString(c.Ctx.ResponseWriter, string(returnJSON))
	if err != nil {
		logs.Error(err)
	}

	c.StopRun()
}

// JsonError writes a BizError (or a generic internal error) as JSON and stops the request.
func (c *BaseController) JsonError(err error) {
	if b, ok := errs.AsBiz(err); ok {
		c.JsonResult(b.Code, b.Msg)
		return
	}
	logs.Error("unhandled error:", err)
	c.JsonResult(errs.CodeInternal, "系统内部错误")
}

// 如果错误不为空，则响应错误信息到浏览器.
func (c *BaseController) CheckJsonError(code int, err error) {

	if err == nil {
		return
	}
	jsonData := make(map[string]any, 3)

	jsonData["errcode"] = code
	jsonData["message"] = err.Error()

	returnJSON, err := json.Marshal(jsonData)
	if err != nil {
		logs.Error(err)
	}

	c.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Ctx.ResponseWriter.Header().Set("Cache-Control", "no-cache, no-store")
	_, err = io.WriteString(c.Ctx.ResponseWriter, string(returnJSON))
	if err != nil {
		logs.Error(err)
	}

	c.StopRun()
}

// ExecuteViewPathTemplate 执行指定的模板并返回执行结果.
func (c *BaseController) ExecuteViewPathTemplate(tplName string, data any) (string, error) {
	var buf bytes.Buffer

	viewPath := c.ViewPath

	if c.ViewPath == "" {
		viewPath = web.BConfig.WebConfig.ViewsPath

	}

	if err := web.ExecuteViewPathTemplate(&buf, tplName, viewPath, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *BaseController) BaseUrl() string {
	baseUrl := config.MustGlobal().BaseURL
	if baseUrl != "" {
		if strings.HasSuffix(baseUrl, "/") {
			baseUrl = strings.TrimSuffix(baseUrl, "/")
		}
	} else {
		baseUrl = c.Ctx.Input.Scheme() + "://" + c.Ctx.Request.Host
	}
	return baseUrl
}

// 显示错误信息页面.
func (c *BaseController) ShowErrorPage(errCode int, errMsg string) {
	c.TplName = "errors/error.tpl"

	c.Data["ErrorMessage"] = errMsg
	c.Data["ErrorCode"] = errCode

	lang := c.Lang
	if lang == "" {
		lang = config.GetDefaultLang()
	}

	var buf bytes.Buffer

	if err := web.ExecuteViewPathTemplate(&buf, "errors/error.tpl", web.BConfig.WebConfig.ViewsPath, map[string]any{
		"ErrorMessage":      errMsg,
		"ErrorCode":         errCode,
		"BaseUrl":           config.BaseUrl,
		"Lang":              lang,
		"site_title_suffix": c.Data["site_title_suffix"],
	}); err != nil {
		c.Abort("500")
	}
	if errCode >= 200 && errCode <= 510 {
		c.CustomAbort(errCode, buf.String())
	} else {
		c.CustomAbort(500, buf.String())
	}
}

func (c *BaseController) CheckErrorResult(code int, err error) {
	if err != nil {
		c.ShowErrorPage(code, err.Error())
	}
}

func (c *BaseController) SetLang() {
	hasCookie := false
	lang := c.GetString("lang")
	if len(lang) == 0 {
		lang = c.Ctx.GetCookie("lang")
		hasCookie = true
	}
	if len(lang) == 0 ||
		!i18n.IsExist(lang) {
		lang = config.GetDefaultLang()
	}
	if !hasCookie {
		c.Ctx.SetCookie("lang", lang, 1<<31-1, "/")
	}
	c.Data["Lang"] = lang
	c.Lang = lang
}
