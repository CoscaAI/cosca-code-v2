package cinexec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// hasFFmpeg reporta se ffmpeg/ffprobe estão disponíveis (skip de testes E2E).
func cineHasFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}
}

// makeVideo gera um vídeo sintético 1s (test source + sine).
func makeVideo(t *testing.T, dir string) string {
	t.Helper()
	cineHasFFmpeg(t)
	path := filepath.Join(dir, "test.mp4")
	args := []string{
		"-y", "-f", "lavfi",
		"-i", "testsrc=duration=1:size=64x64:rate=10",
		"-f", "lavfi",
		"-i", "sine=frequency=440:duration=1",
		"-c:v", "libx264", "-preset", "ultrafast",
		"-c:a", "aac", "-shortest", path,
	}
	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("makeVideo: %v: %s", err, out)
	}
	return path
}

func TestProbeVideo(t *testing.T) {
	cineHasFFmpeg(t)
	dir := t.TempDir()
	vid := makeVideo(t, dir)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "p", Type: "probe", Params: map[string]any{"path": vid}}
	out, err := e.Run(context.Background(), node, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("probe out = %T, want map", out)
	}
	if info["has_video"] != true || info["has_audio"] != true {
		t.Fatalf("probe result: %+v", info)
	}
}

func TestLoadVideoMissing(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "v", Type: "load_video", Params: map[string]any{"path": "/nonexistent.mp4"}}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected error for missing video")
	}
}

func TestExtractAudio(t *testing.T) {
	cineHasFFmpeg(t)
	dir := t.TempDir()
	vid := makeVideo(t, dir)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "a", Type: "extract_audio", Params: map[string]any{"format": "wav"}, Inputs: []string{"src"}}
	out, err := e.Run(context.Background(), node, map[string]any{"src": vid})
	if err != nil {
		t.Fatal(err)
	}
	path, ok := out.(string)
	if !ok || filepath.Ext(path) != ".wav" {
		t.Fatalf("extract_audio out = %v", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("wav missing: %v", err)
	}
}

func TestFrame(t *testing.T) {
	cineHasFFmpeg(t)
	dir := t.TempDir()
	vid := makeVideo(t, dir)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "f", Type: "frame", Params: map[string]any{"time": "00:00:00"}, Inputs: []string{"src"}}
	out, err := e.Run(context.Background(), node, map[string]any{"src": vid})
	if err != nil {
		t.Fatal(err)
	}
	path, ok := out.(string)
	if !ok || filepath.Ext(path) != ".png" {
		t.Fatalf("frame out = %v", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("frame png missing: %v", err)
	}
}

func TestUnsupportedNode(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "x", Type: "image_generation"}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected unsupported node error")
	}
}
