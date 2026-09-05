// Package xexec implementa o executor CROSS-PRODUCT da Fase 9 (§30 do
// manifesto Creative/Scientific/Media): UM node graph que ORQUESTRA todos os
// produtos. Cada nó é roteado para o executor especializado do seu domínio:
//
//	load_image/remove_bg/convert/resize/ocr → mediaexec (COSCA IMAGE)
//	load_video/probe/extract_audio/frame/transcode → cinexec (COSCA CINEMA)
//	load_audio/convert_audio/volume → musexec (COSCA MUSIC)
//	load_3d/probe_3d → td3dexec (COSCA 3D)
//	load_scene/scene_info/add_entity → gameexec (COSCA GAME)
//	run_experiment/lab_best → sciexec (COSCA SCIENTIFIC)
//
// Os produtos CONVERSAM via os resultados: o a.wav do CINEMA alimenta o
// volume do MUSIC; o frame PNG do CINEMA vira imagem; o modelo 3D vira
// asset do jogo — cross-product (§30), um único grafo, um único cache.
package xexec

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CoscaAI/cosca/pkg/engine"

	"github.com/CoscaAI/cosca-code/internal/cinexec"
	"github.com/CoscaAI/cosca-code/internal/diffexec"
	"github.com/CoscaAI/cosca-code/internal/gameexec"
	"github.com/CoscaAI/cosca-code/internal/mediaexec"
	"github.com/CoscaAI/cosca-code/internal/musexec"
	"github.com/CoscaAI/cosca-code/internal/sciexec"
	"github.com/CoscaAI/cosca-code/internal/td3dexec"
)

// Executor roteia cada nó para o executor especializado do domínio.
type Executor struct {
	// Root do projeto (para sciexec e workdirs).
	Root string
	// Executores por domínio.
	media   *mediaexec.Executor
	diff    *diffexec.Executor
	cinema  *cinexec.Executor
	music   *musexec.Executor
	td3d    *td3dexec.Executor
	game    *gameexec.Executor
	sci     *sciexec.Executor
}

// New cria o orquestrador cross-product.
func New(root string) *Executor {
	return &Executor{
		Root:   root,
		media:  mediaexec.New(filepath.Join(root, ".cosca", "media-work")),
		diff:   diffexec.New(filepath.Join(root, ".cosca", "image-work")),
		cinema: cinexec.New(filepath.Join(root, ".cosca", "cinema-work")),
		music:  musexec.New(filepath.Join(root, ".cosca", "music-work")),
		td3d:   td3dexec.New(filepath.Join(root, ".cosca", "td3d-work")),
		game:   gameexec.New(filepath.Join(root, ".cosca", "game-work")),
		sci:    sciexec.New(root),
	}
}

// Route resolve o executor do nó conforme o seu tipo (§30).
func (e *Executor) Route(node *engine.Node) (engine.Executor, error) {
	switch node.Type {
	case "image_generation":
		return e.diff, nil
	case "load_image", "identify", "remove_bg", "convert", "resize", "ocr", "vectorize":
		return e.media, nil
	case "load_video", "probe", "extract_audio", "transcode", "frame":
		return e.cinema, nil
	case "load_audio", "probe_audio", "convert_audio", "volume", "extract_from_video":
		return e.music, nil
	case "load_3d", "probe_3d":
		return e.td3d, nil
	case "load_scene", "scene_info", "add_entity":
		return e.game, nil
	case "run_experiment", "lab_best":
		return e.sci, nil
	default:
		return nil, fmt.Errorf("xexec: domínio desconhecido para o nó %q (%s)", node.ID, node.Type)
	}
}

// Run executa um nó roteado para o executor do domínio (§30).
func (e *Executor) Run(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	exec, err := e.Route(node)
	if err != nil {
		return nil, err
	}
	return exec.Run(ctx, node, input)
}

// Products devolve os produtos do ecossistema e seus nós (para o painel).
func (e *Executor) Products() []Product {
	return []Product{
		{Name: "COSCA IMAGE", Glyph: "▦", Nodes: []string{"load_image", "remove_bg", "convert", "resize", "ocr"}},
		{Name: "COSCA CINEMA", Glyph: "▶", Nodes: []string{"load_video", "probe", "extract_audio", "frame", "transcode"}},
		{Name: "COSCA MUSIC", Glyph: "♪", Nodes: []string{"load_audio", "probe_audio", "convert_audio", "volume"}},
		{Name: "COSCA 3D", Glyph: "◈", Nodes: []string{"load_3d", "probe_3d"}},
		{Name: "COSCA GAME", Glyph: "▣", Nodes: []string{"load_scene", "scene_info", "add_entity"}},
		{Name: "COSCA SCIENTIFIC", Glyph: "Σ", Nodes: []string{"run_experiment", "lab_best"}},
	}
}

// Product descreve um produto do ecossistema (§30).
type Product struct {
	Name  string   `json:"name"`
	Glyph string   `json:"glyph"`
	Nodes []string `json:"nodes"`
}

// Flows devolve os fluxos cross-product canônicos do §30.
func (e *Executor) Flows() []Flow {
	return []Flow{
		{
			Name: "cinema→audio→music",
			Desc: "ExtractAudio (CINEMA) → Volume (MUSIC) — a trilha do vídeo vira música",
			Nodes: []string{"load_video", "extract_audio", "volume"},
		},
		{
			Name: "cinema→image",
			Desc: "Frame (CINEMA) → Resize (IMAGE) — o still vira asset de imagem",
			Nodes: []string{"load_video", "frame", "resize"},
		},
		{
			Name: "3d→game",
			Desc: "Probe3D (3D) → Scene (GAME) — o modelo vira entidade do jogo",
			Nodes: []string{"load_3d", "probe_3d", "scene_info"},
		},
		{
			Name: "scientific→document",
			Desc: "Experiment (SCI) → LabBest — a métrica vira conhecimento",
			Nodes: []string{"run_experiment", "lab_best"},
		},
	}
}

// Flow descreve um fluxo cross-product (§30).
type Flow struct {
	Name  string   `json:"name"`
	Desc  string   `json:"desc"`
	Nodes []string `json:"nodes"`
}

var _ = strings.TrimSpace // reservado
