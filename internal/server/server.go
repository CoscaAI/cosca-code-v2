// Package server expõe o Workspace Engine via HTTP para o frontend (a UI é um
// cliente fino — nunca toca o filesystem; tudo passa por aqui). A porta segue a
// família do ecossistema Cosca (14120 serve, 14123 runtime, 14124 neural-link,
// 14125 node, 14126 code).
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CoscaAI/cosca-code/internal/agent"
	"github.com/CoscaAI/cosca-code/internal/autonomy"
	contextengine "github.com/CoscaAI/cosca-code/internal/context"
	"github.com/CoscaAI/cosca-code/internal/engines"
	"github.com/CoscaAI/cosca-code/internal/event"
	"github.com/CoscaAI/cosca-code/internal/extension"
	"github.com/CoscaAI/cosca-code/internal/intel"
	"github.com/CoscaAI/cosca-code/internal/lsp"
	"github.com/CoscaAI/cosca-code/internal/mcp"
	"github.com/CoscaAI/cosca-code/internal/memory"
	"github.com/CoscaAI/cosca-code/internal/planning"
	"github.com/CoscaAI/cosca-code/internal/provider"
	"github.com/CoscaAI/cosca-code/internal/search"
	"github.com/CoscaAI/cosca-code/internal/skill"
	"github.com/CoscaAI/cosca-code/internal/team"
	"github.com/CoscaAI/cosca-code/internal/testengine"
	"github.com/CoscaAI/cosca-code/internal/tool"
	"github.com/CoscaAI/cosca-code/internal/workflow"
	"github.com/CoscaAI/cosca-code/internal/workspace"
)

// Server serve o workspace e o barramento de eventos ao frontend.
type Server struct {
	ws         *workspace.Workspace
	bus        *event.Bus
	lsp        *lsp.Manager
	router     *provider.Router
	provReg    *provider.Registry
	agent      *agent.Agent
	wf         *workflow.Engine
	team       *team.Team
	mem        *memory.Store
	skills     []skill.Skill
	policy     *autonomy.Policy
	ext        *extension.Registry
	intelIdx   *intel.Index
	projCtx    *contextengine.Context
	toolReg    *tool.Registry
	mcpMu      sync.Mutex
	mcpServers []string
	mcpClients []*mcp.Client
	engines    *engines.Engines
	// Plan Mode (§11): o último plano gerado aguardando aprovação.
	planMu sync.Mutex
	plan   *planning.Plan
}

// New cria o server.
func New(ws *workspace.Workspace, bus *event.Bus) *Server {
	reg := provider.Default()
	router := provider.NewRouter(reg)
	// COSCA solo (ADR-032): se o daemon respondeu, o default é o provider
	// "cosca" (Kernel-First); senão, cai para o Ollama local (fallback).
	if _, ok := reg.Get("cosca"); ok {
		router.SetDefault("cosca", provider.DefaultCoscaModel())
	} else {
		router.SetDefault("ollama", defaultModel())
	}

	mem := memory.NewStore()

	toolReg := tool.NewRegistry()
	tool.Builtin(toolReg, ws)
	registerMemoryTools(toolReg, mem)

	// Extensões: contribuem tools/skills/comandos ao núcleo.
	extReg := extension.NewRegistry()
	for _, e := range extension.Builtin() {
		extReg.Register(e)
	}
	extReg.Apply(toolReg)

	ag := agent.New("cosca", router, toolReg)
	policy := &autonomy.Policy{Level: autonomy.Autonomous}
	ag.SetPolicy(policy)

	// Context Engine: dá ao agente a visão real do projeto (símbolos + git).
	idx, _ := intel.Build(ws.Root())
	projCtx := contextengine.Build(ws, idx)
	ag.SetSystemExtra(projCtx.Prompt(40))

	// Engines da Fase 1 (Cosca Engine): importadas in-process, best-effort.
	// Passamos o router para que gamedesign/scidesign usem o provider selecionado
	// (COSCA solo quando ativo).
	eng, _ := engines.Open(ws.Root(), bus, router)

	return &Server{
		ws:       ws,
		bus:      bus,
		lsp:      lsp.NewManager(ws.Root()),
		router:   router,
		provReg:  reg,
		agent:    ag,
		wf:       workflow.NewEngine(ag),
		team:     team.NewTeam(router, toolReg, ws.Info().TestCommand),
		mem:      mem,
		skills:   skill.Builtin(),
		policy:   policy,
		ext:      extReg,
		intelIdx: idx,
		projCtx:  projCtx,
		toolReg:  toolReg,
		engines:  eng,
	}
}

// defaultModel resolve o modelo padrão (env ou fallback conhecido).
func defaultModel() string {
	if m := os.Getenv("COSCA_CODE_MODEL"); m != "" {
		return m
	}
	return "qwen2.5-coder:14b-128k"
}

// intelIndex devolve o índice (construído no startup).
func (s *Server) intelIndex() *intel.Index {
	return s.intelIdx
}

// Handler devolve o mux HTTP com a API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/info", s.handleInfo)
	mux.HandleFunc("/api/tree", s.handleTree)
	mux.HandleFunc("/api/read", s.handleRead)
	mux.HandleFunc("/api/save", s.handleSave)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/git/status", s.handleGitStatus)
	mux.HandleFunc("/api/git/log", s.handleGitLog)
	mux.HandleFunc("/api/git/branches", s.handleGitBranches)
	mux.HandleFunc("/api/git/diff", s.handleGitDiff)
	mux.HandleFunc("/api/lsp/hover", s.handleLSPHover)
	mux.HandleFunc("/api/lsp/definition", s.handleLSPDefinition)
	mux.HandleFunc("/api/lsp/diagnostics", s.handleLSPDiagnostics)
	mux.HandleFunc("/api/lsp/completion", s.handleLSPCompletion)
	mux.HandleFunc("/api/term", s.handleTerm)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/intel/symbols", s.handleIntelSymbols)
	mux.HandleFunc("/api/intel/dependents", s.handleIntelDependents)
	mux.HandleFunc("/api/test/run", s.handleTestRun)
	mux.HandleFunc("/api/ai/models", s.handleAIModels)
	mux.HandleFunc("/api/ai/chat", s.handleAIChat)
	mux.HandleFunc("/api/ai/inline", s.handleAIInline)
	mux.HandleFunc("/api/agent/run", s.handleAgentRun)
	mux.HandleFunc("/api/plan/generate", s.handlePlanGenerate)
	mux.HandleFunc("/api/plan/approve", s.handlePlanApprove)
	mux.HandleFunc("/api/plan/reject", s.handlePlanReject)
	mux.HandleFunc("/api/workflow/run", s.handleWorkflowRun)
	mux.HandleFunc("/api/memory", s.handleMemory)
	mux.HandleFunc("/api/skills", s.handleSkills)
	mux.HandleFunc("/api/team/run", s.handleTeamRun)
	mux.HandleFunc("/api/autonomy", s.handleAutonomy)
	mux.HandleFunc("/api/extensions", s.handleExtensions)
	mux.HandleFunc("/api/catalog", s.handleCatalog)
	mux.HandleFunc("/api/provider/connect", s.handleProviderConnect)
	mux.HandleFunc("/api/mcp/connect", s.handleMcpConnect)
	mux.HandleFunc("/api/mcp", s.handleMcpList)

	// Engines da Fase 1 (Cosca Engine): asset/nodegraph/project/provenance.
	if s.engines != nil {
		engines.RegisterHandlers(mux, s.engines)
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "service": "cosca-code", "root": s.ws.Root()})
	})
	return cors(mux)
}

// cors é um middleware permissivo — o servidor é loopback (127.0.0.1) e serve
// apenas o frontend local (Vite dev ou o asset server do Wails, que roda em
// origem própria). Nunca exposto publicamente.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Start sobe o servidor HTTP em addr (ex.: "127.0.0.1:14126").
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.ws.Info())
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.ws.Tree().Root)
}

// handleRead devolve o conteúdo de um arquivo relativo ao root (segurança:
// resolve e valida que está dentro do workspace — nunca escapa).
func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, ok := s.resolvePath(rel)
	if !ok {
		http.Error(w, "fora do workspace", http.StatusForbidden)
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"path": rel, "content": string(data)})
}

// handleSave grava o conteúdo de um arquivo (POST {path, content}). Valida que
// o path está dentro do workspace e emite file.saved no Event Bus.
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não suportado", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	abs, ok := s.resolvePath(req.Path)
	if !ok {
		http.Error(w, "fora do workspace", http.StatusForbidden)
		return
	}
	if err := os.WriteFile(abs, []byte(req.Content), 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(event.Event{Type: event.FileSaved, Source: "editor", Payload: req.Path})
	writeJSON(w, map[string]any{"ok": true, "path": req.Path})
}

// resolvePath resolve um path relativo ao root e valida que fica dentro dele.
func (s *Server) resolvePath(rel string) (string, bool) {
	abs := filepath.Join(s.ws.Root(), rel)
	cleanRoot := filepath.Clean(s.ws.Root()) + string(os.PathSeparator)
	if !hasPrefix(filepath.Clean(abs), cleanRoot) {
		return "", false
	}
	return abs, true
}

// handleEvents faz stream SSE dos eventos do barramento.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming não suportado", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan event.Event, 128)
	cancel := s.bus.SubscribeAll(func(ev event.Event) {
		select {
		case ch <- ev:
		default:
		}
	})
	defer cancel()

	for {
		select {
		case ev := <-ch:
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// ── Git endpoints (delegam ao Git Engine do workspace) ────────────────

func (s *Server) git() (bool, error) {
	if s.ws.Git() == nil {
		return false, fmt.Errorf("não é um repositório git")
	}
	return true, nil
}

func (s *Server) handleGitStatus(w http.ResponseWriter, r *http.Request) {
	if _, err := s.git(); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	status, err := s.ws.Git().Status()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	branch, _ := s.ws.Git().CurrentBranch()
	writeJSON(w, map[string]any{"branch": branch, "files": status})
}

func (s *Server) handleGitLog(w http.ResponseWriter, r *http.Request) {
	if _, err := s.git(); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	commits, err := s.ws.Git().Log(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, commits)
}

func (s *Server) handleGitBranches(w http.ResponseWriter, r *http.Request) {
	if _, err := s.git(); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	branches, err := s.ws.Git().Branches()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	current, _ := s.ws.Git().CurrentBranch()
	writeJSON(w, map[string]any{"current": current, "branches": branches})
}

func (s *Server) handleGitDiff(w http.ResponseWriter, r *http.Request) {
	if _, err := s.git(); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	path := r.URL.Query().Get("path")
	diff, err := s.ws.Git().Diff(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	staged, _ := s.ws.Git().DiffStaged(path)
	writeJSON(w, map[string]any{"unstaged": diff, "staged": staged})
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// ── LSP endpoints (delegam ao Language Server) ─────────────────────────

// lspOpen garante que o documento está aberto no language server da linguagem
// correspondente (didOpen com o conteúdo atual).
func (s *Server) lspOpen(rel string) (uri string, err error) {
	lang := langForPath(rel)
	if lang == "" {
		return "", fmt.Errorf("sem language server para %s", rel)
	}
	data, err := os.ReadFile(filepath.Join(s.ws.Root(), rel))
	if err != nil {
		return "", err
	}
	uri = "file://" + filepath.ToSlash(filepath.Join(s.ws.Root(), rel))
	if err := s.lsp.OpenDocument(uri, lang, string(data)); err != nil {
		return "", err
	}
	return uri, nil
}

func (s *Server) handleLSPHover(w http.ResponseWriter, r *http.Request) {
	uri, err := s.lspOpen(r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	line, char := intQuery(r, "line"), intQuery(r, "char")
	hover, err := s.lsp.Hover(uri, line, char)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"hover": hover})
}

func (s *Server) handleLSPDefinition(w http.ResponseWriter, r *http.Request) {
	uri, err := s.lspOpen(r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	line, char := intQuery(r, "line"), intQuery(r, "char")
	locs, err := s.lsp.Definition(uri, line, char)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"locations": locs})
}

func (s *Server) handleLSPDiagnostics(w http.ResponseWriter, r *http.Request) {
	uri, err := s.lspOpen(r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"diagnostics": s.lsp.Diagnostics(uri)})
}

func (s *Server) handleLSPCompletion(w http.ResponseWriter, r *http.Request) {
	uri, err := s.lspOpen(r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	line, char := intQuery(r, "line"), intQuery(r, "char")
	items, err := s.lsp.Completion(uri, line, char)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"items": items})
}

func intQuery(r *http.Request, key string) int {
	var v int
	_, _ = fmt.Sscanf(r.URL.Query().Get(key), "%d", &v)
	return v
}

// handleSearch executa busca text/regex no workspace (Search Engine).
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := search.Options{
		Query:         q.Get("q"),
		Regex:         q.Get("regex") == "1" || q.Get("regex") == "true",
		CaseSensitive: q.Get("case") == "1" || q.Get("case") == "true",
		Include:       q.Get("include"),
	}
	if n, err := strconv.Atoi(q.Get("max")); err == nil && n > 0 {
		opts.MaxResults = n
	}
	if opts.Query == "" {
		writeJSON(w, map[string]any{"error": "query vazia"})
		return
	}
	res, err := search.Search(s.ws.Root(), opts)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

// handleIntelSymbols devolve os símbolos indexados (opcional filtro por nome).
func (s *Server) handleIntelSymbols(w http.ResponseWriter, r *http.Request) {
	idx := s.intelIndex()
	name := r.URL.Query().Get("name")
	if name == "" {
		writeJSON(w, map[string]any{"symbols": idx.Symbols})
		return
	}
	writeJSON(w, map[string]any{"symbols": idx.Usages(name)})
}

// handleIntelDependents devolve os arquivos que dependem de um arquivo/import.
func (s *Server) handleIntelDependents(w http.ResponseWriter, r *http.Request) {
	idx := s.intelIndex()
	file := r.URL.Query().Get("file")
	if file == "" {
		writeJSON(w, map[string]any{"error": "file vazio"})
		return
	}
	writeJSON(w, map[string]any{"dependents": idx.Dependents(file)})
}

// handleTestRun executa os testes do projeto (command opcional via query).
func (s *Server) handleTestRun(w http.ResponseWriter, r *http.Request) {
	command := r.URL.Query().Get("command")
	if command == "" {
		command = s.ws.Info().TestCommand
	}
	res, err := testengine.Run(s.ws.Root(), command, 0, s.bus)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

// handleAIModels lista os modelos do provider (default ollama).
func (s *Server) handleAIModels(w http.ResponseWriter, r *http.Request) {
	p, _, err := s.router.Resolve("", "")
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error(), "models": []string{}})
		return
	}
	models, err := p.Models(r.Context())
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error(), "models": []string{}})
		return
	}
	writeJSON(w, map[string]any{"provider": p.Name(), "models": models})
}

// handleAIChat faz inferência (POST {messages, model?}).
func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não suportado", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Messages []provider.Message `json:"messages"`
		Model    string             `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 {
		http.Error(w, "messages vazio", http.StatusBadRequest)
		return
	}
	reply, err := s.router.Chat(r.Context(), "", req.Model, req.Messages)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"reply": reply})
}

// handlePlanGenerate cria um plano de implementação (§11) via LLM.
func (s *Server) handlePlanGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// COSCA solo: roteia pelo router → provider selecionado (cosca ativo ou
	// fallback). O modelo vem do default do router (SetDefault).
	gen := planning.New(provider.NewRouterLLM(s.router), "")
	plan, err := gen.Generate(r.Context(), req.Goal)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	s.planMu.Lock()
	s.plan = plan
	s.planMu.Unlock()
	writeJSON(w, plan)
}

// handlePlanApprove aprova o plano (§64) — autoriza os agentes a executar.
func (s *Server) handlePlanApprove(w http.ResponseWriter, r *http.Request) {
	s.planMu.Lock()
	defer s.planMu.Unlock()
	if s.plan == nil {
		http.Error(w, "nenhum plano pendente", http.StatusNotFound)
		return
	}
	s.plan.Approve()
	writeJSON(w, s.plan)
}

// handlePlanReject rejeita o plano.
func (s *Server) handlePlanReject(w http.ResponseWriter, r *http.Request) {
	s.planMu.Lock()
	defer s.planMu.Unlock()
	if s.plan == nil {
		http.Error(w, "nenhum plano pendente", http.StatusNotFound)
		return
	}
	s.plan.Reject()
	writeJSON(w, s.plan)
}

// handleAgentRun executa uma tarefa pelo agente (POST {task}).
func (s *Server) handleAgentRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não suportado", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Task string `json:"task"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Task == "" {
		http.Error(w, "task vazia", http.StatusBadRequest)
		return
	}
	final, steps, err := s.agent.Run(r.Context(), req.Task)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error(), "steps": steps})
		return
	}
	writeJSON(w, map[string]any{"final": final, "steps": steps})
}

// handleWorkflowRun executa um workflow (POST {workflow, task}).
func (s *Server) handleWorkflowRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não suportado", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Workflow string `json:"workflow"`
		Task     string `json:"task"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Task == "" {
		http.Error(w, "task vazia", http.StatusBadRequest)
		return
	}

	var wf workflow.Workflow
	switch req.Workflow {
	case "bug_fix":
		wf = workflow.BugFix()
	case "feature":
		wf = workflow.Feature()
	case "refactor":
		wf = workflow.Refactor()
	default:
		http.Error(w, "workflow desconhecido (use bug_fix|feature|refactor)", http.StatusBadRequest)
		return
	}

	res, err := s.wf.Run(r.Context(), wf, req.Task)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error(), "result": res})
		return
	}
	writeJSON(w, res)
}

// handleMemory lista as memórias registradas.
func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"entries": s.mem.List()})
}

// handleSkills lista as skills disponíveis.
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"skills": s.skills})
}

// handleTeamRun executa a equipe multi-agent (POST {task}).
func (s *Server) handleTeamRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não suportado", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Task string `json:"task"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Task == "" {
		http.Error(w, "task vazia", http.StatusBadRequest)
		return
	}
	res, err := s.team.Run(r.Context(), req.Task)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

// handleAutonomy lê (GET) ou define (POST {level}) o nível de autonomia.
func (s *Server) handleAutonomy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Level string `json:"level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.policy.Level = autonomy.ParseLevel(req.Level)
	}
	writeJSON(w, map[string]any{"level": s.policy.Level.String()})
}

// handleExtensions lista as extensões registradas.
func (s *Server) handleExtensions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"extensions": s.ext.List()})
}

// handleCatalog lista o catálogo de providers (models.dev-like) + os conectados.
func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	connected := s.provReg.List()
	writeJSON(w, map[string]any{
		"catalog":   provider.Catalog(),
		"connected": connected,
	})
}

// handleProviderConnect conecta um provider do catálogo (POST {id, api_key}).
func (s *Server) handleProviderConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não suportado", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID     string `json:"id"`
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	entry, ok := provider.FindCatalog(req.ID)
	if !ok {
		http.Error(w, "provider desconhecido", http.StatusBadRequest)
		return
	}

	// Adapter: a maioria é openai-compatible (o Google expõe endpoint openai).
	// O provider "cosca" (COSCA solo, ADR-032) é diferente — conecta no /v1/run
	// do daemon via pkg/cosca, e não fala com Ollama/OpenAI.
	var adapter provider.Provider
	adapter = provider.NewOpenAICompat(req.ID, entry.BaseURL, req.APIKey)
	if entry.Kind == "cosca" {
		pc, err := provider.NewCosca(entry.BaseURL, firstModel(entry))
		if err != nil {
			http.Error(w, "cosca: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// Não setar um default morto: exige o daemon no ar (graceful).
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if !pc.Ping(pingCtx) {
			writeJSON(w, map[string]any{"connected": false, "error": "daemon COSCA não responde em " + entry.BaseURL})
			return
		}
		adapter = pc
	}
	s.provReg.Register(adapter)
	s.router.SetDefault(req.ID, firstModel(entry))
	s.bus.Publish(event.Event{Type: event.ProviderConnected, Source: "provider", Payload: req.ID})

	writeJSON(w, map[string]any{"connected": true, "provider": req.ID, "models": entry.Models})
}

// firstModel devolve o primeiro modelo do entry (default).
func firstModel(e provider.CatalogEntry) string {
	if len(e.Models) > 0 {
		return e.Models[0]
	}
	return ""
}

// langForPath mapeia a extensão do arquivo para a linguagem (LSP languageId).
func langForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	default:
		return ""
	}
}
