// Package diffexec implementa o executor de GERAÇÃO DE IMAGEM por IA (§7:
// PROMPT → MODEL → IMAGE) via diffusion (SD 1.5 + ROCm na RX 6700 XT).
//
// O Go orquestra, o Python processa (divisão de trabalho L211): o executor
// chama generate.py (venv-media) como subprocesso com a GPU ROCm. Integrado
// ao Node Graph com cache por assinatura (§23) — o mesmo prompt não
// re-renderiza.
package diffexec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// PythonBin é o interpretador do venv com torch+diffusers.
const PythonBin = "/home/cosca/.cosca/venv-media/bin/python"

// ScriptPath é o script de geração.
const ScriptPath = "internal/diffexec/generate.py"

// Executor gera imagens por diffusion. Implementa o executor do node graph.
type Executor struct {
	// WorkDir para as imagens geradas.
	WorkDir string
}

// New cria um executor de diffusion.
func New(workDir string) *Executor {
	return &Executor{WorkDir: workDir}
}

// ErrUnsupportedNode é retornado para nós que o executor não conhece.
var ErrUnsupportedNode = fmt.Errorf("diffexec: nó de diffusion não suportado")

// Run executa um nó de geração de imagem. Suporta:
//
//	image_generation: PROMPT → IMAGE (§7). Params: prompt, steps, guidance.
func (e *Executor) Run(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	switch node.Type {
	case "image_generation":
		return e.generate(ctx, node, input)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedNode, node.Type)
	}
}

// paramString devolve um param de string ou default.
func paramString(node *engine.Node, key, def string) string {
	if node.Params == nil {
		return def
	}
	if v, ok := node.Params[key].(string); ok && v != "" {
		return v
	}
	return def
}

// paramFloat devolve um param numérico ou default.
func paramFloat(node *engine.Node, key string, def float64) float64 {
	if node.Params == nil {
		return def
	}
	if v, ok := node.Params[key].(float64); ok {
		return v
	}
	return def
}

// generate: PROMPT → IMAGE (§7). Chama o generate.py com a GPU ROCm.
func (e *Executor) generate(ctx context.Context, node *engine.Node, _ map[string]any) (any, error) {
	prompt := paramString(node, "prompt", "uma paisagem serena")
	steps := int(paramFloat(node, "steps", 25))
	guidance := paramFloat(node, "guidance", 7.5)
	if steps < 1 {
		steps = 25
	}

	_ = os.MkdirAll(e.WorkDir, 0o755)
	out := filepath.Join(e.WorkDir, node.ID+".png")

	// Localiza o script relativo ao repo (ou via GOPATH).
	script := resolveScriptPath()

	cmd := exec.CommandContext(ctx, PythonBin, script, out, prompt,
		fmt.Sprintf("%d", steps), fmt.Sprintf("%.1f", guidance))
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		detail := stderr
		if detail == "" {
			detail = strings.TrimSpace(string(output))
		}
		return nil, fmt.Errorf("diffusion: %w: %s", err, truncate(detail, 300))
	}

	// Extrai o JSON do script (a linha com {"ok":true,...}).
	var result map[string]any
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") {
			if err := json.Unmarshal([]byte(line), &result); err == nil {
				break
			}
		}
	}
	if result == nil {
		result = map[string]any{"ok": true, "path": out}
	}
	result["prompt"] = prompt
	result["steps"] = steps
	// O contrato do node graph: nós PRODUTORES devolvem o CAMINHO (string)
	// para que nós seguintes (remove_bg, resize, convert) possam consumir
	// como input. O objeto detalhado fica acessível via result["meta"].
	if path, ok := result["path"].(string); ok && path != "" {
		result["meta"] = result
		return path, nil
	}
	return result, nil
}

// resolveScriptPath localiza o generate.py (relativo ao cwd do cosca-code).
func resolveScriptPath() string {
	// Tenta caminho relativo ao diretório de trabalho (cosca-code).
	for _, cand := range []string{
		"internal/diffexec/generate.py",
		"/home/cosca/Documents/projects/cosca-code/internal/diffexec/generate.py",
	} {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return ScriptPath
}

// truncate limita o tamanho de uma string (para erros legíveis).
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
