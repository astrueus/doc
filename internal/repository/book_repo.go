package repository

import (
	"context"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

// BookRepo 抽象项目（Book）持久化。
type BookRepo interface {
	Find(ctx context.Context, id int) (*model.Book, error)
	FindByIdentify(ctx context.Context, identify string) (*model.Book, error)
	Create(ctx context.Context, book *model.Book, lang string) error
	Update(ctx context.Context, book *model.Book, cols ...string) error
	IdentifiesByIDs(ctx context.Context, ids []int) (map[int]string, error)
	FindForRoleId(ctx context.Context, bookID, memberID int) (config.BookRole, error)
	ListAllIDs(ctx context.Context) ([]int, error)
	ListVisibleIDs(ctx context.Context, memberID int) ([]int, error)
	FindByIdentifyForMember(ctx context.Context, identify string, memberID int, lang string) (*dto.BookResult, error)
	FindToPagerForMember(ctx context.Context, pageIndex, pageSize, memberID int, lang string) ([]*dto.BookResult, int, error)
	FindToPagerAll(ctx context.Context, pageIndex, pageSize int) ([]*dto.BookResult, int, error)
	FindForHomeToPager(ctx context.Context, pageIndex, pageSize, memberID int) ([]*dto.BookResult, int, error)
	FindForLabelToPager(ctx context.Context, keyword string, pageIndex, pageSize, memberID int) ([]*dto.BookResult, int, error)
	FindToPagerByItemKey(ctx context.Context, key string, pageIndex, pageSize, memberID int) ([]*dto.BookResult, int, error)
	BookToResult(ctx context.Context, book model.Book) *dto.BookResult
}

type bookRepo struct {
	defaultOrm orm.Ormer
}

// NewBookRepo 返回基于 beego/orm 的 BookRepo。
func NewBookRepo(o orm.Ormer) BookRepo {
	return &bookRepo{defaultOrm: o}
}

func (r *bookRepo) ormer(ctx context.Context) orm.Ormer {
	if r.defaultOrm != nil {
		return r.defaultOrm
	}
	return OrmFromContext(ctx)
}

func (r *bookRepo) Find(ctx context.Context, id int) (*model.Book, error) {
	if id <= 0 {
		return nil, model.ErrInvalidParameter
	}
	book := model.NewBook()
	o := r.ormer(ctx)
	err := o.QueryTable(book.TableNameWithPrefix()).Filter("book_id", id).One(book)
	if err != nil {
		return nil, err
	}
	return book, nil
}

func (r *bookRepo) FindByIdentify(ctx context.Context, identify string) (*model.Book, error) {
	if identify == "" {
		return nil, model.ErrInvalidParameter
	}
	book := model.NewBook()
	o := r.ormer(ctx)
	err := o.QueryTable(book.TableNameWithPrefix()).Filter("identify", identify).One(book)
	if err == orm.ErrNoRows {
		return nil, model.ErrDataNotExist
	}
	if err != nil {
		return nil, err
	}
	return book, nil
}

func (r *bookRepo) Create(ctx context.Context, book *model.Book, lang string) error {
	_ = ctx
	return book.Insert(lang)
}

func (r *bookRepo) Update(ctx context.Context, book *model.Book, cols ...string) error {
	_ = ctx
	return book.Update(cols...)
}

func (r *bookRepo) IdentifiesByIDs(ctx context.Context, ids []int) (map[int]string, error) {
	out := make(map[int]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	seen := make([]int, 0, len(ids))
	uniq := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := uniq[id]; ok {
			continue
		}
		uniq[id] = struct{}{}
		seen = append(seen, id)
	}
	if len(seen) == 0 {
		return out, nil
	}

	var books []*model.Book
	tbl := model.NewBook()
	_, err := r.ormer(ctx).QueryTable(tbl.TableNameWithPrefix()).Filter("book_id__in", seen).All(&books, "book_id", "identify")
	if err != nil {
		return nil, err
	}
	for _, item := range books {
		if item == nil {
			continue
		}
		out[item.BookId] = item.Identify
	}
	return out, nil
}
