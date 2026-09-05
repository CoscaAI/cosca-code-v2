package gitengine

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initRepo cria um repo git mínimo em dir e devolve o Engine. Pula o teste se
// o git não estiver disponível (ambiente sem git).
func initRepo(t *testing.T) (*Engine, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não disponível")
	}
	dir := t.TempDir()
	run(t, dir, "init", "-q")
	run(t, dir, "config", "user.email", "test@cosca.dev")
	run(t, dir, "config", "user.name", "Cosca Test")
	run(t, dir, "config", "commit.gpgsign", "false")

	e, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return e, dir
}

func TestStatusAndDiff(t *testing.T) {
	e, dir := initRepo(t)

	// Arquivo novo → untracked.
	write(t, dir, "main.go", "package main\n")
	status, err := e.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 || !status[0].Untracked || status[0].Path != "main.go" {
		t.Fatalf("Status = %+v, want untracked main.go", status)
	}

	// Commit → limpo.
	run(t, dir, "add", "main.go")
	run(t, dir, "commit", "-qm", "init")
	status, _ = e.Status()
	if len(status) != 0 {
		t.Fatalf("após commit, Status = %+v, want vazio", status)
	}

	// Modifica → unstaged modified.
	write(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	status, _ = e.Status()
	if len(status) != 1 || status[0].X != ' ' || status[0].Y != 'M' {
		t.Fatalf("Status = %+v, want ' M main.go'", status)
	}

	// Diff deve conter a linha adicionada.
	diff, err := e.Diff("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if diff == "" || !contains(diff, "func main()") {
		t.Fatalf("Diff não contém a mudança: %q", diff)
	}
}

func TestLogAndBranch(t *testing.T) {
	e, dir := initRepo(t)
	write(t, dir, "a.txt", "a\n")
	run(t, dir, "add", "a.txt")
	run(t, dir, "commit", "-qm", "primeiro commit")

	commits, err := e.Log(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 || commits[0].Message != "primeiro commit" {
		t.Fatalf("Log = %+v", commits)
	}

	branch, err := e.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "master" && branch != "main" {
		t.Fatalf("CurrentBranch = %q", branch)
	}

	branches, err := e.Branches()
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) < 1 {
		t.Fatal("Branches vazio")
	}
}

func TestOpenNonRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não disponível")
	}
	dir := t.TempDir()
	_, err := Open(dir)
	if err == nil {
		t.Fatal("Open de diretório sem .git deveria falhar")
	}
}

// helpers

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
