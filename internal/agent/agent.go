// Package agent é o Agent Engine do COSCA CODE (spec seções 18-19 + agent-
// patterns): agentes sobre infraestrutura comum — capabilities, tools, context
// policy, budget (maxSteps) e verification. O loop é o padrão OBSERVE→PLAN→
// ACT→OBSERVE: o modelo decide qual tool chamar (JSON), o agente executa e
// realimenta o modelo com o resultado, até a resposta final.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CoscaAI/cosca-code/internal/autonomy"
	"github.com/CoscaAI/cosca-code/internal/provider"
	"github.com/CoscaAI/cosca-code/internal/tool"
)

// Step registra um passo do agente (tool chamada + resultado).
type Step struct {
	Tool   string `json:"tool"`
	Args   any    `json:"args"`
	Result string `json:"result"`
}

// Agent é um agente com nome, modelo e um conjunto de tools.
type Agent struct {
	Name        string
	router      *provider.Router
	tools       *tool.Registry
	maxSteps    int
	policy      *autonomy.Policy // nil = sem restrição (allow all)
	systemExtra string           // contexto extra injetado no system prompt
}

// New cria um agente.
func New(name string, router *provider.Router, tools *tool.Registry) *Agent {
	return &Agent{Name: name, router: router, tools: tools, maxSteps: 10}
}

// SetMaxSteps ajusta o orçamento de passos (spec seção 18: budget).
func (a *Agent) SetMaxSteps(n int) { a.maxSteps = n }

// SetPolicy define a política de autonomia (nil remove a restrição).
func (a *Agent) SetPolicy(p *autonomy.Policy) { a.policy = p }

// SetSystemExtra injeta contexto extra (ex.: resumo do projeto) no system prompt.
func (a *Agent) SetSystemExtra(s string) { a.systemExtra = s }

// Run executa uma tarefa e devolve a resposta final + os passos executados.
func (a *Agent) Run(ctx context.Context, task string) (string, []Step, error) {
	messages := []provider.Message{
		{Role: "system", Content: a.systemPrompt()},
		{Role: "user", Content: task},
	}

	var steps []Step
	var (
		lastCallJSON string
		repeatCount  int
		nonProtocol  int
	)

	for i := 0; i < a.maxSteps; i++ {
		reply, err := a.router.Chat(ctx, "", "", messages)
		if err != nil {
			return "", steps, fmt.Errorf("chat: %w", err)
		}
		messages = append(messages, provider.Message{Role: "assistant", Content: reply})

		// Resposta final?
		if final := parseFinal(reply); final != "" {
			return final, steps, nil
		}

		// Chamada de tool?
		call := parseToolCall(reply)
		if call == nil {
			// Texto livre (sem JSON) → resposta final tolerante. Modelos menores
			// frequentemente respondem tarefas de planejamento com texto puro.
			if text := strings.TrimSpace(reply); text != "" {
				return text, steps, nil
			}
			// Vazio → orienta e conta (estagnação).
			nonProtocol++
			if nonProtocol >= 3 {
				return "Não consegui progredir: o modelo não seguiu o protocolo de ferramentas repetidamente.", steps, nil
			}
			messages = append(messages, provider.Message{
				Role:    "user",
				Content: "Responda APENAS com JSON: {\"tool\":\"nome\",\"args\":{...}} ou {\"final\":\"resposta\"}.",
			})
			continue
		}
		nonProtocol = 0

		// Detecção de estagnação: mesma tool + mesmos args repetidos.
		callJSON := fmt.Sprintf("%s|%v", call.Tool, call.Args)
		if callJSON == lastCallJSON {
			repeatCount++
		} else {
			repeatCount = 0
			lastCallJSON = callJSON
		}
		if repeatCount >= 3 {
			return "Não consegui progredir: repeti a mesma ferramenta várias vezes sem avanço. Faltam informações para completar a tarefa.", steps, nil
		}

		// Safe autonomy: consulta a policy antes de executar a tool.
		if a.policy != nil {
			risk := autonomy.ActionRead
			if t, ok := a.tools.Get(call.Tool); ok && t.Risk != "" {
				risk = t.Risk
			}
			if !a.policy.Allow(risk, call.Tool) {
				messages = append(messages, provider.Message{
					Role:    "user",
					Content: fmt.Sprintf("A tool %q foi negada pela política de autonomia (nível %s). Não insista; informe que não tem permissão.", call.Tool, a.policy.Level),
				})
				continue
			}
		}

		result, callErr := a.tools.Call(ctx, call.Tool, call.Args)
		if callErr != nil {
			result = "ERRO: " + callErr.Error()
		}
		steps = append(steps, Step{Tool: call.Tool, Args: call.Args, Result: result})
		messages = append(messages, provider.Message{
			Role:    "user",
			Content: "Resultado da tool " + call.Tool + ":\n" + result,
		})
	}

	return "", steps, fmt.Errorf("orçamento de %d passos esgotado", a.maxSteps)
}

// systemPrompt descreve o agente e as tools disponíveis ao modelo.
func (a *Agent) systemPrompt() string {
	var sb strings.Builder
	if a.systemExtra != "" {
		sb.WriteString(a.systemExtra + "\n\n")
	}
	sb.WriteString("Você é o agente \"" + a.Name + "\" do COSCA CODE, um ambiente de engenharia de software.\n")
	sb.WriteString("Você resolve tarefas usando ferramentas.\n\n")
	sb.WriteString("Ferramentas disponíveis:\n")
	for _, t := range a.tools.List() {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.Description))
	}
	sb.WriteString("\nProtocolo (responda SEMPRE apenas JSON válido, sem markdown):\n")
	sb.WriteString(`Para usar uma ferramenta: {"tool":"nome","args":{...}}` + "\n")
	sb.WriteString(`Para dar a resposta final: {"final":"sua resposta"}` + "\n")
	return sb.String()
}

// toolCall é a forma JSON de uma chamada de tool.
type toolCall struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// finalAnswer é a forma JSON da resposta final.
type finalAnswer struct {
	Final string `json:"final"`
}

// parseFinal extrai a resposta final de um JSON {"final": "..."}.
func parseFinal(reply string) string {
	trimmed := stripFences(reply)
	var fa finalAnswer
	if err := json.Unmarshal([]byte(trimmed), &fa); err == nil && fa.Final != "" {
		return fa.Final
	}
	// Tenta extrair de um JSON embutido em texto.
	start := strings.Index(trimmed, "{")
	if start < 0 {
		return ""
	}
	if err := json.Unmarshal([]byte(trimmed[start:]), &fa); err == nil && fa.Final != "" {
		return fa.Final
	}
	return ""
}

// parseToolCall extrai uma chamada de tool de um JSON {"tool": "...", ...}.
func parseToolCall(reply string) *toolCall {
	trimmed := stripFences(reply)
	var tc toolCall
	if err := json.Unmarshal([]byte(trimmed), &tc); err == nil && tc.Tool != "" {
		return &tc
	}
	start := strings.Index(trimmed, "{")
	if start < 0 {
		return nil
	}
	if err := json.Unmarshal([]byte(trimmed[start:]), &tc); err == nil && tc.Tool != "" {
		return &tc
	}
	return nil
}

// stripFences remove fences markdown (```json ... ```) que modelos costumam
// incluir mesmo quando instruídos a responder só JSON.
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			lines = lines[1:] // remove a linha ```json
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
		}
		s = strings.Join(lines, "\n")
	}
	return strings.TrimSpace(s)
}
