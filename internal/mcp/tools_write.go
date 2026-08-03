package mcp

import (
	"context"
	"strconv"
	"time"

	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/dto/mcpdto"
	"git.itopcms.com/jackliu/doc/internal/errs"
	"git.itopcms.com/jackliu/doc/internal/model"
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
	if in.BookID <= 0 || in.Title == "" {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.New(errs.CodeInvalidParam, "book_id and title required"))
	}
	if err := ensureWritable(m, in.BookID); err != nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](err)
	}

	doc := model.NewDocument()
	doc.BookId = in.BookID
	doc.ParentId = in.ParentID
	doc.DocumentName = in.Title
	doc.Identify = in.Identify
	doc.Markdown = in.Markdown
	doc.MemberId = m.MemberId
	doc.ModifyAt = m.MemberId
	doc.Version = time.Now().Unix()

	if err := documentRepo().Save(ctx, doc); err != nil {
		return toolBizErrorOut[mcpdto.CreateDocumentOut](errs.Wrap(errs.CodeInternal, "create document failed", err))
	}
	return nil, mcpdto.CreateDocumentOut{DocumentID: doc.DocumentId, Version: doc.Version}, nil
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
	if err := ensureWritable(m, doc.BookId); err != nil {
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
		if err := releaseOneDocument(ctx, in.DocumentID); err != nil {
			logs.Warning("auto_release failed for doc %d: %v", in.DocumentID, err)
		}
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
	if err := ensureWritable(m, doc.BookId); err != nil {
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
		if err := releaseOneDocument(ctx, in.DocumentID); err != nil {
			logs.Warning("auto_release failed for doc %d: %v", in.DocumentID, err)
		}
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
	if err := ensureWritable(m, doc.BookId); err != nil {
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
		if err := ensureWritable(m, doc.BookId); err != nil {
			return toolBizErrorOut[mcpdto.ReleaseDocumentOut](err)
		}
		if err := releaseOneDocument(ctx, in.DocumentID); err != nil {
			return toolBizErrorOut[mcpdto.ReleaseDocumentOut](errs.Wrap(errs.CodeInternal, "release failed", err))
		}
		return nil, mcpdto.ReleaseDocumentOut{ReleasedCount: 1}, nil
	}

	if err := ensureWritable(m, in.BookID); err != nil {
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
	if err := ensureWritable(m, doc.BookId); err != nil {
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
