package workflow

import "testing"

func TestBuiltinWorkflows(t *testing.T) {
	cases := []struct {
		name    string
		wf      Workflow
		minStep int
	}{
		{"bug_fix", BugFix(), 4},
		{"feature", Feature(), 3},
		{"refactor", Refactor(), 3},
	}
	for _, c := range cases {
		if c.wf.Name != c.name {
			t.Errorf("Name = %q, want %q", c.wf.Name, c.name)
		}
		if len(c.wf.Steps) < c.minStep {
			t.Errorf("%s: %d passos, want >= %d", c.name, len(c.wf.Steps), c.minStep)
		}
		// Cada passo tem nome e prompt.
		for _, s := range c.wf.Steps {
			if s.Name == "" || s.Prompt == "" {
				t.Errorf("%s: passo malformado %+v", c.name, s)
			}
		}
	}
}
