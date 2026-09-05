package musexec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// musHasFFmpeg reporta se ffmpeg está disponível.
func musHasFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
}

// makeAudio gera um tom de 1s (sine 440Hz) em wav.
func makeAudio(t *testing.T, dir string) string {
	t.Helper()
	musHasFFmpeg(t)
	path := filepath.Join(dir, "tone.wav")
	args := []string{"-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-c:a", "pcm_s16le", path}
	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("makeAudio: %v: %s", err, out)
	}
	return path
}

func TestProbeAudio(t *testing.T) {
	musHasFFmpeg(t)
	dir := t.TempDir()
	audio := makeAudio(t, dir)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "p", Type: "probe_audio", Params: map[string]any{"path": audio}}
	out, err := e.Run(context.Background(), node, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("probe out = %T, want map", out)
	}
	if info["has_audio"] != true {
		t.Fatalf("probe: %+v", info)
	}
	if info["codec"] == "" {
		t.Fatalf("probe missing codec: %+v", info)
	}
}

func TestConvertAudio(t *testing.T) {
	musHasFFmpeg(t)
	dir := t.TempDir()
	audio := makeAudio(t, dir)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "c", Type: "convert_audio", Params: map[string]any{"format": "mp3"}, Inputs: []string{"src"}}
	out, err := e.Run(context.Background(), node, map[string]any{"src": audio})
	if err != nil {
		t.Fatal(err)
	}
	path, ok := out.(string)
	if !ok || filepath.Ext(path) != ".mp3" {
		t.Fatalf("convert out = %v", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("mp3 missing: %v", err)
	}
}

func TestVolume(t *testing.T) {
	musHasFFmpeg(t)
	dir := t.TempDir()
	audio := makeAudio(t, dir)

	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "v", Type: "volume", Params: map[string]any{"volume": "3dB"}, Inputs: []string{"src"}}
	out, err := e.Run(context.Background(), node, map[string]any{"src": audio})
	if err != nil {
		t.Fatal(err)
	}
	path, ok := out.(string)
	if !ok {
		t.Fatalf("volume out = %T", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("volume output missing: %v", err)
	}
}

func TestLoadAudioMissing(t *testing.T) {
	dir := t.TempDir()
	e := New(filepath.Join(dir, "work"))
	node := &engine.Node{ID: "a", Type: "load_audio", Params: map[string]any{"path": "/nonexistent.wav"}}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected error for missing audio")
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
