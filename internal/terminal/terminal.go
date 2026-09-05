// Package terminal é o Terminal Engine do COSCA CODE (spec seção 31): um
// pseudo-terminal (PTY) com um shell. Suporta escrita, leitura incremental e
// resize. Múltiplos terminais, SSH e containers são extensões futuras sobre
// esta base — o núcleo isola o PTY, a UI é cliente.
package terminal

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// Terminal é um shell rodando em PTY.
type Terminal struct {
	cmd  *exec.Cmd
	pty  *os.File
	done chan struct{}
}

// Start inicia um shell (default: $SHELL ou /bin/bash) em PTY com dimensões
// cols×rows.
func Start(shell string, cols, rows int) (*Terminal, error) {
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
	}
	cmd := exec.Command(shell)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	t := &Terminal{cmd: cmd, pty: ptmx, done: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		close(t.done)
	}()
	return t, nil
}

// Write envia dados ao shell (stdin do PTY).
func (t *Terminal) Write(data []byte) (int, error) {
	return t.pty.Write(data)
}

// Read lê a saída do shell (bloqueia até haver dados ou o shell morrer).
func (t *Terminal) Read(p []byte) (int, error) {
	return t.pty.Read(p)
}

// Resize ajusta as dimensões do terminal (cols×rows).
func (t *Terminal) Resize(cols, rows int) error {
	return pty.Setsize(t.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Done fecha quando o shell termina.
func (t *Terminal) Done() <-chan struct{} { return t.done }

// Close encerra o shell e fecha o PTY.
func (t *Terminal) Close() {
	_ = t.pty.Close()
	if t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
}

// Stdout devolve o PTY como um io.Writer (para piping).
func (t *Terminal) Stdout() io.Writer { return t.pty }
