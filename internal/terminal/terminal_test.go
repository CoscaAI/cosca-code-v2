package terminal

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestTerminalEcho(t *testing.T) {
	shell := "/bin/bash"
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "/bin/sh"
	}
	term, err := Start(shell, 80, 24)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer term.Close()

	// Escreve um comando echo e aguarda a saída.
	if _, err := term.Write([]byte("echo __COSCA_TEST__\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	deadline := time.After(5 * time.Second)
	var buf strings.Builder
	b := make([]byte, 256)
	for !strings.Contains(buf.String(), "__COSCA_TEST__") {
		select {
		case <-deadline:
			t.Fatalf("timeout; output até agora: %q", buf.String())
		default:
		}
		_ = term.pty.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := term.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil && !os.IsTimeout(err) {
			// EOF ou erro — pode ser shell sem echo; verifica o que juntou.
			break
		}
	}
	if !strings.Contains(buf.String(), "__COSCA_TEST__") {
		t.Fatalf("echo não retornou o marcador; output: %q", buf.String())
	}
}

func TestTerminalResize(t *testing.T) {
	term, err := Start("/bin/bash", 80, 24)
	if err != nil {
		t.Skipf("bash não disponível: %v", err)
	}
	defer term.Close()
	if err := term.Resize(120, 40); err != nil {
		t.Fatalf("Resize: %v", err)
	}
}
