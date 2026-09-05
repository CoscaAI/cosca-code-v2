package scidesign

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// mockLLM devolve uma resposta fixa.
type mockLLM struct {
	resp string
	err  error
}

func (m *mockLLM) Chat(_ context.Context, _ string, _ []Message) (string, error) {
	return m.resp, m.err
}

const goodExp = `{"id":"exp-001","name":"benchmark-cnn","input":["dataset-001"],
"parameters":{"epochs":10,"lr":0.001},"code_version":"main",
"model_version":"v2","environment":"cosca-runtime","result":0.92,
"kind":"calculated","metrics":{"accuracy":0.92,"loss":0.11}}`

func TestGenerateWithLLM(t *testing.T) {
	llm := &mockLLM{resp: goodExp}
	g := New(llm, "qwen2.5-coder:14b")
	exp, err := g.Generate(context.Background(), "compare dois modelos no dataset de classificação")
	if err != nil {
		t.Fatal(err)
	}
	if exp["id"] != "exp-001" || exp["kind"] != "calculated" {
		t.Fatalf("exp: %+v", exp)
	}
	if _, ok := exp["metrics"].(map[string]any); !ok {
		t.Fatalf("metrics deve ser mapa: %v", exp["metrics"])
	}
}

func TestGenerateWithMarkdownFence(t *testing.T) {
	llm := &mockLLM{resp: "```json\n" + goodExp + "\n```"}
	g := New(llm, "qwen")
	exp, err := g.Generate(context.Background(), "experimento")
	if err != nil {
		t.Fatal(err)
	}
	if exp["id"] != "exp-001" {
		t.Fatal("fence strip falhou")
	}
}

func TestGenerateMissingKind(t *testing.T) {
	llm := &mockLLM{resp: `{"id":"x","name":"y","result":1}`}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "exp"); err == nil {
		t.Fatal("expected missing kind error (§32)")
	}
}

func TestGenerateLLMFailure(t *testing.T) {
	llm := &mockLLM{err: errLLM}
	g := New(llm, "qwen")
	exp, source, err := g.GenerateWithFallback(context.Background(), "experimento")
	if err != nil {
		t.Fatal(err)
	}
	if source != "fallback" {
		t.Fatalf("source = %q", source)
	}
	if exp["kind"] != "calculated" {
		t.Fatalf("fallback kind = %v", exp["kind"])
	}
}

func TestGenerateEmptyDescription(t *testing.T) {
	llm := &mockLLM{resp: goodExp}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty description")
	}
}

func TestGenerateInvalidJSON(t *testing.T) {
	llm := &mockLLM{resp: "não é json"}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "exp"); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestStripCodeFence(t *testing.T) {
	in := "```json\n{\"a\":1}\n```"
	out := stripCodeFence(in)
	var v map[string]int
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("strip fence failed: %v", err)
	}
	if v["a"] != 1 {
		t.Fatalf("value = %v", v)
	}
}

var errLLM = &mockErr{}

type mockErr struct{}

func (e *mockErr) Error() string { return "llm unavailable" }

// garante que o schema inclui os tipos do §32.
func TestSchemaHasKinds(t *testing.T) {
	if !strings.Contains(experimentSchema, "observed") ||
		!strings.Contains(experimentSchema, "calculated") ||
		!strings.Contains(experimentSchema, "hypothesis") {
		t.Fatal("schema deve incluir os kinds §32")
	}
}
