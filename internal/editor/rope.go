// Package editor contém o núcleo de edição do COSCA CODE: o rope (buffer de
// texto imutável) e o Document (modelo com undo/redo). Segue o padrão extraído
// de Monaco/Lapce/Zed: buffer imutável versionado — undo, diff, split e
// colaboração tornam-se consequência natural, não features paralelas.
package editor

// node é um nó de um rope AVL. Leaf guarda o texto; branch concatena duas
// subárvores com weight = nº de runes na subárvore esquerda.
type node struct {
	left, right *node
	text        []rune // apenas leaf
	weight      int    // runes na esquerda (apenas branch)
	height      int    // AVL
}

func leaf(text []rune) *node {
	return &node{text: text, height: 1}
}

func branch(l, r *node) *node {
	return &node{left: l, right: r, weight: nodeLen(l), height: 1 + max(l.height, r.height)}
}

func nodeLen(n *node) int {
	if n == nil {
		return 0
	}
	if n.left == nil && n.right == nil {
		return len(n.text)
	}
	return n.weight + nodeLen(n.right)
}

// rope é um buffer de texto imutável com persistência estrutural.
type rope struct {
	root *node
}

// NewRope cria um rope a partir de uma string.
func NewRope(text string) *rope {
	return &rope{root: leaf([]rune(text))}
}

// Length devolve o nº de runes.
func (r *rope) Length() int {
	if r == nil || r.root == nil {
		return 0
	}
	return nodeLen(r.root)
}

// String devolve o conteúdo completo.
func (r *rope) String() string {
	if r == nil || r.root == nil {
		return ""
	}
	var sb []rune
	flatten(r.root, &sb)
	return string(sb)
}

func flatten(n *node, sb *[]rune) {
	if n == nil {
		return
	}
	if n.left == nil && n.right == nil {
		*sb = append(*sb, n.text...)
		return
	}
	flatten(n.left, sb)
	flatten(n.right, sb)
}

// At devolve o rune na posição offset (0-indexado por rune).
func (r *rope) At(offset int) rune {
	if r == nil || r.root == nil {
		return 0
	}
	n := r.root
	for n != nil {
		if n.left == nil && n.right == nil {
			return n.text[offset]
		}
		if offset < n.weight {
			n = n.left
		} else {
			offset -= n.weight
			n = n.right
		}
	}
	return 0
}

// Insert devolve um novo rope com text inserido em offset.
func (r *rope) Insert(offset int, text string) *rope {
	offset = clamp(offset, 0, r.Length())
	l, rr := split(r.root, offset)
	return &rope{root: concat(concat(l, leaf([]rune(text))), rr)}
}

// Delete devolve um novo rope sem o intervalo [start, end).
func (r *rope) Delete(start, end int) *rope {
	start = clamp(start, 0, r.Length())
	end = clamp(end, start, r.Length())
	l, mid := split(r.root, start)
	_, rr := split(mid, end-start)
	return &rope{root: concat(l, rr)}
}

// Substring devolve o texto do intervalo [start, end).
func (r *rope) Substring(start, end int) string {
	start = clamp(start, 0, r.Length())
	end = clamp(end, start, r.Length())
	_, mid := split(r.root, start)
	mid, _ = split(mid, end-start)
	return (&rope{root: mid}).String()
}

// split divide n em offset, devolvendo (esquerda, direita).
func split(n *node, offset int) (*node, *node) {
	if n == nil {
		return nil, nil
	}
	if offset <= 0 {
		return nil, n
	}
	if offset >= nodeLen(n) {
		return n, nil
	}
	if n.left == nil && n.right == nil {
		return leaf(n.text[:offset]), leaf(n.text[offset:])
	}
	if offset < n.weight {
		l, r := split(n.left, offset)
		return l, concat(r, n.right)
	}
	l, r := split(n.right, offset-n.weight)
	return concat(n.left, l), r
}

// concat junta duas subárvores com rebalanceamento AVL.
func concat(l, r *node) *node {
	if l == nil {
		return r
	}
	if r == nil {
		return l
	}
	n := branch(l, r)
	return rebalance(n)
}

// rebalance reequilibra um nó AVL (rotações simples/duplas).
func rebalance(n *node) *node {
	if n == nil {
		return nil
	}
	n.height = 1 + max(height(n.left), height(n.right))
	bal := height(n.left) - height(n.right)
	if bal > 1 { // pesado à esquerda
		if height(n.left.left) < height(n.left.right) {
			n.left = rotateLeft(n.left)
		}
		return rotateRight(n)
	}
	if bal < -1 { // pesado à direita
		if height(n.right.right) < height(n.right.left) {
			n.right = rotateRight(n.right)
		}
		return rotateLeft(n)
	}
	return n
}

// rotateLeft / rotateRight são COPY-ON-WRITE: criam nós novos em vez de mutar
// os existentes. É essencial para a persistência estrutural — os filhos podem
// ser subárvores compartilhadas com versões anteriores do rope, e mutá-los
// corromperia o histórico (undo) silenciosamente.
func rotateLeft(n *node) *node {
	r := n.right
	newN := &node{
		left:   n.left,
		right:  r.left,
		weight: nodeLen(n.left),
		height: 1 + max(height(n.left), height(r.left)),
	}
	return &node{
		left:   newN,
		right:  r.right,
		weight: nodeLen(newN),
		height: 1 + max(height(newN), height(r.right)),
	}
}

func rotateRight(n *node) *node {
	l := n.left
	newN := &node{
		left:   l.right,
		right:  n.right,
		weight: nodeLen(l.right),
		height: 1 + max(height(l.right), height(n.right)),
	}
	return &node{
		left:   l.left,
		right:  newN,
		weight: nodeLen(l.left),
		height: 1 + max(height(l.left), height(newN)),
	}
}

func height(n *node) int {
	if n == nil {
		return 0
	}
	return n.height
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
