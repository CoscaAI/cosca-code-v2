package doctor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRun verifica o doctor contra um backend SIMULADO (httptest) — o doctor
// só faz GET; o servidor de teste valida as chamadas.
func TestRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			w.Write([]byte(`{"ok":true,"service":"cosca-code"}`))
		case "/api/git/status":
			w.Write([]byte(`{"branch":"master","files":[]}`))
		case "/api/ai/models":
			w.Write([]byte(`{"provider":"ollama","models":[{"id":"qwen"}]}`))
		case "/api/autonomy":
			w.Write([]byte(`{"level":"autonomous"}`))
		case "/api/memory":
			w.Write([]byte(`{"entries":[]}`))
		case "/api/skills":
			w.Write([]byte(`{"skills":[{"name":"go_developer"}]}`))
		case "/api/intel/symbols":
			w.Write([]byte(`{"symbols":[{"name":"App"}]}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	checks, ok := Run(context.Background(), srv.URL)
	if !ok {
		t.Fatalf("doctor deveria reportar tudo ok, checks: %+v", checks)
	}
	if len(checks) != 7 {
		t.Fatalf("esperava 7 checks, got %d", len(checks))
	}
	for _, ch := range checks {
		if ch.Status != "ok" {
			t.Errorf("check %s deveria ser ok, got %s", ch.Name, ch.Status)
		}
	}
}

// TestRunFailsWhenBackendDown: backend morto → todos os checks falham, nunca
// panica.
func TestRunFailsWhenBackendDown(t *testing.T) {
	checks, ok := Run(context.Background(), "http://127.0.0.1:1") // porta morta
	if ok {
		t.Fatal("backend morto deveria reportar !ok")
	}
	if len(checks) == 0 {
		t.Fatal("deveria ter checks de falha")
	}
	for _, ch := range checks {
		if ch.Status != "fail" {
			t.Errorf("esperava fail, got %s", ch.Status)
		}
	}
}

// TestPrint não panica e emite JSON válido quando usado.
func TestPrint(t *testing.T) {
	checks := []Check{{Name: "servidor", Status: "ok"}, {Name: "git", Status: "fail", Detail: "HTTP 500"}}
	Print(checks, false)
	_, err := json.Marshal(checks)
	if err != nil {
		t.Fatalf("checks deveriam serializar: %v", err)
	}
}
