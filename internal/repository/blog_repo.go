package repository

import (
	"context"
	"errors"
	"time"

	"git.itopcms.com/astrueus/doc/internal/cache"
	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

// BlogRepo 博客持久化（阅读路径走 GetOrLoad）。
type BlogRepo interface {
	Find(ctx context.Context, id int) (*model.Blog, error)
}

type blogRepo struct {
	defaultOrm orm.Ormer
}

// NewBlogRepo 返回基于 beego/orm 的 BlogRepo。
func NewBlogRepo(o orm.Ormer) BlogRepo {
	return &blogRepo{defaultOrm: o}
}

func (r *blogRepo) ormer(ctx context.Context) orm.Ormer {
	if r.defaultOrm != nil {
		return r.defaultOrm
	}
	return OrmFromContext(ctx)
}

func localizeBlogTimes(b *model.Blog) *model.Blog {
	if b == nil {
		return b
	}
	if !b.Created.IsZero() {
		b.Created = b.Created.In(time.Local)
	}
	if !b.Modified.IsZero() {
		b.Modified = b.Modified.In(time.Local)
	}
	return b
}

func (r *blogRepo) findFromDB(ctx context.Context, id int) (*model.Blog, error) {
	b := model.NewBlog()
	err := r.ormer(ctx).QueryTable(b.TableNameWithPrefix()).Filter("blog_id", id).One(b)
	if err != nil {
		return nil, err
	}
	if b.BlogType == 1 || b.MemberId > 0 || b.ModifyAt > 0 {
		if linked, lerr := b.Link(); lerr == nil {
			return linked, nil
		}
	}
	return localizeBlogTimes(b), nil
}

func (r *blogRepo) Find(ctx context.Context, id int) (*model.Blog, error) {
	if id <= 0 {
		return nil, model.ErrInvalidParameter
	}
	if a := blogAside(); a != nil {
		k := cacheKeys()
		v, err := a.GetOrLoad(ctx, k.BlogByID(id), metaCacheOptions(k.TagBlog(id)), func(context.Context) (model.Blog, error) {
			b, err := r.findFromDB(ctx, id)
			if err != nil {
				if err == orm.ErrNoRows {
					return model.Blog{}, cache.ErrNotFound
				}
				return model.Blog{}, err
			}
			b.AttachList = nil
			return *b, nil
		})
		if errors.Is(err, cache.ErrNotFound) {
			return nil, orm.ErrNoRows
		}
		if err != nil {
			return nil, err
		}
		out := v
		return localizeBlogTimes(&out), nil
	}
	return r.findFromDB(ctx, id)
}
