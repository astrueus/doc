package dto

import (
	"time"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/i18n"
)

// MemberRelationshipResult 项目成员关系展示结构。
type MemberRelationshipResult struct {
	MemberId       int               `json:"member_id"`
	Account        string            `json:"account"`
	RealName       string            `json:"real_name"`
	Description    string            `json:"description"`
	Email          string            `json:"email"`
	Phone          string            `json:"phone"`
	Avatar         string            `json:"avatar"`
	Role           config.SystemRole `json:"role"`
	Status         int               `json:"status"`
	CreateTime     time.Time         `json:"create_time"`
	CreateAt       int               `json:"create_at"`
	RelationshipId int               `json:"relationship_id"`
	BookId         int               `json:"book_id"`
	RoleId         config.BookRole   `json:"role_id"`
	RoleName       string            `json:"role_name"`
}

// NewMemberRelationshipResult 返回空的成员关系展示结构。
func NewMemberRelationshipResult() *MemberRelationshipResult {
	return &MemberRelationshipResult{}
}

// ResolveRoleName 按项目角色翻译 RoleName。
func (m *MemberRelationshipResult) ResolveRoleName(lang string) *MemberRelationshipResult {
	if m.RoleId == config.BookAdmin {
		m.RoleName = i18n.Tr(lang, "common.administrator")
	} else if m.RoleId == config.BookEditor {
		m.RoleName = i18n.Tr(lang, "common.editor")
	} else if m.RoleId == config.BookObserver {
		m.RoleName = i18n.Tr(lang, "common.obverser")
	}
	return m
}
