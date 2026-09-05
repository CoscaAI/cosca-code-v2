package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OpenAICompatProvider é o adapter para qualquer servidor OpenAI-compatible
// (OpenAI, DeepSeek, Groq, vLLM, LM Studio, etc.). Endpoints:
// POST /v1/chat/completions, GET /v1/models.
type OpenAICompatProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewOpenAICompat cria um adapter OpenAI-compatible.
func NewOpenAICompat(name, baseURL, apiKey string) *OpenAICompatProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAICompatProvider{
		name:    name,
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (o *OpenAICompatProvider) Name() string { return o.name }

// Models lista os modelos via GET /v1/models.
func (o *OpenAICompatProvider) Models(ctx context.Context) ([]ModelInfo, error) {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := o.get(ctx, "/v1/models", &resp); err != nil {
		return nil, err
	}
	var out []ModelInfo
	for _, m := range resp.Data {
		out = append(out, ModelInfo{ID: m.ID, Name: m.ID})
	}
	return out, nil
}

// Chat faz inferência via POST /v1/chat/completions.
func (o *OpenAICompatProvider) Chat(ctx context.Context, model string, messages []Message) (string, error) {
	body := map[string]any{
		"model":    model,
		"messages": messages,
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := o.post(ctx, "/v1/chat/completions", body, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("%s: %s", o.name, resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("%s: resposta vazia", o.name)
	}
	return resp.Choices[0].Message.Content, nil
}

func (o *OpenAICompatProvider) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+path, nil)
	if err != nil {
		return err
	}
	o.auth(req)
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s GET %s: status %d", o.name, path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (o *OpenAICompatProvider) post(ctx context.Context, path string, body, out any) error {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	o.auth(req)
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s POST %s: status %d", o.name, path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (o *OpenAICompatProvider) auth(req *http.Request) {
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
}
