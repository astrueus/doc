package htmlutil

import (
	"strings"
	"testing"
)

func TestStripTags(t *testing.T) {
	in := `<p>Hello</p><script>alert(1)</script><style>.x{}</style><div>World</div>`
	got := StripTags(in)
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Fatalf("StripTags should remove tags: %q", got)
	}
	if !strings.Contains(got, "Hello") || !strings.Contains(got, "World") {
		t.Fatalf("StripTags should keep text: %q", got)
	}
	if strings.Contains(got, "alert") {
		t.Fatalf("StripTags should remove script content: %q", got)
	}
}

func TestAutoSummary(t *testing.T) {
	body := `<p>First paragraph here.</p><p>Second paragraph is longer text.</p>`
	got := AutoSummary(body, 100)
	if got == "" {
		t.Fatal("AutoSummary should not return empty for valid body")
	}
	if !strings.Contains(got, "First paragraph") {
		t.Fatalf("AutoSummary should include first paragraph: %q", got)
	}
	short := `<p>Hi</p><p>Bye there</p>`
	gotShort := AutoSummary(short, 2)
	if strings.Contains(gotShort, "Bye") {
		t.Fatalf("AutoSummary budget should stop before second paragraph: %q", gotShort)
	}
	if AutoSummary("<div>no p tags</div>", 50) != "" {
		t.Fatal("AutoSummary without <p> should return empty")
	}
	if AutoSummary(body, 0) != "" {
		t.Fatal("AutoSummary with l=0 should return empty")
	}
}
