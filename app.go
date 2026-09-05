// App é a ponte entre o core Go e a UI desktop (Wails). Expõe o essencial via
// binding e inicia o Workspace Engine + servidor HTTP no startup. A UI continua
// sendo cliente fino — toda a lógica crítica vive no core (spec seção 70).
package main

import (
	"context"
	"log"
	"os"

	"github.com/CoscaAI/cosca-code/internal/event"
	"github.com/CoscaAI/cosca-code/internal/server"
	"github.com/CoscaAI/cosca-code/internal/workspace"
)

// App é a struct exposta ao frontend via binding Wails.
type App struct {
	ctx  context.Context
	bus  *event.Bus
	ws   *workspace.Workspace
	port string
}

// NewApp cria o app.
func NewApp() *App { return &App{} }

// startup roda quando a janela abre: sobe o Event Bus, abre o workspace
// (cwd) e inicia o servidor HTTP que o frontend consome.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.bus = event.NewBus()

	root, err := os.Getwd()
	if err != nil {
		root = "."
	}

	ws, err := workspace.Open(root, a.bus)
	if err != nil {
		log.Printf("workspace: %v", err)
		return
	}
	a.ws = ws

	// Porta dinâmica: se a preferida (14126) estiver ocupada — ex.: outro app
	// da família rodando — escolhe uma livre e o frontend pergunta via
	// GetPort(). O editor nunca fica morto por conflito de porta.
	a.port = resolveFreePort("14126")
	go func() {
		if err := server.New(ws, a.bus).Start("127.0.0.1:" + a.port); err != nil {
			log.Printf("server: %v", err)
		}
	}()
	log.Printf("COSCA CODE desktop — workspace: %s (API na porta %s)", root, a.port)
}

// shutdown roda quando a janela fecha.
func (a *App) shutdown(ctx context.Context) {
	if a.ws != nil {
		a.ws.Close()
	}
}

// GetPort expõe a porta real da API HTTP ao frontend (binding). Usado para o
// frontend saber onde fazer fetch quando rodando no Wails (fora do proxy dev).
func (a *App) GetPort() string {
	if a.port == "" {
		return "14126"
	}
	return a.port
}

// GetRoot expõe o root do workspace ao frontend.
func (a *App) GetRoot() string {
	if a.ws != nil {
		return a.ws.Root()
	}
	return ""
}
