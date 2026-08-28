package mcp

import (
	"context"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/dto/mcpdto"
	"git.itopcms.com/astrueus/doc/internal/errs"
	"git.itopcms.com/astrueus/doc/internal/model"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func handleSearchDocument(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.SearchDocumentIn) (*sdkmcp.CallToolResult, mcpdto.SearchDocumentOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.SearchDocumentOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}
	bookIDs, err := visibleBookIDs(ctx, m, in.BookID)
	if err != nil {
		return toolBizErrorOut[mcpdto.SearchDocumentOut](err)
	}
	limit := defaultLimit(in.Limit, 10, 50)
	docs, err := newSearchProvider().Search(ctx, in.Query, bookIDs, limit)
	if err != nil {
		return toolBizErrorOut[mcpdto.SearchDocumentOut](errs.Wrap(errs.CodeInternal, "search failed", err))
	}
	idSet := make([]int, 0, len(docs))
	for _, d := range docs {
		idSet = append(idSet, d.BookId)
	}
	idents, err := bookRepo().IdentifiesByIDs(ctx, idSet)
	if err != nil {
		return toolBizErrorOut[mcpdto.SearchDocumentOut](errs.Wrap(errs.CodeInternal, "load book identify failed", err))
	}
	return nil, toSearchOut(docs, idents), nil
}

func handleGetDocument(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.GetDocumentIn) (*sdkmcp.CallToolResult, mcpdto.GetDocumentOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.GetDocumentOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}

	doc, bookIdentify, err := resolveDocument(ctx, m, in)
	if err != nil {
		return toolBizErrorOut[mcpdto.GetDocumentOut](err)
	}
	includeTruncated := true
	if in.IncludeTruncated != nil {
		includeTruncated = *in.IncludeTruncated
	}
	return nil, toGetDocumentOut(doc, bookIdentify, in.MaxChars, includeTruncated), nil
}

func resolveDocument(ctx context.Context, m *model.Member, in mcpdto.GetDocumentIn) (*model.Document, string, error) {
	if in.DocumentID > 0 {
		doc, err := documentRepo().Find(ctx, in.DocumentID)
		if err != nil {
			return nil, "", errs.Wrap(errs.CodeNotFound, "document not found", err)
		}
		if err := canReadBook(ctx, m, doc.BookId); err != nil {
			return nil, "", err
		}
		book, err := bookRepo().Find(ctx, doc.BookId)
		if err != nil {
			return nil, "", errs.Wrap(errs.CodeNotFound, "book not found", err)
		}
		return doc, book.Identify, nil
	}
	if in.BookIdentify == "" || in.DocIdentify == "" {
		return nil, "", errs.New(errs.CodeInvalidParam, "document_id or book_identify+doc_identify required")
	}
	bookID, bookIdentify, err := resolveBookID(ctx, m, 0, in.BookIdentify)
	if err != nil {
		return nil, "", err
	}
	doc, err := documentRepo().FindByIdentify(ctx, in.DocIdentify, bookID)
	if err != nil {
		return nil, "", errs.Wrap(errs.CodeNotFound, "document not found", err)
	}
	return doc, bookIdentify, nil
}

func handleListBooks(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.ListBooksIn) (*sdkmcp.CallToolResult, mcpdto.ListBooksOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.ListBooksOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}
	page := defaultPage(in.Page)
	pageSize := defaultLimit(in.PageSize, 20, 100)
	var (
		books []*dto.BookResult
		total int
		err   error
	)
	if m.IsAdministrator() {
		books, total, err = bookRepo().FindToPagerAll(ctx, page, pageSize)
	} else {
		books, total, err = bookRepo().FindToPagerForMember(ctx, page, pageSize, m.MemberId, config.GetDefaultLang())
	}
	if err != nil {
		return toolBizErrorOut[mcpdto.ListBooksOut](errs.Wrap(errs.CodeInternal, "list books failed", err))
	}
	items := make([]mcpdto.BookBrief, 0, len(books))
	for _, b := range books {
		items = append(items, toBookBrief(b))
	}
	return nil, mcpdto.ListBooksOut{Total: total, Items: items}, nil
}

func handleListDocumentTree(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.ListDocumentTreeIn) (*sdkmcp.CallToolResult, mcpdto.ListDocumentTreeOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.ListDocumentTreeOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}
	bookID, bookIdentify, err := resolveBookID(ctx, m, in.BookID, in.BookIdentify)
	if err != nil {
		return toolBizErrorOut[mcpdto.ListDocumentTreeOut](err)
	}
	docs, err := documentRepo().FindListByBookID(ctx, bookID)
	if err != nil {
		return toolBizErrorOut[mcpdto.ListDocumentTreeOut](errs.Wrap(errs.CodeInternal, "list document tree failed", err))
	}
	return nil, mcpdto.ListDocumentTreeOut{
		BookID:       bookID,
		BookIdentify: bookIdentify,
		Nodes:        toTreeNodes(docs),
	}, nil
}
