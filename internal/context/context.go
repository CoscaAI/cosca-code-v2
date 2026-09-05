// Package context é o Context Engine do COSCA CODE (spec seção 17): constrói
// contexto dinamicamente para o modelo — nunca o projeto inteiro. Um resumo
// orçado do projeto (linguagem, símbolos-chave, dependências, git) que é
// injetado no prompt do agente/AI, com ranking de relevância por símbolo.
package context

import (
	"fmt"
	"strings"

	"github.com/CoscaAI/cosca-code/internal/gitengine"
	"github.com/CoscaAI/cosca-code/internal/intel"
	"github.com/CoscaAI/cosca-code/internal/workspace"
)

// Context é o resumo do projeto montado para o modelo.
type Context struct {
	Info       workspace.ProjectInfo
	Symbols    []intel.Symbol
	GitBranch  string
	GitChanges []gitengine.FileStatus
}

// Build monta o contexto a partir do workspace + índice (símbolos + git).
func Build(ws *workspace.Workspace, idx *intel.Index) *Context {
	c := &Context{Info: ws.Info()}
	if idx != nil {
		c.Symbols = idx.Symbols
	}
	if g := ws.Git(); g != nil {
		c.GitBranch, _ = g.CurrentBranch()
		c.GitChanges, _ = g.Status()
	}
	return c
}

// Prompt devolve o contexto como texto para o system prompt do modelo,
// limitado a maxSymbols símbolos (orçamento — spec seção 17).
func (c *Context) Prompt(maxSymbols int) string {
	if maxSymbols <= 0 {
		maxSymbols = 40
	}
	var sb strings.Builder
	sb.WriteString("=== CONTEXTO DO PROJETO ===\n")
	sb.WriteString(fmt.Sprintf("Root: %s\n", c.Info.Root))
	sb.WriteString(fmt.Sprintf("Linguagem: %s", orDash(c.Info.Language)))
	if c.Info.Framework != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", c.Info.Framework))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Build: %s\n", orDash(c.Info.BuildCommand)))
	sb.WriteString(fmt.Sprintf("Test: %s\n", orDash(c.Info.TestCommand)))

	if c.GitBranch != "" {
		sb.WriteString(fmt.Sprintf("Git branch: %s", c.GitBranch))
		if len(c.GitChanges) > 0 {
			sb.WriteString(fmt.Sprintf(" (%d arquivo(s) modificados)", len(c.GitChanges)))
		}
		sb.WriteString("\n")
	}

	if len(c.Symbols) > 0 {
		sb.WriteString("\nSímbolos principais:\n")
		for i, s := range c.Symbols {
			if i >= maxSymbols {
				sb.WriteString(fmt.Sprintf("... e mais %d símbolos\n", len(c.Symbols)-maxSymbols))
				break
			}
			sb.WriteString(fmt.Sprintf("- %s %s (%s:%d)\n", s.Kind, s.Name, s.File, s.Line))
		}
	}
	return sb.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
