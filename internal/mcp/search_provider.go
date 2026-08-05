package mcp

import (
	"context"
	"fmt"
	"strings"

	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

type searchProvider interface {
	Search(ctx context.Context, q string, bookIDs []int, limit int) ([]*model.Document, error)
}

type sqlLikeProvider struct{}

func newSearchProvider() searchProvider {
	return &sqlLikeProvider{}
}

func (p *sqlLikeProvider) Search(_ context.Context, q string, bookIDs []int, limit int) ([]*model.Document, error) {
	if limit <= 0 {
		limit = 10
	}
	q = strings.TrimSpace(q)
	if q == "" || len(bookIDs) == 0 {
		return []*model.Document{}, nil
	}

	like := "%" + strings.ReplaceAll(q, " ", "%") + "%"
	inClause, inArgs := bookIDInClause(bookIDs)
	table := model.NewDocument().TableNameWithPrefix()
	sql := fmt.Sprintf(
		"SELECT * FROM %s WHERE book_id IN (%s) AND (document_name LIKE ? OR `release` LIKE ?) ORDER BY modify_time DESC LIMIT ?",
		table, inClause,
	)
	args := append(inArgs, like, like, limit)

	o := orm.NewOrm()
	var docs []*model.Document
	_, err := o.Raw(sql, args...).QueryRows(&docs)
	return docs, err
}
