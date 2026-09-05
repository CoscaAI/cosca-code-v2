package team

import (
	"testing"

	"github.com/CoscaAI/cosca-code/internal/provider"
	"github.com/CoscaAI/cosca-code/internal/tool"
)

func TestNewTeam(t *testing.T) {
	reg := provider.NewRegistry()
	router := provider.NewRouter(reg)
	tools := tool.NewRegistry()

	tm := NewTeam(router, tools, "go test ./...")
	if tm == nil {
		t.Fatal("NewTeam retornou nil")
	}
	if tm.maxRetries != 2 {
		t.Fatalf("maxRetries = %d, want 2", tm.maxRetries)
	}
	if tm.testCmd != "go test ./..." {
		t.Fatalf("testCmd = %q", tm.testCmd)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"curto", 100, "curto"},
		{"1234567890", 5, "12345..."},
	}
	for _, c := range cases {
		if got := truncate(c.in, c.n); got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}
