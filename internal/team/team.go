// Package team é o Multi-Agent Engine do COSCA CODE (spec seção 18/19 + Fase
// 5): uma equipe de agentes com papéis distintos (planner, coder, reviewer)
// que trabalham em sequência com verificação e recovery. O coder não "termina"
// por escrever código — a equipe VERIFICA (roda testes) e RECUPERA (re-tenta
// com o feedback) até convergir ou esgotar as tentativas.
package team

import (
	"context"
	"fmt"

	"github.com/CoscaAI/cosca-code/internal/agent"
	"github.com/CoscaAI/cosca-code/internal/provider"
	"github.com/CoscaAI/cosca-code/internal/testengine"
	"github.com/CoscaAI/cosca-code/internal/tool"
)

// Result é o resultado da execução da equipe.
type Result struct {
	Plan       string `json:"plan"`
	Code       string `json:"code"`
	TestPass   bool   `json:"test_pass"`
	TestOut    string `json:"test_output"`
	Recoveries int    `json:"recoveries"`
	Review     string `json:"review"`
}

// Team é uma equipe de agentes com papéis distintos + verificação.
type Team struct {
	router     *provider.Router
	tools      *tool.Registry
	testCmd    string
	maxRetries int
}

// NewTeam cria a equipe (testCmd = comando de teste para verificação).
func NewTeam(router *provider.Router, tools *tool.Registry, testCmd string) *Team {
	return &Team{router: router, tools: tools, testCmd: testCmd, maxRetries: 2}
}

// Run executa: planner → coder → verify (test) → recovery → reviewer.
func (t *Team) Run(ctx context.Context, task string) (*Result, error) {
	result := &Result{}

	// 1. Planner: planeja a implementação.
	plan, _, err := t.agent("planner").Run(ctx, fmt.Sprintf(
		"Você é o PLANEJADOR. Planeje a implementação para: %s. Descreva o que criar/alterar e como verificar.", task,
	))
	if err != nil {
		return result, fmt.Errorf("planner: %w", err)
	}
	result.Plan = plan

	// 2. Coder + verificação + recovery.
	code, err := t.codeWithRecovery(ctx, task, plan)
	if err != nil {
		return result, err
	}
	result.Code = code

	// 3. Reviewer: revisa o resultado.
	review, _, err := t.agent("reviewer").Run(ctx, fmt.Sprintf(
		"Você é o REVISOR. Revise a implementação para: %s\n\nPlano:\n%s\n\nImplementação:\n%s\n\nAponte problemas ou aprove.", task, plan, code,
	))
	if err != nil {
		return result, fmt.Errorf("reviewer: %w", err)
	}
	result.Review = review
	return result, nil
}

// codeWithRecovery roda o coder, verifica (teste) e recupera em caso de falha.
func (t *Team) codeWithRecovery(ctx context.Context, task, plan string) (string, error) {
	feedback := ""
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		prompt := fmt.Sprintf(
			"Você é o CODER. Implemente: %s\n\nPlano:\n%s", task, plan,
		)
		if feedback != "" {
			prompt += "\n\nFeedback da verificação anterior:\n" + feedback
		}

		code, _, err := t.agent("coder").Run(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("coder: %w", err)
		}

		// Verificação: roda os testes do projeto (se houver comando).
		if t.testCmd == "" {
			return code, nil
		}
		testRes, testErr := testengine.Run(".", t.testCmd, 0, nil)
		if testErr == nil && testRes.Passed {
			return code, nil
		}
		if attempt < t.maxRetries {
			out := "falha"
			if testRes != nil {
				out = testRes.Output
			}
			feedback = "Os testes falharam:\n" + truncate(out, 2000)
			continue // recovery: re-tenta com o feedback
		}
		// Última tentativa falhou: devolve o código com a falha registrada.
		return code, nil
	}
	return "", fmt.Errorf("recovery esgotado")
}

// agent cria um agente com nome e o tool registry da equipe.
func (t *Team) agent(name string) *agent.Agent {
	return agent.New(name, t.router, t.tools)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
