// Package sciexec implementa o executor da Fase 8 (COSCA SCIENTIFIC): executa
// os nós do node graph científico (§11: DOCUMENT + DATA + CODE + SIMULATION +
// VISUALIZATION + MODEL) via a Scientific Engine (§11/§12) e o COSCA LAB.
//
// Nós suportados: run_experiment (calcula/registra), lab_best (comparação).
// Integrado ao Node Graph com cache por assinatura (§23).
package sciexec

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/CoscaAI/cosca/pkg/engine"
)

// Executor executa nós científicos via a Scientific Engine.
type Executor struct {
	// Root do projeto (para o registry de experimentos).
	Root string
}

// New cria um executor científico.
func New(root string) *Executor {
	return &Executor{Root: root}
}

// ErrUnsupportedNode é retornado para nós que o executor não conhece.
var ErrUnsupportedNode = fmt.Errorf("sciexec: nó científico não suportado")

// Run executa um nó científico.
func (e *Executor) Run(ctx context.Context, node *engine.Node, input map[string]any) (any, error) {
	switch node.Type {
	case "run_experiment":
		return e.runExperiment(node)
	case "lab_best":
		return e.labBest(node)
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

// runExperiment: calcula um resultado determinístico e REGISTRA o experimento
// (§11/§12). Suporta expressões de simulação simples via params.
// Ex.: params {expr: "monte_carlo", samples: 1000} → média de amostras.
// Ex.: params {expr: "constant", value: 0.95} → valor fixo (calculated).
func (e *Executor) runExperiment(node *engine.Node) (any, error) {
	id := paramString(node, "id", "exp-"+node.ID)
	name := paramString(node, "name", id)
	kindStr := paramString(node, "kind", "calculated")
	kind := engine.ResultKind(kindStr)

	expr := paramString(node, "expr", "constant")
	var result float64
	switch expr {
	case "constant":
		result = paramFloat(node, "value", 0)
	case "monte_carlo":
		// Simulação determinística com seed: média de N amostras seno.
		samples := int(paramFloat(node, "samples", 100))
		seed := paramFloat(node, "seed", 42)
		if samples <= 0 {
			samples = 100
		}
		var sum float64
		for i := 0; i < samples; i++ {
			sum += math.Abs(math.Sin(seed + float64(i)))
		}
		result = sum / float64(samples)
		kind = engine.SciSimulated
	default:
		return nil, fmt.Errorf("expressão desconhecida %q (constant, monte_carlo)", expr)
	}

	exp, err := engine.NewExperiment(id, name, kind)
	if err != nil {
		return nil, err
	}
	exp.Result = result
	exp.SetMetric("result", result)
	exp.Parameters["expr"] = expr
	if v, ok := node.Params["seed"]; ok {
		exp.Parameters["seed"] = v
	}

	// Persiste no registry do projeto (COSCA LAB §12).
	reg, err := engine.OpenExperiments(e.Root)
	if err != nil {
		return nil, err
	}
	if _, exists := reg.Get(id); exists {
		return nil, fmt.Errorf("experimento %q já existe", id)
	}
	if err := engine.ExperimentAdd(reg, exp); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":        exp.ID,
		"kind":      string(exp.Kind),
		"result":    result,
		"expr":      expr,
		"timestamp": exp.Timestamp,
	}, nil
}

// labBest: compara experimentos (melhor métrica — A/B/benchmark §12).
func (e *Executor) labBest(node *engine.Node) (any, error) {
	metric := paramString(node, "metric", "result")
	reg, err := engine.OpenExperiments(e.Root)
	if err != nil {
		return nil, err
	}
	best, ok := engine.ExperimentBest(reg, metric)
	if !ok {
		return map[string]any{"found": false, "metric": metric, "count": reg.Count()}, nil
	}
	return map[string]any{
		"found":  true,
		"metric": metric,
		"id":     best.ID,
		"name":   best.Name,
		"value":  best.Metric(metric),
		"count":  reg.Count(),
	}, nil
}

var _ = os.Stat // reservado
var _ = filepath.Join // reservado
