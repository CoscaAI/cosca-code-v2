package skill

import "testing"

func TestBuiltin(t *testing.T) {
	skills := Builtin()
	if len(skills) < 3 {
		t.Fatalf("Builtin = %d, want >= 3", len(skills))
	}
	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}
	for _, want := range []string{"go_developer", "security_auditor", "tester"} {
		if !names[want] {
			t.Errorf("skill %q ausente", want)
		}
	}
}

func TestSkillPrompt(t *testing.T) {
	s := Builtin()[0]
	p := s.Prompt()
	if p == "" {
		t.Fatal("Prompt vazio")
	}
	if len(s.Tools) == 0 {
		t.Fatal("skill sem tools recomendadas")
	}
}
