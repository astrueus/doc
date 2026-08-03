package krand

import (
	"testing"
	"unicode"
)

func isDigits(b []byte) bool {
	for _, c := range b {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isLower(b []byte) bool {
	for _, c := range b {
		if !unicode.IsLower(rune(c)) {
			return false
		}
	}
	return true
}

func isUpper(b []byte) bool {
	for _, c := range b {
		if !unicode.IsUpper(rune(c)) {
			return false
		}
	}
	return true
}

func isAlphaNum(b []byte) bool {
	for _, c := range b {
		if !unicode.IsDigit(rune(c)) && !unicode.IsLetter(rune(c)) {
			return false
		}
	}
	return true
}

func TestKrand(t *testing.T) {
	cases := []struct {
		name  string
		size  int
		kind  int
		check func([]byte) bool
	}{
		{"num 8", 8, KC_RAND_KIND_NUM, isDigits},
		{"lower 10", 10, KC_RAND_KIND_LOWER, isLower},
		{"upper 10", 10, KC_RAND_KIND_UPPER, isUpper},
		{"all 16", 16, KC_RAND_KIND_ALL, isAlphaNum},
		{"size zero", 0, KC_RAND_KIND_NUM, func(b []byte) bool { return len(b) == 0 }},
	}
	for _, tc := range cases {
		got := Krand(tc.size, tc.kind)
		if len(got) != tc.size {
			t.Fatalf("%s: length got %d want %d", tc.name, len(got), tc.size)
		}
		if !tc.check(got) {
			t.Fatalf("%s: charset check failed for %q", tc.name, got)
		}
	}
}
