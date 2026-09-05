// Package provider é o Provider Engine do COSCA CODE (spec seções 8-11): uma
// abstração universal de provedores de modelo. O núcleo nunca depende de um
// provider específico — adapters independentes (ollama, openai-compatible,
// etc.) implementam a mesma interface. O usuário escolhe Provider + Model e
// começa a programar.
package provider

import "context"

// Message é uma mensagem de chat.
type Message struct {
	Role    string `json:"role"` // system | user | assistant
	Content string `json:"content"`
}

// ModelInfo descreve um modelo disponível no provider.
type ModelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Provider é a interface de um provedor de modelos.
type Provider interface {
	// Name devolve o identificador do provider (ex.: "ollama", "openai").
	Name() string
	// Models lista os modelos disponíveis.
	Models(ctx context.Context) ([]ModelInfo, error)
	// Chat faz uma inferência não-streaming e devolve a resposta.
	Chat(ctx context.Context, model string, messages []Message) (string, error)
}

// Config é a configuração de um provider.
type Config struct {
	Name    string // identificador (ex.: "ollama", "my-openai")
	BaseURL string
	APIKey  string
	Model   string // modelo padrão
}
