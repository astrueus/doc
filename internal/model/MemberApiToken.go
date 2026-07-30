package model

import (
	"time"

	"git.itopcms.com/jackliu/doc/internal/config"
	"github.com/beego/beego/v2/client/orm"
)

// MemberApiToken stores hashed API tokens for MCP HTTP Bearer auth.
// Plaintext is shown only once at creation; only TokenHash is persisted.
type MemberApiToken struct {
	TokenId    int       `orm:"column(token_id);pk;auto" json:"token_id"`
	MemberId   int       `orm:"column(member_id);type(int);index" json:"member_id"`
	TokenHash  string    `orm:"column(token_hash);size(64);unique" json:"-"`
	Name       string    `orm:"column(name);size(100)" json:"name"`
	Scopes     string    `orm:"column(scopes);size(255)" json:"scopes"`
	ExpiresAt  time.Time `orm:"column(expires_at);type(datetime);null" json:"expires_at"`
	LastUsedAt time.Time `orm:"column(last_used_at);type(datetime);null" json:"last_used_at"`
	LastUsedIP string    `orm:"column(last_used_ip);size(45);null" json:"last_used_ip"`
	RevokedAt  time.Time `orm:"column(revoked_at);type(datetime);null" json:"revoked_at"`
	CreatedAt  time.Time `orm:"column(created_at);type(datetime);auto_now_add" json:"created_at"`
}

func (m *MemberApiToken) TableName() string { return "member_api_tokens" }

func (m *MemberApiToken) TableEngine() string { return "INNODB" }

func (m *MemberApiToken) TableNameWithPrefix() string {
	return config.GetDatabasePrefix() + m.TableName()
}

func NewMemberApiToken() *MemberApiToken {
	return &MemberApiToken{}
}

func (m *MemberApiToken) IsRevoked() bool {
	return !m.RevokedAt.IsZero()
}

func (m *MemberApiToken) IsExpired(now time.Time) bool {
	return !m.ExpiresAt.IsZero() && m.ExpiresAt.Before(now)
}

func (m *MemberApiToken) FindByID(id int) (*MemberApiToken, error) {
	o := orm.NewOrm()
	err := o.QueryTable(m.TableNameWithPrefix()).Filter("token_id", id).One(m)
	return m, err
}

func (m *MemberApiToken) FindByHash(hash string) (*MemberApiToken, error) {
	o := orm.NewOrm()
	err := o.QueryTable(m.TableNameWithPrefix()).Filter("token_hash", hash).One(m)
	return m, err
}

func (m *MemberApiToken) ListByMember(memberID int) ([]*MemberApiToken, error) {
	o := orm.NewOrm()
	var list []*MemberApiToken
	_, err := o.QueryTable(m.TableNameWithPrefix()).
		Filter("member_id", memberID).
		OrderBy("-token_id").
		All(&list)
	return list, err
}

func (m *MemberApiToken) Insert() error {
	o := orm.NewOrm()
	id, err := o.Insert(m)
	if err != nil {
		return err
	}
	m.TokenId = int(id)
	return nil
}

func (m *MemberApiToken) Revoke(now time.Time) error {
	o := orm.NewOrm()
	m.RevokedAt = now
	_, err := o.Update(m, "revoked_at")
	return err
}
