package sciexec

import (
	"context"
	"testing"

	"github.com/CoscaAI/cosca/pkg/engine"
)

func TestRunExperimentConstant(t *testing.T) {
	root := t.TempDir()
	e := New(root)
	node := &engine.Node{
		ID: "exp1", Type: "run_experiment",
		Params: map[string]any{
			"id": "exp-001", "name": "baseline", "kind": "calculated",
			"expr": "constant", "value": 0.95,
		},
	}
	out, err := e.Run(context.Background(), node, nil)
	if err != nil {
		t.Fatal(err)
	}
	info := out.(map[string]any)
	if info["result"] != 0.95 || info["kind"] != "calculated" {
		t.Fatalf("constant experiment: %+v", info)
	}
}

func TestRunExperimentMonteCarlo(t *testing.T) {
	root := t.TempDir()
	e := New(root)
	node := &engine.Node{
		ID: "exp2", Type: "run_experiment",
		Params: map[string]any{
			"id": "exp-002", "name": "sim", "kind": "simulated",
			"expr": "monte_carlo", "samples": 1000, "seed": 42,
		},
	}
	out, err := e.Run(context.Background(), node, nil)
	if err != nil {
		t.Fatal(err)
	}
	info := out.(map[string]any)
	if info["kind"] != "simulated" {
		t.Fatalf("monte carlo kind = %v, want simulated", info["kind"])
	}
	val, ok := info["result"].(float64)
	if !ok || val <= 0 || val > 1 {
		t.Fatalf("monte carlo result = %v (esperado 0<r<=1)", info["result"])
	}
}

func TestRunExperimentDuplicate(t *testing.T) {
	root := t.TempDir()
	e := New(root)
	node := &engine.Node{
		ID: "a", Type: "run_experiment",
		Params: map[string]any{"id": "dup", "name": "x", "expr": "constant", "value": 0.5},
	}
	if _, err := e.Run(context.Background(), node, nil); err != nil {
		t.Fatal(err)
	}
	// Mesmo ID de novo → erro.
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected duplicate experiment error")
	}
}

func TestLabBest(t *testing.T) {
	root := t.TempDir()
	e := New(root)
	// Cria dois experimentos: 0.7 e 0.92.
	n1 := &engine.Node{ID: "a", Type: "run_experiment", Params: map[string]any{"id": "e-a", "name": "a", "expr": "constant", "value": 0.7}}
	n2 := &engine.Node{ID: "b", Type: "run_experiment", Params: map[string]any{"id": "e-b", "name": "b", "expr": "constant", "value": 0.92}}
	if _, err := e.Run(context.Background(), n1, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Run(context.Background(), n2, nil); err != nil {
		t.Fatal(err)
	}

	best := &engine.Node{ID: "best", Type: "lab_best", Params: map[string]any{"metric": "result"}}
	out, err := e.Run(context.Background(), best, nil)
	if err != nil {
		t.Fatal(err)
	}
	info := out.(map[string]any)
	if info["found"] != true || info["id"] != "e-b" {
		t.Fatalf("lab_best: %+v", info)
	}
}

func TestUnsupportedNode(t *testing.T) {
	root := t.TempDir()
	e := New(root)
	node := &engine.Node{ID: "x", Type: "quantum_sim"}
	if _, err := e.Run(context.Background(), node, nil); err == nil {
		t.Fatal("expected unsupported node error")
	}
}
