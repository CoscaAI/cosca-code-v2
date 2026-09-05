// Package benchmark mede a performance do COSCA CODE em grandes repositórios
// (spec seção 68/99): gera um repo sintético com N arquivos e mede o tempo de
// árvore, busca e indexação. É a prova de escala da Fase 7 — não adivinha,
// mede.
package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CoscaAI/cosca-code/internal/intel"
	"github.com/CoscaAI/cosca-code/internal/search"
	"github.com/CoscaAI/cosca-code/internal/workspace"
)

// Result é o resultado do benchmark.
type Result struct {
	Files    int   `json:"files"`
	TreeMs   int64 `json:"tree_ms"`
	SearchMs int64 `json:"search_ms"`
	IndexMs  int64 `json:"index_ms"`
}

// Run gera um repo sintético com nFiles arquivos Go (distribuídos em pacotes)
// e mede o tempo de cada operação. Limpa o diretório temporário ao final.
func Run(nFiles int) (*Result, error) {
	if nFiles <= 0 {
		nFiles = 1000
	}

	root, err := os.MkdirTemp("", "cosca-bench-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)

	// Gera nFiles arquivos Go em pacotes de ~50 arquivos.
	genStart := time.Now()
	perPkg := 50
	for i := 0; i < nFiles; i++ {
		pkg := i / perPkg
		dir := filepath.Join(root, fmt.Sprintf("pkg%04d", pkg))
		_ = os.MkdirAll(dir, 0o755)
		name := filepath.Join(dir, fmt.Sprintf("file%04d.go", i))
		content := fmt.Sprintf("package pkg%04d\n\nimport \"fmt\"\n\nfunc fn%d() int {\n\tfmt.Println(%d)\n\treturn %d\n}\n", pkg, i, i, i)
		_ = os.WriteFile(name, []byte(content), 0o644)
	}
	_ = genStart

	// 1. Árvore (ScanTree).
	t0 := time.Now()
	ws, err := workspace.Open(root, nil)
	if err != nil {
		return nil, err
	}
	defer ws.Close()
	treeMs := time.Since(t0).Milliseconds()
	_ = len(ws.Files())

	// 2. Busca (texto).
	t1 := time.Now()
	res, err := search.Search(root, search.Options{Query: "fmt.Println", MaxResults: 100})
	if err != nil {
		return nil, err
	}
	searchMs := time.Since(t1).Milliseconds()
	_ = res.Total

	// 3. Indexação (Project Intelligence).
	t2 := time.Now()
	idx, err := intel.Build(root)
	if err != nil {
		return nil, err
	}
	indexMs := time.Since(t2).Milliseconds()
	_ = len(idx.Symbols)

	return &Result{Files: nFiles, TreeMs: treeMs, SearchMs: searchMs, IndexMs: indexMs}, nil
}
