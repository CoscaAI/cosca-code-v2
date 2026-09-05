package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	cosca "github.com/CoscaAI/cosca/pkg/cosca"
)

// =============================================================================
// COSCA (Kernel) — o provider "solo"
// =============================================================================
//
// O Don: "COSCA solo = o provider que os dois (Desktop e Code) se conectam".
// A IA do COSCA CODE deve vir do **COSCA daemon** (`POST /v1/run` — Kernel-First,
// deliberação ADR-032: resolve sem LLM ou escala→Router→LLM), e NÃO de um
// Ollama/OpenAI direto. Este adapter envolve `pkg/cosca` (o SDK oficial do
// cérebro) e injeta a requisição no /v1/run. Não duplica router/registry e não
// fala com Ollama/OpenAI.
//
// Contrato do /v1/run: `prompt string` (não `[]Message`). Por isso `Chat`
// FLATTENA as mensagens do Provider Engine para um prompt texto — a decisão de
// como reconstruir o contexto é explicitada em flattenMessages (ver GAP).

// CoscaProvider é o adapter para o COSCA daemon (Kernel-First). Expõe a
// interface `Provider` e delega a inferência ao cérebro via pkg/cosca.
type CoscaProvider struct {
	client *cosca.Client
	addr   string // "host:port" do runtime (ex.: "127.0.0.1:14120")
	model  string // modo/modelo padrão (informacional — o daemon resolve o LLM)
}

// NewCosca cria o adapter do COSCA. baseURL aceita "http://host:port",
// "https://host:port" ou "host:port" (default http://127.0.0.1:14120); model é
// o modo padrão (default cosca/kernel). Reusa pkg/cosca (sem dep circular —
// o repo cosca não importa o cosca-code).
func NewCosca(baseURL, model string) (*CoscaProvider, error) {
	addr := runtimeAddr(baseURL)
	if addr == "" {
		addr = "127.0.0.1:14120"
	}
	if model == "" {
		model = DefaultCoscaModel()
	}
	client, err := cosca.NewClient(cosca.ClientConfig{
		RuntimeAddr: addr,
		Timeout:     60 * time.Second, // /v1/run pode deliberar (docs + agentes)
	})
	if err != nil {
		return nil, err
	}
	return &CoscaProvider{client: client, addr: addr, model: model}, nil
}

// Name devolve o identificador do provider.
func (c *CoscaProvider) Name() string { return "cosca" }

// Address devolve o host:port do daemon (para diagnóstico / UI).
func (c *CoscaProvider) Address() string { return c.addr }

// Ping verifica se o daemon COSCA responde (GET /v1/health). Usado no
// Default() para registrar o provider "cosca" somente quando o cérebro está
// no ar (graceful), e no connect para não setar um default morto.
func (c *CoscaProvider) Ping(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+c.addr+"/v1/health", nil)
	if err != nil {
		return false
	}
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Models lista os modelos/modos disponíveis. O daemon (Fase 1/2) é
// Kernel-First e não expõe /v1/models de chat — a deliberação resolve sem LLM
// ou escala para o Router→LLM interno. Então:
//   - Se GET /v1/providers responde, deriva os modelos reais do cérebro
//     (mais honesto — os LLMs que o Router do daemon pode acionar).
//   - Senão devolve uma lista ESTÁVEL (modos de operação do kernel).
//
// GAP: se o daemon expuser /v1/models/chat, passamos a devolver o real.
func (c *CoscaProvider) Models(ctx context.Context) ([]ModelInfo, error) {
	if c.client != nil && c.client.Providers != nil {
		if ps, err := c.client.Providers.List(); err == nil && len(ps) > 0 {
			var out []ModelInfo
			for _, p := range ps {
				name := p.Model
				if name == "" && len(p.Models) > 0 {
					name = p.Models[0]
				}
				if name == "" {
					name = "cosca/" + p.Name
				}
				out = append(out, ModelInfo{ID: name, Name: name})
			}
			if len(out) > 0 {
				return out, nil
			}
		}
	}
	return stableCoscaModels(), nil
}

// Chat injeta a conversa no COSCA via /v1/run (Kernel-First). NÃO chama
// Ollama/OpenAI — o cérebro decide (deliberação ADR-032). O /v1/run espera
// `prompt string`, então `messages []Message` é FLATTENADO (ver flattenMessages).
// O `model` é um modo informacional aqui (o daemon resolve o LLM via seu
// próprio Router); usamos WithProvider("") para "solo" (default do daemon).
func (c *CoscaProvider) Chat(ctx context.Context, model string, messages []Message) (string, error) {
	prompt := flattenMessages(messages)
	res, err := c.client.Orchestration.Run(ctx, prompt, cosca.WithProvider(""))
	if err != nil {
		return "", fmt.Errorf("cosca: %w", err)
	}
	if res == nil {
		return "", fmt.Errorf("cosca: resposta vazia do /v1/run")
	}
	return res.Response, nil
}

// ---------------------------------------------------------------------------
// Flatten []Message -> prompt texto
// ---------------------------------------------------------------------------

// coscaModes é a lista ESTÁVEL de "modelos" (modos de operação) do provider
// cosca. O daemon é Kernel-First (ADR-032) e não tem um catálogo fixo de chat
// models no /v1/run — a deliberação resolve o request. Cada entrada representa
// um MODO de operação, não um checkpoint de pesos.
var coscaModes = []string{
	"cosca/kernel", // Kernel-First: delibera; resolve sem LLM ou escala para o LLM do router do daemon (default).
	"cosca/auto",   // Automático: o kernel decide o caminho.
}

// DefaultCoscaModel devolve o modo/modelo padrão do provider cosca.
func DefaultCoscaModel() string { return coscaModes[0] }

// stableCoscaModels transforma coscaModes em []ModelInfo.
func stableCoscaModels() []ModelInfo {
	out := make([]ModelInfo, 0, len(coscaModes))
	for _, m := range coscaModes {
		out = append(out, ModelInfo{ID: m, Name: m})
	}
	return out
}

// flattenMessages converte um chat (`[]Message`) em um prompt texto para o
// contrato `prompt string` do /v1/run. Preserva os roles como marcadores
// ([system]/[user]/[assistant]) para não perder o turno.
//
// DECISÃO (documentada): o /v1/run é consoante com o pipeline Kernel-First e
// recebe apenas `prompt`. Flattenar um chat multi-turn para texto é LOSSY
// (não carrega tool_calls/refusals/structured metadata). Para o caso comum do
// IDE (system + user, ou system + user/assistant intercalados) o resultado é
// fiel o bastante. Se o contexto multi-turn rico for crítico, o caminho certo
// é o daemon expor `/v1/run` aceitando `messages[]` (GAP futuro) — aí este
// flatten vira pass-through.
func flattenMessages(messages []Message) string {
	var b strings.Builder
	for i, m := range messages {
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		switch m.Role {
		case "system":
			b.WriteString("[system]\n" + strings.TrimSpace(m.Content))
		case "assistant":
			b.WriteString("[assistant]\n" + strings.TrimSpace(m.Content))
		default: // user / qualquer outro
			b.WriteString("[user]\n" + strings.TrimSpace(m.Content))
		}
	}
	return strings.TrimSpace(b.String())
}

// runtimeAddr normaliza uma base URL ("http://host:port", "https://host:port"
// ou "host:port") para o formato "host:port" esperado pelo cosca.ClientConfig.
func runtimeAddr(baseURL string) string {
	s := strings.TrimSpace(baseURL)
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "https://")
	return strings.TrimRight(s, "/")
}
