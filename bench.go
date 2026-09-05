// Modo benchmark (Fase 7): `cosca-code bench <n>` mede a performance em um
// repositório sintético com n arquivos — prova de escala.
package main

import (
	"fmt"

	"github.com/CoscaAI/cosca-code/internal/benchmark"
)

func runBench(n int) {
	fmt.Printf("COSCA CODE — benchmark com %d arquivos...\n", n)
	res, err := benchmark.Run(n)
	if err != nil {
		fmt.Printf("erro: %v\n", err)
		return
	}
	fmt.Printf("  arquivos:  %d\n", res.Files)
	fmt.Printf("  árvore:    %dms\n", res.TreeMs)
	fmt.Printf("  busca:     %dms\n", res.SearchMs)
	fmt.Printf("  indexação: %dms\n", res.IndexMs)
}
