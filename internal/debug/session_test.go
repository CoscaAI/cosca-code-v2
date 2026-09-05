package debug

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSessionIntegrationGo usa o dlv real (se disponível) para validar o fluxo
// end-to-end: launch → breakpoint → stopped → stackTrace → variables.
func TestSessionIntegrationGo(t *testing.T) {
	if detectDlv() == "" {
		t.Skip("dlv não disponível")
	}

	dir := t.TempDir()
	src := `package main

import "fmt"

func add(a, b int) int {
	return a + b
}

func main() {
	x := add(2, 3)
	fmt.Println(x)
}
`
	mainPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Launch(mainPath, 38697)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	defer s.Disconnect()

	// Breakpoint na linha do "return a + b" (linha 6, 1-indexado).
	if err := s.SetBreakpoint(mainPath, 6); err != nil {
		t.Fatalf("SetBreakpoint: %v", err)
	}
	if err := s.ConfigurationDone(); err != nil {
		t.Fatalf("ConfigurationDone: %v", err)
	}

	threadID, err := s.WaitStop(20 * 1e9)
	if err != nil {
		t.Fatalf("WaitStop: %v", err)
	}

	frames, err := s.StackTrace(threadID)
	if err != nil {
		t.Fatalf("StackTrace: %v", err)
	}
	if len(frames) == 0 {
		t.Fatal("stack trace vazia")
	}
	// O frame topo deve ser "add" (parado dentro da função).
	if frames[0].Name == "" {
		t.Fatal("frame sem nome")
	}
	t.Logf("frame topo: %s (linha %d)", frames[0].Name, frames[0].Line)

	vars, err := s.Variables(frames[0].ID)
	if err != nil {
		t.Fatalf("Variables: %v", err)
	}
	t.Logf("variáveis: %+v", vars)

	// Advanced: evaluate — avalia "a + b" no contexto do frame.
	result, err := s.Evaluate(frames[0].ID, "a + b")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	t.Logf("evaluate a + b = %s", result)
	if !strings.Contains(result, "5") {
		t.Fatalf("evaluate a + b = %q, want contém 5", result)
	}
}
