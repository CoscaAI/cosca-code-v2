package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestManagerIntegrationGo usa o gopls real (se disponível) para validar o
// fluxo end-to-end: didOpen → publishDiagnostics → hover.
func TestManagerIntegrationGo(t *testing.T) {
	if _, _, err := Detect("go"); err != nil {
		t.Skipf("gopls não disponível: %v", err)
	}

	dir := t.TempDir()
	src := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	writeFile(t, dir, "main.go", src)

	m := NewManager(dir)
	defer m.Close()

	uri := uriFor(filepath.Join(dir, "main.go"))
	if err := m.OpenDocument(uri, "go", src); err != nil {
		t.Fatalf("OpenDocument: %v", err)
	}

	// Aguarda o gopls processar e publicar diagnostics (deve ser limpo).
	time.Sleep(3 * time.Second)
	diags := m.Diagnostics(uri)
	if len(diags) != 0 {
		t.Fatalf("diagnostics inesperados em código válido: %+v", diags)
	}

	// Hover em "Println" (linha 6, coluna ~8 → dentro do identificador).
	hover, err := m.Hover(uri, 5, 10)
	if err != nil {
		t.Fatalf("Hover: %v", err)
	}
	if hover == "" {
		t.Fatal("hover vazio para fmt.Println")
	}

	// Definition de "Println" deve apontar para fmt.
	locs, err := m.Definition(uri, 5, 10)
	if err != nil {
		t.Fatalf("Definition: %v", err)
	}
	if len(locs) == 0 {
		t.Fatal("definition vazia para fmt.Println")
	}
	if !strings.Contains(locs[0].URI, "fmt") {
		t.Fatalf("definition deveria apontar para fmt, got %q", locs[0].URI)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
