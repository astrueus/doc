package mcp

import (
	"context"
	"strconv"
	"strings"
	"time"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/dto/mcpdto"
	"git.itopcms.com/astrueus/doc/internal/errs"
	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/russross/blackfriday/v2"
)

func handleCreateDocument(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.CreateDocumentIn) (*sdkmcp.CallToolResult, mcpdto.CreateDocumentOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}

	bookID, _, err := resolveBookID(ctx, m, in.BookID, in.BookIdentify)
	if err != nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](err)
	}
	if err := ensureWritable(ctx, m, bookID); err != nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](err)
	}

	ident, err := resolveDocIdentify(in.Identify, in.DocIdentify)
	if err != nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](err)
	}
	ifExists, err := parseIfExists(in.IfExists)
	if err != nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](err)
	}
	if ifExists == "update" && ident == "" {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.New(errs.CodeInvalidParam, "identify required when if_exists=update"))
	}

	parentID, err := resolveParentID(ctx, bookID, in.ParentID, in.ParentIdentify)
	if err != nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](err)
	}

	var existing *model.Document
	if ident != "" {
		doc, findErr := documentRepo().FindByIdentify(ctx, ident, bookID)
		if findErr != nil && findErr != orm.ErrNoRows {
			return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.Wrap(errs.CodeInternal, "lookup document failed", findErr))
		}
		if findErr == nil {
			existing = doc
		}
	}

	if existing != nil {
		if ifExists != "update" {
			return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.New(errs.CodeInvalidParam, "document identify already exists"))
		}
		return updateExistingOnCreate(ctx, m, existing, in)
	}

	if strings.TrimSpace(in.Title) == "" {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.New(errs.CodeInvalidParam, "title required"))
	}

	doc := model.NewDocument()
	doc.BookId = bookID
	doc.ParentId = parentID
	doc.DocumentName = in.Title
	doc.Identify = ident
	doc.Markdown = in.Markdown
	doc.MemberId = m.MemberId
	doc.ModifyAt = m.MemberId
	doc.Version = time.Now().Unix()

	if err := documentRepo().Save(ctx, doc); err != nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.Wrap(errs.CodeInternal, "create document failed", err))
	}
	maybeAutoRelease(ctx, in.AutoRelease, doc.DocumentId)
	return nil, mcpdto.CreateDocumentOut{DocumentID: doc.DocumentId, Version: doc.Version, Updated: false}, nil
}

func resolveDocIdentify(identify, docIdentify string) (string, error) {
	a := strings.TrimSpace(identify)
	b := strings.TrimSpace(docIdentify)
	if a != "" && b != "" && a != b {
		return "", errs.New(errs.CodeInvalidParam, "identify and doc_identify mismatch")
	}
	if a != "" {
		return a, nil
	}
	return b, nil
}

func parseIfExists(v string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "" {
		return "error", nil
	}
	if s != "error" && s != "update" {
		return "", errs.New(errs.CodeInvalidParam, "if_exists must be error or update")
	}
	return s, nil
}

func resolveParentID(ctx context.Context, bookID, parentID int, parentIdentify string) (int, error) {
	if parentID > 0 {
		parent, err := documentRepo().Find(ctx, parentID)
		if err != nil {
			return 0, errs.Wrap(errs.CodeNotFound, "parent document not found", err)
		}
		if parent.BookId != bookID {
			return 0, errs.New(errs.CodeInvalidParam, "parent document not in book")
		}
		return parent.DocumentId, nil
	}
	parentIdentify = strings.TrimSpace(parentIdentify)
	if parentIdentify == "" {
		return 0, nil
	}
	parent, err := documentRepo().FindByIdentify(ctx, parentIdentify, bookID)
	if err != nil {
		return 0, errs.Wrap(errs.CodeNotFound, "parent document not found", err)
	}
	return parent.DocumentId, nil
}

func updateExistingOnCreate(ctx context.Context, m *model.Member, existing *model.Document, in mcpdto.CreateDocumentIn) (*sdkmcp.CallToolResult, mcpdto.CreateDocumentOut, error) {
	version := existing.Version
	if in.Markdown != "" {
		if in.ExpectVersion == 0 {
			return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.New(errs.CodeInvalidParam, "expect_version required when if_exists=update with markdown"))
		}
		newVersion := time.Now().Unix()
		aff, err := documentRepo().UpdateMarkdownWithVersion(ctx, existing.DocumentId, in.ExpectVersion, in.Markdown, m.MemberId, newVersion)
		if err != nil {
			return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.Wrap(errs.CodeInternal, "update document failed", err))
		}
		if aff == 0 {
			return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.New(errs.CodeVersionConflict, "version conflict: please refetch with get_document and retry"))
		}
		version = newVersion
	}
	if title := strings.TrimSpace(in.Title); title != "" && title != existing.DocumentName {
		existing.DocumentName = title
		existing.ModifyAt = m.MemberId
		if err := existing.InsertOrUpdate("document_name", "modify_at"); err != nil {
			return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.Wrap(errs.CodeInternal, "update title failed", err))
		}
	}
	maybeAutoRelease(ctx, in.AutoRelease, existing.DocumentId)
	return nil, mcpdto.CreateDocumentOut{DocumentID: existing.DocumentId, Version: version, Updated: true}, nil
}

func maybeAutoRelease(ctx context.Context, auto bool, documentID int) {
	if !auto || documentID <= 0 {
		return
	}
	if err := releaseOneDocument(ctx, documentID); err != nil {
		logs.Warning("auto_release failed for doc %d: %v", documentID, err)
	}
}

func handleUpdateDocumentContent(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.UpdateDocumentContentIn) (*sdkmcp.CallToolResult, mcpdto.UpdateDocumentContentOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.UpdateDocumentContentOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}
	if in.DocumentID <= 0 {
		return toolBizErrorOut[mcpdto.UpdateDocumentContentOut](errs.New(errs.CodeInvalidParam, "document_id required"))
	}

	repo := documentRepo()
	doc, err := repo.Find(ctx, in.DocumentID)
	if err != nil {
		return toolBizErrorOut[mcpdto.UpdateDocumentContentOut](errs.Wrap(errs.CodeNotFound, "document not found", err))
	}
	if err := ensureWritable(ctx, m, doc.BookId); err != nil {
		return toolBizErrorOut[mcpdto.UpdateDocumentContentOut](err)
	}

	newVersion := time.Now().Unix()
	aff, err := repo.UpdateMarkdownWithVersion(ctx, in.DocumentID, in.ExpectVersion, in.Markdown, m.MemberId, newVersion)
	if err != nil {
		return toolBizErrorOut[mcpdto.UpdateDocumentContentOut](errs.Wrap(errs.CodeInternal, "db error", err))
	}
	if aff == 0 {
		return toolBizErrorOut[mcpdto.UpdateDocumentContentOut](errs.New(errs.CodeVersionConflict, "version conflict: please refetch with get_document and retry"))
	}

	if in.AutoRelease {
		maybeAutoRelease(ctx, true, in.DocumentID)
	}

	return nil, mcpdto.UpdateDocumentContentOut{
		DocumentID: in.DocumentID,
		Version:    newVersion,
		Message:    "updated",
	}, nil
}

func handleAppendDocumentContent(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.AppendDocumentContentIn) (*sdkmcp.CallToolResult, mcpdto.AppendDocumentContentOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.AppendDocumentContentOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}
	if in.DocumentID <= 0 {
		return toolBizErrorOut[mcpdto.AppendDocumentContentOut](errs.New(errs.CodeInvalidParam, "document_id required"))
	}

	repo := documentRepo()
	doc, err := repo.Find(ctx, in.DocumentID)
	if err != nil {
		return toolBizErrorOut[mcpdto.AppendDocumentContentOut](errs.Wrap(errs.CodeNotFound, "document not found", err))
	}
	if err := ensureWritable(ctx, m, doc.BookId); err != nil {
		return toolBizErrorOut[mcpdto.AppendDocumentContentOut](err)
	}

	newVersion := time.Now().Unix()
	newMarkdown := doc.Markdown + in.MarkdownAppend
	aff, err := repo.UpdateMarkdownWithVersion(ctx, in.DocumentID, in.ExpectVersion, newMarkdown, m.MemberId, newVersion)
	if err != nil {
		return toolBizErrorOut[mcpdto.AppendDocumentContentOut](errs.Wrap(errs.CodeInternal, "append failed", err))
	}
	if aff == 0 {
		return toolBizErrorOut[mcpdto.AppendDocumentContentOut](errs.New(errs.CodeVersionConflict, "version conflict: please refetch with get_document and retry"))
	}

	if in.AutoRelease {
		maybeAutoRelease(ctx, true, in.DocumentID)
	}

	return nil, mcpdto.AppendDocumentContentOut{DocumentID: in.DocumentID, Version: newVersion}, nil
}

func handleUpdateDocumentMeta(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.UpdateDocumentMetaIn) (*sdkmcp.CallToolResult, mcpdto.UpdateDocumentMetaOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.UpdateDocumentMetaOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}
	if in.DocumentID <= 0 {
		return toolBizErrorOut[mcpdto.UpdateDocumentMetaOut](errs.New(errs.CodeInvalidParam, "document_id required"))
	}

	doc, err := documentRepo().Find(ctx, in.DocumentID)
	if err != nil {
		return toolBizErrorOut[mcpdto.UpdateDocumentMetaOut](errs.Wrap(errs.CodeNotFound, "document not found", err))
	}
	if err := ensureWritable(ctx, m, doc.BookId); err != nil {
		return toolBizErrorOut[mcpdto.UpdateDocumentMetaOut](err)
	}

	params := orm.Params{"modify_at": m.MemberId}
	if in.Title != nil {
		params["document_name"] = *in.Title
	}
	if in.Identify != nil {
		params["identify"] = *in.Identify
	}
	if in.OrderSort != nil {
		params["order_sort"] = *in.OrderSort
	}
	if in.ParentID != nil {
		params["parent_id"] = *in.ParentID
	}
	if len(params) == 1 {
		return toolBizErrorOut[mcpdto.UpdateDocumentMetaOut](errs.New(errs.CodeInvalidParam, "no meta fields to update"))
	}

	o := orm.NewOrm()
	_, err = o.QueryTable(doc.TableNameWithPrefix()).Filter("document_id", in.DocumentID).Update(params)
	if err != nil {
		return toolBizErrorOut[mcpdto.UpdateDocumentMetaOut](errs.Wrap(errs.CodeInternal, "update meta failed", err))
	}
	return nil, mcpdto.UpdateDocumentMetaOut{DocumentID: in.DocumentID}, nil
}

func handleReleaseDocument(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.ReleaseDocumentIn) (*sdkmcp.CallToolResult, mcpdto.ReleaseDocumentOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.ReleaseDocumentOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}
	if in.DocumentID <= 0 && in.BookID <= 0 {
		return toolBizErrorOut[mcpdto.ReleaseDocumentOut](errs.New(errs.CodeInvalidParam, "document_id or book_id required"))
	}

	if in.DocumentID > 0 {
		doc, err := documentRepo().Find(ctx, in.DocumentID)
		if err != nil {
			return toolBizErrorOut[mcpdto.ReleaseDocumentOut](errs.Wrap(errs.CodeNotFound, "document not found", err))
		}
		if err := ensureWritable(ctx, m, doc.BookId); err != nil {
			return toolBizErrorOut[mcpdto.ReleaseDocumentOut](err)
		}
		if err := releaseOneDocument(ctx, in.DocumentID); err != nil {
			return toolBizErrorOut[mcpdto.ReleaseDocumentOut](errs.Wrap(errs.CodeInternal, "release failed", err))
		}
		return nil, mcpdto.ReleaseDocumentOut{ReleasedCount: 1}, nil
	}

	if err := ensureWritable(ctx, m, in.BookID); err != nil {
		return toolBizErrorOut[mcpdto.ReleaseDocumentOut](err)
	}
	docs, err := documentRepo().FindListByBookID(ctx, in.BookID)
	if err != nil {
		return toolBizErrorOut[mcpdto.ReleaseDocumentOut](errs.Wrap(errs.CodeInternal, "list documents failed", err))
	}
	count := 0
	for _, d := range docs {
		if err := releaseOneDocument(ctx, d.DocumentId); err != nil {
			logs.Warning("release doc %d failed: %v", d.DocumentId, err)
			continue
		}
		count++
	}
	return nil, mcpdto.ReleaseDocumentOut{ReleasedCount: count}, nil
}

func handleDeleteDocument(ctx context.Context, _ *sdkmcp.CallToolRequest, in mcpdto.DeleteDocumentIn) (*sdkmcp.CallToolResult, mcpdto.DeleteDocumentOut, error) {
	m := memberFromCtx(ctx)
	if m == nil {
		return toolBizErrorOut[mcpdto.DeleteDocumentOut](errs.New(errs.CodeUnauthorized, "unauthorized"))
	}
	if in.DocumentID <= 0 {
		return toolBizErrorOut[mcpdto.DeleteDocumentOut](errs.New(errs.CodeInvalidParam, "document_id required"))
	}
	if !in.Confirm {
		return toolBizErrorOut[mcpdto.DeleteDocumentOut](errs.New(errs.CodeConfirmRequired, "confirm required: set confirm=true to delete"))
	}

	doc, err := documentRepo().Find(ctx, in.DocumentID)
	if err != nil {
		return toolBizErrorOut[mcpdto.DeleteDocumentOut](errs.Wrap(errs.CodeNotFound, "document not found", err))
	}
	if err := ensureWritable(ctx, m, doc.BookId); err != nil {
		return toolBizErrorOut[mcpdto.DeleteDocumentOut](err)
	}

	history := model.NewDocumentHistory()
	history.DocumentId = doc.DocumentId
	history.DocumentName = doc.DocumentName
	history.ParentId = doc.ParentId
	history.Markdown = doc.Markdown
	history.Content = doc.Content
	history.MemberId = doc.MemberId
	history.ModifyAt = m.MemberId
	history.Version = doc.Version
	history.IsOpen = doc.IsOpen
	history.Action = "mcp_delete"
	history.ActionName = "MCP delete snapshot"
	if _, err := history.InsertOrUpdate(); err != nil {
		return toolBizErrorOut[mcpdto.DeleteDocumentOut](errs.Wrap(errs.CodeInternal, "snapshot history failed", err))
	}

	deleted, err := recursiveDeleteKeepHistory(in.DocumentID)
	if err != nil {
		return toolBizErrorOut[mcpdto.DeleteDocumentOut](errs.Wrap(errs.CodeInternal, "delete failed", err))
	}
	model.NewBook().ResetDocumentNumber(doc.BookId)

	return nil, mcpdto.DeleteDocumentOut{
		DeletedCount:      deleted,
		SnapshotHistoryID: history.HistoryId,
	}, nil
}

func releaseOneDocument(ctx context.Context, documentID int) error {
	doc, err := documentRepo().Find(ctx, documentID)
	if err != nil {
		return err
	}
	doc.Lang = config.GetDefaultLang()
	doc.Content = string(blackfriday.Run([]byte(doc.Markdown)))
	return doc.ReleaseContent()
}

// recursiveDeleteKeepHistory deletes a document tree without clearing DocumentHistory
// (unlike Document.RecursiveDocument), so the MCP delete snapshot remains.
func recursiveDeleteKeepHistory(docID int) (int, error) {
	o := orm.NewOrm()
	doc := model.NewDocument()
	count := 0

	var maps []orm.Params
	_, err := o.Raw("SELECT document_id FROM "+doc.TableNameWithPrefix()+" WHERE parent_id=?", docID).Values(&maps)
	if err != nil {
		return 0, err
	}
	for _, param := range maps {
		raw, ok := param["document_id"].(string)
		if !ok {
			continue
		}
		id, _ := strconv.Atoi(raw)
		n, err := recursiveDeleteKeepHistory(id)
		if err != nil {
			return count, err
		}
		count += n
	}

	if existing, err := documentRepo().Find(context.Background(), docID); err == nil {
		if _, err := o.Delete(existing); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
