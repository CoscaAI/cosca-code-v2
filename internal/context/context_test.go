package context

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/CoscaAI/cosca-code/internal/intel"
	"github.com/CoscaAI/cosca-code/internal/workspace"
)

func TestBuildAndPrompt(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hi\") }\n"), 0o644)

	ws, err := workspace.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	idx, err := intel.Build(dir)
	if err != nil {
		t.Fatal(err)
	}

	c := Build(ws, idx)
	if c.Info.Language != "go" {
		t.Fatalf("Language = %q, want go", c.Info.Language)
	}

	p := c.Prompt(10)
	if p == "" {
		t.Fatal("Prompt vazio")
	}
	if !contains(p, "Linguagem: go") {
		t.Fatalf("Prompt não contém linguagem: %q", p)
	}
	if !contains(p, "main") {
		t.Fatalf("Prompt não contém símbolo main: %q", p)
	}
}

func TestPromptBudget(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644)
	for i := 0; i < 100; i++ {
		name := filepath.Join(dir, fmt.Sprintf("f%03d.go", i))
		os.WriteFile(name, []byte(fmt.Sprintf("package main\n\nfunc fn%d() {}\n", i)), 0o644)
	}
	ws, _ := workspace.Open(dir, nil)
	defer ws.Close()
	idx, _ := intel.Build(dir)

	c := Build(ws, idx)
	p := c.Prompt(5)
	if !contains(p, "... e mais") {
		t.Fatalf("Prompt deveria truncar símbolos (orçamento): %q", p)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
