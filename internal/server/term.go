package server

import (
	"context"
	"io"
	"net/http"

	"nhooyr.io/websocket"

	"github.com/CoscaAI/cosca-code/internal/terminal"
)

// handleTerm faz a ponte bidirecional WebSocket ↔ PTY. O cliente envia input
// (bytes) e recebe a saída do shell em streaming. Spawna o shell na conexão.
func (s *Server) handleTerm(w http.ResponseWriter, r *http.Request) {
	term, err := terminal.Start("", 100, 30)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer term.Close()

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusInternalError, "closed")

	ctx := r.Context()
	conn.SetReadLimit(1 << 20)

	// Goroutine: shell → cliente.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := term.Read(buf)
			if n > 0 {
				_ = writeTermWS(ctx, conn, buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					return
				}
			}
		}
	}()

	// Loop: cliente → shell.
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if _, err := term.Write(data); err != nil {
			return
		}
	}
}

// writeTermWS escreve bytes como mensagem binária no WebSocket.
func writeTermWS(ctx context.Context, conn *websocket.Conn, data []byte) error {
	return conn.Write(ctx, websocket.MessageBinary, data)
}
