// Package cinexec implementa o executor de VÍDEO da Fase 4 (COSCA CINEMA):
// executa os nós do node graph cinematográfico (§8 pipeline:
// SCRIPT → STORYBOARD → SHOT LIST → ASSETS → SCENES → TIMELINE → AUDIO →
// COLOR → VFX → RENDER) via a Media Engine (ffmpeg/ffprobe — Fase 1.8).
//
// Nós suportados: load_video, probe, extract_audio, transcode, frame.
// Integrado ao Node Graph com cache por assinatura (§23).
package cinexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// Executor executa nós de vídeo via a Media Engine. Implementa o contrato de
// executor do node graph (Fase 1.6).
type Executor struct {
	// WorkDir é o diretório de artefatos intermediários.
	WorkDir string
}

// New cria um executor de cinema.
func New(workDir string) *Executor {
	return &Executor{WorkDir: workDir}
}

// ErrUnsupportedNode é retornado para nós que o executor de vídeo não conhece.
var ErrUnsupportedNode = fmt.Errorf("cinexec: nó de vídeo não suportado")

// Run executa um nó de vídeo. Cada tipo produz um caminho de saída (ou texto).
func (e *Executor) Run(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	switch node.Type {
	case "load_video":
		return e.loadVideo(node)
	case "probe":
		return e.probe(ctx, node, input)
	case "extract_audio":
		return e.extractAudio(ctx, node, input)
	case "transcode":
		return e.transcode(ctx, node, input)
	case "frame":
		return e.frame(ctx, node, input)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedNode, node.Type)
	}
}

// hasFFmpeg reporta se ffmpeg está disponível (para mensagens claras).
func hasFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// firstInput resolve o caminho do primeiro input (ou param path p/ entrada).
func (e *Executor) firstInput(node *engine.Node, input map[string]any) (string, error) {
	if len(node.Inputs) == 0 {
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

// outPath devolve um caminho de saída no workdir.
func (e *Executor) outPath(node *engine.Node, ext string) string {
	_ = os.MkdirAll(e.WorkDir, 0o755)
	return filepath.Join(e.WorkDir, node.ID+"."+ext)
}

// loadVideo: nó de ENTRADA — valida o arquivo de vídeo e devolve o caminho.
func (e *Executor) loadVideo(node *engine.Node) (any, error) {
	path, ok := node.Params["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("nó load_video requer param path")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("nó load_video: %w", err)
	}
	return path, nil
}

// probe: analisa o vídeo (ffprobe — codecs, streams, duração §16).
func (e *Executor) probe(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	info, err := engine.ProbeMedia(ctx, in)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"format":   info.Format.Name,
		"duration": info.DurationSeconds(),
		"streams":  len(info.Streams),
		"has_video": info.HasVideo,
		"has_audio": info.HasAudio,
	}, nil
}

// extractAudio: VIDEO → EXTRACT_AUDIO (§21). Params: out (wav/mp3/aac).
func (e *Executor) extractAudio(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	ext := paramString(node, "format", "wav")
	out := e.outPath(node, ext)
	if err := engine.MediaExtractAudio(ctx, in, out, engine.MediaPipeOptions{}); err != nil {
		return nil, err
	}
	return out, nil
}

// transcode: re-encoda (crf por qualidade §22). Params: quality, format.
func (e *Executor) transcode(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	quality := paramString(node, "quality", "draft")
	format := paramString(node, "format", "mkv")
	out := e.outPath(node, format)
	if err := engine.MediaTranscode(ctx, in, out, engine.MediaPipeOptions{Quality: quality}); err != nil {
		return nil, err
	}
	return out, nil
}

// frame: extrai um still em t (thumbnail/storyboard §8). Params: time.
func (e *Executor) frame(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	t := paramString(node, "time", "00:00:01")
	out := e.outPath(node, "png")
	if err := engine.MediaExtractFrame(ctx, in, out, t, engine.MediaPipeOptions{}); err != nil {
		return nil, err
	}
	return out, nil
}

var _ = strings.TrimSpace // reservado
