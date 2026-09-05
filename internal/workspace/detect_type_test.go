package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectType_NoManifest(t *testing.T) {
	dir := t.TempDir()
	info := Detect(dir)
	if info.ProjectType != "editor" {
		t.Fatalf("ProjectType = %q, want editor (default)", info.ProjectType)
	}
	if info.ProjectTypeSource != "default" {
		t.Fatalf("ProjectTypeSource = %q, want default", info.ProjectTypeSource)
	}
}

func TestDetectProjectType_FromManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".cosca"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ProjectManifestFile), []byte("name: meu-curta\ntype: cinema\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info := Detect(dir)
	if info.ProjectType != "cinema" {
		t.Fatalf("ProjectType = %q, want cinema", info.ProjectType)
	}
	if info.ProjectTypeSource != "manifest" {
		t.Fatalf("ProjectTypeSource = %q, want manifest", info.ProjectTypeSource)
	}
}

func TestDetectProjectType_InvalidFallsBack(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".cosca"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Tipo inválido no manifest → default editor.
	if err := os.WriteFile(filepath.Join(dir, ProjectManifestFile), []byte("type: holograma\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info := Detect(dir)
	if info.ProjectType != "editor" {
		t.Fatalf("ProjectType = %q, want editor (fallback para tipo inválido)", info.ProjectType)
	}
	if info.ProjectTypeSource != "default" {
		t.Fatalf("ProjectTypeSource = %q, want default", info.ProjectTypeSource)
	}
}

func TestDetectProjectType_AllValidTypes(t *testing.T) {
	for typ := range validProjectTypes {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".cosca"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ProjectManifestFile), []byte("type: "+typ+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		info := Detect(dir)
		if info.ProjectType != typ {
			t.Fatalf("ProjectType = %q, want %q", info.ProjectType, typ)
		}
		if info.ProjectTypeSource != "manifest" {
			t.Fatalf("ProjectTypeSource = %q, want manifest (type %s)", info.ProjectTypeSource, typ)
		}
	}
}

func TestDetectProjectType_ManifestWithQuotes(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".cosca"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ProjectManifestFile), []byte("type: \"game\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info := Detect(dir)
	if info.ProjectType != "game" {
		t.Fatalf("ProjectType = %q, want game (com aspas)", info.ProjectType)
	}
}

func TestDetectCombinesLanguageAndType(t *testing.T) {
	dir := t.TempDir()
	// Projeto Go + manifest cinema: linguagem e modo coexistem.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".cosca"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ProjectManifestFile), []byte("type: cinema\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info := Detect(dir)
	if info.Language != "go" {
		t.Fatalf("Language = %q, want go", info.Language)
	}
	if info.ProjectType != "cinema" {
		t.Fatalf("ProjectType = %q, want cinema", info.ProjectType)
	}
}
