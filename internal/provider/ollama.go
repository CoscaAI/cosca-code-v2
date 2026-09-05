package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaProvider é o adapter para o Ollama (local, OpenAI-compatible-ish).
// Endpoints: POST /api/chat (chat), GET /api/tags (modelos).
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

// NewOllama cria o adapter Ollama (baseURL default http://127.0.0.1:11434).
func NewOllama(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434"
	}
	return &OllamaProvider{baseURL: baseURL, client: &http.Client{Timeout: 120 * time.Second}}
}

func (o *OllamaProvider) Name() string { return "ollama" }

// Models lista os modelos via GET /api/tags.
func (o *OllamaProvider) Models(ctx context.Context) ([]ModelInfo, error) {
	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := o.get(ctx, "/api/tags", &resp); err != nil {
		return nil, err
	}
	var out []ModelInfo
	for _, m := range resp.Models {
		out = append(out, ModelInfo{ID: m.Name, Name: m.Name})
	}
	return out, nil
}

// Chat faz inferência via POST /api/chat (non-streaming).
func (o *OllamaProvider) Chat(ctx context.Context, model string, messages []Message) (string, error) {
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	var resp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error string `json:"error"`
	}
	if err := o.post(ctx, "/api/chat", body, &resp); err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("ollama: %s", resp.Error)
	}
	return resp.Message.Content, nil
}

func (o *OllamaProvider) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama GET %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (o *OllamaProvider) post(ctx context.Context, path string, body, out any) error {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama POST %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
