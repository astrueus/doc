package controller

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"git.itopcms.com/jackliu/doc/internal/mcp"
	"git.itopcms.com/jackliu/doc/internal/model"
	"github.com/beego/i18n"
)

type MemberApiTokenController struct {
	BaseController
}

func (c *MemberApiTokenController) Prepare() {
	c.BaseController.Prepare()
	if !c.isUserLoggedIn() {
		c.Abort("403")
	}
}

func (c *MemberApiTokenController) Index() {
	c.TplName = "member/api_tokens.tpl"
	list, err := model.NewMemberApiToken().ListByMember(c.Member.MemberId)
	if err != nil {
		c.ShowErrorPage(500, err.Error())
		return
	}
	c.Data["Tokens"] = list
	c.Data["Now"] = time.Now()
}

func (c *MemberApiTokenController) Create() {
	if !c.Ctx.Input.IsPost() {
		c.Abort("405")
		return
	}
	name := strings.TrimSpace(c.GetString("name"))
	if name == "" {
		c.JsonResult(6001, i18n.Tr(c.Lang, "message.param_error"))
		return
	}
	scopes := strings.TrimSpace(c.GetString("scopes"))
	if scopes == "" {
		scopes = "read,write"
	}
	expiresAt, err := parseAPITokenExpires(c.GetString("expires_at"))
	if err != nil {
		c.JsonResult(6002, "expires_at format invalid, use YYYY-MM-DD or leave empty")
		return
	}

	raw, err := randomAPITokenRaw(48)
	if err != nil {
		c.JsonResult(6003, "generate token failed")
		return
	}
	sum := sha256.Sum256([]byte(raw))
	token := model.NewMemberApiToken()
	token.MemberId = c.Member.MemberId
	token.TokenHash = hex.EncodeToString(sum[:])
	token.Name = name
	token.Scopes = scopes
	token.ExpiresAt = expiresAt
	token.CreatedAt = time.Now()

	if err := token.Insert(); err != nil {
		c.JsonResult(6004, err.Error())
		return
	}

	c.JsonResult(0, "ok", map[string]any{
		"token":    "doc_" + raw,
		"token_id": token.TokenId,
		"name":     token.Name,
		"scopes":   token.Scopes,
	})
}

func (c *MemberApiTokenController) Revoke() {
	if !c.Ctx.Input.IsPost() {
		c.Abort("405")
		return
	}
	id, err := c.GetInt(":id")
	if err != nil || id <= 0 {
		c.JsonResult(6001, i18n.Tr(c.Lang, "message.param_error"))
		return
	}
	token, err := model.NewMemberApiToken().FindByID(id)
	if err != nil || token.MemberId != c.Member.MemberId {
		c.JsonResult(6005, "token not found")
		return
	}
	if token.IsRevoked() {
		c.JsonResult(0, "ok")
		return
	}
	if err := token.Revoke(time.Now()); err != nil {
		c.JsonResult(6006, err.Error())
		return
	}
	mcp.InvalidateAPITokenCache(c.Ctx.Request.Context(), token.TokenHash)
	c.JsonResult(0, "ok")
}

func parseAPITokenExpires(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		// end of day
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local), nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func randomAPITokenRaw(n int) (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out), nil
}
