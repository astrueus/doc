package repository

import (
	"context"

	"github.com/beego/beego/v2/client/orm"
)

type ormContextKey struct{}

// WithOrm 将 Ormer 写入 ctx，供调用链内共用同一 DB 会话。
func WithOrm(ctx context.Context, o orm.Ormer) context.Context {
	return context.WithValue(ctx, ormContextKey{}, o)
}

// OrmFromContext 从 ctx 取 Ormer；不存在时返回 orm.NewOrm()。
func OrmFromContext(ctx context.Context) orm.Ormer {
	if ctx != nil {
		if o, ok := ctx.Value(ormContextKey{}).(orm.Ormer); ok && o != nil {
			return o
		}
	}
	return orm.NewOrm()
}

// UnitOfWork 在逻辑单元内执行 fn；当前为轻量透传，事务可后续接入。
type UnitOfWork interface {
	Run(ctx context.Context, fn func(ctx context.Context) error) error
}

type unitOfWork struct{}

// NewUnitOfWork 返回 UnitOfWork 辅助对象。
func NewUnitOfWork() UnitOfWork {
	return &unitOfWork{}
}

func (u *unitOfWork) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	o := OrmFromContext(ctx)
	return fn(WithOrm(ctx, o))
}
