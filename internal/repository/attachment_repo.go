package repository

import (
	"context"
	"strings"

	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/model"
	"git.itopcms.com/astrueus/doc/pkg/filetil"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// AttachmentRepo 抽象附件查询（展示结构走 dto）。
type AttachmentRepo interface {
	FindResult(ctx context.Context, id int) (*dto.AttachmentResult, error)
	FindToPager(ctx context.Context, pageIndex, pageSize int) ([]*dto.AttachmentResult, int, error)
}

type attachmentRepo struct {
	defaultOrm orm.Ormer
}

// NewAttachmentRepo 返回基于 beego/orm 的 AttachmentRepo。
func NewAttachmentRepo(o orm.Ormer) AttachmentRepo {
	return &attachmentRepo{defaultOrm: o}
}

func (r *attachmentRepo) ormer(ctx context.Context) orm.Ormer {
	if r.defaultOrm != nil {
		return r.defaultOrm
	}
	return OrmFromContext(ctx)
}

func copyAttachmentFields(dst *dto.AttachmentResult, src *model.Attachment) {
	if dst == nil || src == nil {
		return
	}
	dst.AttachmentId = src.AttachmentId
	dst.BookId = src.BookId
	dst.DocumentId = src.DocumentId
	dst.FileName = src.FileName
	dst.FilePath = src.FilePath
	dst.FileSize = src.FileSize
	dst.HttpPath = src.HttpPath
	dst.FileExt = src.FileExt
	dst.CreateTime = src.CreateTime
	dst.CreateAt = src.CreateAt
}

func (r *attachmentRepo) fillAttachmentNames(o orm.Ormer, m *dto.AttachmentResult) {
	if m.BookId == 0 && m.DocumentId > 0 {
		blog := model.NewBlog()
		if err := o.QueryTable(blog.TableNameWithPrefix()).Filter("blog_id", m.DocumentId).One(blog, "blog_title"); err == nil {
			m.BookName = blog.BlogTitle
		} else {
			m.BookName = "[文章不存在]"
		}
		return
	}
	book := model.NewBook()
	if e := o.QueryTable(book.TableNameWithPrefix()).Filter("book_id", m.BookId).One(book, "book_name"); e == nil {
		m.BookName = book.BookName
	} else {
		m.BookName = "[不存在]"
	}
	doc := model.NewDocument()
	if e := o.QueryTable(doc.TableNameWithPrefix()).Filter("document_id", m.DocumentId).One(doc, "document_name"); e == nil {
		m.DocumentName = doc.DocumentName
	} else {
		m.DocumentName = "[不存在]"
	}
}

func (r *attachmentRepo) FindResult(ctx context.Context, id int) (*dto.AttachmentResult, error) {
	o := r.ormer(ctx)
	attach := model.NewAttachment()
	m := dto.NewAttachmentResult()
	err := o.QueryTable(attach.TableNameWithPrefix()).Filter("attachment_id", id).One(attach)
	if err != nil {
		return m, err
	}
	copyAttachmentFields(m, attach)
	r.fillAttachmentNames(o, m)
	if attach.CreateAt > 0 {
		member := model.NewMember()
		if e := o.QueryTable(member.TableNameWithPrefix()).Filter("member_id", attach.CreateAt).One(member, "account"); e == nil {
			m.Account = member.Account
		}
	}
	m.FileShortSize = filetil.FormatBytes(int64(attach.FileSize))
	m.LocalHttpPath = strings.Replace(m.FilePath, "\\", "/", -1)
	return m, nil
}

func (r *attachmentRepo) FindToPager(ctx context.Context, pageIndex, pageSize int) ([]*dto.AttachmentResult, int, error) {
	o := r.ormer(ctx)
	tbl := model.NewAttachment()
	total, err := o.QueryTable(tbl.TableNameWithPrefix()).Count()
	if err != nil {
		return nil, 0, err
	}
	totalCount := int(total)
	offset := (pageIndex - 1) * pageSize
	var list []*model.Attachment
	_, err = o.QueryTable(tbl.TableNameWithPrefix()).OrderBy("-attachment_id").Offset(offset).Limit(pageSize).All(&list)
	if err != nil {
		if err == orm.ErrNoRows {
			logs.Info("没有查到附件 ->", err)
			err = nil
		}
		return nil, totalCount, err
	}

	attachList := make([]*dto.AttachmentResult, 0, len(list))
	for _, item := range list {
		attach := dto.NewAttachmentResult()
		copyAttachmentFields(attach, item)
		attach.FileShortSize = filetil.FormatBytes(int64(attach.FileSize))
		if item.BookId == 0 && item.DocumentId > 0 {
			blog := model.NewBlog()
			if err := o.QueryTable(blog.TableNameWithPrefix()).Filter("blog_id", item.DocumentId).One(blog, "blog_title"); err == nil {
				attach.BookName = blog.BlogTitle
			} else {
				attach.BookName = "[文章不存在]"
			}
		} else {
			book := model.NewBook()
			if e := o.QueryTable(book.TableNameWithPrefix()).Filter("book_id", item.BookId).One(book, "book_name"); e == nil {
				attach.BookName = book.BookName
				doc := model.NewDocument()
				if e := o.QueryTable(doc.TableNameWithPrefix()).Filter("document_id", item.DocumentId).One(doc, "document_name"); e == nil {
					attach.DocumentName = doc.DocumentName
				} else {
					attach.DocumentName = "[文档不存在]"
				}
			} else {
				attach.BookName = "[项目不存在]"
			}
		}
		attach.LocalHttpPath = strings.Replace(item.FilePath, "\\", "/", -1)
		attachList = append(attachList, attach)
	}
	return attachList, totalCount, nil
}
