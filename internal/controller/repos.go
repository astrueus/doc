package controller

import (
	"context"

	"git.itopcms.com/astrueus/doc/internal/repository"
	"github.com/beego/beego/v2/client/orm"
)

func (c *BaseController) requestContext() context.Context {
	if c.Ctx != nil && c.Ctx.Request != nil {
		return c.Ctx.Request.Context()
	}
	return context.Background()
}

func documentRepo() repository.DocumentRepo {
	return repository.NewDocumentRepo(orm.NewOrm())
}

func bookRepo() repository.BookRepo {
	return repository.NewBookRepo(orm.NewOrm())
}

func memberRepo() repository.MemberRepo {
	return repository.NewMemberRepo(orm.NewOrm())
}

func attachmentRepo() repository.AttachmentRepo {
	return repository.NewAttachmentRepo(orm.NewOrm())
}

func blogRepo() repository.BlogRepo {
	return repository.NewBlogRepo(orm.NewOrm())
}
