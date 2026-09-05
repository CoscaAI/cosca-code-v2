package tool

import (
	"context"
	"testing"

	"github.com/CoscaAI/cosca-code/internal/autonomy"
)

func TestRegistryRegisterGetCall(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{
		Name:        "echo",
		Description: "eco",
		InputSchema: map[string]any{"type": "object"},
		Risk:        autonomy.ActionRead,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return args["text"].(string), nil
		},
	})

	if _, ok := r.Get("echo"); !ok {
		t.Fatal("echo não registrada")
	}
	out, err := r.Call(context.Background(), "echo", map[string]any{"text": "olá"})
	if err != nil || out != "olá" {
		t.Fatalf("Call = %q, %v", out, err)
	}
}

func TestRegistryCallUnknown(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Call(context.Background(), "nope", nil); err == nil {
		t.Fatal("tool inexistente deveria retornar erro")
	}
}

func TestRegistryListOrdered(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{Name: "b"})
	r.Register(&Tool{Name: "a"})
	r.Register(&Tool{Name: "c"})

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("List = %d, want 3", len(list))
	}
	// Ordenado alfabeticamente.
	if list[0].Name != "a" || list[1].Name != "b" || list[2].Name != "c" {
		t.Fatalf("ordem errada: %v", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}
