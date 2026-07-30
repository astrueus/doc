package mcp

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"git.itopcms.com/jackliu/doc/internal/dto/mcpdto"
	"git.itopcms.com/jackliu/doc/internal/model"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func documentSnippet(doc *model.Document, maxRunes int) string {
	text := strings.TrimSpace(doc.Markdown)
	if text == "" {
		text = strings.TrimSpace(doc.Release)
		text = htmlTagRe.ReplaceAllString(text, " ")
		text = strings.Join(strings.Fields(text), " ")
	}
	return truncateRunes(text, maxRunes)
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var n int
	for i := range s {
		if n == max {
			return s[:i] + "..."
		}
		n++
	}
	return s
}

func toDocumentBrief(doc *model.Document) mcpdto.DocumentBrief {
	return mcpdto.DocumentBrief{
		ID:      doc.DocumentId,
		BookID:  doc.BookId,
		Title:   doc.DocumentName,
		Snippet: documentSnippet(doc, 200),
		Version: doc.Version,
	}
}

func toSearchOut(docs []*model.Document) mcpdto.SearchDocumentOut {
	items := make([]mcpdto.DocumentBrief, 0, len(docs))
	for _, d := range docs {
		items = append(items, toDocumentBrief(d))
	}
	return mcpdto.SearchDocumentOut{
		Total: len(items),
		Items: items,
	}
}

func toGetDocumentOut(doc *model.Document, bookIdentify string) mcpdto.GetDocumentOut {
	return mcpdto.GetDocumentOut{
		DocumentID:   doc.DocumentId,
		BookID:       doc.BookId,
		BookIdentify: bookIdentify,
		Title:        doc.DocumentName,
		Identify:     doc.Identify,
		Markdown:     doc.Markdown,
		Release:      doc.Release,
		Version:      doc.Version,
		ParentID:     doc.ParentId,
	}
}

func toBookBrief(b *model.BookResult) mcpdto.BookBrief {
	return mcpdto.BookBrief{
		BookID:      b.BookId,
		Identify:    b.Identify,
		Title:       b.BookName,
		Description: b.Description,
		Private:     b.PrivatelyOwned == 1,
		RoleID:      int(b.RoleId),
		DocCount:    b.DocCount,
	}
}

func toTreeNodes(docs []*model.Document) []mcpdto.DocumentTreeNode {
	nodes := make([]mcpdto.DocumentTreeNode, 0, len(docs))
	for _, d := range docs {
		nodes = append(nodes, mcpdto.DocumentTreeNode{
			DocumentID: d.DocumentId,
			ParentID:   d.ParentId,
			Title:      d.DocumentName,
			Identify:   d.Identify,
			Version:    d.Version,
			OrderSort:  d.OrderSort,
		})
	}
	return nodes
}
