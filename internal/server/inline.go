package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/CoscaAI/cosca-code/internal/provider"
)

// inlineSystemPrompt guia o modelo para o Inline AI: transformação retorna SÓ
// código; explicação retorna SÓ texto. Sem markdown nas transformações — o
// resultado vai direto pro editor.
const inlineSystemPrompt = `Você é um assistente de engenharia de software embutido em um IDE (COSCA CODE).
Recebe uma INSTRUÇÃO e um trecho de CÓDIGO (entre marcadores <code>).

Regras:
- Se a instrução pedir TRANSFORMAÇÃO (refatorar, corrigir, otimizar, simplificar, gerar testes, completar, migrar), responda APENAS com o código transformado. Sem markdown, sem fences, sem explicação antes ou depois.
- Se a instrução pedir EXPLICAÇÃO (explicar, o que faz, descrever, documentar), responda APENAS com a explicação em texto puro.
- Preserve a indentação e o idioma do código de entrada.
- Se a instrução for ambígua, faça a transformação mais conservadora possível.`

// handleAIInline transforma/explica um trecho de código (POST).
// Body: {code, instruction} → {result}.
func (s *Server) handleAIInline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não suportado", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Code        string `json:"code"`
		Instruction string `json:"instruction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Instruction = strings.TrimSpace(req.Instruction)
	if req.Instruction == "" {
		http.Error(w, "instruction vazia", http.StatusBadRequest)
		return
	}

	user := "INSTRUÇÃO: " + req.Instruction + "\n<code>\n" + req.Code + "\n</code>"

	reply, err := s.router.Chat(r.Context(), "", "", []provider.Message{
		{Role: "system", Content: inlineSystemPrompt},
		{Role: "user", Content: user},
	})
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	// Higieniza fences de markdown que o modelo possa ter incluído.
	reply = stripMarkdownFences(reply)
	writeJSON(w, map[string]any{"result": reply})
}

// stripMarkdownFences remove ```lang ... ``` se o modelo ignorou a regra.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove a primeira linha (```lang) e a última (```).
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
		}
		s = strings.Join(lines, "\n")
	}
	return strings.TrimSpace(s)
}
