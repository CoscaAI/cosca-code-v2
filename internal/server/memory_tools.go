package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/CoscaAI/cosca-code/internal/autonomy"
	"github.com/CoscaAI/cosca-code/internal/memory"
	"github.com/CoscaAI/cosca-code/internal/tool"
)

// registerMemoryTools registra as tools de memória (remember/recall) no
// registry, ligadas à Memory Store do server.
func registerMemoryTools(r *tool.Registry, mem *memory.Store) {
	r.Register(&tool.Tool{
		Name:        "remember",
		Description: "Registra uma memória sobre o projeto (key e value) para consulta futura.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key":   map[string]any{"type": "string"},
				"value": map[string]any{"type": "string"},
			},
			"required": []string{"key", "value"},
		},
		Risk: autonomy.ActionWrite,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			key, _ := args["key"].(string)
			value, _ := args["value"].(string)
			if key == "" || value == "" {
				return "", fmt.Errorf("key e value obrigatórios")
			}
			e := mem.Set(key, value, "agent", 0.8)
			return fmt.Sprintf("memória registrada: %s (v%d)", e.Key, e.Version), nil
		},
	})

	r.Register(&tool.Tool{
		Name:        "recall",
		Description: "Recupera uma memória pelo key (ou lista todas se key vazio).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{"type": "string"},
			},
		},
		Risk: autonomy.ActionRead,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			key, _ := args["key"].(string)
			if key != "" {
				if e, ok := mem.Get(key); ok {
					return e.Value, nil
				}
				return "nenhuma memória para " + key, nil
			}
			entries := mem.List()
			if len(entries) == 0 {
				return "nenhuma memória registrada", nil
			}
			var sb strings.Builder
			for _, e := range entries {
				fmt.Fprintf(&sb, "%s = %s\n", e.Key, e.Value)
			}
			return sb.String(), nil
		},
	})
}
