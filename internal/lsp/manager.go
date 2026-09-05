package lsp

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Position é uma posição (linha/coluna, 0-indexadas) no documento.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range é um intervalo [start, end).
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Diagnostic é um erro/aviso reportado pelo language server.
type Diagnostic struct {
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	Range    Range  `json:"range"`
}

// Location é o resultado de "definition".
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// CompletionItem é um item de autocomplete do language server.
type CompletionItem struct {
	Label      string `json:"label"`
	Detail     string `json:"detail,omitempty"`
	Kind       int    `json:"kind,omitempty"`
	InsertText string `json:"insert_text,omitempty"`
}

// Manager gerencia language servers por linguagem: um Client por linguagem,
// com os documentos abertos e os diagnostics capturados por notification.
type Manager struct {
	mu       sync.Mutex
	root     string
	clients  map[string]*Client
	docs     map[string]string       // uri → texto
	docLangs map[string]string       // uri → linguagem
	diags    map[string][]Diagnostic // uri → diagnostics
}

// NewManager cria um manager vazio.
func NewManager(root string) *Manager {
	return &Manager{
		root:     root,
		clients:  map[string]*Client{},
		docs:     map[string]string{},
		docLangs: map[string]string{},
		diags:    map[string][]Diagnostic{},
	}
}

// OpenDocument abre (spawn se preciso) o server da linguagem e envia didOpen.
func (m *Manager) OpenDocument(uri, lang, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[lang]; !ok {
		bin, args, err := Detect(lang)
		if err != nil {
			return err
		}
		client, err := Start(bin, args, uriFor(m.root), m.handleNotification)
		if err != nil {
			return err
		}
		m.clients[lang] = client
	}

	m.docs[uri] = text
	m.docLangs[uri] = lang

	params := map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": lang,
			"version":    1,
			"text":       text,
		},
	}
	return m.clients[lang].notify("textDocument/didOpen", params)
}

// DidChange envia a mudança de conteúdo (full sync) para o server.
func (m *Manager) DidChange(uri, text string) error {
	m.mu.Lock()
	m.docs[uri] = text
	client := m.clientForURI(uri)
	m.mu.Unlock()

	if client == nil {
		return nil
	}
	params := map[string]any{
		"textDocument":   map[string]any{"uri": uri, "version": 2},
		"contentChanges": []map[string]any{{"text": text}},
	}
	return client.notify("textDocument/didChange", params)
}

// Hover devolve a documentação/assinatura do símbolo na posição.
func (m *Manager) Hover(uri string, line, char int) (string, error) {
	m.mu.Lock()
	client := m.clientForURI(uri)
	m.mu.Unlock()
	if client == nil {
		return "", fmt.Errorf("sem language server para %s", uri)
	}

	raw, err := client.request("textDocument/hover", textDocPos(uri, line, char))
	if err != nil {
		return "", err
	}
	var hover struct {
		Contents any `json:"contents"`
	}
	if err := json.Unmarshal(raw, &hover); err != nil {
		return "", err
	}
	return hoverMarkdown(hover.Contents), nil
}

// Definition devolve a localização(ões) da definição do símbolo.
func (m *Manager) Definition(uri string, line, char int) ([]Location, error) {
	m.mu.Lock()
	client := m.clientForURI(uri)
	m.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("sem language server para %s", uri)
	}

	raw, err := client.request("textDocument/definition", textDocPos(uri, line, char))
	if err != nil {
		return nil, err
	}
	if string(raw) == "null" {
		return nil, nil
	}
	var locs []Location
	if err := json.Unmarshal(raw, &locs); err == nil {
		return locs, nil
	}
	var single Location
	if err := json.Unmarshal(raw, &single); err == nil && single.URI != "" {
		return []Location{single}, nil
	}
	return nil, nil
}

// Diagnostics devolve os diagnostics capturados para o documento.
func (m *Manager) Diagnostics(uri string) []Diagnostic {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.diags[uri]
}

// Completion devolve os itens de autocomplete do language server na posição.
func (m *Manager) Completion(uri string, line, char int) ([]CompletionItem, error) {
	m.mu.Lock()
	client := m.clientForURI(uri)
	m.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("sem language server para %s", uri)
	}

	raw, err := client.request("textDocument/completion", textDocPos(uri, line, char))
	if err != nil {
		return nil, err
	}
	if string(raw) == "null" {
		return nil, nil
	}
	// Resposta pode ser CompletionList{items} ou array de CompletionItem.
	var list struct {
		Items []CompletionItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err == nil && list.Items != nil {
		return list.Items, nil
	}
	var items []CompletionItem
	if err := json.Unmarshal(raw, &items); err == nil {
		return items, nil
	}
	return nil, nil
}

// Close encerra todos os language servers.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.clients {
		c.Close()
	}
	m.clients = map[string]*Client{}
}

// handleNotification captura notifications do servidor (publishDiagnostics).
func (m *Manager) handleNotification(method string, params json.RawMessage) {
	if method != "textDocument/publishDiagnostics" {
		return
	}
	var p struct {
		URI         string       `json:"uri"`
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	m.mu.Lock()
	m.diags[p.URI] = p.Diagnostics
	m.mu.Unlock()
}

// clientForURI acha o client que serve uma URI.
func (m *Manager) clientForURI(uri string) *Client {
	if lang, ok := m.docLangs[uri]; ok {
		return m.clients[lang]
	}
	return nil
}

// uriFor converte um path em file URI.
func uriFor(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return "file://" + filepath.ToSlash(abs)
}

// textDocPos monta o params comum {textDocument, position}.
func textDocPos(uri string, line, char int) map[string]any {
	return map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line, "character": char},
	}
}

// hoverMarkdown extrai texto legível do contents do hover (markdown ou string).
func hoverMarkdown(contents any) string {
	switch c := contents.(type) {
	case string:
		return c
	case map[string]any:
		if v, ok := c["value"].(string); ok {
			return v
		}
	case []any:
		// array de MarkedString/MarkupContent: concatena os valores.
		var parts []string
		for _, item := range c {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			} else if obj, ok := item.(map[string]any); ok {
				if v, ok := obj["value"].(string); ok {
					parts = append(parts, v)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}
