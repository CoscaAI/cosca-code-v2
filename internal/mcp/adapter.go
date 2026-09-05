package mcp

import (
	"context"

	"github.com/CoscaAI/cosca-code/internal/tool"
)

// RegisterTools lista as tools de um servidor MCP e as registra no Tool
// Registry do COSCA (interop — o COSCA mantém seu próprio registry, mas as
// tools externas entram como tools nativas, com handler delegando ao MCP).
func RegisterTools(client *Client, registry *tool.Registry) error {
	tools, err := client.ListTools()
	if err != nil {
		return err
	}
	for _, mt := range tools {
		mt := mt // capture
		registry.Register(&tool.Tool{
			Name:        mt.Name,
			Description: mt.Description,
			InputSchema: mt.InputSchema,
			Handler: func(_ context.Context, args map[string]any) (string, error) {
				return client.CallTool(mt.Name, args)
			},
		})
	}
	return nil
}
