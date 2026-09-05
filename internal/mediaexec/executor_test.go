package mediaexec

import (
	"context"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// hasMedia reporta se a cosca-media está no PATH (skip de testes E2E).
func hasMedia(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cosca-media"); err != nil {
		t.Skip("cosca-media not available")
	}
}

// makePNG gera um PNG real 16x16 vermelho.
func makePNG(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for i := range img.Pix {
		img.Pix[i] = 255 // branco
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestIdentifyNode(t *testing.T) {
	hasMedia(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "logo.png")
	makePNG(t, src)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "n1", Type: "identify", Inputs: []string{"src"}}
	input := map[string]any{"src": src}

	out, err := e.Run(context.Background(), node, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(string); !ok {
		t.Fatalf("identify out = %T, want string", out)
	}
}

func TestResizeNode(t *testing.T) {
	hasMedia(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "logo.png")
	makePNG(t, src)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{
		ID: "n2", Type: "resize",
		Params: map[string]any{"width": 8.0, "height": 8.0},
		Inputs: []string{"src"},
	}
	out, err := e.Run(context.Background(), node, map[string]any{"src": src})
	if err != nil {
		t.Fatal(err)
	}
	path, ok := out.(string)
	if !ok {
		t.Fatalf("resize out = %T, want string", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("resized file missing: %v", err)
	}
}

func TestConvertNode(t *testing.T) {
	hasMedia(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "logo.png")
	makePNG(t, src)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{
		ID: "n3", Type: "convert",
		Params: map[string]any{"format": "webp"},
		Inputs: []string{"src"},
	}
	out, err := e.Run(context.Background(), node, map[string]any{"src": src})
	if err != nil {
		t.Fatal(err)
	}
	path, ok := out.(string)
	if !ok || filepath.Ext(path) != ".webp" {
		t.Fatalf("convert out = %v, want .webp path", out)
	}
}

func TestUnsupportedNode(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "x", Type: "video_generation"}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected unsupported node error")
	}
}

func TestNodeRequiresInput(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "x", Type: "resize"}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected missing input error")
	}
}
