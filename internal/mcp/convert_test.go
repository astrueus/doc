package mcp

import (
	"testing"

	"git.itopcms.com/astrueus/doc/internal/model"
)

func TestDocumentSnippetMarkdown(t *testing.T) {
	doc := &model.Document{Markdown: "hello world"}
	if got := documentSnippet(doc, 200); got != "hello world" {
		t.Fatalf("snippet=%q", got)
	}
}

func TestDocumentSnippetHTMLRelease(t *testing.T) {
	doc := &model.Document{Release: "<p>hi <b>there</b></p>"}
	got := documentSnippet(doc, 200)
	if got != "hi there" {
		t.Fatalf("snippet=%q", got)
	}
}
