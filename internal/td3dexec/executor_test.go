package td3dexec

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/CoscaAI/cosca/pkg/engine"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad3DMissing(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "m", Type: "load_3d", Params: map[string]any{"path": "/nonexistent.obj"}}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected error for missing 3d file")
	}
}

func TestProbeOBJ(t *testing.T) {
	dir := t.TempDir()
	obj := writeFile(t, dir, "cube.obj", `v -1 -1 -1
v 1 -1 -1
v 1 1 -1
v -1 1 -1
usemtl bronze
f 1 2 3 4
`)
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "p", Type: "probe_3d", Params: map[string]any{"path": obj}}
	out, err := e.Run(context.Background(), node, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("probe out = %T", out)
	}
	if info["format"] != "obj" || info["vertices"] != 4 || info["faces"] != 1 {
		t.Fatalf("probe obj: %+v", info)
	}
	if len(info["materials"].([]string)) != 1 {
		t.Fatalf("materials: %+v", info["materials"])
	}
}

func TestProbeGLTF(t *testing.T) {
	dir := t.TempDir()
	gltf := writeFile(t, dir, "scene.gltf", `{"meshes":[{"name":"M","primitives":[{}]}],"materials":[{"name":"Gold"}]}`)
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "p", Type: "probe_3d", Params: map[string]any{"path": gltf}}
	out, err := e.Run(context.Background(), node, nil)
	if err != nil {
		t.Fatal(err)
	}
	info := out.(map[string]any)
	if info["format"] != "gltf" || info["faces"] != 1 {
		t.Fatalf("probe gltf: %+v", info)
	}
}

func TestUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	bad := writeFile(t, dir, "doc.txt", "not 3d")
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "p", Type: "probe_3d", Params: map[string]any{"path": bad}}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func TestUnsupportedNode(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "x", Type: "game_scene"}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected unsupported node error")
	}
}
