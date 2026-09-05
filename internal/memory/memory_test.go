package memory

import "testing"

func TestStoreSetGetVersion(t *testing.T) {
	s := NewStore()

	e1 := s.Set("key", "value1", "agent", 0.8)
	if e1.Version != 1 {
		t.Fatalf("Version = %d, want 1", e1.Version)
	}

	e2 := s.Set("key", "value2", "agent", 0.9)
	if e2.Version != 2 {
		t.Fatalf("Version = %d, want 2 (incrementa)", e2.Version)
	}
	if e2.Value != "value2" {
		t.Fatalf("Value = %q, want value2", e2.Value)
	}

	got, ok := s.Get("key")
	if !ok || got.Value != "value2" {
		t.Fatalf("Get = %+v ok=%v", got, ok)
	}

	if _, ok := s.Get("missing"); ok {
		t.Fatal("Get de key inexistente deveria ser false")
	}
}

func TestStoreList(t *testing.T) {
	s := NewStore()
	s.Set("a", "1", "t", 0.5)
	s.Set("b", "2", "t", 0.5)
	if len(s.List()) != 2 {
		t.Fatalf("List = %d, want 2", len(s.List()))
	}
}
