// Package doctor implementa o `cosca-code doctor` (mandamento 23 da doutrina):
// verifica a saúde de cada subsistema do COSCA CODE consultando os endpoints
// de LEITURA existentes. SOMENTE LEITURA — não executa, não altera, não envia
// POST. A P14 + a ordem do Don (read-only total) são respeitadas: o doctor
// apenas OBSERVA o backend rodando.
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Check é o resultado de UMA verificação.
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok | warn | fail
	Detail  string `json:"detail,omitempty"`
}

// Run consulta o backend (base, ex: http://127.0.0.1:14126) e reporta a
// saúde de cada subsistema. Retorna os checks e se tudo ok.
func Run(ctx context.Context, base string) ([]Check, bool) {
	c := &http.Client{Timeout: 5 * time.Second}
	var checks []Check
	allOK := true

	get := func(name, path string, parse func([]byte) error) {
		ch := Check{Name: name}
		resp, err := c.Get(base + path)
		if err != nil {
			ch.Status = "fail"
			ch.Detail = "sem conexão"
			allOK = false
			checks = append(checks, ch)
			return
		}
		defer resp.Body.Close()
		// Lê até 1MB — o /api/intel/symbols pode ser grande (todo o projeto);
		// o LimitReader de 4KB truncava o JSON e gerava warning falso.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if resp.StatusCode != http.StatusOK {
			ch.Status = "fail"
			ch.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
			allOK = false
			checks = append(checks, ch)
			return
		}
		if parse != nil {
			if err := parse(body); err != nil {
				ch.Status = "warn"
				ch.Detail = "resposta inesperada: " + err.Error()
				allOK = false
			}
		}
		if ch.Status == "" {
			ch.Status = "ok"
		}
		checks = append(checks, ch)
	}

	// Health — o servidor está no ar.
	get("servidor", "/health", func(b []byte) error {
		var h struct {
			Service string `json:"service"`
		}
		if err := json.Unmarshal(b, &h); err != nil {
			return err
		}
		if h.Service != "cosca-code" {
			return fmt.Errorf("serviço %q", h.Service)
		}
		return nil
	})

	// Git — status e branch.
	get("git", "/api/git/status", func(b []byte) error {
		var g struct {
			Branch string `json:"branch"`
		}
		return json.Unmarshal(b, &g)
	})

	// Modelos de IA — provider e lista.
	get("modelos IA", "/api/ai/models", func(b []byte) error {
		var m struct {
			Provider string `json:"provider"`
			Models   []any  `json:"models"`
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return err
		}
		if len(m.Models) == 0 {
			return fmt.Errorf("nenhum modelo disponível")
		}
		return nil
	})

	// Autonomia — o nível de permissão.
	get("autonomia", "/api/autonomy", func(b []byte) error {
		var a struct {
			Level string `json:"level"`
		}
		return json.Unmarshal(b, &a)
	})

	// Memória — registros aprendidos.
	get("memória", "/api/memory", func(b []byte) error {
		var m struct {
			Entries []any `json:"entries"`
		}
		return json.Unmarshal(b, &m)
	})

	// Skills — capacidades instaladas.
	get("skills", "/api/skills", func(b []byte) error {
		var s struct {
			Skills []any `json:"skills"`
		}
		return json.Unmarshal(b, &s)
	})

	// Símbolos (intel) — índice do projeto.
	get("índice de símbolos", "/api/intel/symbols", func(b []byte) error {
		var s struct {
			Symbols []any `json:"symbols"`
		}
		return json.Unmarshal(b, &s)
	})

	return checks, allOK
}

// Print imprime o relatório do doctor no terminal (formato amigável).
func Print(checks []Check, allOK bool) {
	fmt.Println("── COSCA CODE DOCTOR ──")
	for _, ch := range checks {
		icon := "✅"
		if ch.Status == "warn" {
			icon = "⚠️"
		}
		if ch.Status == "fail" {
			icon = "❌"
		}
		line := fmt.Sprintf("  %s %-18s", icon, ch.Name)
		if ch.Detail != "" {
			line += " — " + ch.Detail
		}
		fmt.Println(line)
	}
	if allOK {
		fmt.Println("  ✔ todos os subsistemas saudáveis")
	} else {
		fmt.Println("  ✖ alguns subsistemas precisam de atenção")
	}
	_ = os.Stdout
}
