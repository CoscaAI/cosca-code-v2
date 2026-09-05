package provider

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Registry é o registro de providers disponíveis.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry cria um registro vazio.
func NewRegistry() *Registry {
	return &Registry{providers: map[string]Provider{}}
}

// Register adiciona um provider ao registro.
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Get devolve um provider pelo nome.
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List devolve os nomes dos providers registrados.
func (r *Registry) List() []string {
	var out []string
	for name := range r.providers {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Default cria o registry padrão: detecta o Ollama local (se responder) e
// registra o adapter. É o caminho zero-config (spec seção 49).
//
// COSCA solo (ADR-032): se o daemon COSCA (127.0.0.1:14120) responder ao
// /v1/health, o provider "cosca" (Kernel-First) é registrado — a IA do
// COSCA CODE passa a vir do /v1/run do cérebro. Se não responder, mantém os
// atuais (graceful, sem erro).
func Default() *Registry {
	r := NewRegistry()
	ollama := NewOllama("http://127.0.0.1:11434")
	// Detecta o Ollama local com um timeout curto (não-bloqueante).
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := ollama.Models(ctx); err == nil {
		r.Register(ollama)
	}

	// COSCA daemon: registra "cosca" só se o cérebro estiver no ar.
	if pc, err := NewCosca("", ""); err == nil {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel2()
		if pc.Ping(ctx2) {
			r.Register(pc)
		}
	}
	return r
}

// Router resolve provider+modelo para uma tarefa (spec seção 10). Suporta
// MANUAL (provider/modelo explícitos) e AUTO (default do registry).
type Router struct {
	registry     *Registry
	defaultName  string
	defaultModel string
}

// NewRouter cria um router sobre o registry.
func NewRouter(registry *Registry) *Router {
	return &Router{registry: registry}
}

// SetDefault define o provider/modelo padrão (usado no modo AUTO).
func (r *Router) SetDefault(provider, model string) {
	r.defaultName = provider
	r.defaultModel = model
}

// Resolve devolve (provider, modelo, erro) para uma requisição. Se provider ou
// model vierem vazios, usa o default (AUTO).
func (r *Router) Resolve(provider, model string) (Provider, string, error) {
	if provider == "" {
		provider = r.defaultName
	}
	if provider == "" {
		// Fallback: primeiro provider registrado.
		for name := range r.registry.providers {
			provider = name
			break
		}
	}
	p, ok := r.registry.Get(provider)
	if !ok {
		return nil, "", fmt.Errorf("provider %q não registrado", provider)
	}
	if model == "" {
		model = r.defaultModel
	}
	if model == "" {
		return nil, "", fmt.Errorf("nenhum modelo definido para %q", provider)
	}
	return p, model, nil
}

// Chat encaminha uma conversa para o provider/modelo resolvidos.
func (r *Router) Chat(ctx context.Context, provider, model string, messages []Message) (string, error) {
	p, m, err := r.Resolve(provider, model)
	if err != nil {
		return "", err
	}
	return p.Chat(ctx, m, messages)
}
