package mcp

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/dto/mcpdto"
	"git.itopcms.com/astrueus/doc/internal/errs"
	"git.itopcms.com/astrueus/doc/internal/model"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

var bookIdentifyRe = regexp.MustCompile(`^[a-z]+[a-zA-Z0-9_\-]*$`)

func handleCreateBook(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.CreateBookIn) (*sdkmcp.CallToolResult, mcpdto.BookBrief, error) {
	m := memberFromCtx(ctx)
	if err := ensureCanCreateBook(m); err != nil {
		return toolBizErrorOut[mcpdto.BookBrief](err)
	}

	title := strings.TrimSpace(in.Title)
	identify := strings.TrimSpace(in.Identify)
	description := strings.TrimSpace(in.Description)
	if title == "" {
		return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "title required"))
	}
	if identify == "" {
		return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "identify required"))
	}
	if !bookIdentifyRe.MatchString(identify) {
		return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "invalid identify: must match ^[a-z]+[a-zA-Z0-9_-]*$"))
	}
	if utf8.RuneCountInString(identify) > 50 {
		return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "identify too long"))
	}
	if utf8.RuneCountInString(description) > 500 {
		return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "description too long"))
	}

	itemID, err := resolveItemID(in.ItemIdentify)
	if err != nil {
		return toolBizErrorOut[mcpdto.BookBrief](err)
	}

	if _, err := bookRepo().FindByIdentify(ctx, identify); err == nil {
		return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "book identify already exists"))
	} else if err != model.ErrDataNotExist {
		return toolBizErrorOut[mcpdto.BookBrief](errs.Wrap(errs.CodeInternal, "lookup book failed", err))
	}

	privatelyOwned := 0
	if in.Private {
		privatelyOwned = 1
	}

	book := model.NewBook()
	book.Cover = config.GetDefaultCover()
	book.BookName = title
	book.Description = description
	book.CommentCount = 0
	book.PrivatelyOwned = privatelyOwned
	book.CommentStatus = "closed"
	book.Identify = identify
	book.DocCount = 0
	book.MemberId = m.MemberId
	book.Version = time.Now().Unix()
	book.IsEnableShare = 0
	book.IsUseFirstDocument = 1
	book.IsDownload = 1
	book.AutoRelease = 0
	book.ItemId = itemID
	book.Editor = "markdown"
	book.Theme = "default"

	if err := bookRepo().Create(ctx, book, config.GetDefaultLang()); err != nil {
		return toolBizErrorOut[mcpdto.BookBrief](errs.Wrap(errs.CodeInternal, "create book failed", err))
	}

	created, err := bookRepo().Find(ctx, book.BookId)
	if err != nil {
		created = book
	}
	return nil, toBookBriefFromBook(created, int(config.BookFounder)), nil
}

func handleUpdateBook(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.UpdateBookIn) (*sdkmcp.CallToolResult, mcpdto.BookBrief, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}

	bookID, _, err := resolveBookID(m, in.BookID, in.BookIdentify)
	if err != nil {
		return toolBizErrorOut[mcpdto.BookBrief](err)
	}
	if err := ensureBookMetaWritable(m, bookID, in.Private != nil); err != nil {
		return toolBizErrorOut[mcpdto.BookBrief](err)
	}

	book, err := bookRepo().Find(ctx, bookID)
	if err != nil {
		return toolBizErrorOut[mcpdto.BookBrief](errs.Wrap(errs.CodeNotFound, "book not found", err))
	}

	cols := make([]string, 0, 3)
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "title cannot be empty"))
		}
		book.BookName = title
		cols = append(cols, "book_name")
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if utf8.RuneCountInString(desc) > 500 {
			return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "description too long"))
		}
		book.Description = desc
		cols = append(cols, "description")
	}
	if in.Private != nil {
		if *in.Private {
			book.PrivatelyOwned = 1
		} else {
			book.PrivatelyOwned = 0
		}
		cols = append(cols, "privately_owned")
	}
	if len(cols) == 0 {
		return toolBizErrorOut[mcpdto.BookBrief](errs.New(errs.CodeInvalidParam, "no fields to update"))
	}

	if err := bookRepo().Update(ctx, book, cols...); err != nil {
		return toolBizErrorOut[mcpdto.BookBrief](errs.Wrap(errs.CodeInternal, "update book failed", err))
	}

	role, err := bookRoleOf(m, book.BookId)
	if err != nil {
		role = config.BookFounder
	}
	return nil, toBookBriefFromBook(book, int(role)), nil
}

func resolveItemID(itemIdentify string) (int, error) {
	itemIdentify = strings.TrimSpace(itemIdentify)
	if itemIdentify == "" {
		if !model.NewItemsets().Exist(1) {
			return 0, errs.New(errs.CodeInvalidParam, "default itemsets not found")
		}
		return 1, nil
	}
	item, err := model.NewItemsets().FindFirst(itemIdentify)
	if err != nil || item == nil || item.ItemId <= 0 {
		return 0, errs.Wrap(errs.CodeInvalidParam, "itemsets not found", err)
	}
	return item.ItemId, nil
}

func toBookBriefFromBook(b *model.Book, roleID int) mcpdto.BookBrief {
	if b == nil {
		return mcpdto.BookBrief{}
	}
	return mcpdto.BookBrief{
		BookID:      b.BookId,
		Identify:    b.Identify,
		Title:       b.BookName,
		Description: b.Description,
		Private:     b.PrivatelyOwned == 1,
		RoleID:      roleID,
		DocCount:    b.DocCount,
	}
}
