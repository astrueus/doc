package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.itopcms.com/astrueus/doc/internal/cache"
	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

// DocumentRepo 抽象文档持久化。
type DocumentRepo interface {
	Find(ctx context.Context, id int) (*model.Document, error)
	Save(ctx context.Context, d *model.Document) error
	UpdateMarkdownWithVersion(ctx context.Context, id int, expectVersion int64, markdown string, modifyAt int, newVersion int64) (int64, error)
	FindListByBookID(ctx context.Context, bookID int) ([]*model.Document, error)
	FindByIdentify(ctx context.Context, identify string, bookID int) (*model.Document, error)
	FindFirstByBookID(ctx context.Context, bookID int) (*model.Document, error)
	SearchLike(ctx context.Context, q string, bookIDs []int, limit int) ([]*model.Document, error)
	SearchToPager(ctx context.Context, keyword string, pageIndex, pageSize, memberID int) ([]*dto.DocumentSearchResult, int, error)
	SearchInBook(ctx context.Context, keyword string, bookID int) ([]*dto.DocumentSearchResult, error)
	FindHistoryToPager(ctx context.Context, docID, pageIndex, pageSize int) ([]*dto.DocumentHistorySimpleResult, int, error)
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

func (r *documentRepo) findFromDB(ctx context.Context, id int) (*model.Document, error) {
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

func (r *documentRepo) Find(ctx context.Context, id int) (*model.Document, error) {
	if a := documentAside(); a != nil {
		k := cacheKeys()
		v, err := a.GetOrLoad(ctx, k.DocumentByID(id), metaCacheOptions(k.TagDocument(id)), func(context.Context) (model.Document, error) {
			d, err := r.findFromDB(ctx, id)
			if err != nil {
				if errors.Is(err, model.ErrDataNotExist) || errors.Is(err, model.ErrInvalidParameter) {
					return model.Document{}, cache.ErrNotFound
				}
				return model.Document{}, err
			}
			d.AttachList = nil
			d.Lang = ""
			return *d, nil
		})
		if errors.Is(err, cache.ErrNotFound) {
			if id <= 0 {
				return nil, model.ErrInvalidParameter
			}
			return nil, model.ErrDataNotExist
		}
		if err != nil {
			return nil, err
		}
		out := v
		return localizeDocumentTimes(&out), nil
	}
	return r.findFromDB(ctx, id)
}

func (r *documentRepo) Save(ctx context.Context, d *model.Document) error {
	_ = ctx
	return d.InsertOrUpdate()
}

func (r *documentRepo) UpdateMarkdownWithVersion(ctx context.Context, id int, expectVersion int64, markdown string, modifyAt int, newVersion int64) (int64, error) {
	doc := model.NewDocument()
	o := r.ormer(ctx)
	aff, err := o.QueryTable(doc.TableNameWithPrefix()).
		Filter("document_id", id).
		Filter("version", expectVersion).
		Update(orm.Params{
			"markdown":  markdown,
			"version":   newVersion,
			"modify_at": modifyAt,
		})
	if err == nil && aff > 0 && cache.Kernel() != nil {
		if d, ferr := r.findFromDB(ctx, id); ferr == nil {
			InvalidateDocument(d)
		} else {
			InvalidateDocument(&model.Document{DocumentId: id})
		}
	}
	return aff, err
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

func (r *documentRepo) findByIdentifyDB(ctx context.Context, identify string, bookID int) (*model.Document, error) {
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

func (r *documentRepo) FindByIdentify(ctx context.Context, identify string, bookID int) (*model.Document, error) {
	if a := documentAside(); a != nil {
		k := cacheKeys()
		tags := []string{k.TagBook(bookID)}
		v, err := a.GetOrLoad(ctx, k.DocumentByIdentify(bookID, identify), metaCacheOptions(tags...), func(context.Context) (model.Document, error) {
			d, err := r.findByIdentifyDB(ctx, identify, bookID)
			if err != nil {
				if err == orm.ErrNoRows {
					return model.Document{}, cache.ErrNotFound
				}
				return model.Document{}, err
			}
			d.AttachList = nil
			d.Lang = ""
			return *d, nil
		})
		if errors.Is(err, cache.ErrNotFound) {
			return nil, orm.ErrNoRows
		}
		if err != nil {
			return nil, err
		}
		out := v
		return localizeDocumentTimes(&out), nil
	}
	return r.findByIdentifyDB(ctx, identify, bookID)
}

func (r *documentRepo) FindFirstByBookID(ctx context.Context, bookID int) (*model.Document, error) {
	doc := model.NewDocument()
	err := r.ormer(ctx).QueryTable(doc.TableNameWithPrefix()).
		Filter("book_id", bookID).Filter("parent_id", 0).OrderBy("order_sort").One(doc)
	if err != nil {
		return nil, err
	}
	return localizeDocumentTimes(doc), nil
}

func (r *documentRepo) SearchLike(ctx context.Context, q string, bookIDs []int, limit int) ([]*model.Document, error) {
	if limit <= 0 {
		limit = 10
	}
	q = strings.TrimSpace(q)
	if q == "" || len(bookIDs) == 0 {
		return []*model.Document{}, nil
	}

	like := "%" + strings.ReplaceAll(q, " ", "%") + "%"
	inClause, inArgs := intInClause(bookIDs)
	table := model.NewDocument().TableNameWithPrefix()
	sql := fmt.Sprintf(
		"SELECT * FROM %s WHERE book_id IN (%s) AND (document_name LIKE ? OR `release` LIKE ?) ORDER BY modify_time DESC LIMIT ?",
		table, inClause,
	)
	args := append(inArgs, like, like, limit)

	var docs []*model.Document
	_, err := r.ormer(ctx).Raw(sql, args...).QueryRows(&docs)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		localizeDocumentTimes(d)
	}
	return docs, nil
}

func (r *documentRepo) SearchInBook(ctx context.Context, keyword string, bookID int) ([]*dto.DocumentSearchResult, error) {
	sql := fmt.Sprintf("SELECT * FROM %s WHERE book_id = ? AND (document_name LIKE ? OR `release` LIKE ?) ", model.NewDocument().TableNameWithPrefix())
	keyword = "%" + keyword + "%"
	var docs []*dto.DocumentSearchResult
	_, err := r.ormer(ctx).Raw(sql, bookID, keyword, keyword).QueryRows(&docs)
	return docs, err
}

func (r *documentRepo) FindHistoryToPager(ctx context.Context, docID, pageIndex, pageSize int) ([]*dto.DocumentHistorySimpleResult, int, error) {
	offset := (pageIndex - 1) * pageSize
	p := config.GetDatabasePrefix()
	sql := fmt.Sprintf(`SELECT history.*,m1.account,m2.account as modify_name
FROM %sdocument_history AS history
LEFT JOIN %smembers AS m1 ON history.member_id = m1.member_id
LEFT JOIN %smembers AS m2 ON history.modify_at = m2.member_id
WHERE history.document_id = ? ORDER BY history.history_id DESC LIMIT ?,?;`, p, p, p)

	var docs []*dto.DocumentHistorySimpleResult
	_, err := r.ormer(ctx).Raw(sql, docID, offset, pageSize).QueryRows(&docs)
	if err != nil {
		return nil, 0, err
	}
	for _, doc := range docs {
		if doc != nil && !doc.ModifyTime.IsZero() {
			doc.ModifyTime = doc.ModifyTime.In(time.Local)
		}
	}
	count, err := r.ormer(ctx).QueryTable(model.NewDocumentHistory().TableNameWithPrefix()).Filter("document_id", docID).Count()
	if err != nil {
		return docs, 0, err
	}
	return docs, int(count), nil
}

func intInClause(ids []int) (string, []any) {
	if len(ids) == 0 {
		return "NULL", nil
	}
	parts := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		parts[i] = "?"
		args[i] = id
	}
	return strings.Join(parts, ","), args
}
