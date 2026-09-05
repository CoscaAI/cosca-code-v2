package server

import "testing"

func TestStripMarkdownFences(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"```go\nfunc a() {}\n```", "func a() {}"},
		{"func a() {}", "func a() {}"},
		{"```\nplain\n```", "plain"},
		{"  \n code \n  ", "code"},
	}
	for _, c := range cases {
		if got := stripMarkdownFences(c.in); got != c.want {
			t.Errorf("stripMarkdownFences(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
