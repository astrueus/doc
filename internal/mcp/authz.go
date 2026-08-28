package mcp

import (
	"context"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/errs"
	"git.itopcms.com/astrueus/doc/internal/model"
)

type memberCtxKey struct{}

func withMember(ctx context.Context, m *model.Member) context.Context {
	return context.WithValue(ctx, memberCtxKey{}, m)
}

func memberFromCtx(ctx context.Context) *model.Member {
	m, _ := ctx.Value(memberCtxKey{}).(*model.Member)
	return m
}

func canReadBook(ctx context.Context, m *model.Member, bookID int) error {
	if m == nil {
		return errs.New(errs.CodeUnauthorized, "unauthorized")
	}
	if m.IsAdministrator() {
		return nil
	}
	book, err := bookRepo().Find(ctx, bookID)
	if err != nil {
		return errs.Wrap(errs.CodeNotFound, "book not found", err)
	}
	if book.PrivatelyOwned == 0 {
		return nil
	}
	role, err := bookRepo().FindForRoleId(ctx, bookID, m.MemberId)
	if err != nil {
		return errs.New(errs.CodeForbidden, "forbidden")
	}
	if role > config.BookObserver {
		return errs.New(errs.CodeForbidden, "forbidden")
	}
	return nil
}

func ensureCanCreateBook(m *model.Member) error {
	if m == nil {
		return errs.New(errs.CodeUnauthorized, "unauthorized")
	}
	if m.Status != 0 {
		return errs.New(errs.CodeForbidden, "forbidden: member disabled")
	}
	return nil
}

func bookRoleOf(ctx context.Context, m *model.Member, bookID int) (config.BookRole, error) {
	if m == nil {
		return config.BookObserver, errs.New(errs.CodeUnauthorized, "unauthorized")
	}
	if m.IsAdministrator() {
		return config.BookFounder, nil
	}
	if _, err := bookRepo().Find(ctx, bookID); err != nil {
		return config.BookObserver, errs.Wrap(errs.CodeNotFound, "book not found", err)
	}
	role, err := bookRepo().FindForRoleId(ctx, bookID, m.MemberId)
	if err != nil {
		return config.BookObserver, errs.New(errs.CodeForbidden, "forbidden")
	}
	return role, nil
}

// ensureBookMetaWritable 对齐 Web：改标题/简介需创始人或项目管理员；改公开/私有仅创始人。
func ensureBookMetaWritable(ctx context.Context, m *model.Member, bookID int, changePrivate bool) error {
	role, err := bookRoleOf(ctx, m, bookID)
	if err != nil {
		return err
	}
	if changePrivate {
		if role != config.BookFounder {
			return errs.New(errs.CodeForbidden, "forbidden: only founder can change private")
		}
		return nil
	}
	if role != config.BookFounder && role != config.BookAdmin {
		return errs.New(errs.CodeForbidden, "forbidden: need BookAdmin or founder")
	}
	return nil
}

// ensureWritable requires BookRole <= BookEditor (founder/admin/editor).
func ensureWritable(ctx context.Context, m *model.Member, bookID int) error {
	if m == nil {
		return errs.New(errs.CodeUnauthorized, "unauthorized")
	}
	if m.IsAdministrator() {
		return nil
	}
	if _, err := bookRepo().Find(ctx, bookID); err != nil {
		return errs.Wrap(errs.CodeNotFound, "book not found", err)
	}
	role, err := bookRepo().FindForRoleId(ctx, bookID, m.MemberId)
	if err != nil {
		return errs.New(errs.CodeForbidden, "forbidden: need BookEditor or higher")
	}
	if role > config.BookEditor {
		return errs.New(errs.CodeForbidden, "forbidden: need BookEditor or higher")
	}
	return nil
}

func resolveBookID(ctx context.Context, m *model.Member, bookID int, bookIdentify string) (int, string, error) {
	if bookID > 0 {
		book, err := bookRepo().Find(ctx, bookID)
		if err != nil {
			return 0, "", errs.Wrap(errs.CodeNotFound, "book not found", err)
		}
		if err := canReadBook(ctx, m, book.BookId); err != nil {
			return 0, "", err
		}
		return book.BookId, book.Identify, nil
	}
	if bookIdentify == "" {
		return 0, "", errs.New(errs.CodeInvalidParam, "book_id or book_identify required")
	}
	if m.IsAdministrator() {
		book, err := bookRepo().FindByIdentify(ctx, bookIdentify)
		if err != nil {
			return 0, "", errs.Wrap(errs.CodeNotFound, "book not found", err)
		}
		return book.BookId, book.Identify, nil
	}
	br, err := bookRepo().FindByIdentifyForMember(ctx, bookIdentify, m.MemberId, config.GetDefaultLang())
	if err != nil {
		return 0, "", errs.Wrap(errs.CodeForbidden, "book not found or forbidden", err)
	}
	if br.RoleId > config.BookObserver {
		return 0, "", errs.New(errs.CodeForbidden, "forbidden")
	}
	return br.BookId, br.Identify, nil
}

func visibleBookIDs(ctx context.Context, m *model.Member, filterBookID int) ([]int, error) {
	if filterBookID > 0 {
		if err := canReadBook(ctx, m, filterBookID); err != nil {
			return nil, err
		}
		return []int{filterBookID}, nil
	}
	if m.IsAdministrator() {
		ids, err := bookRepo().ListAllIDs(ctx)
		if err != nil {
			return nil, errs.Wrap(errs.CodeInternal, "list books failed", err)
		}
		return ids, nil
	}
	ids, err := bookRepo().ListVisibleIDs(ctx, m.MemberId)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list visible books failed", err)
	}
	return ids, nil
}
