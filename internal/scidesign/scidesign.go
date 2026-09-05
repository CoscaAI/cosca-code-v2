// Package scidesign implementa o §31 real do COSCA SCIENTIFIC: o Don descreve
// o experimento em linguagem natural ("compare dois modelos no dataset de
// classificação") e a IA local (qwen via Ollama) gera o experiment.json no
// schema do sciengine (§11/§12).
//
// O LLM recebe o schema como contrato e devolve o experimento — o mesmo
// schema que o COSCA LAB abre. Fallback determinístico se o LLM falhar.
package scidesign

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CoscaAI/cosca-code/internal/provider"
)

// LLM é a interface mínima do chat (implementada pelo OllamaProvider).
type LLM interface {
	Chat(ctx context.Context, model string, messages []provider.Message) (string, error)
}

// Message é uma mensagem de chat (mesma shape do provider).
type Message = provider.Message

// Generator gera experimentos a partir de descrições em linguagem natural.
type Generator struct {
	llm   LLM
	model string
}

// New cria um gerador com o LLM e modelo dados.
func New(llm LLM, model string) *Generator {
	return &Generator{llm: llm, model: model}
}

// experimentSchema define o schema do experimento (§11) que o LLM preenche.
const experimentSchema = `{
  "id": string,          // id único (ex.: "exp-001")
  "name": string,        // nome do experimento
  "input": [string],     // assets de entrada (dataset)
  "parameters": {        // parâmetros do experimento
    "key": number | string
  },
  "code_version": string, // versão do código (ex.: "main")
  "model_version": string,// versão do modelo (se aplicável)
  "environment": string,  // runtime (ex.: "cosca-runtime")
  "result": number,       // resultado principal
  "kind": string,         // observed | calculated | simulated | generated | hypothesis
  "metrics": {            // métricas
    "key": number
  }
}`

// systemPrompt instrui o LLM a produzir SOMENTE o JSON do experimento.
const systemPrompt = `Você é o Cientista Chefe do COSCA SCIENTIFIC. A partir da descrição 
do usuário, crie um experimento no EXATO schema JSON abaixo. Responda APENAS 
com o JSON válido, sem markdown, sem texto extra, sem comentários.

REGRA DE INTEGRIDADE (§32):
- kind calculated: resultado derivado por cálculo determinístico
- kind simulated: resultado de simulação
- kind observed: resultado medido
- kind hypothesis: ainda não verificado
- Nunca invente um observed como se fosse medido; se a descrição não diz
  que foi medido, use calculated, simulated ou hypothesis.

SCHEMA DO EXPERIMENTO:
` + experimentSchema

// Generate pede ao LLM o experimento e valida o JSON.
func (g *Generator) Generate(ctx context.Context, description string) (map[string]any, error) {
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("descrição do experimento é obrigatória")
	}
	raw, err := g.llm.Chat(ctx, g.model, []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: description},
	})
	if err != nil {
		return nil, fmt.Errorf("scidesign: llm: %w", err)
	}
	cleaned := stripCodeFence(raw)
	var exp map[string]any
	if err := json.Unmarshal([]byte(cleaned), &exp); err != nil {
		return nil, fmt.Errorf("scidesign: JSON inválido do LLM: %w", err)
	}
	if _, ok := exp["id"]; !ok {
		return nil, fmt.Errorf("scidesign: experimento sem id")
	}
	if _, ok := exp["kind"]; !ok {
		return nil, fmt.Errorf("scidesign: experimento sem kind (§32)")
	}
	// Normaliza metrics ausente → mapa vazio (contrato do sciengine).
	if exp["metrics"] == nil {
		exp["metrics"] = map[string]any{}
	}
	return exp, nil
}

// GenerateWithFallback tenta o LLM; se falhar, usa o gerador determinístico.
func (g *Generator) GenerateWithFallback(ctx context.Context, description string) (map[string]any, string, error) {
	exp, err := g.Generate(ctx, description)
	if err == nil {
		return exp, "llm", nil
	}
	return fallbackExperiment(description), "fallback", nil
}

// stripCodeFence remove ```json ... ``` se o LLM envolver a resposta.
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

// fallbackExperiment gera um experimento mínimo determinístico.
func fallbackExperiment(description string) map[string]any {
	name := "exp-baseline"
	if words := strings.Fields(description); len(words) > 0 {
		name = strings.ToLower(words[0]) + "-exp"
	}
	return map[string]any{
		"id":            "exp-baseline",
		"name":          name,
		"input":         []string{},
		"parameters":    map[string]any{"expr": "constant", "value": 0.5},
		"code_version":  "main",
		"environment":   "cosca-runtime",
		"result":        0.5,
		"kind":          "calculated",
		"metrics":       map[string]any{"result": 0.5},
	}
}
