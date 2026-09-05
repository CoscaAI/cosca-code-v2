// Package planning implementa o PLAN MODE (§11 do manifesto COSCA CODE
// ENTERPRISE): antes de alterar código, agentes complexos produzem um plano
// estruturado — GOAL · REQUIREMENTS · AFFECTED FILES · ARCHITECTURE ·
// IMPLEMENTATION STEPS · RISKS · TESTS · ROLLBACK — que o usuário aprova,
// edita ou rejeita. Nada toca o código sem aprovação (human-in-the-loop §64).
package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CoscaAI/cosca-code/internal/provider"
)

// Plan é um plano de implementação (§11).
type Plan struct {
	ID         string   `json:"id"`
	Goal       string   `json:"goal"`
	Requirements []string `json:"requirements,omitempty"`
	AffectedFiles []string `json:"affected_files,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Steps      []string `json:"steps,omitempty"`
	Risks      string   `json:"risks,omitempty"`
	Tests      []string `json:"tests,omitempty"`
	Rollback   string   `json:"rollback,omitempty"`
	// Estado de aprovação (§64).
	Status string `json:"status"` // "proposed" | "approved" | "rejected" | "revising"
	// EstimatedChanges é o número estimado de arquivos.
	EstimatedChanges int `json:"estimated_changes,omitempty"`
}

// planSchema é o contrato que o LLM preenche (§11).
const planSchema = `{
  "goal": string,
  "requirements": [string],
  "affected_files": [string],
  "architecture": string,
  "steps": [string],
  "risks": string,
  "tests": [string],
  "rollback": string,
  "estimated_changes": number
}`

const planPrompt = `Você é o AI ARCHITECT do COSCA. Analise a solicitação do usuário 
e produza um PLANO DE IMPLEMENTAÇÃO no EXATO schema JSON abaixo. Responda 
APENAS com o JSON válido, sem markdown, sem texto extra.

Antes de propor alterações, considere: quais arquivos serão criados/modificados 
(+ para novo, ~ para modificado), a arquitetura, os passos de implementação, 
os riscos, os testes necessários e o plano de rollback.

SCHEMA DO PLANO:
` + planSchema

// Generator produz planos via LLM.
type Generator struct {
	llm   LLM
	model string
}

// LLM é a interface mínima do chat.
type LLM interface {
	Chat(ctx context.Context, model string, messages []provider.Message) (string, error)
}

// New cria um gerador de planos.
func New(llm LLM, model string) *Generator {
	return &Generator{llm: llm, model: model}
}

// Generate cria um plano a partir de um objetivo (§11).
func (g *Generator) Generate(ctx context.Context, goal string) (*Plan, error) {
	if strings.TrimSpace(goal) == "" {
		return nil, fmt.Errorf("objetivo é obrigatório")
	}
	raw, err := g.llm.Chat(ctx, g.model, []provider.Message{
		{Role: "system", Content: planPrompt},
		{Role: "user", Content: goal},
	})
	if err != nil {
		return nil, fmt.Errorf("planning: llm: %w", err)
	}
	cleaned := stripCodeFence(raw)
	var p Plan
	if err := json.Unmarshal([]byte(cleaned), &p); err != nil {
		return nil, fmt.Errorf("planning: JSON inválido do LLM: %w", err)
	}
	if p.Goal == "" {
		p.Goal = goal
	}
	p.ID = fmt.Sprintf("PLAN-%d", idCounter())
	p.Status = "proposed"
	return &p, nil
}

// Approve marca o plano como aprovado (§64).
func (p *Plan) Approve() { p.Status = "approved" }

// Reject marca como rejeitado.
func (p *Plan) Reject() { p.Status = "rejected" }

// stripCodeFence remove ```json ... ```.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.Contains(lines[len(lines)-1], "```") {
			lines = lines[:len(lines)-1]
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}
	return s
}

var planCounter int

func idCounter() int {
	planCounter++
	return planCounter
}
