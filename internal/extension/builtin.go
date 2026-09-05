package extension

import (
	"context"
	"os/exec"
	"strings"

	"github.com/CoscaAI/cosca-code/internal/tool"
)

// Builtin devolve as extensões nativas do COSCA CODE. Cada uma é um exemplo
// de contribuição declarativa: tools que entram no Tool Registry do núcleo.
func Builtin() []*Extension {
	return []*Extension{
		formatterExtension(),
		gitExtension(),
	}
}

// formatterExtension contribui uma tool "gofmt" (formata um arquivo Go).
func formatterExtension() *Extension {
	return &Extension{
		Manifest: Manifest{
			ID:          "cosca.formatter",
			Name:        "Formatter",
			Version:     "0.1.0",
			Description: "Formatação de código (gofmt para Go).",
			Category:    "Formatter",
		},
		Tools: []*tool.Tool{
			{
				Name:        "gofmt",
				Description: "Formata um arquivo Go usando o gofmt (retorna o código formatado).",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code": map[string]any{"type": "string"},
					},
					"required": []string{"code"},
				},
				Handler: func(_ context.Context, args map[string]any) (string, error) {
					code, _ := args["code"].(string)
					cmd := exec.Command("gofmt")
					cmd.Stdin = strings.NewReader(code)
					out, err := cmd.Output()
					if err != nil {
						return code, nil // gofmt falhou: devolve o código original
					}
					return string(out), nil
				},
			},
		},
	}
}

// gitExtension contribui uma tool "git_log" (últimos commits do repo).
func gitExtension() *Extension {
	return &Extension{
		Manifest: Manifest{
			ID:          "cosca.git",
			Name:        "Git",
			Version:     "0.1.0",
			Description: "Integração Git (log).",
			Category:    "Git",
		},
		Tools: []*tool.Tool{
			{
				Name:        "git_log",
				Description: "Lista os últimos commits do repositório.",
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				Handler: func(_ context.Context, _ map[string]any) (string, error) {
					cmd := exec.Command("git", "log", "--oneline", "-n", "10")
					out, err := cmd.Output()
					if err != nil {
						return "", err
					}
					return strings.TrimSpace(string(out)), nil
				},
			},
		},
	}
}
