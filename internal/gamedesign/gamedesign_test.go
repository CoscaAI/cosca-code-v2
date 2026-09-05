package gamedesign

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// mockLLM devolve uma resposta fixa (simula o qwen gerando a cena).
type mockLLM struct {
	resp string
	err  error
}

func (m *mockLLM) Chat(_ context.Context, _ string, _ []Message) (string, error) {
	return m.resp, m.err
}

const goodScene = `{"name":"level-1","width":800,"height":600,"entities":[
  {"id":"player","name":"player","components":[
    {"type":"transform","params":{"x":0,"y":0}},
    {"type":"physics","params":{"gravity":true}},
    {"type":"input"}]},
  {"id":"enemy","name":"enemy_1","components":[
    {"type":"transform"},{"type":"ai"},{"type":"health","params":{"hp":100}}]}]}`

func TestGenerateWithLLM(t *testing.T) {
	llm := &mockLLM{resp: goodScene}
	g := New(llm, "qwen2.5-coder:14b")
	scene, err := g.Generate(context.Background(), "crie um jogo de plataforma")
	if err != nil {
		t.Fatal(err)
	}
	ents, ok := scene["entities"].([]any)
	if !ok || len(ents) != 2 {
		t.Fatalf("entities = %v", scene["entities"])
	}
	first := ents[0].(map[string]any)
	if first["id"] != "player" {
		t.Fatalf("first entity = %v", first["id"])
	}
}

func TestGenerateWithMarkdownFence(t *testing.T) {
	// LLM devolve com fence ```json ... ```
	llm := &mockLLM{resp: "```json\n" + goodScene + "\n```"}
	g := New(llm, "qwen")
	scene, err := g.Generate(context.Background(), "crie um jogo")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := scene["entities"]; !ok {
		t.Fatal("fence stripping failed")
	}
}

func TestGenerateLLMFailure(t *testing.T) {
	llm := &mockLLM{err: errLLM}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "jogo"); err == nil {
		t.Fatal("expected llm error")
	}
	// Com fallback, não falha — devolve cena mínima.
	scene, source, err := g.GenerateWithFallback(context.Background(), "jogo")
	if err != nil {
		t.Fatal(err)
	}
	if source != "fallback" {
		t.Fatalf("source = %q, want fallback", source)
	}
	if _, ok := scene["entities"]; !ok {
		t.Fatal("fallback scene missing entities")
	}
}

func TestGenerateEmptyDescription(t *testing.T) {
	llm := &mockLLM{resp: goodScene}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty description")
	}
}

func TestGenerateInvalidJSON(t *testing.T) {
	llm := &mockLLM{resp: "isto não é json"}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "jogo"); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestGenerateSceneMissingEntities(t *testing.T) {
	llm := &mockLLM{resp: `{"name":"x"}`}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "jogo"); err == nil {
		t.Fatal("expected missing entities error")
	}
}

func TestStripCodeFence(t *testing.T) {
	in := "```json\n{\"a\":1}\n```"
	out := stripCodeFence(in)
	var v map[string]int
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("strip fence failed: %v — out=%q", err, out)
	}
	if v["a"] != 1 {
		t.Fatalf("value = %v", v)
	}
	// Sem fence: intacto.
	if got := stripCodeFence(`{"a":1}`); !strings.Contains(got, "a") {
		t.Fatal("no-fence input altered")
	}
}

var errLLM = &mockErr{}

type mockErr struct{}

func (e *mockErr) Error() string { return "llm unavailable" }

// TestSanitizeScene — refinamento pós-dogfooding (L239): turbo/coin com ai
// perde o ai; enemy com input perde o input; player preserva tudo.
func TestSanitizeScene(t *testing.T) {
	scene := map[string]any{
		"entities": []any{
			map[string]any{ // turbo (item coletável) — ai e input sobram
				"id": "turbo",
				"components": []any{
					map[string]any{"type": "transform"},
					map[string]any{"type": "render"},
					map[string]any{"type": "score"},
					map[string]any{"type": "ai"},
					map[string]any{"type": "input"},
				},
			},
			map[string]any{ // enemy — input sobra
				"id": "enemy",
				"components": []any{
					map[string]any{"type": "transform"},
					map[string]any{"type": "ai"},
					map[string]any{"type": "input"},
				},
			},
			map[string]any{ // player — preserva tudo
				"id": "player",
				"components": []any{
					map[string]any{"type": "transform"},
					map[string]any{"type": "input"},
				},
			},
		},
	}
	SanitizeScene(scene)

	ents := scene["entities"].([]any)
	turbo := ents[0].(map[string]any)["components"].([]any)
	if hasComponentType(turbo, "ai") || hasComponentType(turbo, "input") {
		t.Fatalf("turbo deve perder ai/input: %v", turbo)
	}
	if !hasComponentType(turbo, "score") {
		t.Fatalf("turbo deve manter score: %v", turbo)
	}
	enemy := ents[1].(map[string]any)["components"].([]any)
	if hasComponentType(enemy, "input") {
		t.Fatalf("enemy deve perder input: %v", enemy)
	}
	if !hasComponentType(enemy, "ai") {
		t.Fatalf("enemy deve manter ai: %v", enemy)
	}
	player := ents[2].(map[string]any)["components"].([]any)
	if !hasComponentType(player, "input") || !hasComponentType(player, "transform") {
		t.Fatalf("player preservado: %v", player)
	}
}
