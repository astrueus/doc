package sqltil

import "testing"

func TestEscapeLike(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "hello", "hello"},
		{"percent", "100%", "100\\%"},
		{"underscore", "a_b", "a\\_b"},
		{"both", "%_x%_", "\\%\\_x\\%\\_"},
		{"backslash percent", "\\%", "\\\\%"},
	}
	for _, tc := range cases {
		got := EscapeLike(tc.in)
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}
