package ui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestMiddleEllipsize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"short string untouched", "Zelda.nsp", 40, "Zelda.nsp"},
		{"exactly max untouched", "abcdef", 6, "abcdef"},
		{"tiny max untouched", "abcdefghij", 3, "abcdefghij"},
		{"keeps both ends", "The Legend of Zelda TOTK (DLC Pack 2).nsp", 20, "The Legen...k 2).nsp"},
	}
	for _, c := range cases {
		got := middleEllipsize(c.in, c.max)
		if got != c.want {
			t.Errorf("%s: middleEllipsize(%q, %d) = %q, want %q", c.name, c.in, c.max, got, c.want)
		}
	}
}

// Truncated results must not exceed the budget, and must preserve the tail so
// files differing only near the end stay distinguishable.
func TestMiddleEllipsizeBudgetAndTail(t *testing.T) {
	long := "Super Long Game Name (Update v1.2.1) [Region].nsp"
	out := middleEllipsize(long, 24)
	if n := utf8.RuneCountInString(out); n > 24 {
		t.Errorf("result %q has %d runes, exceeds max 24", out, n)
	}
	if !strings.HasSuffix(out, ".nsp") {
		t.Errorf("expected tail preserved, got %q", out)
	}
	if !strings.Contains(out, "...") {
		t.Errorf("expected ellipsis in %q", out)
	}
}
