package intel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildGoSymbols(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", `package main

import "fmt"

func main() {
	fmt.Println("hi")
}

type Server struct {
	Port int
}

func (s *Server) Start() error {
	return nil
}
`)
	write(t, dir, "util.go", `package main

import "os"

func helper() string {
	return os.Getenv("X")
}
`)

	idx, err := Build(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Símbolos esperados.
	names := map[string]bool{}
	for _, s := range idx.Symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"main", "Server", "Start", "helper"} {
		if !names[want] {
			t.Errorf("símbolo %q não indexado", want)
		}
	}

	// Dependências: main.go importa fmt; util.go importa os.
	deps := idx.Dependencies["main.go"]
	if !contains(deps, "fmt") {
		t.Errorf("main.go deveria importar fmt, imports = %v", deps)
	}
	if !contains(idx.Dependencies["util.go"], "os") {
		t.Errorf("util.go deveria importar os")
	}
}

func TestUsagesAndDependents(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package main\n\nimport \"b\"\n\nfunc alpha() {}\n")
	write(t, dir, "b.go", "package main\n\nfunc beta() {}\n")

	idx, _ := Build(dir)

	usages := idx.Usages("alpha")
	if len(usages) != 1 || usages[0].File != "a.go" {
		t.Fatalf("Usages(alpha) = %+v", usages)
	}

	// a.go importa "b" → quem depende de "b"?
	deps := idx.Dependents("b")
	if !contains(deps, "a.go") {
		t.Fatalf("Dependents(b) = %v, want incluir a.go", deps)
	}
}

func TestBuildSkipsIgnored(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc real() {}\n")
	write(t, dir, "node_modules/x.js", "function fake() {}\n")

	idx, _ := Build(dir)
	for _, s := range idx.Symbols {
		if s.File == "node_modules/x.js" {
			t.Fatal("node_modules não deveria ser indexado")
		}
	}
}

// helpers

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
