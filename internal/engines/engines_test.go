package engines

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CoscaAI/cosca/pkg/engine"
)

func TestOpenCreatesRegistries(t *testing.T) {
	root := t.TempDir()
	e, err := Open(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if e.AssetRegistry() == nil {
		t.Fatal("asset registry should be created")
	}
	if e.Provenance() == nil {
		t.Fatal("provenance registry should be created")
	}
	if e.Root() != root {
		t.Fatalf("Root = %q, want %q", e.Root(), root)
	}
}

func TestProjectManifest(t *testing.T) {
	root := t.TempDir()
	// Sem manifest → erro (os.IsNotExist).
	if _, err := engine.ReadProjectManifest(root); err == nil {
		t.Fatal("expected error for missing manifest")
	}
	// Cria manifest e lê via engine.
	if _, err := engine.NewProjectManifest("meu-curta", engine.TypeCinema); err != nil {
		t.Fatal(err)
	}
	// Write usa a API interna — aqui simulamos via bridge: escreve project.yaml.
	dir := filepath.Join(root, ".cosca")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.yaml"), []byte("name: meu-curta\ntype: cinema\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := Open(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.ProjectManifest()
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != string(engine.TypeCinema) {
		t.Fatalf("type = %q, want cinema", got.Type)
	}
}

func TestAssetAddAndList(t *testing.T) {
	root := t.TempDir()
	e, err := Open(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Arquivo de teste dentro do workspace.
	src := filepath.Join(root, "logo.png")
	if err := os.WriteFile(src, []byte("fake-png"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := e.assets.AddFile(src, engine.AssetType(engine.TypeImage), src)
	if err != nil {
		t.Fatal(err)
	}
	if a.Type != engine.AssetType(engine.TypeImage) {
		t.Fatalf("type = %q, want image", a.Type)
	}
	if e.assets.Count() != 1 {
		t.Fatalf("Count = %d, want 1", e.assets.Count())
	}
	// Dedup: mesmo conteúdo → mesmo ID.
	a2, err := e.assets.AddFile(src, engine.AssetType(engine.TypeImage), "outra-fonte")
	if err != nil {
		t.Fatal(err)
	}
	if a2.ID != a.ID {
		t.Fatalf("dedup failed: %s != %s", a2.ID, a.ID)
	}
}

func TestLoadGraph(t *testing.T) {
	data := []byte(`{"name":"wf","nodes":[{"id":"a","type":"load"},{"id":"b","type":"seg","inputs":["a"]}]}`)
	g, err := LoadGraph(data)
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "wf" || len(g.Nodes) != 2 {
		t.Fatalf("graph mismatch: %+v", g)
	}
}
