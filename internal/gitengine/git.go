// Package gitengine é o Git Engine do COSCA CODE (spec seção 7): integração
// Git de primeira classe. Encapsula as operações sobre o binário git do
// sistema (não reimplementa o Git — abstrai o fluxo: status, diff, log,
// branches). A UI nunca roda `git` diretamente.
package gitengine

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// FileStatus representa o estado de um arquivo no working tree (parse de
// `git status --porcelain`). X é o status no índice (staged); Y no working tree.
type FileStatus struct {
	Path      string `json:"path"`
	X         byte   `json:"x"`
	Y         byte   `json:"y"`
	Staged    bool   `json:"staged"`
	Untracked bool   `json:"untracked"`
}

// Commit é um registro do histórico.
type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// Engine encapsula operações git sobre um root.
type Engine struct {
	root string
}

// Open abre um Engine para o diretório root. Retorna erro se root não for um
// repositório git (ou git não estiver disponível).
func Open(root string) (*Engine, error) {
	e := &Engine{root: root}
	if !e.IsRepo() {
		return nil, fmt.Errorf("%s não é um repositório git", root)
	}
	return e, nil
}

// IsRepo reporta se root está dentro de um repositório git.
func (e *Engine) IsRepo() bool {
	_, err := e.run("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// CurrentBranch devolve o nome do branch atual.
func (e *Engine) CurrentBranch() (string, error) {
	out, err := e.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Status devolve o estado de todos os arquivos alterados/novos/removidos.
func (e *Engine) Status() ([]FileStatus, error) {
	out, err := e.run("status", "--porcelain")
	if err != nil {
		return nil, err
	}
	var result []FileStatus
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		path := strings.TrimSpace(line[3:])
		result = append(result, FileStatus{
			Path:      path,
			X:         x,
			Y:         y,
			Staged:    x != ' ' && x != '?',
			Untracked: x == '?',
		})
	}
	return result, nil
}

// Diff devolve o diff (unstaged) de um arquivo, ou de todo o repo se path == "".
func (e *Engine) Diff(path string) (string, error) {
	args := []string{"diff", "--"}
	if path != "" {
		args = append(args, path)
	}
	out, err := e.run(args...)
	if err != nil {
		return "", err
	}
	return out, nil
}

// DiffStaged devolve o diff do que está staged (index).
func (e *Engine) DiffStaged(path string) (string, error) {
	args := []string{"diff", "--cached", "--"}
	if path != "" {
		args = append(args, path)
	}
	out, err := e.run(args...)
	if err != nil {
		return "", err
	}
	return out, nil
}

// Log devolve os últimos limit commits.
func (e *Engine) Log(limit int) ([]Commit, error) {
	out, err := e.run("log", "--pretty=format:%H%x09%an%x09%ad%x09%s", "--date=short", fmt.Sprintf("-n%d", limit))
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, Commit{Hash: parts[0], Author: parts[1], Date: parts[2], Message: parts[3]})
	}
	return commits, nil
}

// Branches devolve a lista de branches locais.
func (e *Engine) Branches() ([]string, error) {
	out, err := e.run("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, b := range strings.Split(out, "\n") {
		if b = strings.TrimSpace(b); b != "" {
			branches = append(branches, b)
		}
	}
	return branches, nil
}

// run executa o git com os argumentos dados, a partir do root do engine.
func (e *Engine) run(args ...string) (string, error) {
	full := append([]string{"-C", e.root}, args...)
	cmd := exec.Command("git", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
