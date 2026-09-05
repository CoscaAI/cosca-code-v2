// Package lsp é o cliente Language Server Protocol do COSCA CODE (spec seção
// 14): JSON-RPC sobre stdio com framing Content-Length. Fala com qualquer
// language server (gopls, rust-analyzer, pyright...) via o mesmo contrato —
// a inteligência de código (completion, hover, definition, diagnostics) nunca
// é reimplementada no núcleo; é delegada ao server real.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// request é uma mensagem JSON-RPC de request (tem id + method).
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// response é a resposta a um request.
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// notification é uma mensagem JSON-RPC sem id (do servidor).
type notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Client é um cliente LSP sobre stdio.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	mu      sync.Mutex
	nextID  int
	pending map[int]chan *response

	onNotify  map[string]func(method string, params json.RawMessage)
	closeOnce sync.Once
}

// Start inicia o language server (binário + args) e executa o handshake
// initialize. Notifica onNotify para cada notification recebida.
func Start(bin string, args []string, rootURI string, onNotify func(method string, params json.RawMessage)) (*Client, error) {
	cmd := exec.Command(bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = nil // descarta (servers logam no stderr)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("iniciar %s: %w", bin, err)
	}

	c := &Client{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   bufio.NewReader(stdout),
		pending:  map[int]chan *response{},
		onNotify: map[string]func(string, json.RawMessage){},
	}
	if onNotify != nil {
		c.onNotify["*"] = onNotify
	}

	go c.readLoop()

	// Handshake initialize.
	initParams := map[string]any{
		"processId": nil,
		"rootUri":   rootURI,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"hover":              map[string]any{"contentFormat": []string{"markdown", "plaintext"}},
				"definition":         map[string]any{"linkSupport": true},
				"completion":         map[string]any{},
				"publishDiagnostics": map[string]any{},
			},
		},
	}
	if _, err := c.request("initialize", initParams); err != nil {
		c.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}
	c.notify("initialized", map[string]any{})
	return c, nil
}

// Close encerra o servidor (shutdown + exit + kill do processo).
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		_, _ = c.request("shutdown", nil)
		_ = c.notify("exit", nil)
		_ = c.stdin.Close()
		_ = c.cmd.Wait()
	})
}

// request envia um request e bloqueia até a resposta (com timeout).
func (c *Client) request(method string, params any) (json.RawMessage, error) {
	raw, _ := json.Marshal(params)
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan *response, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	msg, _ := json.Marshal(request{JSONRPC: "2.0", ID: id, Method: method, Params: raw})
	if err := c.write(msg); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp == nil {
			return nil, fmt.Errorf("sem resposta para %s", method)
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("%s: %s (code %d)", method, resp.Error.Message, resp.Error.Code)
		}
		return resp.Result, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout em %s", method)
	}
}

// notify envia uma notification (sem resposta).
func (c *Client) notify(method string, params any) error {
	raw, _ := json.Marshal(params)
	msg, _ := json.Marshal(notification{JSONRPC: "2.0", Method: method, Params: raw})
	return c.write(msg)
}

// write envia uma mensagem com framing Content-Length.
func (c *Client) write(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(msg))
	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return err
	}
	_, err := c.stdin.Write(msg)
	return err
}

// readLoop lê mensagens do servidor e faz dispatch (response → pending;
// notification → onNotify).
func (c *Client) readLoop() {
	for {
		msg, err := c.readMessage()
		if err != nil {
			// Encerra: resolve todos os pending com nil (timeout/sem resposta).
			c.mu.Lock()
			for id, ch := range c.pending {
				ch <- nil
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}

		// Tenta parse como response (tem id).
		var resp response
		if json.Unmarshal(msg, &resp) == nil && resp.ID != 0 {
			c.mu.Lock()
			ch, ok := c.pending[resp.ID]
			if ok {
				ch <- &resp
				delete(c.pending, resp.ID)
			}
			c.mu.Unlock()
			continue
		}

		// Tenta parse como notification.
		var notif notification
		if json.Unmarshal(msg, &notif) == nil && notif.Method != "" {
			c.mu.Lock()
			handler, ok := c.onNotify["*"]
			c.mu.Unlock()
			if ok {
				handler(notif.Method, notif.Params)
			}
		}
	}
}

// readMessage lê uma mensagem com framing Content-Length.
func (c *Client) readMessage() ([]byte, error) {
	var contentLength int
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // fim dos headers
		}
		if strings.HasPrefix(line, "Content-Length:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(v)
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("Content-Length inválido")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(c.stdout, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
