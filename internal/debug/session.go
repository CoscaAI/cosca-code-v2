package debug

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// StackFrame é um frame da call stack.
type StackFrame struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Source struct {
		Path string `json:"path"`
	} `json:"source"`
	Line int `json:"line"`
}

// Variable é uma variável em um escopo.
type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// Session é uma sessão de debug sobre um debug adapter DAP (dlv p/ Go).
type Session struct {
	client *Client
	cmd    *exec.Cmd
}

// Launch inicia uma sessão de debug: spawna `dlv dap`, conecta via TCP,
// inicializa e dispara o launch (build + debug do programa).
func Launch(program string, port int) (*Session, error) {
	bin := detectDlv()
	if bin == "" {
		return nil, fmt.Errorf("dlv não encontrado (instale: go install github.com/go-delve/delve/cmd/dlv@latest)")
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	cmd := exec.Command(bin, "dap", "--listen="+addr)
	cmd.Dir = filepath.Dir(program)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("spawn dlv: %w", err)
	}

	// Aguarda o dlv dap escutar (retry no dial).
	var client *Client
	var err error
	for i := 0; i < 50; i++ {
		client, err = Dial(addr)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if client == nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("conectar ao dlv dap: %w", err)
	}

	s := &Session{client: client, cmd: cmd}

	// Handshake DAP.
	if _, err := client.request("initialize", map[string]any{
		"adapterID": "go", "clientID": "cosca-code", "linesStartAt1": true,
		"columnsStartAt1": true,
	}); err != nil {
		s.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	// Launch (mode debug: build + run).
	if _, err := client.request("launch", map[string]any{
		"mode": "debug", "program": program, "stopOnEntry": false,
	}); err != nil {
		s.Close()
		return nil, fmt.Errorf("launch: %w", err)
	}
	return s, nil
}

// SetBreakpoint define um breakpoint em file:line.
func (s *Session) SetBreakpoint(file string, line int) error {
	return s.SetConditionalBreakpoint(file, line, "", "")
}

// SetConditionalBreakpoint define um breakpoint com condição (só dispara quando
// verdadeira) e/ou logMessage (logpoint: loga e não para).
func (s *Session) SetConditionalBreakpoint(file string, line int, condition, logMessage string) error {
	bp := map[string]any{"line": line}
	if condition != "" {
		bp["condition"] = condition
	}
	if logMessage != "" {
		bp["logMessage"] = logMessage
	}
	_, err := s.client.request("setBreakpoints", map[string]any{
		"source":      map[string]any{"path": file},
		"breakpoints": []map[string]any{bp},
	})
	return err
}

// Evaluate avalia uma expressão no contexto do frame (watch/repl).
func (s *Session) Evaluate(frameID int, expression string) (string, error) {
	raw, err := s.client.request("evaluate", map[string]any{
		"expression": expression,
		"frameId":    frameID,
		"context":    "repl",
	})
	if err != nil {
		return "", err
	}
	var body struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", err
	}
	return body.Result, nil
}

// ConfigurationDone sinaliza que a configuração terminou (resume execução).
func (s *Session) ConfigurationDone() error {
	_, err := s.client.request("configurationDone", nil)
	return err
}

// WaitStop espera o evento stopped e devolve a thread parada (ou erro em
// terminated/timeout).
func (s *Session) WaitStop(timeout time.Duration) (int, error) {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-s.client.Events():
			if !ok {
				return 0, fmt.Errorf("adapter encerrou")
			}
			if ev.Name == "stopped" {
				var body struct {
					ThreadID int `json:"threadId"`
				}
				_ = json.Unmarshal(ev.Body, &body)
				return body.ThreadID, nil
			}
			if ev.Name == "terminated" {
				return 0, fmt.Errorf("programa terminou sem parar no breakpoint")
			}
		case <-deadline:
			return 0, fmt.Errorf("timeout esperando stopped")
		}
	}
}

// StackTrace devolve a call stack da thread.
func (s *Session) StackTrace(threadID int) ([]StackFrame, error) {
	raw, err := s.client.request("stackTrace", map[string]any{"threadId": threadID})
	if err != nil {
		return nil, err
	}
	var body struct {
		StackFrames []StackFrame `json:"stackFrames"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	return body.StackFrames, nil
}

// Variables devolve as variáveis de um escopo (frameID → scopes → variables).
func (s *Session) Variables(frameID int) ([]Variable, error) {
	raw, err := s.client.request("scopes", map[string]any{"frameId": frameID})
	if err != nil {
		return nil, err
	}
	var scopes struct {
		Scopes []struct {
			Name               string `json:"name"`
			VariablesReference int    `json:"variablesReference"`
		} `json:"scopes"`
	}
	if err := json.Unmarshal(raw, &scopes); err != nil {
		return nil, err
	}

	var vars []Variable
	for _, sc := range scopes.Scopes {
		if sc.VariablesReference == 0 {
			continue
		}
		vraw, err := s.client.request("variables", map[string]any{"variablesReference": sc.VariablesReference})
		if err != nil {
			continue
		}
		var vbody struct {
			Variables []Variable `json:"variables"`
		}
		if json.Unmarshal(vraw, &vbody) == nil {
			vars = append(vars, vbody.Variables...)
		}
	}
	return vars, nil
}

// Continue retoma a execução.
func (s *Session) Continue() error {
	_, err := s.client.request("continue", map[string]any{"threadId": 0})
	return err
}

// Next avança uma linha (step over).
func (s *Session) Next(threadID int) error {
	_, err := s.client.request("next", map[string]any{"threadId": threadID})
	return err
}

// Disconnect encerra a sessão (disconnect + mata o dlv).
func (s *Session) Disconnect() error {
	_, _ = s.client.request("disconnect", map[string]any{"terminateDebuggee": true})
	s.Close()
	return nil
}

// Close encerra cliente e processo.
func (s *Session) Close() {
	if s.client != nil {
		s.client.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}

// detectDlv resolve o binário do dlv (PATH + ~/go/bin).
func detectDlv() string {
	if p, err := exec.LookPath("dlv"); err == nil {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, "go", "bin", "dlv")
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}
