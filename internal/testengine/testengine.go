// Package testengine é o Test Engine do COSCA CODE (spec seção 23): detecta e
// executa os testes do projeto, parseando o resultado e emitindo eventos no
// Event Bus. Não reimplementa runners — encapsula o comando de teste do
// projeto (go test, cargo test, npm test, pytest...).
package testengine

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/CoscaAI/cosca-code/internal/event"
)

// PackageResult é o resultado de um pacote/suite (parse do go test).
type PackageResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // ok | fail | no-tests
	Time   string `json:"time"`
}

// Result é o resultado completo da execução de testes.
type Result struct {
	Command  string          `json:"command"`
	Passed   bool            `json:"passed"`
	Output   string          `json:"output"`
	Duration string          `json:"duration"`
	Packages []PackageResult `json:"packages,omitempty"`
}

// Run executa o comando de teste no diretório root, com timeout. Emite
// test.started / test.passed / test.failed no bus (se fornecido).
func Run(root, command string, timeout time.Duration, bus *event.Bus) (*Result, error) {
	if command == "" {
		command = "go test ./..."
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	if bus != nil {
		bus.Publish(event.Event{Type: event.TestStarted, Source: "testengine", Payload: command})
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start).Round(time.Millisecond)

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	// "exit status N" também conta como falha, não só err != nil.
	passed := err == nil
	if exitErr, ok := err.(*exec.ExitError); ok {
		passed = exitErr.ExitCode() == 0
	}

	result := &Result{
		Command:  command,
		Passed:   passed,
		Output:   output,
		Duration: duration.String(),
		Packages: parseGoTest(output),
	}

	if bus != nil {
		if passed {
			bus.Publish(event.Event{Type: event.TestPassed, Source: "testengine", Payload: command})
		} else {
			bus.Publish(event.Event{Type: event.TestFailed, Source: "testengine", Payload: command})
		}
	}
	return result, nil
}

// parseGoTest extrai os resultados por pacote do output do `go test`.
var (
	goOkRe   = regexp.MustCompile(`^ok\s+(\S+)\s+(\S+)`)
	goFailRe = regexp.MustCompile(`^FAIL\s+(\S+)\s+(\S+)`)
	goNoRe   = regexp.MustCompile(`^\?\s+(\S+)\s+\[no test files\]`)
)

func parseGoTest(output string) []PackageResult {
	var out []PackageResult
	for _, line := range strings.Split(output, "\n") {
		if m := goOkRe.FindStringSubmatch(line); m != nil {
			out = append(out, PackageResult{Name: m[1], Status: "ok", Time: m[2]})
		} else if m := goFailRe.FindStringSubmatch(line); m != nil {
			out = append(out, PackageResult{Name: m[1], Status: "fail", Time: m[2]})
		} else if m := goNoRe.FindStringSubmatch(line); m != nil {
			out = append(out, PackageResult{Name: m[1], Status: "no-tests"})
		}
	}
	return out
}
