package extension

import (
	"context"
	"testing"

	"github.com/CoscaAI/cosca-code/internal/tool"
)

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	for _, e := range Builtin() {
		r.Register(e)
	}
	list := r.List()
	if len(list) < 2 {
		t.Fatalf("List = %d, want >= 2", len(list))
	}
}

func TestApplyRegistersTools(t *testing.T) {
	r := NewRegistry()
	for _, e := range Builtin() {
		r.Register(e)
	}
	tools := tool.NewRegistry()
	r.Apply(tools)

	if _, ok := tools.Get("gofmt"); !ok {
		t.Fatal("tool gofmt não foi aplicada ao registry")
	}
	if _, ok := tools.Get("git_log"); !ok {
		t.Fatal("tool git_log não foi aplicada ao registry")
	}
}

func TestGofmtTool(t *testing.T) {
	for _, e := range Builtin() {
		for _, tl := range e.Tools {
			if tl.Name == "gofmt" {
				out, err := tl.Handler(context.Background(), map[string]any{"code": "package main\nfunc main(){x:=1\n}"})
				if err != nil {
					t.Fatal(err)
				}
				if out == "" {
					t.Fatal("gofmt vazio")
				}
				return
			}
		}
	}
	t.Fatal("tool gofmt não encontrada")
}
