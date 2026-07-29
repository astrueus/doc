package controller

import (
	"git.itopcms.com/jackliu/doc/pkg/htmlutil"
	"strconv"
	"strings"

	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/errs"
	"git.itopcms.com/jackliu/doc/internal/model"
	"git.itopcms.com/jackliu/doc/pkg/pagination"
	"git.itopcms.com/jackliu/doc/pkg/sqltil"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/i18n"
)

type SearchController struct {
	BaseController
}

// 搜索首页
func (c *SearchController) Index() {
	c.Prepare()
	c.TplName = "search/index.tpl"

	//如果没有开启你们访问则跳转到登录
	if !c.EnableAnonymous && c.Member == nil {
		c.Redirect(config.URLFor("AccountController.Login"), 302)
		return
	}

	keyword := c.GetString("keyword")
	pageIndex, _ := c.GetInt("page", 1)

	c.Data["BaseUrl"] = c.BaseUrl()

	if keyword != "" {
		c.Data["Keyword"] = keyword
		memberId := 0
		if c.Member != nil {
			memberId = c.Member.MemberId
		}
		searchResult, totalCount, err := model.NewDocumentSearchResult().FindToPager(sqltil.EscapeLike(keyword), pageIndex, config.PageSize, memberId)

		if err != nil {
			logs.Error("搜索失败 ->", err)
			return
		}
		if totalCount > 0 {
			pager := pagination.NewPagination(c.Ctx.Request, totalCount, config.PageSize, c.BaseUrl())
			c.Data["PageHtml"] = pager.HtmlPages()
		} else {
			c.Data["PageHtml"] = ""
		}
		if len(searchResult) > 0 {
			keywords := strings.Split(keyword, " ")

			for _, item := range searchResult {
				for _, word := range keywords {
					item.DocumentName = strings.Replace(item.DocumentName, word, "<em>"+word+"</em>", -1)
					if item.Description != "" {
						src := item.Description

						r := []rune(htmlutil.StripTags(item.Description))

						if len(r) > 100 {
							src = string(r[:100])
						} else {
							src = string(r)
						}
						item.Description = strings.Replace(src, word, "<em>"+word+"</em>", -1)
					}
				}
				if item.Identify == "" {
					item.Identify = strconv.Itoa(item.DocumentId)
				}
				if item.ModifyTime.IsZero() {
					item.ModifyTime = item.CreateTime
				}
			}
		}
		c.Data["Lists"] = searchResult
	}
}

// 搜索用户
func (c *SearchController) User() {
	c.Prepare()
	key := c.Ctx.Input.Param(":key")
	keyword := strings.TrimSpace(c.GetString("q"))
	if key == "" || keyword == "" {
		c.JsonError(errs.New(errs.CodeInvalidParam, i18n.Tr(c.Lang, "message.param_error")))
		return
	}
	keyword = sqltil.EscapeLike(keyword)

	book, err := model.NewBookResult().FindByIdentify(key, c.Member.MemberId)
	if err != nil {
		if err == model.ErrPermissionDenied {
			c.JsonError(errs.New(errs.CodeForbidden, i18n.Tr(c.Lang, "message.no_permission")))
			return
		}
		c.JsonError(errs.New(errs.CodeNotFound, i18n.Tr(c.Lang, "message.item_not_exist")))
		return
	}

	//members, err := model.NewMemberRelationshipResult().FindNotJoinUsersByAccount(book.BookId, 10, "%"+keyword+"%")
	members, err := model.NewMemberRelationshipResult().FindNotJoinUsersByAccountOrRealName(book.BookId, 10, "%"+keyword+"%")
	if err != nil {
		logs.Error("查询用户列表出错：" + err.Error())
		c.JsonError(errs.Wrap(errs.CodeInternal, err.Error(), err))
		return
	}
	result := model.SelectMemberResult{}
	items := make([]model.KeyValueItem, 0)

	for _, member := range members {
		item := model.KeyValueItem{}
		item.Id = member.MemberId
		item.Text = member.Account + "[" + member.RealName + "]"
		items = append(items, item)
	}

	result.Result = items

	c.JsonResult(0, "OK", result)
}
