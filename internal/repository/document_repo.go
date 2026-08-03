package repository

import (
	"context"
	"time"

	"git.itopcms.com/jackliu/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

// DocumentRepo 抽象文档持久化。
type DocumentRepo interface {
	Find(ctx context.Context, id int) (*model.Document, error)
	Save(ctx context.Context, d *model.Document) error
	UpdateMarkdownWithVersion(ctx context.Context, id int, expectVersion int64, markdown string, modifyAt int, newVersion int64) (int64, error)
	FindListByBookID(ctx context.Context, bookID int) ([]*model.Document, error)
	FindByIdentify(ctx context.Context, identify string, bookID int) (*model.Document, error)
}

type documentRepo struct {
	defaultOrm orm.Ormer
}

// NewDocumentRepo 返回基于 beego/orm 的 DocumentRepo；传入 nil 时使用 OrmFromContext / 默认库。
func NewDocumentRepo(o orm.Ormer) DocumentRepo {
	return &documentRepo{defaultOrm: o}
}

func (r *documentRepo) ormer(ctx context.Context) orm.Ormer {
	if r.defaultOrm != nil {
		return r.defaultOrm
	}
	return OrmFromContext(ctx)
}

func localizeDocumentTimes(d *model.Document) *model.Document {
	if d == nil {
		return d
	}
	if !d.CreateTime.IsZero() {
		d.CreateTime = d.CreateTime.In(time.Local)
	}
	if !d.ModifyTime.IsZero() {
		d.ModifyTime = d.ModifyTime.In(time.Local)
	}
	return d
}

func (r *documentRepo) Find(ctx context.Context, id int) (*model.Document, error) {
	if id <= 0 {
		return nil, model.ErrInvalidParameter
	}
	doc := model.NewDocument()
	o := r.ormer(ctx)
	err := o.QueryTable(doc.TableNameWithPrefix()).Filter("document_id", id).One(doc)
	if err == orm.ErrNoRows {
		return nil, model.ErrDataNotExist
	}
	if err != nil {
		return nil, err
	}
	return localizeDocumentTimes(doc), nil
}

func (r *documentRepo) Save(ctx context.Context, d *model.Document) error {
	_ = ctx
	return d.InsertOrUpdate()
}

func (r *documentRepo) UpdateMarkdownWithVersion(ctx context.Context, id int, expectVersion int64, markdown string, modifyAt int, newVersion int64) (int64, error) {
	doc := model.NewDocument()
	o := r.ormer(ctx)
	return o.QueryTable(doc.TableNameWithPrefix()).
		Filter("document_id", id).
		Filter("version", expectVersion).
		Update(orm.Params{
			"markdown":  markdown,
			"version":   newVersion,
			"modify_at": modifyAt,
		})
}

func (r *documentRepo) FindListByBookID(ctx context.Context, bookID int) ([]*model.Document, error) {
	doc := model.NewDocument()
	var docs []*model.Document
	o := r.ormer(ctx)
	_, err := o.QueryTable(doc.TableNameWithPrefix()).Filter("book_id", bookID).OrderBy("order_sort").All(&docs)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		localizeDocumentTimes(d)
	}
	return docs, nil
}

func (r *documentRepo) FindByIdentify(ctx context.Context, identify string, bookID int) (*model.Document, error) {
	doc := model.NewDocument()
	o := r.ormer(ctx)
	err := o.QueryTable(doc.TableNameWithPrefix()).Filter("book_id", bookID).Filter("identify", identify).One(doc)
	if err == orm.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return localizeDocumentTimes(doc), nil
}
