package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CoscaAI/cosca-code/internal/provider"
	"github.com/CoscaAI/cosca-code/internal/tool"
	"github.com/CoscaAI/cosca-code/internal/workspace"
)

func TestParseFinal(t *testing.T) {
	if got := parseFinal(`{"final":"resposta final"}`); got != "resposta final" {
		t.Fatalf("parseFinal = %q", got)
	}
	if got := parseFinal(`algum texto {"final":"embedded"}`); got != "embedded" {
		t.Fatalf("parseFinal embedded = %q", got)
	}
	if got := parseFinal(`{"tool":"read_file","args":{"path":"x"}}`); got != "" {
		t.Fatalf("parseFinal de tool call deveria ser vazio, got %q", got)
	}
}

func TestParseToolCall(t *testing.T) {
	call := parseToolCall(`{"tool":"read_file","args":{"path":"a.go"}}`)
	if call == nil || call.Tool != "read_file" || call.Args["path"] != "a.go" {
		t.Fatalf("parseToolCall = %+v", call)
	}
	if parseToolCall(`{"final":"x"}`) != nil {
		t.Fatal("parseToolCall de final deveria ser nil")
	}
}

func TestParseToolCallWithFences(t *testing.T) {
	// Modelo respondeu JSON dentro de fences markdown.
	reply := "```json\n{\n  \"tool\": \"list_files\",\n  \"args\": {}\n}\n```"
	call := parseToolCall(reply)
	if call == nil || call.Tool != "list_files" {
		t.Fatalf("parseToolCall com fences = %+v, want list_files", call)
	}
	if got := parseFinal("```json\n{\"final\":\"ok\"}\n```"); got != "ok" {
		t.Fatalf("parseFinal com fences = %q", got)
	}
}

// TestAgentRunIntegration roda o agente real (ollama + tools) numa tarefa
// simples de listagem. Pula se o ollama não estiver disponível.
func TestAgentRunIntegration(t *testing.T) {
	ollama := provider.NewOllama("http://127.0.0.1:11434")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := ollama.Models(ctx); err != nil {
		t.Skipf("ollama não disponível: %v", err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "util.go"), []byte("package main\n\nfunc helper() {}\n"), 0o644)

	ws, err := workspace.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	reg := tool.NewRegistry()
	tool.Builtin(reg, ws)

	provReg := provider.NewRegistry()
	provReg.Register(ollama)
	router := provider.NewRouter(provReg)
	router.SetDefault("ollama", "qwen2.5-coder:14b-128k")

	a := New("tester", router, reg)
	a.SetMaxSteps(6)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel2()

	final, steps, err := a.Run(ctx2, "Use a ferramenta list_files e me diga quantos arquivos .go existem no projeto.")
	if err != nil {
		t.Fatalf("Run: %v (steps: %+v)", err, steps)
	}
	if final == "" {
		t.Fatal("resposta final vazia")
	}
	t.Logf("final: %s", final)
	t.Logf("passos: %d", len(steps))
}
