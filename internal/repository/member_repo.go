package repository

import (
	"context"
	"fmt"
	"time"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

// MemberRepo 抽象成员持久化。
type MemberRepo interface {
	Find(ctx context.Context, id int) (*model.Member, error)
	FindByAccount(ctx context.Context, account string) (*model.Member, error)
	FindAPITokenByHash(ctx context.Context, hash string) (*model.MemberApiToken, error)
	TouchAPITokenLastUsed(ctx context.Context, tokenID int, ip string) error
	// ResolveAPIToken 按 token 哈希解析成员；Aside 开启时走 GetOrLoad。
	ResolveAPIToken(ctx context.Context, tokenHash string) (*APITokenIdentity, error)
	FindForUsersByBookID(ctx context.Context, lang string, bookID, pageIndex, pageSize int) ([]*dto.MemberRelationshipResult, int, error)
	FindNotJoinUsersByAccount(ctx context.Context, bookID, limit int, account string) ([]*model.Member, error)
	FindNotJoinUsersByAccountOrRealName(ctx context.Context, bookID, limit int, keyWord string) ([]*model.Member, error)
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

func (r *memberRepo) FindByAccount(ctx context.Context, account string) (*model.Member, error) {
	member := model.NewMember()
	err := r.ormer(ctx).QueryTable(member.TableNameWithPrefix()).Filter("account", account).One(member)
	if err != nil {
		return nil, err
	}
	member.ResolveRoleName()
	return member, nil
}

func (r *memberRepo) FindAPITokenByHash(ctx context.Context, hash string) (*model.MemberApiToken, error) {
	token := model.NewMemberApiToken()
	err := r.ormer(ctx).QueryTable(token.TableNameWithPrefix()).Filter("token_hash", hash).One(token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (r *memberRepo) TouchAPITokenLastUsed(ctx context.Context, tokenID int, ip string) error {
	_, err := r.ormer(ctx).QueryTable(model.NewMemberApiToken().TableNameWithPrefix()).
		Filter("token_id", tokenID).
		Update(orm.Params{
			"last_used_at": time.Now(),
			"last_used_ip": ip,
		})
	return err
}

func (r *memberRepo) FindForUsersByBookID(ctx context.Context, lang string, bookID, pageIndex, pageSize int) ([]*dto.MemberRelationshipResult, int, error) {
	o := r.ormer(ctx)
	p := config.GetDatabasePrefix()
	sql1 := fmt.Sprintf("SELECT * FROM %srelationship AS rel LEFT JOIN %smembers as member ON rel.member_id = member.member_id WHERE rel.book_id = ? ORDER BY rel.relationship_id DESC  LIMIT ?,?", p, p)
	sql2 := fmt.Sprintf("SELECT count(*) AS total_count FROM %srelationship AS rel LEFT JOIN %smembers as member ON rel.member_id = member.member_id WHERE rel.book_id = ?", p, p)

	var totalCount int
	err := o.Raw(sql2, bookID).QueryRow(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	offset := (pageIndex - 1) * pageSize
	var members []*dto.MemberRelationshipResult
	_, err = o.Raw(sql1, bookID, offset, pageSize).QueryRows(&members)
	if err != nil {
		return nil, 0, err
	}
	for _, item := range members {
		item.ResolveRoleName(lang)
	}
	return members, totalCount, nil
}

func (r *memberRepo) FindNotJoinUsersByAccount(ctx context.Context, bookID, limit int, account string) ([]*model.Member, error) {
	p := config.GetDatabasePrefix()
	sql := fmt.Sprintf("SELECT m.* FROM %smembers as m LEFT JOIN %srelationship as rel ON m.member_id=rel.member_id AND rel.book_id = ? WHERE rel.relationship_id IS NULL AND m.account LIKE ? LIMIT 0,?;", p, p)
	var members []*model.Member
	_, err := r.ormer(ctx).Raw(sql, bookID, account, limit).QueryRows(&members)
	return members, err
}

func (r *memberRepo) FindNotJoinUsersByAccountOrRealName(ctx context.Context, bookID, limit int, keyWord string) ([]*model.Member, error) {
	p := config.GetDatabasePrefix()
	sql := fmt.Sprintf("SELECT m.* FROM %smembers as m LEFT JOIN %srelationship as rel ON rel.member_id = m.member_id AND rel.book_id = ? WHERE rel.relationship_id IS NULL AND (m.real_name LIKE ? OR m.account LIKE ?) LIMIT 0,?;", p, p)
	var members []*model.Member
	_, err := r.ormer(ctx).Raw(sql, bookID, keyWord, keyWord, limit).QueryRows(&members)
	return members, err
}

// MemberToRelationship 把成员实体转为项目成员展示结构。
func MemberToRelationship(member *model.Member) *dto.MemberRelationshipResult {
	m := dto.NewMemberRelationshipResult()
	if member == nil {
		return m
	}
	m.MemberId = member.MemberId
	m.Account = member.Account
	m.Description = member.Description
	m.Email = member.Email
	m.Phone = member.Phone
	m.Avatar = member.Avatar
	m.Role = member.Role
	m.Status = member.Status
	m.CreateTime = member.CreateTime
	m.CreateAt = member.CreateAt
	m.RealName = member.RealName
	return m
}
