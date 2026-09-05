package provider

import "context"

// RouterLLM é um adapter que expõe o Router como uma interface LLM mínima
// (`Chat(ctx, model, messages)` — a mesma assinatura usada por planning,
// gamedesign e scidesign). Assim esses consumidores roteiam pelo provider
// SELECIONADO (ollama, openai ou **cosca**) em vez de criar um provider
// hardcoded (NewOllama) que ignora a escolha do usuário.
//
// "COSCA solo, fornecendo tudo": quando o default é "cosca", o Chat é
// injetado no /v1/run do cérebro; senão, cai para o provider ativo. Se `model`
// vier vazio, o Router.Resolve usa o default (provider + modelo ativos do
// SetDefault).
type RouterLLM struct {
	router *Router
}

// NewRouterLLM cria o adapter sobre o router.
func NewRouterLLM(router *Router) *RouterLLM {
	return &RouterLLM{router: router}
}

// Chat roteia a inferência pelo Router (provider default / modelo default).
func (r *RouterLLM) Chat(ctx context.Context, model string, messages []Message) (string, error) {
	return r.router.Chat(ctx, "", model, messages)
}
