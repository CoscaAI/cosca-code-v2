package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchText(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {\n\tprintln(\"hello cosmos\")\n}\n")
	write(t, dir, "util.go", "package main\n\nfunc helper() {}\n")
	mkdir(t, dir, "node_modules")
	write(t, dir, "node_modules/junk.go", "package junk\n\nprintln(\"hello cosmos\")\n")

	res, err := Search(dir, Options{Query: "hello cosmos"})
	if err != nil {
		t.Fatal(err)
	}
	// Deve achar só em main.go (node_modules ignorado).
	if res.Total != 1 || len(res.Matches) != 1 {
		t.Fatalf("Total = %d, Matches = %v, want 1 em main.go", res.Total, res.Matches)
	}
	if res.Matches[0].Path != "main.go" {
		t.Fatalf("Path = %q, want main.go", res.Matches[0].Path)
	}
	if res.Matches[0].Line != 4 {
		t.Fatalf("Line = %d, want 4", res.Matches[0].Line)
	}
}

func TestSearchRegex(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package main\n\nfunc foo1() {}\nfunc bar2() {}\n")

	res, err := Search(dir, Options{Query: `func (foo|bar)\d`, Regex: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("Total = %d, want 2", res.Total)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package main\n\nconst NAME = \"x\"\n")

	res, err := Search(dir, Options{Query: "name"}) // case-insensitive por padrão
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 {
		t.Fatalf("Total = %d, want 1 (case-insensitive)", res.Total)
	}
}

func TestSearchInclude(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package main\n\nvar x = target\n")
	write(t, dir, "b.txt", "target em txt\n")

	res, err := Search(dir, Options{Query: "target", Include: ".go"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Matches[0].Path != "a.go" {
		t.Fatalf("Total = %d, Matches = %v, want 1 em a.go", res.Total, res.Matches)
	}
}

func TestSearchInvalidRegex(t *testing.T) {
	_, err := Search(t.TempDir(), Options{Query: "[", Regex: true})
	if err == nil {
		t.Fatal("regex inválida deveria retornar erro")
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

func mkdir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}
