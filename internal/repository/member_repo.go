package repository

import (
	"context"

	"git.itopcms.com/jackliu/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

// MemberRepo 抽象成员持久化。
type MemberRepo interface {
	Find(ctx context.Context, id int) (*model.Member, error)
}

type memberRepo struct {
	defaultOrm orm.Ormer
}

// NewMemberRepo 返回基于 beego/orm 的 MemberRepo。
func NewMemberRepo(o orm.Ormer) MemberRepo {
	return &memberRepo{defaultOrm: o}
}

func (r *memberRepo) ormer(ctx context.Context) orm.Ormer {
	if r.defaultOrm != nil {
		return r.defaultOrm
	}
	return OrmFromContext(ctx)
}

func (r *memberRepo) Find(ctx context.Context, id int) (*model.Member, error) {
	member := model.NewMember()
	o := r.ormer(ctx)
	if err := o.QueryTable(member.TableNameWithPrefix()).Filter("member_id", id).One(member); err != nil {
		return nil, err
	}
	member.ResolveRoleName()
	return member, nil
}
