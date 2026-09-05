package engines

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CoscaAI/cosca/pkg/engine"

	"github.com/CoscaAI/cosca-code/internal/cinexec"
	"github.com/CoscaAI/cosca-code/internal/diffexec"
	"github.com/CoscaAI/cosca-code/internal/event"
	"github.com/CoscaAI/cosca-code/internal/gamedesign"
	"github.com/CoscaAI/cosca-code/internal/gameexec"
	"github.com/CoscaAI/cosca-code/internal/mediaexec"
	"github.com/CoscaAI/cosca-code/internal/musexec"
	"github.com/CoscaAI/cosca-code/internal/provider"
	"github.com/CoscaAI/cosca-code/internal/scidesign"
	"github.com/CoscaAI/cosca-code/internal/sciexec"
	"github.com/CoscaAI/cosca-code/internal/td3dexec"
	"github.com/CoscaAI/cosca-code/internal/xexec"
)

// RegisterHandlers registra os endpoints das engines da Fase 1 no mux.
//
//	GET  /api/engine/project    — Project Manifest (§34) + tipo/modo
//	GET  /api/engine/assets     — lista de assets (§2)
//	POST /api/engine/asset/add  — registra um asset (path + type)
//	GET  /api/engine/asset      — detalhes de um asset (?id=)
//	POST /api/engine/graph/validate — valida um node graph (§21)
//	POST /api/engine/graph/sig      — assinaturas (cache keys §23)
//	GET  /api/engine/provenance — provenance (§32-35)
func RegisterHandlers(mux *http.ServeMux, e *Engines) {
	mux.HandleFunc("/api/engine/project", e.handleProject)
	mux.HandleFunc("/api/engine/assets", e.handleAssets)
	mux.HandleFunc("/api/engine/asset/add", e.handleAssetAdd)
	mux.HandleFunc("/api/engine/asset", e.handleAssetInfo)
	mux.HandleFunc("/api/engine/graph/validate", e.handleGraphValidate)
	mux.HandleFunc("/api/engine/graph/sig", e.handleGraphSig)
	mux.HandleFunc("/api/engine/graph/run", e.handleGraphRun)
	mux.HandleFunc("/api/engine/cinema/run", e.handleCinemaRun)
	mux.HandleFunc("/api/engine/music/run", e.handleMusicRun)
	mux.HandleFunc("/api/engine/td3d/run", e.handleTD3DRun)
	mux.HandleFunc("/api/engine/game/run", e.handleGameRun)
	mux.HandleFunc("/api/engine/game/design", e.handleGameDesign)
	mux.HandleFunc("/api/engine/sci/run", e.handleSciRun)
	mux.HandleFunc("/api/engine/sci/design", e.handleSciDesign)
	mux.HandleFunc("/api/engine/x/run", e.handleXRun)
	mux.HandleFunc("/api/engine/x/flows", e.handleXFlows)
	mux.HandleFunc("/api/engine/image/generate", e.handleImageGenerate)
	mux.HandleFunc("/api/engine/provenance", e.handleProvenance)
}

// handleProject devolve o Project Manifest + tipo/modo do editor.
func (e *Engines) handleProject(w http.ResponseWriter, r *http.Request) {
	m, err := e.ProjectManifest()
	if err != nil {
		// Sem manifest → modo default editor.
		JSON(w, http.StatusOK, map[string]any{
			"type":   "editor",
			"source": "default",
			"path":   e.ManifestPath(),
		})
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"type":     m.Type,
		"source":   "manifest",
		"path":     e.ManifestPath(),
		"name":     m.Name,
		"version":  m.Version,
		"models":   m.Models,
		"assets":   m.Assets,
		"workflows": m.Workflows,
	})
}

// handleAssets devolve a lista de assets (§2).
func (e *Engines) handleAssets(w http.ResponseWriter, r *http.Request) {
	assets := e.assets.List()
	JSON(w, http.StatusOK, map[string]any{
		"count":  len(assets),
		"assets": assets,
	})
}

// handleAssetAdd registra um asset: POST ?path=<abs>&type=image.
func (e *Engines) handleAssetAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	path := r.URL.Query().Get("path")
	typ := r.URL.Query().Get("type")
	if path == "" || typ == "" {
		Error(w, http.StatusBadRequest, "path and type are required")
		return
	}
	// Segurança: path deve estar dentro do workspace.
	abs, err := filepath.Abs(path)
	if err != nil || !strings.HasPrefix(abs, e.root) {
		Error(w, http.StatusForbidden, "path fora do workspace")
		return
	}
	if _, err := os.Stat(abs); err != nil {
		Error(w, http.StatusNotFound, "arquivo não encontrado")
		return
	}
	a, err := e.assets.AddFile(abs, engine.AssetType(typ), path)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Evento no barramento (observabilidade — Fase 2.3).
	e.publish(event.AssetAdded, map[string]string{"id": a.ID, "type": string(a.Type), "path": path})
	JSON(w, http.StatusOK, a)
}

// handleAssetInfo devolve os detalhes de um asset: GET ?id=<hash>.
func (e *Engines) handleAssetInfo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		Error(w, http.StatusBadRequest, "id is required")
		return
	}
	a, ok := e.assets.Get(id)
	if !ok {
		Error(w, http.StatusNotFound, "asset não encontrado")
		return
	}
	JSON(w, http.StatusOK, a)
}

// handleGraphValidate valida um node graph (§21): POST body JSON.
func (e *Engines) handleGraphValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	order, err := engine.GraphTopoOrder(g)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ids := make([]string, len(order))
	for i, n := range order {
		ids[i] = n.ID
	}
	// Evento no barramento (Fase 2.3).
	e.publish(event.GraphValidated, map[string]any{"name": g.Name, "nodes": len(g.Nodes), "order": ids})
	JSON(w, http.StatusOK, map[string]any{
		"valid": true,
		"name":  g.Name,
		"nodes": len(g.Nodes),
		"order": ids,
	})
}

// handleGraphSig devolve as assinaturas (cache keys §23).
func (e *Engines) handleGraphSig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	sigs := make(map[string]string, len(g.Nodes))
	for _, n := range g.Nodes {
		sig, err := engine.GraphSignature(g, n.ID)
		if err != nil {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		sigs[n.ID] = sig
	}
	JSON(w, http.StatusOK, map[string]any{"signatures": sigs})
}

// loadGraphRequest lê o JSON do corpo e carrega o node graph (§21).
func loadGraphRequest(w http.ResponseWriter, r *http.Request) (*engine.Graph, error) {
	defer r.Body.Close()
	data := make([]byte, 1<<20) // 1 MiB limite
	n, err := r.Body.Read(data)
	if err != nil && err.Error() != "EOF" {
		Error(w, http.StatusBadRequest, "corpo inválido")
		return nil, err
	}
	g, err := engine.UnmarshalGraph(data[:n])
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return nil, err
	}
	return g, nil
}

// publish emite um evento no barramento (nil-safe: sem bus, no-op).
func (e *Engines) publish(typ event.Type, payload any) {
	if e.bus == nil {
		return
	}
	e.bus.Publish(event.Event{Type: typ, Source: "engine", Payload: payload})
}

// handleGraphRun executa um node graph de imagem (§7) com o mediaexec e cache
// por assinatura (§23). POST body JSON do grafo. Resultados por nó + stats.
func (e *Engines) handleGraphRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	// Executor de imagem (Fase 3): nós viram comandos da cosca-media.
	exec := mediaexec.New(filepath.Join(e.root, ".cosca", "media-work"))
	stats, err := engine.GraphRun(r.Context(), g, exec, nil)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Evento de execução no barramento.
	e.publish(event.GraphExecuted, map[string]any{"name": g.Name, "executed": stats.Executed, "hits": stats.CachedHits})
	JSON(w, http.StatusOK, map[string]any{
		"name":        g.Name,
		"executed":    stats.Executed,
		"cached_hits": stats.CachedHits,
		"results":     stats.Results,
	})
}

// handleCinemaRun executa um node graph de VÍDEO (§8) com o cinexec (Media
// Engine ffmpeg) e cache por assinatura (§23). POST body JSON do grafo.
func (e *Engines) handleCinemaRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	exec := cinexec.New(filepath.Join(e.root, ".cosca", "cinema-work"))
	stats, err := engine.GraphRun(r.Context(), g, exec, nil)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	e.publish(event.GraphExecuted, map[string]any{"name": g.Name, "executed": stats.Executed, "hits": stats.CachedHits})
	JSON(w, http.StatusOK, map[string]any{
		"name":        g.Name,
		"executed":    stats.Executed,
		"cached_hits": stats.CachedHits,
		"results":     stats.Results,
	})
}

// handleMusicRun executa um node graph de ÁUDIO (§9) com o musexec (Media
// Engine ffmpeg) e cache por assinatura (§23). POST body JSON do grafo.
func (e *Engines) handleMusicRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	exec := musexec.New(filepath.Join(e.root, ".cosca", "music-work"))
	stats, err := engine.GraphRun(r.Context(), g, exec, nil)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	e.publish(event.GraphExecuted, map[string]any{"name": g.Name, "executed": stats.Executed, "hits": stats.CachedHits})
	JSON(w, http.StatusOK, map[string]any{
		"name":        g.Name,
		"executed":    stats.Executed,
		"cached_hits": stats.CachedHits,
		"results":     stats.Results,
	})
}

// handleTD3DRun executa um node graph 3D (§13) com o td3dexec (3D Engine
// pura Go) e cache por assinatura (§23). POST body JSON do grafo.
func (e *Engines) handleTD3DRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	exec := td3dexec.New(filepath.Join(e.root, ".cosca", "td3d-work"))
	stats, err := engine.GraphRun(r.Context(), g, exec, nil)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	e.publish(event.GraphExecuted, map[string]any{"name": g.Name, "executed": stats.Executed, "hits": stats.CachedHits})
	JSON(w, http.StatusOK, map[string]any{
		"name":        g.Name,
		"executed":    stats.Executed,
		"cached_hits": stats.CachedHits,
		"results":     stats.Results,
	})
}

// handleGameRun executa um node graph de JOGO (§10) com o gameexec (Game
// Engine ECS pura Go) e cache por assinatura (§23). POST body JSON do grafo.
func (e *Engines) handleGameRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	exec := gameexec.New(filepath.Join(e.root, ".cosca", "game-work"))
	stats, err := engine.GraphRun(r.Context(), g, exec, nil)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	e.publish(event.GraphExecuted, map[string]any{"name": g.Name, "executed": stats.Executed, "hits": stats.CachedHits})
	JSON(w, http.StatusOK, map[string]any{
		"name":        g.Name,
		"executed":    stats.Executed,
		"cached_hits": stats.CachedHits,
		"results":     stats.Results,
	})
}

// handleGameDesign implementa o §31 real: o Don descreve o jogo em linguagem
// natural e a IA local (qwen via Ollama) gera o level.json no schema ECS.
// POST body: {"description": "crie um jogo de plataforma com 2 inimigos..."}.
func (e *Engines) handleGameDesign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var req struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if e.router == nil {
		Error(w, http.StatusInternalServerError, "engine sem provider configurado")
		return
	}
	// COSCA solo: roteia pelo router → provider selecionado (cosca ativo ou
	// fallback). Não cria um LLM hardcoded.
	gen := gamedesign.New(provider.NewRouterLLM(e.router), "")
	scene, source, err := gen.GenerateWithFallback(r.Context(), req.Description)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Persiste em level.json (o editor abre direto).
	path := filepath.Join(e.root, "level.json")
	data, _ := json.MarshalIndent(scene, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		Error(w, http.StatusInternalServerError, "salvar level.json")
		return
	}
	ents, _ := scene["entities"].([]any)
	JSON(w, http.StatusOK, map[string]any{
		"source":   source,
		"path":     path,
		"entities": len(ents),
		"scene":    scene,
	})
}

// handleSciRun executa um node graph CIENTÍFICO (§11/§12) com o sciexec
// (Scientific Engine + COSCA LAB) e cache por assinatura (§23).
func (e *Engines) handleSciRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	exec := sciexec.New(e.root)
	stats, err := engine.GraphRun(r.Context(), g, exec, nil)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	e.publish(event.GraphExecuted, map[string]any{"name": g.Name, "executed": stats.Executed, "hits": stats.CachedHits})
	JSON(w, http.StatusOK, map[string]any{
		"name":        g.Name,
		"executed":    stats.Executed,
		"cached_hits": stats.CachedHits,
		"results":     stats.Results,
	})
}

// handleSciDesign implementa o §31 real do COSCA SCIENTIFIC: o Don descreve
// o experimento em linguagem natural e a IA local (qwen via Ollama) gera o
// experiment.json no schema do sciengine (§11). POST body: {"description"}.
func (e *Engines) handleSciDesign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var req struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if e.router == nil {
		Error(w, http.StatusInternalServerError, "engine sem provider configurado")
		return
	}
	// COSCA solo: roteia pelo router → provider selecionado (cosca ativo ou
	// fallback). Não cria um LLM hardcoded.
	gen := scidesign.New(provider.NewRouterLLM(e.router), "")
	exp, source, err := gen.GenerateWithFallback(r.Context(), req.Description)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Persiste em experiment.json (o LAB abre direto).
	path := filepath.Join(e.root, "experiment.json")
	data, _ := json.MarshalIndent(exp, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		Error(w, http.StatusInternalServerError, "salvar experiment.json")
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"source": source,
		"path":   path,
		"kind":   exp["kind"],
		"experiment": exp,
	})
}

// handleXRun executa um node graph CROSS-PRODUCT (§30) com o xexec — UM grafo
// que roteia cada nó para o executor do seu domínio (image/cinema/music/3d/
// game/sci). POST body JSON do grafo.
func (e *Engines) handleXRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	g, err := loadGraphRequest(w, r)
	if err != nil {
		return
	}
	exec := xexec.New(e.root)
	stats, err := engine.GraphRun(r.Context(), g, exec, nil)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	e.publish(event.GraphExecuted, map[string]any{"name": g.Name, "executed": stats.Executed, "hits": stats.CachedHits})
	JSON(w, http.StatusOK, map[string]any{
		"name":        g.Name,
		"executed":    stats.Executed,
		"cached_hits": stats.CachedHits,
		"results":     stats.Results,
	})
}

// handleXFlows devolve os produtos e fluxos cross-product do §30.
func (e *Engines) handleXFlows(w http.ResponseWriter, r *http.Request) {
	exec := xexec.New(e.root)
	JSON(w, http.StatusOK, map[string]any{
		"products": exec.Products(),
		"flows":    exec.Flows(),
	})
}

// handleImageGenerate implementa a GERAÇÃO DE IMAGEM por IA (§7: PROMPT →
// IMAGE) via diffusion (SD 1.5 + ROCm na RX 6700 XT). POST body JSON:
// {"prompt": "...", "steps": 25, "guidance": 7.5}.
func (e *Engines) handleImageGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var req struct {
		Prompt   string  `json:"prompt"`
		Steps    float64 `json:"steps"`
		Guidance float64 `json:"guidance"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if req.Prompt == "" {
		Error(w, http.StatusBadRequest, "prompt é obrigatório")
		return
	}
	exec := diffexec.New(filepath.Join(e.root, ".cosca", "image-work"))
	node := &engine.Node{
		ID: "gen-" + shortHash(req.Prompt),
		Type: "image_generation",
		Params: map[string]any{
			"prompt": req.Prompt, "steps": req.Steps, "guidance": req.Guidance,
		},
	}
	result, err := exec.Run(r.Context(), node, nil)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// O executor devolve o CAMINHO (string) como contrato do node graph.
	path, _ := result.(string)
	if path == "" {
		Error(w, http.StatusInternalServerError, "diffusion não devolveu caminho")
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"prompt": req.Prompt,
		"path":   path,
	})
}

// shortHash gera um id curto e estável a partir de uma string.
func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:6])
}

// handleProvenance devolve claims, generations e licenças (§32-35).
func (e *Engines) handleProvenance(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{
		"claims":      e.provenance.SortedClaims(),
		"generations": e.provenance.SortedGenerations(),
		"licenses":    e.provenance.SortedLicenses(),
	})
}
