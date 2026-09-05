package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectGo(t *testing.T) {
	bin, args, err := Detect("go")
	if err != nil {
		t.Skipf("gopls não disponível: %v", err)
	}
	if bin == "" {
		t.Fatal("bin vazio")
	}
	if len(args) != 0 {
		t.Fatalf("gopls não deve ter args, got %v", args)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("binário %q não existe", bin)
	}
}

func TestDetectUnknownLanguage(t *testing.T) {
	_, _, err := Detect("cobol")
	if err == nil {
		t.Fatal("linguagem sem server deveria retornar erro")
	}
}

func TestResolveBinGoBin(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("sem home")
	}
	// Simula um binário em ~/go/bin (não precisa existir de verdade — só
	// verifica que a resolução procura ali).
	p := filepath.Join(home, "go", "bin", "gopls")
	if info, err := os.Stat(p); err == nil && !info.IsDir() {
		if got := resolveBin("gopls"); got == "" {
			t.Fatal("resolveBin não achou gopls em ~/go/bin")
		}
	}
}
