package mcp

import "testing"

func TestClassifyToolKindFromBody(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{``, "read"},
		{`{"method":"tools/list"}`, "read"},
		{`{"method":"tools/call","params":{"name":"search_document"}}`, "read"},
		{`{"method":"tools/call","params":{"name":"create_document"}}`, "write"},
		{`{"method":"tools/call","params":{"name":"delete_document"}}`, "delete"},
	}
	for _, c := range cases {
		if got := classifyToolKindFromBody([]byte(c.body)); got != c.want {
			t.Fatalf("body=%q got=%s want=%s", c.body, got, c.want)
		}
	}
}

func TestAllowByTokenBuckets(t *testing.T) {
	globalRateLimiter = newTokenRateLimiter(6) // write=3 delete=1
	if !AllowByToken(42, "read") {
		t.Fatal("first read should allow")
	}
	if !AllowByToken(42, "write") {
		t.Fatal("first write should allow")
	}
	if !AllowByToken(42, "delete") {
		t.Fatal("first delete should allow")
	}
	if AllowByToken(42, "delete") {
		t.Fatal("second delete should rate-limit with burst=1")
	}
}
