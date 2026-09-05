// Modo CLI (dev): sobe o Workspace Engine e expõe a API HTTP que o frontend
// Vite consome em desenvolvimento (npm run dev). O modo desktop (Wails) é o
// caminho padrão do main.go.
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/CoscaAI/cosca-code/internal/event"
	"github.com/CoscaAI/cosca-code/internal/server"
	"github.com/CoscaAI/cosca-code/internal/workspace"
)

// runServe inicia o servidor HTTP para o diretório root (default: cwd).
func runServe(root string) {
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}

	bus := event.NewBus()
	ws, err := workspace.Open(abs, bus)
	if err != nil {
		log.Fatalf("abrir workspace: %v", err)
	}
	defer ws.Close()

	preferred := os.Getenv("COSCA_CODE_PORT")
	if preferred == "" {
		preferred = "14126"
	}
	// Porta dinâmica: se a preferida estiver ocupada (ex.: outro app da
	// família na mesma porta), escolhe automaticamente uma porta livre e
	// reporta — o editor nunca fica morto por conflito de porta.
	port := resolveFreePort(preferred)

	info := ws.Info()
	log.Printf("COSCA CODE (serve) — workspace: %s", abs)
	log.Printf("  linguagem: %s · framework: %s · arquivos: %d",
		info.Language, orDash(info.Framework), len(ws.Files()))
	log.Printf("  API em http://127.0.0.1:%s (frontend: npm run dev)", port)

	if err := server.New(ws, bus).Start("127.0.0.1:" + port); err != nil {
		log.Fatal(err)
	}
}

// resolveFreePort devolve a porta preferida se livre; caso contrário, uma
// porta livre qualquer (net.Listen :0). Nunca falha.
func resolveFreePort(preferred string) string {
	// Tenta a preferida.
	if preferred != "" {
		l, err := net.Listen("tcp", "127.0.0.1:"+preferred)
		if err == nil {
			_ = l.Close()
			return preferred
		}
		log.Printf("  porta %s ocupada — procurando porta livre...", preferred)
	}
	// Porta livre aleatória.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		// Último recurso: retorna a preferida e deixa o erro real aparecer.
		return preferred
	}
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
