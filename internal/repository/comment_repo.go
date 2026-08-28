package repository

import (
	"context"
	"fmt"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

// CommentRepo 抽象评论展示查询。Web 评论页目前为空实现，本方法供 T8 迁出后调用。
type CommentRepo interface {
	FindForDocumentToPager(ctx context.Context, docID, pageIndex, pageSize int) ([]*dto.CommentResult, int, error)
}

type commentRepo struct {
	defaultOrm orm.Ormer
}

// NewCommentRepo 返回基于 beego/orm 的 CommentRepo。
func NewCommentRepo(o orm.Ormer) CommentRepo {
	return &commentRepo{defaultOrm: o}
}

func (r *commentRepo) ormer(ctx context.Context) orm.Ormer {
	if r.defaultOrm != nil {
		return r.defaultOrm
	}
	return OrmFromContext(ctx)
}

func (r *commentRepo) FindForDocumentToPager(ctx context.Context, docID, pageIndex, pageSize int) ([]*dto.CommentResult, int, error) {
	o := r.ormer(ctx)
	p := config.GetDatabasePrefix()
	sql1 := fmt.Sprintf(`
SELECT
  comment.* ,
  parent.* ,
  member.account AS author,
  p_member.account AS reply_account
FROM %scomments AS comment
  LEFT JOIN %smembers AS member ON comment.member_id = member.member_id
  LEFT JOIN %scomments AS parent ON comment.parent_id = parent.comment_id
  LEFT JOIN %smembers AS p_member ON p_member.member_id = parent.member_id

WHERE comment.document_id = ? ORDER BY comment.comment_id DESC LIMIT 0,10`, p, p, p, p)

	offset := (pageIndex - 1) * pageSize
	var comments []*dto.CommentResult
	_, err := o.Raw(sql1, docID, offset, pageSize).QueryRows(&comments)
	if err != nil {
		return nil, 0, err
	}

	v, err := o.QueryTable(model.NewComment().TableNameWithPrefix()).Filter("document_id", docID).Count()
	totalCount := 0
	if err == nil {
		totalCount = int(v)
	}
	return comments, totalCount, err
}
