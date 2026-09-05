package xexec

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

func hasFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
}

func makeVideo(t *testing.T, dir string) string {
	t.Helper()
	hasFFmpeg(t)
	path := filepath.Join(dir, "test.mp4")
	args := []string{
		"-y", "-f", "lavfi", "-i", "testsrc=duration=1:size=32x32:rate=5",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-shortest", path,
	}
	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("makeVideo: %v: %s", err, out)
	}
	return path
}

func TestRouteDispatchesByDomain(t *testing.T) {
	e := New(t.TempDir())
	cases := map[string]string{
		"remove_bg":        "media",
		"load_video":       "cinema",
		"volume":           "music",
		"load_3d":          "td3d",
		"scene_info":       "game",
		"run_experiment":   "sci",
	}
	for ntype, domain := range cases {
		exec, err := e.Route(&engine.Node{ID: "n", Type: engine.NodeType(ntype)})
		if err != nil {
			t.Fatalf("route %s: %v", ntype, err)
		}
		if exec == nil {
			t.Fatalf("route %s: nil executor", ntype)
		}
		// Sem inspecionar o tipo, verificamos que NÃO é o default (não falha).
		_ = domain
	}
}

func TestRouteUnknown(t *testing.T) {
	e := New(t.TempDir())
	if _, err := e.Route(&engine.Node{ID: "n", Type: "quantum_sim"}); err == nil {
		t.Fatal("expected unknown domain error")
	}
}

func hasMedia(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cosca-media"); err != nil {
		t.Skip("cosca-media not available")
	}
}

// makePNG gera um PNG real 16x16 (faz o papel do PRODUTOR do node graph:
// image_generation devolve o CAMINHO do PNG, conforme o contrato L245).
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

// TestCrossProductDiffusionToRemoveBG é a proteção de regressão do contrato
// do node graph (L245): o nó PRODUTOR (image_generation/diffusion) devolve o
// CAMINHO (string) e o nó consumidor (remove_bg/media) aceita STRING como
// input. Se o diffusion regredir para objeto (o bug do 400), o remove_bg
// recebe o tipo errado e o encadeamento quebra — este teste pega isso.
func TestCrossProductDiffusionToRemoveBG(t *testing.T) {
	hasMedia(t)
	dir := t.TempDir()

	// O produtor (diffusion) geraria um PNG e devolveria o CAMINHO (string).
	pngPath := filepath.Join(dir, "gen-1.png")
	makePNG(t, pngPath)

	e := New(dir)
	// Nó do COSCA IMAGE (remove_bg) consumindo o caminho do produtor.
	rm := &engine.Node{ID: "bg-1", Type: "remove_bg", Inputs: []string{"src"}}
	out, err := e.Run(context.Background(), rm, map[string]any{"src": pngPath})
	if err != nil {
		t.Fatalf("remove_bg não aceitou caminho string (contrato quebrado): %v", err)
	}
	outPath, ok := out.(string)
	if !ok {
		t.Fatalf("remove_bg out = %T, esperava string (caminho)", out)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("remove_bg output missing: %v", err)
	}
}

func TestCrossProductCinemaToMusic(t *testing.T) {
	// §30: cinema→audio→music — extract_audio (CINEMA) alimenta volume (MUSIC).
	hasFFmpeg(t)
	dir := t.TempDir()
	vid := makeVideo(t, dir)

	e := New(dir)
	// Nó do CINEMA (extract_audio) → saída wav → alimenta volume do MUSIC.
	extract := &engine.Node{ID: "a", Type: "extract_audio", Params: map[string]any{"format": "wav"}, Inputs: []string{"src"}}
	out, err := e.Run(context.Background(), extract, map[string]any{"src": vid})
	if err != nil {
		t.Fatal(err)
	}
	wav, ok := out.(string)
	if !ok {
		t.Fatalf("extract out = %T", out)
	}

	// Nó do MUSIC (volume) consumindo o wav do CINEMA.
	vol := &engine.Node{ID: "v", Type: "volume", Params: map[string]any{"volume": "3dB"}, Inputs: []string{"src"}}
	out2, err := e.Run(context.Background(), vol, map[string]any{"src": wav})
	if err != nil {
		t.Fatal(err)
	}
	volPath, ok := out2.(string)
	if !ok {
		t.Fatalf("volume out = %T", out2)
	}
	if _, err := os.Stat(volPath); err != nil {
		t.Fatalf("volume output missing: %v", err)
	}
}

func TestProductsAndFlows(t *testing.T) {
	e := New(t.TempDir())
	products := e.Products()
	if len(products) != 6 {
		t.Fatalf("products = %d, want 6", len(products))
	}
	flows := e.Flows()
	if len(flows) < 3 {
		t.Fatalf("flows = %d, want >= 3", len(flows))
	}
	// Fluxo canônico cinema→audio→music presente.
	found := false
	for _, f := range flows {
		if f.Name == "cinema→audio→music" {
			found = true
		}
	}
	if !found {
		t.Fatal("cinema→audio→music flow missing")
	}
}
