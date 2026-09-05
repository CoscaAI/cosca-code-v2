package provider

// CatalogEntry é um provider do catálogo (padrão models.dev): nome, base URL,
// tipo de API e modelos principais. O usuário escolhe provider + modelo + chave
// e o COSCA conecta via adapter (a maioria é openai-compatible).
type CatalogEntry struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	BaseURL  string   `json:"base_url"`
	Kind     string   `json:"kind"` // "openai" | "anthropic" | "cosca"
	Models   []string `json:"models"`
	NeedsKey bool     `json:"needs_key"`
	Local    bool     `json:"local"`
}

// Catalog devolve o catálogo embutido de providers (curadoria dos principais
// do models.dev, relevantes para um IDE de código).
func Catalog() []CatalogEntry {
	return []CatalogEntry{
		{
			// COSCA solo (ADR-032): o provider que os dois (Desktop e Code) se
			// conectam. Kind "cosca" → handleProviderConnect cria o adapter via
			// pkg/cosca (/v1/run Kernel-First), não NewOpenAICompat.
			ID: "cosca", Name: "COSCA (Kernel)", BaseURL: "http://127.0.0.1:14120",
			Kind: "cosca", Models: append([]string(nil), coscaModes...),
			NeedsKey: false, Local: true,
		},
		{
			ID: "ollama", Name: "Ollama (local)", BaseURL: "http://127.0.0.1:11434",
			Kind: "openai", Models: []string{"qwen2.5-coder:14b", "llama3.3", "deepseek-r1"},
			NeedsKey: false, Local: true,
		},
		{
			ID: "openai", Name: "OpenAI", BaseURL: "https://api.openai.com/v1",
			Kind: "openai", Models: []string{"gpt-4o", "gpt-4o-mini", "o3-mini", "gpt-4.1"},
			NeedsKey: true,
		},
		{
			ID: "anthropic", Name: "Anthropic", BaseURL: "https://api.anthropic.com/v1",
			Kind: "anthropic", Models: []string{"claude-sonnet-4-20250514", "claude-3-7-sonnet-latest", "claude-3-5-haiku-latest"},
			NeedsKey: true,
		},
		{
			ID: "google", Name: "Google (Gemini)", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
			Kind: "openai", Models: []string{"gemini-2.5-pro", "gemini-2.0-flash", "gemini-2.5-flash"},
			NeedsKey: true,
		},
		{
			ID: "deepseek", Name: "DeepSeek", BaseURL: "https://api.deepseek.com",
			Kind: "openai", Models: []string{"deepseek-chat", "deepseek-reasoner"},
			NeedsKey: true,
		},
		{
			ID: "groq", Name: "Groq", BaseURL: "https://api.groq.com/openai/v1",
			Kind: "openai", Models: []string{"llama-3.3-70b-versatile", "deepseek-r1-distill-llama-70b"},
			NeedsKey: true,
		},
		{
			ID: "mistral", Name: "Mistral", BaseURL: "https://api.mistral.ai/v1",
			Kind: "openai", Models: []string{"mistral-large-latest", "codestral-latest", "mistral-small-latest"},
			NeedsKey: true,
		},
		{
			ID: "openrouter", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1",
			Kind: "openai", Models: []string{"anthropic/claude-sonnet-4", "openai/gpt-4o", "google/gemini-2.5-pro"},
			NeedsKey: true,
		},
		{
			ID: "togetherai", Name: "Together AI", BaseURL: "https://api.together.xyz/v1",
			Kind: "openai", Models: []string{"deepseek-r1", "llama-3.3-70b-instruct-turbo"},
			NeedsKey: true,
		},
		{
			ID: "fireworks", Name: "Fireworks AI", BaseURL: "https://api.fireworks.ai/inference/v1",
			Kind: "openai", Models: []string{"accounts/fireworks/models/deepseek-r1", "accounts/fireworks/models/qwen2p5-coder-32b-instruct"},
			NeedsKey: true,
		},
		{
			ID: "xai", Name: "xAI (Grok)", BaseURL: "https://api.x.ai/v1",
			Kind: "openai", Models: []string{"grok-3-mini", "grok-2-latest"},
			NeedsKey: true,
		},
	}
}

// FindCatalog devolve um entry do catálogo pelo ID.
func FindCatalog(id string) (CatalogEntry, bool) {
	for _, e := range Catalog() {
		if e.ID == id {
			return e, true
		}
	}
	return CatalogEntry{}, false
}
