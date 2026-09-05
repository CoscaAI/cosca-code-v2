package benchmark

import "testing"

func TestRunSmall(t *testing.T) {
	res, err := Run(100)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Files != 100 {
		t.Fatalf("Files = %d, want 100", res.Files)
	}
	if res.TreeMs < 0 || res.SearchMs < 0 || res.IndexMs < 0 {
		t.Fatal("tempos negativos")
	}
	t.Logf("100 arquivos: árvore %dms, busca %dms, index %dms", res.TreeMs, res.SearchMs, res.IndexMs)
}
