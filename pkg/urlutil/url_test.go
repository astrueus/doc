package urlutil

import "testing"

func TestJoinURI(t *testing.T) {
	cases := []struct {
		name string
		elem []string
		want string
	}{
		{"empty", nil, ""},
		{"single no slash", []string{"http://example.com"}, "http://example.com/"},
		{"single with slash", []string{"http://example.com/"}, "http://example.com/"},
		{"two parts", []string{"http://example.com", "path/to"}, "http://example.com/path/to"},
		{"leading slash on second", []string{"http://example.com/", "/api/v1"}, "http://example.com/api/v1"},
		{"backslash", []string{"http://example.com", "path\\to\\file"}, "http://example.com/path/to/file"},
		{"double slash in segment", []string{"http://example.com", "//api//v1"}, "http://example.com/api/v1"},
		{"three parts concat", []string{"http://host/", "docs", "page.html"}, "http://host/docspage.html"},
		{"path in second segment", []string{"http://host/", "docs/page.html"}, "http://host/docs/page.html"},
	}
	for _, tc := range cases {
		got := JoinURI(tc.elem...)
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}
