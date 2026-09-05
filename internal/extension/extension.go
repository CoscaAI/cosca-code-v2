// Package extension é o sistema de Extensões do COSCA CODE (spec seções 12/52/
// 72): contribuições declarativas (manifest) que registram tools, skills e
// comandos no núcleo. A API de extensão é estável e as capacidades são
// autorizadas — uma extensão só acessa o que o manifest declara. É a fundação
// do plugin ecosystem (Fase 6).
package extension

import (
	"context"
	"sync"

	"github.com/CoscaAI/cosca-code/internal/skill"
	"github.com/CoscaAI/cosca-code/internal/tool"
)

// Command é um comando contribuído por uma extensão.
type Command struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Run   func(ctx context.Context) (string, error)
}

// Manifest é o manifesto declarativo de uma extensão.
type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// Extension é uma extensão carregada: manifesto + contribuições.
type Extension struct {
	Manifest Manifest
	Tools    []*tool.Tool
	Skills   []skill.Skill
	Commands []Command
}

// Registry registra e consulta extensões.
type Registry struct {
	mu   sync.RWMutex
	exts map[string]*Extension
}

// NewRegistry cria um registry vazio.
func NewRegistry() *Registry {
	return &Registry{exts: map[string]*Extension{}}
}

// Register adiciona uma extensão.
func (r *Registry) Register(e *Extension) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exts[e.Manifest.ID] = e
}

// List devolve os manifestos registrados.
func (r *Registry) List() []Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Manifest, 0, len(r.exts))
	for _, e := range r.exts {
		out = append(out, e.Manifest)
	}
	return out
}

// Apply registra as contribuições de todas as extensões nos registries alvo.
func (r *Registry) Apply(tools *tool.Registry) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.exts {
		for _, t := range e.Tools {
			tools.Register(t)
		}
	}
}

// Skills devolve as skills de todas as extensões.
func (r *Registry) Skills() []skill.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []skill.Skill
	for _, e := range r.exts {
		out = append(out, e.Skills...)
	}
	return out
}
