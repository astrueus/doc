package mcp

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/dto/mcpdto"
	"git.itopcms.com/astrueus/doc/internal/model"
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

const (
	truncatedMarker = "…[truncated]"
	maxGetCharsCap  = 200_000
)

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

func clampMaxChars(n int) int {
	if n <= 0 {
		return 0
	}
	if n > maxGetCharsCap {
		return maxGetCharsCap
	}
	return n
}

// truncateForGet 按 Unicode 字符数截断；maxChars<=0 表示不截断。
func truncateForGet(s string, maxChars int, includeMarker bool) (string, bool) {
	maxChars = clampMaxChars(maxChars)
	if maxChars <= 0 {
		return s, false
	}
	if utf8.RuneCountInString(s) <= maxChars {
		return s, false
	}
	var n int
	cut := s
	for i := range s {
		if n == maxChars {
			cut = s[:i]
			break
		}
		n++
	}
	if includeMarker {
		return cut + truncatedMarker, true
	}
	return cut, true
}

func toDocumentBrief(doc *model.Document, bookIdentify string) mcpdto.DocumentBrief {
	return mcpdto.DocumentBrief{
		ID:           doc.DocumentId,
		BookID:       doc.BookId,
		BookIdentify: bookIdentify,
		DocIdentify:  doc.Identify,
		Title:        doc.DocumentName,
		Snippet:      documentSnippet(doc, 200),
		Version:      doc.Version,
	}
}

func toSearchOut(docs []*model.Document, bookIdentifies map[int]string) mcpdto.SearchDocumentOut {
	items := make([]mcpdto.DocumentBrief, 0, len(docs))
	for _, d := range docs {
		ident := ""
		if bookIdentifies != nil {
			ident = bookIdentifies[d.BookId]
		}
		items = append(items, toDocumentBrief(d, ident))
	}
	return mcpdto.SearchDocumentOut{
		Total: len(items),
		Items: items,
	}
}

func toGetDocumentOut(doc *model.Document, bookIdentify string, maxChars int, includeTruncated bool) mcpdto.GetDocumentOut {
	md, t1 := truncateForGet(doc.Markdown, maxChars, includeTruncated)
	rel, t2 := truncateForGet(doc.Release, maxChars, includeTruncated)
	return mcpdto.GetDocumentOut{
		DocumentID:   doc.DocumentId,
		BookID:       doc.BookId,
		BookIdentify: bookIdentify,
		Title:        doc.DocumentName,
		Identify:     doc.Identify,
		Markdown:     md,
		Release:      rel,
		Version:      doc.Version,
		ParentID:     doc.ParentId,
		Truncated:    t1 || t2,
	}
}

func toBookBrief(b *dto.BookResult) mcpdto.BookBrief {
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
