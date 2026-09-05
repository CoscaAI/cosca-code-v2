package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CoscaAI/cosca-code/internal/autonomy"
	"github.com/CoscaAI/cosca-code/internal/search"
	"github.com/CoscaAI/cosca-code/internal/workspace"
)

// Builtin registra as tools nativas do COSCA CODE no registry, todas com
// acesso controlado ao workspace (nunca escapam do root).
func Builtin(r *Registry, ws *workspace.Workspace) {
	r.Register(&Tool{
		Name:        "read_file",
		Description: "Lê o conteúdo de um arquivo do workspace (relativo ao root).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "caminho relativo do arquivo"},
			},
			"required": []string{"path"},
		},
		Risk: autonomy.ActionRead,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return "", fmt.Errorf("path obrigatório")
			}
			abs := filepath.Join(ws.Root(), path)
			data, err := os.ReadFile(abs)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	})

	r.Register(&Tool{
		Name:        "list_files",
		Description: "Lista os arquivos do workspace (git-aware, ignora dependências).",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Risk:        autonomy.ActionRead,
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			files := ws.Files()
			return strings.Join(files, "\n"), nil
		},
	})

	r.Register(&Tool{
		Name:        "search",
		Description: "Busca text/regex no workspace e retorna os resultados (path:linha:conteúdo).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
				"regex": map[string]any{"type": "boolean"},
			},
			"required": []string{"query"},
		},
		Risk: autonomy.ActionRead,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			query, _ := args["query"].(string)
			if query == "" {
				return "", fmt.Errorf("query obrigatória")
			}
			isRegex, _ := args["regex"].(bool)
			res, err := search.Search(ws.Root(), search.Options{Query: query, Regex: isRegex, MaxResults: 50})
			if err != nil {
				return "", err
			}
			var sb strings.Builder
			for _, m := range res.Matches {
				fmt.Fprintf(&sb, "%s:%d: %s\n", m.Path, m.Line, m.Content)
			}
			if sb.Len() == 0 {
				return "nenhum resultado", nil
			}
			return sb.String(), nil
		},
	})
}
