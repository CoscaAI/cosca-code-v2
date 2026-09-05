package testengine

import (
	"testing"
)

func TestParseGoTest(t *testing.T) {
	output := `ok  	github.com/x/a	0.123s
FAIL	github.com/x/b	0.456s
?   	github.com/x/c	[no test files]
`
	pkgs := parseGoTest(output)
	if len(pkgs) != 3 {
		t.Fatalf("len = %d, want 3", len(pkgs))
	}
	if pkgs[0].Status != "ok" || pkgs[0].Name != "github.com/x/a" {
		t.Fatalf("pkg[0] = %+v", pkgs[0])
	}
	if pkgs[1].Status != "fail" {
		t.Fatalf("pkg[1] = %+v", pkgs[1])
	}
	if pkgs[2].Status != "no-tests" {
		t.Fatalf("pkg[2] = %+v", pkgs[2])
	}
}

func TestRunEchoCommand(t *testing.T) {
	// Usa um comando simples (não depende de go test).
	res, err := Run(t.TempDir(), "echo ok && exit 0", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed {
		t.Fatalf("Passed = false, output = %q", res.Output)
	}
	if res.Duration == "" {
		t.Fatal("Duration vazio")
	}
}

func TestRunFailingCommand(t *testing.T) {
	res, err := Run(t.TempDir(), "exit 1", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed {
		t.Fatal("Passed = true para comando que falha")
	}
}
