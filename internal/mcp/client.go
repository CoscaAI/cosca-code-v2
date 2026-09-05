// Package mcp é o adaptador MCP do COSCA CODE (spec seção 4 + mcp-patterns):
// interoperabilidade com o ecossistema Model Context Protocol. O COSCA tem seu
// PRÓPRIO Tool Registry (seguro, tipado), e este cliente traduz tools de um
// servidor MCP para o registry — interop, não dependência.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// MCPTool é uma tool descrita por um servidor MCP.
type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Client é um cliente MCP sobre stdio (newline-delimited JSON-RPC).
type Client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  int
	pending map[int]chan map[string]any
}

// Start inicia um servidor MCP (binário + args) e faz o handshake initialize.
func Start(bin string, args ...string) (*Client, error) {
	cmd := exec.Command(bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("iniciar servidor MCP %s: %w", bin, err)
	}

	c := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		pending: map[int]chan map[string]any{},
	}
	go c.readLoop()

	// Handshake initialize (com capabilities vazias — não implementamos recursos).
	if _, err := c.request("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "cosca-code", "version": "0.1"},
	}); err != nil {
		c.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}
	c.notify("notifications/initialized", map[string]any{})
	return c, nil
}

// Close encerra o servidor MCP.
func (c *Client) Close() {
	_ = c.stdin.Close()
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

// ListTools devolve as tools do servidor (tools/list).
func (c *Client) ListTools() ([]MCPTool, error) {
	raw, err := c.request("tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

// CallTool chama uma tool do servidor e devolve o conteúdo textual.
func (c *Client) CallTool(name string, args map[string]any) (string, error) {
	raw, err := c.request("tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return "", err
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("tool MCP retornou erro")
	}
	if len(result.Content) == 0 {
		return "", nil
	}
	return result.Content[0].Text, nil
}

// request envia um request JSON-RPC e espera a resposta.
func (c *Client) request(method string, params any) (json.RawMessage, error) {
	raw, _ := json.Marshal(params)
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan map[string]any, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	msg, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": id, "method": method, "params": json.RawMessage(raw),
	})
	if err := c.write(msg); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp == nil {
			return nil, fmt.Errorf("sem resposta para %s", method)
		}
		if e, ok := resp["error"]; ok {
			return nil, fmt.Errorf("%s: %v", method, e)
		}
		result, _ := json.Marshal(resp["result"])
		return result, nil
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("timeout em %s", method)
	}
}

// notify envia uma notification (sem resposta).
func (c *Client) notify(method string, params any) {
	raw, _ := json.Marshal(params)
	msg, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": method, "params": json.RawMessage(raw),
	})
	_ = c.write(msg)
}

// write envia uma mensagem (newline-delimited JSON).
func (c *Client) write(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.stdin.Write(append(msg, '\n')); err != nil {
		return err
	}
	return nil
}

// readLoop lê mensagens newline-delimited e faz dispatch de responses.
func (c *Client) readLoop() {
	for {
		line, err := c.stdout.ReadBytes('\n')
		if err != nil {
			return
		}
		var msg map[string]any
		if json.Unmarshal(line, &msg) != nil {
			continue
		}
		// Só responses (têm "id") importam; notifications são ignoradas.
		if id, ok := msg["id"].(float64); ok {
			c.mu.Lock()
			ch, found := c.pending[int(id)]
			if found {
				ch <- msg
				delete(c.pending, int(id))
			}
			c.mu.Unlock()
		}
	}
}
