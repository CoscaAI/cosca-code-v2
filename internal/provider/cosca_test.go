package provider

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCoscaFlattenMessages(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "Você é um assistente de engenharia."},
		{Role: "user", Content: "Refatore esse código."},
		{Role: "assistant", Content: "Claro, aqui está."},
		{Role: "user", Content: "Agora adicione testes."},
	}
	got := flattenMessages(msgs)
	for _, want := range []string{"[system]", "Você é um assistente", "[user]", "Refatore esse código.", "[assistant]", "Agora adicione testes."} {
		if !strings.Contains(got, want) {
			t.Fatalf("flattenMessages: esperava conter %q; got:\n%s", want, got)
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(got), "[system]") {
		t.Fatalf("flattenMessages: deveria começar com [system]; got:\n%s", got)
	}
	// Mensagens vazias devem ser ignoradas.
	got2 := flattenMessages([]Message{{Role: "user", Content: "   "}})
	if got2 != "" {
		t.Fatalf("flattenMessages: conteúdo em branco deveria resultar em vazio; got %q", got2)
	}
}

func TestCoscaStableModelsAndDefault(t *testing.T) {
	if DefaultCoscaModel() != "cosca/kernel" {
		t.Fatalf("DefaultCoscaModel = %q, esperava cosca/kernel", DefaultCoscaModel())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Aponta para uma porta quase certamente sem daemon → cai para a lista
	// estável (fallback, sem erro). Não aborta se por acaso houver um daemon.
	c, err := NewCosca("http://127.0.0.1:14121", "")
	if err != nil {
		t.Fatalf("NewCosca: %v", err)
	}
	if c.Name() != "cosca" {
		t.Fatalf("Name() = %q, esperava cosca", c.Name())
	}
	models, err := c.Models(ctx)
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("Models: lista estável vazia")
	}
	found := false
	for _, m := range models {
		if m.ID == "cosca/kernel" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Models: não contém cosca/kernel; got %v", models)
	}
}

func TestCoscaRuntimeAddr(t *testing.T) {
	cases := map[string]string{
		"http://127.0.0.1:14120":  "127.0.0.1:14120",
		"https://127.0.0.1:14120": "127.0.0.1:14120",
		"127.0.0.1:14120":         "127.0.0.1:14120",
		"  http://x:1/  ":         "x:1",
		"":                        "",
	}
	for in, want := range cases {
		if got := runtimeAddr(in); got != want {
			t.Fatalf("runtimeAddr(%q) = %q, esperava %q", in, got, want)
		}
	}
}

func TestCoscaPingDown(t *testing.T) {
	// Porta alta improvável de ter um daemon → Ping deve ser false (rápido).
	c, err := NewCosca("http://127.0.0.1:1", "")
	if err != nil {
		t.Fatalf("NewCosca: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if c.Ping(ctx) {
		t.Fatal("Ping deveria ser false com daemon fora do ar")
	}
}
