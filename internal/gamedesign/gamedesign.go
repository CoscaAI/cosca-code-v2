// Package gamedesign implementa o §31 real do COSCA GAME: o Don descreve o
// jogo em linguagem natural ("crie um jogo de plataforma com 3 inimigos e
// moedas") e a IA local (qwen via Ollama) gera a cena ECS no schema do
// gameengine (§10).
//
// O LLM recebe o schema como contrato (JSON de entidades/componentes) e
// devolve o level.json — o mesmo schema que o editor abre. Se o LLM falhar,
// um fallback determinístico gera uma cena mínima.
package gamedesign

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

// Generator gera cenas de jogo a partir de descrições em linguagem natural.
type Generator struct {
	llm   LLM
	model string
}

// New cria um gerador com o LLM e modelo dados.
func New(llm LLM, model string) *Generator {
	return &Generator{llm: llm, model: model}
}

// sceneSchema define o schema ECS que o LLM deve preencher (contrato §10).
const sceneSchema = `{
  "name": string,        // nome da cena (ex.: "level-1")
  "width": number,       // largura do mundo (ex.: 800)
  "height": number,      // altura do mundo (ex.: 600)
  "entities": [          // entidades da cena
    {
      "id": string,      // id único (ex.: "player")
      "name": string,    // nome legível
      "components": [    // componentes válidos: transform, physics, render,
                         // animation, audio, input, ai, collider, health, score
        {"type": "transform", "params": {"x": number, "y": number}},
        {"type": "physics", "params": {"gravity": boolean}},
        {"type": "render", "params": {"sprite": string}},
        {"type": "input"},
        {"type": "ai"},
        {"type": "health", "params": {"hp": number}},
        {"type": "score", "params": {"value": number}}
      ]
    }
  ]
}`

// systemPrompt instrui o LLM a produzir SOMENTE o JSON da cena, com regras
// por papel (refinamento pós-dogfooding L239: componentes coerentes com o
// papel da entidade).
const systemPrompt = `Você é o Game Designer do COSCA GAME. A partir da descrição do usuário, 
crie uma cena de jogo no EXATO schema JSON abaixo. Responda APENAS com o JSON 
válido, sem markdown, sem texto extra, sem comentários.

REGRA DE PAPEL (componentes coerentes com o papel da entidade):
- player/jogador: transform + physics + input + render (opcional: health)
- enemy/inimigo: transform + ai + render (opcional: health, physics)
- item/coletável (moeda, turbo, power-up): transform + render + score
  — NUNCA inclua 'ai' nem 'input' em itens coletáveis
- obstáculo: transform + collider + render
- Use apenas os tipos: transform, physics, render, animation, audio,
  input, ai, collider, health, score.

SCHEMA DA CENA:
` + sceneSchema

// Generate pede ao LLM a cena e valida o JSON resultante.
func (g *Generator) Generate(ctx context.Context, description string) (map[string]any, error) {
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("descrição do jogo é obrigatória")
	}
	raw, err := g.llm.Chat(ctx, g.model, []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: description},
	})
	if err != nil {
		return nil, fmt.Errorf("gamedesign: llm: %w", err)
	}
	// Limpa possíveis fences de markdown que o LLM insira.
	cleaned := stripCodeFence(raw)
	var scene map[string]any
	if err := json.Unmarshal([]byte(cleaned), &scene); err != nil {
		return nil, fmt.Errorf("gamedesign: JSON inválido do LLM: %w", err)
	}
	if _, ok := scene["entities"]; !ok {
		return nil, fmt.Errorf("gamedesign: cena sem entidades")
	}
	// Refinamento pós-dogfooding (L239): coerência de componentes por papel —
	// itens coletáveis não levam ai/input; inimigos não levam input.
	SanitizeScene(scene)
	return scene, nil
}

// SanitizeScene aplica coerência de componentes por papel (refinamento
// pós-dogfooding L239). Heurísticas conservadoras:
//
//	item coletável (score, sem ai/input) → remove ai e input
//	inimigo (ai, sem input)             → remove input
//	jogador (input)                     → mantém
func SanitizeScene(scene map[string]any) {
	entities, ok := scene["entities"].([]any)
	if !ok {
		return
	}
	for _, raw := range entities {
		e, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		comps, ok := e["components"].([]any)
		if !ok {
			continue
		}
		hasAI := hasComponentType(comps, "ai")
		hasScore := hasComponentType(comps, "score")
		// Papel inferido por assinatura de componentes:
		//   item coletável = tem score (moeda/turbo/power-up — nunca tem
		//     comportamento ai nem input; são estáticos e coletáveis)
		//   inimigo = tem ai (controle é IA, nunca input humano)
		isCollectible := hasScore
		isEnemy := hasAI
		var cleaned []any
		for _, rc := range comps {
			c, ok := rc.(map[string]any)
			if !ok {
				cleaned = append(cleaned, rc)
				continue
			}
			typ, _ := c["type"].(string)
			switch {
			// Item coletável: remove ai e input.
			case isCollectible && (typ == "ai" || typ == "input"):
				continue
			// Inimigo: remove input (controle é da IA, não do jogador).
			case isEnemy && typ == "input":
				continue
			}
			cleaned = append(cleaned, rc)
		}
		e["components"] = cleaned
	}
}

// hasComponentType reports se a lista tem o tipo dado.
func hasComponentType(comps []any, typ string) bool {
	for _, rc := range comps {
		if c, ok := rc.(map[string]any); ok {
			if t, _ := c["type"].(string); t == typ {
				return true
			}
		}
	}
	return false
}

// GenerateWithFallback tenta o LLM; se falhar, usa o gerador determinístico.
func (g *Generator) GenerateWithFallback(ctx context.Context, description string) (map[string]any, string, error) {
	scene, err := g.Generate(ctx, description)
	if err == nil {
		return scene, "llm", nil
	}
	return fallbackScene(description), "fallback", nil
}

// stripCodeFence remove ```json ... ``` se o LLM envolver a resposta.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove a primeira linha (fence) e a última.
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

// fallbackScene gera uma cena mínima determinística quando o LLM falha.
func fallbackScene(description string) map[string]any {
	name := "level-1"
	if words := strings.Fields(description); len(words) > 0 {
		name = strings.ToLower(words[0]) + "-level"
	}
	return map[string]any{
		"name":   name,
		"width":  800,
		"height": 600,
		"entities": []any{
			map[string]any{
				"id": "player", "name": "player",
				"components": []any{
					map[string]any{"type": "transform", "params": map[string]any{"x": 0, "y": 0}},
					map[string]any{"type": "physics", "params": map[string]any{"gravity": true}},
					map[string]any{"type": "input"},
				},
			},
			map[string]any{
				"id": "enemy", "name": "enemy_1",
				"components": []any{
					map[string]any{"type": "transform"},
					map[string]any{"type": "ai"},
				},
			},
		},
	}
}
