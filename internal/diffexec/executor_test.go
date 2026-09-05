package diffexec

import (
	"context"
	"testing"

	"github.com/CoscaAI/cosca/pkg/engine"
)

func TestParamHelpers(t *testing.T) {
	node := &engine.Node{ID: "n", Type: "image_generation", Params: map[string]any{
		"prompt": "um gato", "steps": 15.0, "guidance": 8.0,
	}}
	if got := paramString(node, "prompt", "default"); got != "um gato" {
		t.Fatalf("prompt = %q", got)
	}
	if got := paramString(node, "missing", "default"); got != "default" {
		t.Fatalf("missing = %q", got)
	}
	if got := paramFloat(node, "steps", 25); got != 15 {
		t.Fatalf("steps = %v", got)
	}
	if got := paramFloat(node, "missing", 7.5); got != 7.5 {
		t.Fatalf("missing float = %v", got)
	}
}

func TestUnsupportedNode(t *testing.T) {
	e := New(t.TempDir())
	node := &engine.Node{ID: "x", Type: "video_generation"}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected unsupported node error")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcdef", 3); got != "abc..." {
		t.Fatalf("truncate = %q", got)
	}
	if got := truncate("ab", 3); got != "ab" {
		t.Fatalf("truncate short = %q", got)
	}
}

func TestResolveScriptPath(t *testing.T) {
	// Deve resolver sem erro (existe no repo).
	p := resolveScriptPath()
	if p == "" {
		t.Fatal("script path vazio")
	}
}
