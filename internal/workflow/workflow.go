// Package workflow é o Workflow Engine do COSCA CODE (spec seção 64):
// orquestra agentes em fluxos definidos (bug fix, feature, refactor, ...).
// Cada passo é uma instrução para o agente, com o contexto dos passos
// anteriores acumulado — o agente tem acesso às tools (ler/editar/buscar/testar).
package workflow

import (
	"context"
	"fmt"

	"github.com/CoscaAI/cosca-code/internal/agent"
)

// Step é um passo do workflow.
type Step struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

// Workflow é uma sequência de passos.
type Workflow struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Steps       []Step `json:"steps"`
}

// StepResult é o resultado de um passo.
type StepResult struct {
	Name   string `json:"name"`
	Output string `json:"output"`
}

// Result é o resultado completo da execução do workflow.
type Result struct {
	Workflow string       `json:"workflow"`
	Steps    []StepResult `json:"steps"`
	Final    string       `json:"final"`
}

// Engine executa workflows usando um agente (com tools).
type Engine struct {
	agent *agent.Agent
}

// NewEngine cria o engine.
func NewEngine(a *agent.Agent) *Engine {
	return &Engine{agent: a}
}

// Run executa o workflow para uma tarefa, passando o contexto acumulado.
func (e *Engine) Run(ctx context.Context, wf Workflow, task string) (*Result, error) {
	contextSoFar := task
	result := &Result{Workflow: wf.Name}

	for _, step := range wf.Steps {
		prompt := fmt.Sprintf(
			"Tarefa original: %s\n\nPasso atual (%s): %s\n\nContexto acumulado:\n%s",
			task, step.Name, step.Prompt, contextSoFar,
		)
		output, _, err := e.agent.Run(ctx, prompt)
		if err != nil {
			return result, fmt.Errorf("passo %s: %w", step.Name, err)
		}
		result.Steps = append(result.Steps, StepResult{Name: step.Name, Output: output})
		contextSoFar += "\n\n[" + step.Name + "]: " + output
	}

	if len(result.Steps) > 0 {
		result.Final = result.Steps[len(result.Steps)-1].Output
	}
	return result, nil
}

// BugFix é o workflow de correção de bug.
func BugFix() Workflow {
	return Workflow{
		Name:        "bug_fix",
		Description: "Investiga, localiza e corrige um bug.",
		Steps: []Step{
			{Name: "investigar", Prompt: "Investigue o problema e identifique os arquivos/funções relevantes (use search e read_file)."},
			{Name: "localizar", Prompt: "Localize a causa provável do bug e explique o raciocínio."},
			{Name: "corrigir", Prompt: "Proponha a correção concreta (descreva o que mudar)."},
			{Name: "verificar", Prompt: "Verifique se a correção resolve o problema sem efeitos colaterais."},
		},
	}
}

// Feature é o workflow de nova funcionalidade.
func Feature() Workflow {
	return Workflow{
		Name:        "feature",
		Description: "Planeja e implementa uma nova funcionalidade.",
		Steps: []Step{
			{Name: "planejar", Prompt: "Planeje a implementação: o que mudar e onde."},
			{Name: "implementar", Prompt: "Descreva a implementação concreta (funções/arquivos a criar ou alterar)."},
			{Name: "testar", Prompt: "Descreva como testar e o que verificar."},
		},
	}
}

// Refactor é o workflow de refatoração.
func Refactor() Workflow {
	return Workflow{
		Name:        "refactor",
		Description: "Analisa e refatora código existente.",
		Steps: []Step{
			{Name: "analisar", Prompt: "Analise o código atual e identifique oportunidades de melhoria."},
			{Name: "refatorar", Prompt: "Descreva a refatoração concreta."},
			{Name: "verificar", Prompt: "Verifique que o comportamento foi preservado."},
		},
	}
}
