// Package gameexec implementa o executor da Fase 7 (COSCA GAME): executa os
// nós do node graph de jogo (§10: ENGINE → PROJECT → SCENE → ENTITY →
// COMPONENT → SYSTEM) via a Game Engine (ECS em Go puro).
//
// Nós suportados: load_scene, scene_info, add_entity. Integrado ao Node
// Graph com cache por assinatura (§23).
package gameexec

import (
	"context"
	"fmt"
	"os"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// Executor executa nós de jogo via a Game Engine. Implementa o contrato de
// executor do node graph (Fase 1.6).
type Executor struct {
	// WorkDir é o diretório de artefatos.
	WorkDir string
}

// New cria um executor de jogo.
func New(workDir string) *Executor {
	return &Executor{WorkDir: workDir}
}

// ErrUnsupportedNode é retornado para nós que o executor de jogo não conhece.
var ErrUnsupportedNode = fmt.Errorf("gameexec: nó de jogo não suportado")

// Run executa um nó de jogo.
func (e *Executor) Run(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	switch node.Type {
	case "load_scene":
		return e.loadScene(node)
	case "scene_info":
		return e.sceneInfo(ctx, node, input)
	case "add_entity":
		return e.addEntity(ctx, node, input)
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

// loadScene: nó de ENTRADA — valida o arquivo de cena JSON e devolve o caminho.
func (e *Executor) loadScene(node *engine.Node) (any, error) {
	path, ok := node.Params["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("nó load_scene requer param path")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("nó load_scene: %w", err)
	}
	return path, nil
}

// sceneInfo: carrega e resume a cena (entidades/componentes §10).
func (e *Executor) sceneInfo(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	path, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("scene_info: %w", err)
	}
	scene, err := engine.UnmarshalGameScene(data)
	if err != nil {
		return nil, err
	}
	// Conta componentes por tipo.
	compCount := map[string]int{}
	for _, e := range scene.SortEntities() {
		for _, c := range e.Components {
			compCount[string(c.Type)]++
		}
	}
	return map[string]any{
		"name":        scene.Name,
		"entities":    scene.EntityCount(),
		"components":  compCount,
		"players":     len(engine.GameSceneEntitiesWith(scene, engine.GameInput)),
		"enemies":     len(engine.GameSceneEntitiesWith(scene, engine.GameAI)),
	}, nil
}

// addEntity: adiciona uma entidade à cena (param entity JSON) — protótipo.
func (e *Executor) addEntity(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	path, err := e.firstInput(node, input)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("add_entity: %w", err)
	}
	scene, err := engine.UnmarshalGameScene(data)
	if err != nil {
		return nil, err
	}
	// Constrói a entidade a partir dos params (id, name, components).
	id := paramString(node, "id", "entity_" + fmt.Sprint(scene.EntityCount()+1))
	ent := &engine.GameEntity{ID: id, Name: paramString(node, "name", id)}
	components := node.Params["components"]
	if comps, ok := components.([]any); ok {
		for _, raw := range comps {
			if m, ok := raw.(map[string]any); ok {
				t := engine.GameComponentType(fmt.Sprint(m["type"]))
				c := engine.GameComponent{Type: t}
				if p, ok := m["params"].(map[string]any); ok {
					c.Params = p
				}
				ent.Components = append(ent.Components, c)
			}
		}
	}
	if err := engine.GameSceneAddEntity(scene, ent); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":       ent.ID,
		"name":     ent.Name,
		"added":    true,
		"entities": scene.EntityCount(),
	}, nil
}
