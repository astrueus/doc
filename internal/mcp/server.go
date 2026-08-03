package mcp

import (
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func newServer() *sdkmcp.Server {
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "doc-mcp", Version: "1.0.0"}, nil)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "search_document",
		Description: "Search documents by keyword; returns brief items with book/title/snippet.",
	}, handleSearchDocument)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "get_document",
		Description: "Get a document by document_id or book_identify+doc_identify.",
	}, handleGetDocument)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "list_books",
		Description: "List books accessible to the current member.",
	}, handleListBooks)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "list_document_tree",
		Description: "List document tree nodes for a book.",
	}, handleListDocumentTree)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "create_document",
		Description: "Create a document in a book. Requires BookEditor or higher.",
	}, handleCreateDocument)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "update_document_content",
		Description: "Overwrite document Markdown with optimistic locking via expect_version.",
	}, handleUpdateDocumentContent)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "append_document_content",
		Description: "Append Markdown with optimistic locking via expect_version; optional auto_release.",
	}, handleAppendDocumentContent)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "update_document_meta",
		Description: "Update document title/identify/order_sort/parent_id.",
	}, handleUpdateDocumentMeta)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "release_document",
		Description: "Release Markdown to HTML for one document or an entire book.",
	}, handleReleaseDocument)

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "delete_document",
		Description: "Delete a document tree. Requires confirm=true; writes a history snapshot first.",
	}, handleDeleteDocument)

	return srv
}
