package mcp

import (
	"strings"
	"testing"
	"unicode/utf8"

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

func TestTruncateForGet(t *testing.T) {
	s := "你好世界ABC"
	got, truncated := truncateForGet(s, 0, true)
	if truncated || got != s {
		t.Fatalf("no truncate: got=%q truncated=%v", got, truncated)
	}

	got, truncated = truncateForGet(s, 4, true)
	if !truncated {
		t.Fatal("expected truncated")
	}
	if got != "你好世界"+truncatedMarker {
		t.Fatalf("got=%q", got)
	}

	got, truncated = truncateForGet(s, 4, false)
	if !truncated || got != "你好世界" {
		t.Fatalf("no marker: got=%q truncated=%v", got, truncated)
	}

	got, truncated = truncateForGet(s, 500, true)
	if truncated || got != s {
		t.Fatalf("short text should not truncate: got=%q", got)
	}
}

func TestClampMaxChars(t *testing.T) {
	if clampMaxChars(0) != 0 || clampMaxChars(-1) != 0 {
		t.Fatal("<=0 should be 0")
	}
	if clampMaxChars(maxGetCharsCap+1) != maxGetCharsCap {
		t.Fatal("should clamp")
	}
}

func TestParseIfExists(t *testing.T) {
	got, err := parseIfExists("")
	if err != nil || got != "error" {
		t.Fatalf("default: %q %v", got, err)
	}
	got, err = parseIfExists("UPDATE")
	if err != nil || got != "update" {
		t.Fatalf("update: %q %v", got, err)
	}
	if _, err := parseIfExists("upsert"); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveDocIdentify(t *testing.T) {
	got, err := resolveDocIdentify("a", "")
	if err != nil || got != "a" {
		t.Fatalf("a: %q %v", got, err)
	}
	got, err = resolveDocIdentify("", "b")
	if err != nil || got != "b" {
		t.Fatalf("b: %q %v", got, err)
	}
	if _, err := resolveDocIdentify("a", "b"); err == nil {
		t.Fatal("mismatch should error")
	}
}

func TestToSearchOutIdentifies(t *testing.T) {
	docs := []*model.Document{{
		DocumentId:   12,
		BookId:       3,
		Identify:     "intro",
		DocumentName: "简介",
		Markdown:     "hello",
		Version:      1,
	}}
	out := toSearchOut(docs, map[int]string{3: "my-book"})
	if len(out.Items) != 1 {
		t.Fatalf("len=%d", len(out.Items))
	}
	item := out.Items[0]
	if item.BookIdentify != "my-book" || item.DocIdentify != "intro" {
		t.Fatalf("%+v", item)
	}
}

func TestToGetDocumentOutTruncated(t *testing.T) {
	doc := &model.Document{
		DocumentId:   1,
		BookId:       2,
		DocumentName: "t",
		Identify:     "x",
		Markdown:     "abcdefghij",
		Release:      "<p>abcdefghij</p>",
		Version:      9,
	}
	out := toGetDocumentOut(doc, "bk", 4, true)
	if !out.Truncated {
		t.Fatal("expected truncated")
	}
	if out.Title != "t" || out.Identify != "x" || out.Version != 9 {
		t.Fatalf("meta changed: %+v", out)
	}
	if utf8.RuneCountInString(strings.TrimSuffix(out.Markdown, truncatedMarker)) != 4 {
		t.Fatalf("markdown=%q", out.Markdown)
	}
}
