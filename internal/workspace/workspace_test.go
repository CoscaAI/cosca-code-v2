package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CoscaAI/cosca-code/internal/event"
)

func TestDetectGoProject(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/foo\n\ngo 1.25\n")
	write(t, dir, "main.go", "package main\nfunc main(){}\n")
	write(t, dir, "README.md", "# foo\n")
	mkdir(t, dir, ".git")

	info := Detect(dir)
	if info.Language != "go" {
		t.Fatalf("Language = %q, want go", info.Language)
	}
	if info.PackageManager != "go mod" {
		t.Fatalf("PackageManager = %q, want go mod", info.PackageManager)
	}
	if info.BuildCommand != "go build ./..." {
		t.Fatalf("BuildCommand = %q", info.BuildCommand)
	}
	if info.TestCommand != "go test ./..." {
		t.Fatalf("TestCommand = %q", info.TestCommand)
	}
	if !info.HasGit || !info.HasDocs {
		t.Fatalf("HasGit=%v HasDocs=%v, want true", info.HasGit, info.HasDocs)
	}
}

func TestDetectNodeReact(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"name":"x","dependencies":{"react":"^18","next":"^14"}}`)

	info := Detect(dir)
	if info.Language != "javascript" {
		t.Fatalf("Language = %q", info.Language)
	}
	if info.Framework != "next" {
		t.Fatalf("Framework = %q, want next", info.Framework)
	}
}

func TestDetectEmptyDir(t *testing.T) {
	dir := t.TempDir()
	info := Detect(dir)
	if info.Language != "" {
		t.Fatalf("Language = %q, want vazio", info.Language)
	}
	if info.HasGit || info.HasDocker || info.HasCI {
		t.Fatal("dir vazio não deve ter git/docker/ci")
	}
}

func TestScanTree(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, dir, "src")
	write(t, dir, "src/main.go", "package main\n")
	write(t, dir, "README.md", "# x\n")
	mkdir(t, dir, "node_modules") // deve ser ignorado
	write(t, dir, "node_modules/x.js", "// ignore me\n")

	tree := ScanTree(dir)
	files := tree.Files()

	want := map[string]bool{"src/main.go": true, "README.md": true}
	got := map[string]bool{}
	for _, f := range files {
		got[f] = true
	}
	for f := range want {
		if !got[f] {
			t.Errorf("falta %q em Files()", f)
		}
	}
	if got["node_modules/x.js"] {
		t.Error("node_modules não deveria estar na árvore")
	}
}

func TestWorkspaceOpenAndWatcher(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.txt", "hello")

	bus := event.NewBus()
	events := make(chan event.Event, 8)
	bus.SubscribeAll(func(ev event.Event) { events <- ev })

	w, err := Open(dir, bus)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer w.Close()
	w.watcher.SetInterval(50 * time.Millisecond)

	if w.Info().Language != "" {
		t.Fatalf("Language = %q, want vazio", w.Info().Language)
	}
	if len(w.Files()) != 1 {
		t.Fatalf("Files = %v, want 1 arquivo", w.Files())
	}

	// Modifica o arquivo → deve emitir file.changed.
	write(t, dir, "a.txt", "hello world")

	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Type == event.FileChanged && ev.Payload == "a.txt" {
				return // sucesso
			}
		case <-deadline:
			t.Fatal("timeout esperando file.changed")
		}
	}
}

func TestWorkspaceRejectsFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "notadir.txt")
	write(t, dir, "notadir.txt", "x")

	_, err := Open(f, event.NewBus())
	if err == nil {
		t.Fatal("Open de arquivo deveria falhar")
	}
}

// helpers

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}
