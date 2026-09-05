// Package musexec implementa o executor de ÁUDIO da Fase 5 (COSCA MUSIC):
// executa os nós do node graph musical (§9 — DAW básica: AUDIO/MIDI/TRACKS/
// INSTRUMENTS/EFFECTS/MIXER/MASTER/SAMPLES) via a Media Engine (ffmpeg).
//
// Nós suportados: load_audio, probe, convert, volume, extract_from_video.
// Integrado ao Node Graph com cache por assinatura (§23).
package musexec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// Executor executa nós de áudio via a Media Engine. Implementa o contrato de
// executor do node graph (Fase 1.6).
type Executor struct {
	// WorkDir é o diretório de artefatos intermediários.
	WorkDir string
}

// New cria um executor de música.
func New(workDir string) *Executor {
	return &Executor{WorkDir: workDir}
}

// ErrUnsupportedNode é retornado para nós que o executor de áudio não conhece.
var ErrUnsupportedNode = fmt.Errorf("musexec: nó de áudio não suportado")

// Run executa um nó de áudio. Cada tipo produz um caminho (ou texto).
func (e *Executor) Run(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	switch node.Type {
	case "load_audio":
		return e.loadAudio(node)
	case "probe_audio":
		return e.probeAudio(ctx, node, input)
	case "convert_audio":
		return e.convertAudio(ctx, node, input)
	case "volume":
		return e.volume(ctx, node, input)
	case "extract_from_video":
		return e.extractFromVideo(ctx, node, input)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedNode, node.Type)
	}
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

// loadAudio: nó de ENTRADA — valida o arquivo de áudio e devolve o caminho.
func (e *Executor) loadAudio(node *engine.Node) (any, error) {
	path, ok := node.Params["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("nó load_audio requer param path")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("nó load_audio: %w", err)
	}
	return path, nil
}

// probeAudio: analisa o áudio (ffprobe — codec, canais, sample rate §16).
func (e *Executor) probeAudio(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	info, err := engine.ProbeMedia(ctx, in)
	if err != nil {
		return nil, err
	}
	as := info.AudioStream()
	res := map[string]any{
		"format":    info.Format.Name,
		"duration":  info.DurationSeconds(),
		"has_audio": info.HasAudio,
	}
	if as != nil {
		res["codec"] = as.CodecName
		res["channels"] = as.Channels
		res["sample_rate"] = as.SampleRate
	}
	return res, nil
}

// convertAudio: converte formato/codec (§9 — exportação de trilha).
// Params: format (mp3/wav/flac/ogg), sample_rate, channels.
func (e *Executor) convertAudio(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	format := paramString(node, "format", "mp3")
	out := e.outPath(node, format)
	opts := engine.AudioPipeOptions{
		SampleRate: paramString(node, "sample_rate", ""),
		Channels:   paramString(node, "channels", ""),
	}
	if err := engine.MediaConvertAudio(ctx, in, out, opts); err != nil {
		return nil, err
	}
	return out, nil
}

// volume: ajusta o volume (db ou fator) — MIXER/MASTER básico (§9).
// Params: volume (ex.: "3dB" ou "0.5").
func (e *Executor) volume(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	vol := paramString(node, "volume", "0dB")
	if vol == "" {
		vol = "0dB"
	}
	ext := strings.TrimPrefix(filepath.Ext(in), ".")
	if ext == "" {
		ext = "wav"
	}
	out := e.outPath(node, ext)
	opts := engine.AudioPipeOptions{Volume: vol}
	if err := engine.MediaConvertAudio(ctx, in, out, opts); err != nil {
		return nil, err
	}
	return out, nil
}

// extractFromVideo: extrai a trilha de áudio de um vídeo (§9 samples).
// Params: format (wav/mp3/aac).
func (e *Executor) extractFromVideo(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	in, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	format := paramString(node, "format", "wav")
	out := e.outPath(node, format)
	if err := engine.MediaExtractAudio(ctx, in, out, engine.MediaPipeOptions{}); err != nil {
		return nil, err
	}
	return out, nil
}
