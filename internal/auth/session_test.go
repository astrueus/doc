package auth

import (
	"testing"

	"git.itopcms.com/jackliu/doc/internal/model"
)

func TestMemberIDFromSession(t *testing.T) {
	cases := []struct {
		name string
		v    any
		id   int
		ok   bool
	}{
		{"int", 7, 7, true},
		{"zero", 0, 0, false},
		{"int64", int64(3), 3, true},
		{"float64", float64(9), 9, true},
		{"member", model.Member{MemberId: 5}, 5, true},
		{"ptr", &model.Member{MemberId: 2}, 2, true},
		{"nil", nil, 0, false},
		{"string", "1", 0, false},
	}
	for _, tc := range cases {
		id, ok := MemberIDFromSession(tc.v)
		if id != tc.id || ok != tc.ok {
			t.Fatalf("%s: got (%d,%v) want (%d,%v)", tc.name, id, ok, tc.id, tc.ok)
		}
	}
}
