package provider

import (
	"context"
	"testing"
	"time"
)

func TestOllamaDetectAndModels(t *testing.T) {
	o := NewOllama("http://127.0.0.1:11434")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	models, err := o.Models(ctx)
	if err != nil {
		t.Skipf("ollama não disponível: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("ollama sem modelos")
	}
	t.Logf("modelos: %v", models)
}

func TestRouterResolveFallback(t *testing.T) {
	r := NewRegistry()
	r.Register(NewOllama("http://127.0.0.1:11434"))

	router := NewRouter(r)
	router.SetDefault("ollama", "qwen2.5-coder:14b-128k")

	// Resolve com provider/modelo explícitos.
	p, m, err := router.Resolve("ollama", "qwen2.5-coder:14b-128k")
	if err != nil || p.Name() != "ollama" || m != "qwen2.5-coder:14b-128k" {
		t.Fatalf("Resolve explícito = %v %v %v", p, m, err)
	}

	// Resolve com AUTO (vazios → default).
	p, m, err = router.Resolve("", "")
	if err != nil || p.Name() != "ollama" || m != "qwen2.5-coder:14b-128k" {
		t.Fatalf("Resolve AUTO = %v %v %v", p, m, err)
	}
}

func TestRouterUnknownProvider(t *testing.T) {
	r := NewRegistry()
	router := NewRouter(r)
	if _, _, err := router.Resolve("nope", "x"); err == nil {
		t.Fatal("provider inexistente deveria retornar erro")
	}
}
