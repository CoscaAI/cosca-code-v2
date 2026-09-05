package server

import (
	"encoding/json"
	"net/http"

	"github.com/CoscaAI/cosca-code/internal/mcp"
)

// handleMcpConnect conecta um servidor MCP (POST {command, args[]}) e registra
// suas tools no Tool Registry do núcleo (o agente pode usá-las).
func (s *Server) handleMcpConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não suportado", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Command == "" {
		http.Error(w, "command vazio", http.StatusBadRequest)
		return
	}

	client, err := mcp.Start(req.Command, req.Args...)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if err := mcp.RegisterTools(client, s.toolReg); err != nil {
		client.Close()
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	s.mcpMu.Lock()
	s.mcpServers = append(s.mcpServers, req.Command)
	s.mcpClients = append(s.mcpClients, client)
	s.mcpMu.Unlock()

	writeJSON(w, map[string]any{"connected": true, "command": req.Command})
}

// handleMcpList lista os servidores MCP conectados.
func (s *Server) handleMcpList(w http.ResponseWriter, r *http.Request) {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()
	writeJSON(w, map[string]any{"servers": s.mcpServers})
}
