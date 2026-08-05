package mcp

import (
	"git.itopcms.com/astrueus/doc/internal/repository"
	"github.com/beego/beego/v2/client/orm"
)

func documentRepo() repository.DocumentRepo {
	return repository.NewDocumentRepo(orm.NewOrm())
}
