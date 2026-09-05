// Package engines integra as engines do Cosca Engine (Fase 1) ao COSCA CODE
// editor. Usa o bridge público github.com/CoscaAI/cosca/pkg/engine (as engines
// vivem em internal/ do módulo cosca, inacessível por regra do Go). Importadas
// in-process e expostas via HTTP handlers — UI cliente fino, tudo pelo server.
package engines

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/CoscaAI/cosca/pkg/engine"

	"github.com/CoscaAI/cosca-code/internal/event"
	"github.com/CoscaAI/cosca-code/internal/provider"
)

// Engines agrega as engines da Fase 1 para um projeto/workspace.
type Engines struct {
	root       string
	assets     *engine.AssetRegistry
	provenance *engine.ProvenanceRegistry
	bus        *event.Bus
	router     *provider.Router // fonte do provider (gamedesign/scidesign → COSCA solo)
}

// Open inicializa as engines para o root do workspace. Best-effort: uma
// engine sem dados abre vazia, nunca falha o editor. O router é a fonte do
// provider para os design LLMs; pode ser nil (então os handlers de design
// respondem erro claro em vez de panic).
func Open(root string, bus *event.Bus, router *provider.Router) (*Engines, error) {
	assets, err := engine.OpenAssetRegistry(root)
	if err != nil {
		assets, _ = engine.OpenAssetRegistry(root)
	}
	prov, err := engine.OpenProvenance(root)
	if err != nil {
		prov, _ = engine.OpenProvenance(root)
	}
	return &Engines{root: root, assets: assets, provenance: prov, bus: bus, router: router}, nil
}

// Router devolve o router (fonte do provider selecionado).
func (e *Engines) Router() *provider.Router { return e.router }

// Root devolve o root do workspace.
func (e *Engines) Root() string { return e.root }

// AssetRegistry devolve o registry de assets (§2).
func (e *Engines) AssetRegistry() *engine.AssetRegistry { return e.assets }

// Provenance devolve o registry de provenance (§32-35).
func (e *Engines) Provenance() *engine.ProvenanceRegistry { return e.provenance }

// ManifestPath devolve o caminho do Project Manifest (project.yaml).
func (e *Engines) ManifestPath() string {
	return filepath.Join(e.root, ".cosca", "project.yaml")
}

// ProjectManifest lê o Project Manifest (§34) do projeto.
func (e *Engines) ProjectManifest() (*engine.ProjectManifest, error) {
	return engine.ReadProjectManifest(e.root)
}

// LoadGraph carrega um node graph a partir de bytes JSON (§21).
func LoadGraph(data []byte) (*engine.Graph, error) {
	return engine.UnmarshalGraph(data)
}

// JSON é um helper de resposta (evita repetir marshal em cada handler).
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

// Error responde com erro JSON estruturado.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}
