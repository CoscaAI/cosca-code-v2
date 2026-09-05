// Package mediaexec implementa o executor de IMAGEM da Fase 3 (COSCA IMAGE):
// executa os nós do node graph de imagem (§7 pipeline:
// PROMPT → MODEL → IMAGE → SEGMENTATION → EDIT → UPSCALE → COLOR → EXPORT)
// chamando a cosca-media (Python) como processo — a Visual Media Engine da
// família (Stack 18/19, L210/L211).
//
// Os nós suportados são os comandos da cosca-media: remove_bg, convert,
// resize, identify, ocr, svg_optimize, svg_to_png, vectorize. A execução é
// integrada ao Node Graph com cache por assinatura (§23).
package mediaexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// Executor executa nós de imagem via cosca-media. Implementa o contrato de
// executor do node graph (Fase 1.6): Run(ctx, node, inputs).
type Executor struct {
	// MediaBin é o caminho do binário cosca-media (default: $PATH).
	MediaBin string
	// WorkDir é o diretório de trabalho (assets temporários).
	WorkDir string
}

// New cria um executor. workDir é onde os artefatos intermediários vivem.
func New(workDir string) *Executor {
	return &Executor{MediaBin: "cosca-media", WorkDir: workDir}
}

// ErrUnsupportedNode é retornado para nós que o executor de imagem não conhece.
var ErrUnsupportedNode = fmt.Errorf("mediaexec: nó de imagem não suportado")

// Run executa um nó de imagem. O input carrega os caminhos dos assets de
// entrada (inputID → path); cada tipo de nó produz um caminho de saída.
func (e *Executor) Run(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	switch node.Type {
	case "load_image":
		return e.loadImage(node)
	case "identify":
		return e.identify(ctx, node, input)
	case "remove_bg":
		return e.removeBG(ctx, node, input)
	case "convert":
		return e.convert(ctx, node, input)
	case "resize":
		return e.resize(ctx, node, input)
	case "ocr":
		return e.ocr(ctx, node, input)
	case "vectorize":
		return e.vectorize(ctx, node, input)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedNode, node.Type)
	}
}

// loadImage: nó de ENTRADA — valida o arquivo (path) e devolve o caminho.
func (e *Executor) loadImage(node *engine.Node) (any, error) {
	path, ok := node.Params["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("nó load_image requer param path")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("nó load_image: %w", err)
	}
	return path, nil
}

// firstInput resolve o primeiro input do nó para um caminho no disco. Nós de
// ENTRADA (sem inputs) usam o param "path" — ex.: load_image com path=logo.png.
func (e *Executor) firstInput(node *engine.Node, input map[string]any) (string, error) {
	if len(node.Inputs) == 0 {
		// Nó de entrada: caminho vem do param path.
		if p, ok := node.Params["path"].(string); ok && p != "" {
			return p, nil
		}
		return "", fmt.Errorf("nó %q (%s) requer input ou param path", node.ID, node.Type)
	}
	raw, ok := input[node.Inputs[0]]
	if !ok {
		return "", fmt.Errorf("nó %q: input %q não disponível", node.ID, node.Inputs[0])
	}
	path, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("nó %q: input %q não é um caminho", node.ID, node.Inputs[0])
	}
	return path, nil
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

// runMedia executa um comando da cosca-media e captura o erro.
func (e *Executor) runMedia(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, e.MediaBin, args...)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		if stderr != "" {
			return "", fmt.Errorf("cosca-media %s: %w: %s", args[0], err, stderr)
		}
		return "", fmt.Errorf("cosca-media %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}

// outPath devolve um caminho de saída para o nó (no workdir).
func (e *Executor) outPath(node *engine.Node, ext string) string {
	if err := os.MkdirAll(e.WorkDir, 0o755); err == nil {
		// best-effort
	}
	return filepath.Join(e.WorkDir, node.ID+"."+ext)
}

// removeBG: IMAGE → SEGMENT → REMOVE_BG (§7). Params: model (u2net), feather.
func (e *Executor) removeBG(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	model := paramString(node, "model", "u2net")
	out := e.outPath(node, "png")
	args := []string{"remove-bg", in, out, "--model", model}
	if f, ok := node.Params["feather"].(float64); ok && f > 0 {
		args = append(args, "--feather", fmt.Sprintf("%.0f", f))
	}
	if _, err := e.runMedia(ctx, args...); err != nil {
		return nil, err
	}
	return out, nil
}

// convert: IMAGE → FILTER → CONVERT (formato-aware). Params: format/quality.
func (e *Executor) convert(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	format := paramString(node, "format", "webp")
	quality := "90"
	if q, ok := node.Params["quality"].(float64); ok && q > 0 {
		quality = fmt.Sprintf("%.0f", q)
	}
	out := e.outPath(node, format)
	if _, err := e.runMedia(ctx, "convert", in, out, "--quality", quality); err != nil {
		return nil, err
	}
	return out, nil
}

// resize: IMAGE → RESIZE (LANCZOS). Params: width/height.
func (e *Executor) resize(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	w := "512"
	h := "512"
	if v, ok := node.Params["width"].(float64); ok && v > 0 {
		w = fmt.Sprintf("%.0f", v)
	}
	if v, ok := node.Params["height"].(float64); ok && v > 0 {
		h = fmt.Sprintf("%.0f", v)
	}
	ext := strings.TrimPrefix(filepath.Ext(in), ".")
	if ext == "" {
		ext = "png"
	}
	out := e.outPath(node, ext)
	if _, err := e.runMedia(ctx, "resize", in, out, w, h); err != nil {
		return nil, err
	}
	return out, nil
}

// identify: detecta formato pelo conteúdo (MAGIC — doutrina Stack 18).
func (e *Executor) identify(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	out, err := e.runMedia(ctx, "identify", in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ocr: IMAGE → OCR → TEXT. Params: lang (por).
func (e *Executor) ocr(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	lang := paramString(node, "lang", "por")
	out, err := e.runMedia(ctx, "ocr", in, "--lang", lang)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// vectorize: IMAGE → VECTOR (raster→vetor). Params: threshold.
func (e *Executor) vectorize(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	out := e.outPath(node, "svg")
	args := []string{"vectorize", in, out}
	if t, ok := node.Params["threshold"].(float64); ok && t > 0 {
		args = append(args, "--threshold", fmt.Sprintf("%.0f", t))
	}
	if _, err := e.runMedia(ctx, args...); err != nil {
		return nil, err
	}
	return out, nil
}
