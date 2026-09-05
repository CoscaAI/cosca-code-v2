// Package tool é o Tool Engine do COSCA CODE (spec seções 3-4 + mcp-patterns):
// um registry de ferramentas com contrato declarativo. Cada tool é
// name + description + inputSchema (JSON Schema) + handler tipado — o schema é
// a fronteira de validação e a "API do LLM" (padrão extraído do MCP).
package tool

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/CoscaAI/cosca-code/internal/autonomy"
)

// Tool é uma ferramenta (contrato declarativo).
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema map[string]any  `json:"input_schema"` // JSON Schema
	Risk        autonomy.Action `json:"-"`            // risco da operação (safe autonomy)
	Handler     func(ctx context.Context, args map[string]any) (string, error)
}

// Registry registra e executa tools, com acesso thread-safe.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
	order []string
}

// NewRegistry cria um registry vazio.
func NewRegistry() *Registry {
	return &Registry{tools: map[string]*Tool{}}
}

// Register adiciona uma tool (sobrescreve se o nome já existir).
func (r *Registry) Register(t *Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[t.Name]; !exists {
		r.order = append(r.order, t.Name)
	}
	r.tools[t.Name] = t
}

// Get devolve uma tool pelo nome.
func (r *Registry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List devolve as tools em ordem de registro.
func (r *Registry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sorted := append([]string(nil), r.order...)
	sort.Strings(sorted)
	out := make([]*Tool, 0, len(sorted))
	for _, name := range sorted {
		out = append(out, r.tools[name])
	}
	return out
}

// Call executa uma tool (valida existência e delega ao handler).
func (r *Registry) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("tool %q não existe", name)
	}
	return t.Handler(ctx, args)
}
