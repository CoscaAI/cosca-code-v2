package gameexec

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/CoscaAI/cosca/pkg/engine"
)

const levelJSON = `{"name":"level-1","width":800,"height":600,"entities":[
  {"id":"player","name":"player","components":[
    {"type":"transform","params":{"x":0,"y":0}},
    {"type":"physics","params":{"gravity":true}},
    {"type":"input"}]},
  {"id":"enemy","name":"enemy_1","components":[
    {"type":"transform"},
    {"type":"ai"},
    {"type":"health","params":{"hp":100}}]}]}`

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSceneInfo(t *testing.T) {
	dir := t.TempDir()
	scene := writeFile(t, dir, "level.json", levelJSON)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "s", Type: "scene_info", Params: map[string]any{"path": scene}}
	out, err := e.Run(context.Background(), node, nil)
	if err != nil {
		t.Fatal(err)
	}
	info := out.(map[string]any)
	if info["name"] != "level-1" || info["entities"] != 2 {
		t.Fatalf("scene_info: %+v", info)
	}
	if info["players"] != 1 || info["enemies"] != 1 {
		t.Fatalf("players/enemies: %+v", info)
	}
	comps := info["components"].(map[string]int)
	if comps["transform"] != 2 || comps["input"] != 1 || comps["ai"] != 1 {
		t.Fatalf("components: %+v", comps)
	}
}

func TestAddEntity(t *testing.T) {
	dir := t.TempDir()
	scene := writeFile(t, dir, "level.json", levelJSON)

	e := New(filepath.Join(dir, "work"))
	// Constrói components via JSON no param (o executor lê de map[string]any
	// após deserialização — no teste, monta o node com params tipados).
	node := &engine.Node{
		ID: "a", Type: "add_entity",
		Params: map[string]any{
			"id": "coin_1", "name": "coin",
			"components": []any{
				map[string]any{"type": "transform"},
				map[string]any{"type": "score", "params": map[string]any{"value": 10}},
			},
		},
		Inputs: []string{"src"},
	}
	out, err := e.Run(context.Background(), node, map[string]any{"src": scene})
	if err != nil {
		t.Fatal(err)
	}
	info := out.(map[string]any)
	if info["id"] != "coin_1" || info["entities"] != 3 {
		t.Fatalf("add_entity: %+v", info)
	}
}

func TestLoadSceneMissing(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "s", Type: "load_scene", Params: map[string]any{"path": "/nonexistent.json"}}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected error for missing scene")
	}
}

func TestUnsupportedNode(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "x", Type: "render_final"}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected unsupported node error")
	}
}
