// Package td3dexec implementa o executor 3D da Fase 6 (COSCA 3D): executa os
// nós do node graph 3D (§13: MODELING → MATERIALS → LIGHTING → CAMERAS →
// SCENE → RENDERING) via a 3D Engine (parsers puros Go — OBJ/glTF).
//
// Nós suportados: load_3d, probe_3d. Integrado ao Node Graph com cache por
// assinatura (§23).
package td3dexec

import (
	"context"
	"fmt"
	"os"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// Executor executa nós 3D via a 3D Engine. Implementa o contrato de executor
// do node graph (Fase 1.6).
type Executor struct {
	// WorkDir é o diretório de artefatos.
	WorkDir string
}

// New cria um executor 3D.
func New(workDir string) *Executor {
	return &Executor{WorkDir: workDir}
}

// ErrUnsupportedNode é retornado para nós que o executor 3D não conhece.
var ErrUnsupportedNode = fmt.Errorf("td3dexec: nó 3D não suportado")

// Run executa um nó 3D.
func (e *Executor) Run(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	switch node.Type {
	case "load_3d":
		return e.load3D(node)
	case "probe_3d":
		return e.probe3D(node, input)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedNode, node.Type)
	}
}

// firstInput resolve o caminho (input ou param path).
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

// load3D: nó de ENTRADA — valida o arquivo 3D e devolve o caminho.
func (e *Executor) load3D(node *engine.Node) (any, error) {
	path, ok := node.Params["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("nó load_3d requer param path")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("nó load_3d: %w", err)
	}
	return path, nil
}

// probe3D: inspeciona o modelo (OBJ/glTF) — vértices, faces, materiais §13.
func (e *Executor) probe3D(node *engine.Node, input map[string]any) (any, error) {
	path, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	format := engine.DetectTDFormat(path)
	switch format {
	case engine.TDObject:
		info, err := engine.ParseOBJ(path)
		if err != nil {
			return nil, err
		}
		return toMap(info), nil
	case engine.TDGLTF, engine.TDGLB:
		info, err := engine.ParseGLTF(path)
		if err != nil {
			return nil, err
		}
		return toMap(info), nil
	default:
		return nil, fmt.Errorf("formato 3D não suportado: %s (suportados: obj, gltf, glb)", format)
	}
}

func toMap(info *engine.MeshInfo) map[string]any {
	return map[string]any{
		"format":    string(info.Format),
		"vertices":  info.Vertices,
		"faces":     info.Faces,
		"normals":   info.Normals,
		"materials": info.Materials,
		"supported": info.Supported,
		"warnings":  info.Warnings,
	}
}
