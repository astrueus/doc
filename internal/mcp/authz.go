package mcp

import (
	"context"
	"fmt"

	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/errs"
	"git.itopcms.com/jackliu/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
)

type memberCtxKey struct{}

func withMember(ctx context.Context, m *model.Member) context.Context {
	return context.WithValue(ctx, memberCtxKey{}, m)
}

func memberFromCtx(ctx context.Context) *model.Member {
	m, _ := ctx.Value(memberCtxKey{}).(*model.Member)
	return m
}

func canReadBook(m *model.Member, bookID int) error {
	if m == nil {
		return errs.New(errs.CodeUnauthorized, "unauthorized")
	}
	if m.IsAdministrator() {
		return nil
	}
	book, err := model.NewBook().Find(bookID)
	if err != nil {
		return errs.Wrap(errs.CodeNotFound, "book not found", err)
	}
	if book.PrivatelyOwned == 0 {
		return nil
	}
	role, err := model.NewBook().FindForRoleId(bookID, m.MemberId)
	if err != nil {
		return errs.New(errs.CodeForbidden, "forbidden")
	}
	if role > config.BookObserver {
		return errs.New(errs.CodeForbidden, "forbidden")
	}
	return nil
}

// ensureWritable requires BookRole <= BookEditor (founder/admin/editor).
func ensureWritable(m *model.Member, bookID int) error {
	if m == nil {
		return errs.New(errs.CodeUnauthorized, "unauthorized")
	}
	if m.IsAdministrator() {
		return nil
	}
	if _, err := model.NewBook().Find(bookID); err != nil {
		return errs.Wrap(errs.CodeNotFound, "book not found", err)
	}
	role, err := model.NewBook().FindForRoleId(bookID, m.MemberId)
	if err != nil {
		return errs.New(errs.CodeForbidden, "forbidden: need BookEditor or higher")
	}
	if role > config.BookEditor {
		return errs.New(errs.CodeForbidden, "forbidden: need BookEditor or higher")
	}
	return nil
}

func resolveBookID(m *model.Member, bookID int, bookIdentify string) (int, string, error) {
	if bookID > 0 {
		book, err := model.NewBook().Find(bookID)
		if err != nil {
			return 0, "", errs.Wrap(errs.CodeNotFound, "book not found", err)
		}
		if err := canReadBook(m, book.BookId); err != nil {
			return 0, "", err
		}
		return book.BookId, book.Identify, nil
	}
	if bookIdentify == "" {
		return 0, "", errs.New(errs.CodeInvalidParam, "book_id or book_identify required")
	}
	if m.IsAdministrator() {
		book, err := model.NewBook().FindByIdentify(bookIdentify)
		if err != nil {
			return 0, "", errs.Wrap(errs.CodeNotFound, "book not found", err)
		}
		return book.BookId, book.Identify, nil
	}
	br, err := model.NewBookResult().FindByIdentify(bookIdentify, m.MemberId)
	if err != nil {
		return 0, "", errs.Wrap(errs.CodeForbidden, "book not found or forbidden", err)
	}
	if br.RoleId > config.BookObserver {
		return 0, "", errs.New(errs.CodeForbidden, "forbidden")
	}
	return br.BookId, br.Identify, nil
}

func visibleBookIDs(m *model.Member, filterBookID int) ([]int, error) {
	if filterBookID > 0 {
		if err := canReadBook(m, filterBookID); err != nil {
			return nil, err
		}
		return []int{filterBookID}, nil
	}
	if m.IsAdministrator() {
		return allBookIDs()
	}
	o := orm.NewOrm()
	var ids []int
	p := config.GetDatabasePrefix()
	_, err := o.Raw(fmt.Sprintf(`SELECT DISTINCT book.book_id
FROM %sbooks AS book
  LEFT JOIN %srelationship AS rel1 ON book.book_id = rel1.book_id AND rel1.member_id = ?
  LEFT JOIN (
    SELECT book_id, team_member_id
    FROM (
      SELECT book_id, team_member_id, role_id
      FROM %steam_relationship AS mtr
        LEFT JOIN %steam_member AS mtm ON mtm.team_id = mtr.team_id AND mtm.member_id = ?
      ORDER BY role_id DESC
    ) AS t
    GROUP BY t.role_id, t.team_member_id, t.book_id
  ) AS team ON team.book_id = book.book_id
WHERE book.privately_owned = 0 OR rel1.relationship_id > 0 OR team.team_member_id > 0`, p, p, p, p)).QueryRows(&ids)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list visible books failed", err)
	}
	return ids, nil
}

func allBookIDs() ([]int, error) {
	o := orm.NewOrm()
	var ids []int
	_, err := o.Raw(fmt.Sprintf("SELECT book_id FROM %s ORDER BY book_id", model.NewBook().TableNameWithPrefix())).QueryRows(&ids)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list books failed", err)
	}
	return ids, nil
}
