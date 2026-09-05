package editor

import "testing"

func TestDocumentEdit(t *testing.T) {
	d := NewDocument("/tmp/a.go", "func main() {}")
	d.Edit(Change{Start: d.Length(), End: d.Length(), Text: "\n"})
	if d.Text() != "func main() {}\n" {
		t.Fatalf("Text = %q", d.Text())
	}
	if d.Version() != 1 {
		t.Fatalf("Version = %d, want 1", d.Version())
	}
}

func TestDocumentUndoRedo(t *testing.T) {
	d := NewDocument("/tmp/a.go", "abc")
	d.Edit(Change{Start: 3, End: 3, Text: "def"}) // "abcdef"
	d.Edit(Change{Start: 0, End: 3, Text: "XYZ"}) // "XYZdef"

	if !d.CanUndo() {
		t.Fatal("CanUndo = false")
	}
	d.Undo()
	if d.Text() != "abcdef" {
		t.Fatalf("após undo = %q, want abcdef", d.Text())
	}
	d.Undo()
	if d.Text() != "abc" {
		t.Fatalf("após 2º undo = %q, want abc", d.Text())
	}
	if d.Undo() {
		t.Fatal("undo além do histórico retornou true")
	}
	d.Redo()
	if d.Text() != "abcdef" {
		t.Fatalf("após redo = %q", d.Text())
	}
}

func TestDocumentDirty(t *testing.T) {
	d := NewDocument("/tmp/a.go", "x")
	if d.IsDirty() {
		t.Fatal("novo documento não deve estar dirty")
	}
	d.Edit(Change{Start: 1, End: 1, Text: "y"})
	if !d.IsDirty() {
		t.Fatal("após edição deve estar dirty")
	}
	d.MarkSaved()
	if d.IsDirty() {
		t.Fatal("após MarkSaved não deve estar dirty")
	}
}

func TestDocumentEditResetsRedo(t *testing.T) {
	d := NewDocument("/tmp/a.go", "abc")
	d.Edit(Change{Start: 3, End: 3, Text: "d"})
	d.Undo()
	if !d.CanRedo() {
		t.Fatal("CanRedo deve ser true após undo")
	}
	d.Edit(Change{Start: 0, End: 0, Text: "Z"}) // nova edição limpa o redo
	if d.CanRedo() {
		t.Fatal("redo deve ser limpo após nova edição")
	}
}
