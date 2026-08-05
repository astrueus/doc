package repository

import (
	"context"

	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

// BookRepo 抽象项目（Book）持久化。
type BookRepo interface {
	Find(ctx context.Context, id int) (*model.Book, error)
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
