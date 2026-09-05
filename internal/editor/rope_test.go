package editor

import (
	"strings"
	"testing"
)

func TestRopeBasics(t *testing.T) {
	r := NewRope("hello world")
	if r.Length() != 11 {
		t.Fatalf("Length = %d, want 11", r.Length())
	}
	if r.String() != "hello world" {
		t.Fatalf("String = %q", r.String())
	}
	if r.At(0) != 'h' || r.At(5) != ' ' || r.At(10) != 'd' {
		t.Fatalf("At devolveu valores errados")
	}
}

func TestRopeInsert(t *testing.T) {
	cases := []struct {
		initial string
		offset  int
		insert  string
		want    string
	}{
		{"hello", 0, "X", "Xhello"},
		{"hello", 5, "X", "helloX"},
		{"hello", 2, "XY", "heXYllo"},
		{"", 0, "abc", "abc"},
	}
	for _, c := range cases {
		r := NewRope(c.initial).Insert(c.offset, c.insert)
		if r.String() != c.want {
			t.Errorf("Insert(%q, %d, %q) = %q, want %q", c.initial, c.offset, c.insert, r.String(), c.want)
		}
	}
}

func TestRopeDelete(t *testing.T) {
	r := NewRope("hello world").Delete(5, 6)
	if r.String() != "helloworld" {
		t.Fatalf("Delete = %q, want %q", r.String(), "helloworld")
	}
	r = NewRope("hello").Delete(0, 5)
	if r.String() != "" {
		t.Fatalf("Delete total = %q, want vazio", r.String())
	}
}

func TestRopeSubstring(t *testing.T) {
	r := NewRope("hello world")
	if r.Substring(0, 5) != "hello" {
		t.Fatalf("Substring(0,5) = %q", r.Substring(0, 5))
	}
	if r.Substring(6, 11) != "world" {
		t.Fatalf("Substring(6,11) = %q", r.Substring(6, 11))
	}
}

func TestRopeImmutability(t *testing.T) {
	orig := NewRope("hello")
	orig.Insert(0, "XX")
	if orig.String() != "hello" {
		t.Fatalf("rope foi mutado: %q", orig.String())
	}
}

func TestRopeUnicode(t *testing.T) {
	r := NewRope("café ☕")
	if r.Length() != 6 { // c,a,f,é,espaço,☕
		t.Fatalf("Length = %d, want 6", r.Length())
	}
	if r.At(3) != 'é' || r.At(5) != '☕' {
		t.Fatalf("At unicode errado")
	}
	r = r.Insert(6, "☕") // "café ☕☕"
	if r.String() != "café ☕☕" {
		t.Fatalf("Insert unicode = %q", r.String())
	}
}

func TestRopeLargeTextStress(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 50000; i++ {
		sb.WriteString("linha de texto com conteúdo variado 1234567890\n")
	}
	big := sb.String()

	r := NewRope(big)
	if r.Length() != len([]rune(big)) {
		t.Fatalf("Length divergente")
	}

	// Muitas edições incrementais devem manter integridade.
	for i := 0; i < 1000; i++ {
		r = r.Insert(i*7, "x")
	}
	if r.Length() != len([]rune(big))+1000 {
		t.Fatalf("Length após edições = %d", r.Length())
	}

	// Substring no meio deve bater com a versão ingênua.
	got := r.Substring(100, 6000)
	want := []rune(r.String())[100:6000]
	if got != string(want) {
		t.Fatalf("Substring diverge após estresse")
	}
}
