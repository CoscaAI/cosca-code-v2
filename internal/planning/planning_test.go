package planning

import (
	"context"
	"testing"

	"github.com/CoscaAI/cosca-code/internal/provider"
)

type mockLLM struct {
	resp string
	err  error
}

func (m *mockLLM) Chat(_ context.Context, _ string, _ []provider.Message) (string, error) {
	return m.resp, m.err
}

const goodPlan = `{"goal":"Implementar auth multi-tenant","requirements":["JWT","RBAC"],
"affected_files":["+ auth/middleware.go","~ api/router.go"],
"architecture":"Middleware JWT + tabela de tenants","steps":["Criar middleware","Ligar router"],
"risks":"Médio","tests":["auth/middleware_test.go"],"rollback":"git revert",
"estimated_changes":14}`

func TestGeneratePlan(t *testing.T) {
	llm := &mockLLM{resp: goodPlan}
	g := New(llm, "qwen")
	p, err := g.Generate(context.Background(), "implementar auth")
	if err != nil {
		t.Fatal(err)
	}
	if p.Goal == "" || p.Status != "proposed" {
		t.Fatalf("plan: %+v", p)
	}
	if p.ID == "" || !strings_HasPrefix(p.ID, "PLAN-") {
		t.Fatalf("id = %q", p.ID)
	}
	if len(p.AffectedFiles) != 2 {
		t.Fatalf("files = %v", p.AffectedFiles)
	}
}

func TestGenerateWithFence(t *testing.T) {
	llm := &mockLLM{resp: "```json\n" + goodPlan + "\n```"}
	g := New(llm, "qwen")
	p, err := g.Generate(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if p.Goal == "" {
		t.Fatal("fence strip falhou")
	}
}

func TestGenerateEmptyGoal(t *testing.T) {
	llm := &mockLLM{resp: goodPlan}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "  "); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateLLMFailure(t *testing.T) {
	llm := &mockLLM{err: errMock}
	g := New(llm, "qwen")
	if _, err := g.Generate(context.Background(), "x"); err == nil {
		t.Fatal("expected llm error")
	}
}

func TestPlanStateMachine(t *testing.T) {
	p := &Plan{ID: "P1", Goal: "g"}
	p.Approve()
	if p.Status != "approved" {
		t.Fatalf("status = %q", p.Status)
	}
	p.Reject()
	if p.Status != "rejected" {
		t.Fatalf("status = %q", p.Status)
	}
}

func TestStripCodeFence(t *testing.T) {
	out := stripCodeFence("```json\n{\"a\":1}\n```")
	if out != `{"a":1}` {
		t.Fatalf("out = %q", out)
	}
}

type mockErr struct{}

func (e *mockErr) Error() string { return "llm down" }

var errMock = &mockErr{}

func strings_HasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
