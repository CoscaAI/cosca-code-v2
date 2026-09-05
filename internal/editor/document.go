package editor

// Change é uma operação de edição atômica: substitui o intervalo [Start, End)
// por Text. Insert puro = Start==End; delete puro = Text=="".
type Change struct {
	Start int    // offset (runes) inicial
	End   int    // offset (runes) final (exclusivo)
	Text  string // texto de substituição
}

// Apply devolve um novo rope com a alteração aplicada (imutável).
func (c Change) Apply(r *rope) *rope {
	if c.Text == "" && c.Start == c.End {
		return r
	}
	result := r
	if c.End > c.Start {
		result = result.Delete(c.Start, c.End)
	}
	if c.Text != "" {
		result = result.Insert(c.Start, c.Text)
	}
	return result
}

// Document é o modelo de um arquivo aberto: um rope imutável + histórico de
// versões (undo/redo). A persistência estrutural faz cada snapshot custar só
// as subárvores tocadas — undo é O(1) de memória incremental.
type Document struct {
	path    string
	current *rope
	saved   *rope // conteúdo da última gravação (dirty = current != saved)
	version int64
	undo    []*rope
	redo    []*rope
}

// NewDocument cria um documento com o conteúdo inicial text.
func NewDocument(path, text string) *Document {
	r := NewRope(text)
	return &Document{path: path, current: r, saved: r}
}

// Path devolve o caminho do arquivo.
func (d *Document) Path() string { return d.path }

// Text devolve o conteúdo completo atual.
func (d *Document) Text() string { return d.current.String() }

// Length devolve o nº de runes.
func (d *Document) Length() int { return d.current.Length() }

// Version devolve a versão monotônica (incrementa a cada mutação/undo/redo).
func (d *Document) Version() int64 { return d.version }

// IsDirty reporta se há alterações não gravadas.
func (d *Document) IsDirty() bool { return d.current != d.saved }

// Edit aplica uma alteração, empurrando a versão anterior ao undo.
func (d *Document) Edit(c Change) {
	d.undo = append(d.undo, d.current)
	d.redo = nil
	d.current = c.Apply(d.current)
	d.version++
}

// Undo desfaz a última edição. Retorna false se o histórico estiver vazio.
func (d *Document) Undo() bool {
	if len(d.undo) == 0 {
		return false
	}
	d.redo = append(d.redo, d.current)
	d.current = d.undo[len(d.undo)-1]
	d.undo = d.undo[:len(d.undo)-1]
	d.version++
	return true
}

// Redo refaz a última edição desfeita. Retorna false se não houver nada.
func (d *Document) Redo() bool {
	if len(d.redo) == 0 {
		return false
	}
	d.undo = append(d.undo, d.current)
	d.current = d.redo[len(d.redo)-1]
	d.redo = d.redo[:len(d.redo)-1]
	d.version++
	return true
}

// MarkSaved marca o estado atual como gravado (limpa o dirty).
func (d *Document) MarkSaved() { d.saved = d.current }

// CanUndo / CanRedo reportam a disponibilidade de operações.
func (d *Document) CanUndo() bool { return len(d.undo) > 0 }
func (d *Document) CanRedo() bool { return len(d.redo) > 0 }
