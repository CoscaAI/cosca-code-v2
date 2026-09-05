package workspace

import (
	"os"
	"path/filepath"
	"sort"
)

// FileNode é um nó da árvore de arquivos (explorer, spec seção 32).
type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"` // relativo ao root
	IsDir    bool        `json:"is_dir"`
	Children []*FileNode `json:"children,omitempty"`
}

// Tree é a árvore de arquivos do workspace.
type Tree struct {
	RootPath string
	Root     *FileNode
}

// defaultIgnores são diretórios/padrões que nunca entram na árvore (evita
// indexar node_modules, .git, build, etc.). É a filtragem "git-aware" do
// explorer — o usuário não quer lixo de dependência no explorador.
var defaultIgnores = map[string]bool{
	".git":         true,
	"node_modules": true,
	"target":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".cache":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	".idea":        true,
	".vscode":      true,
	"coverage":     true,
}

// ScanTree constrói a árvore de arquivos a partir do root, ignorando
// diretórios de dependência/artefato (defaultIgnores).
func ScanTree(root string) *Tree {
	t := &Tree{RootPath: root}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	t.Root = scanDir(abs, "")
	return t
}

func scanDir(absDir, relDir string) *FileNode {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return &FileNode{Name: filepath.Base(absDir), Path: relDir, IsDir: true}
	}
	node := &FileNode{Name: filepath.Base(absDir), Path: relDir, IsDir: true}
	if relDir == "" {
		node.Name = filepath.Base(absDir)
		if node.Name == "/" || node.Name == "." {
			node.Name = absDir
		}
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if defaultIgnores[name] {
				continue
			}
			child := scanDir(filepath.Join(absDir, name), joinRel(relDir, name))
			node.Children = append(node.Children, child)
		} else {
			node.Children = append(node.Children, &FileNode{
				Name:  name,
				Path:  joinRel(relDir, name),
				IsDir: false,
			})
		}
	}
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].IsDir != node.Children[j].IsDir {
			return node.Children[i].IsDir // diretórios primeiro
		}
		return node.Children[i].Name < node.Children[j].Name
	})
	return node
}

func joinRel(rel, name string) string {
	if rel == "" {
		return name
	}
	return rel + "/" + name
}

// Walk percorre a árvore em pré-ordem, chamando fn para cada nó.
func (t *Tree) Walk(fn func(n *FileNode)) {
	walkNode(t.Root, fn)
}

func walkNode(n *FileNode, fn func(*FileNode)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Children {
		walkNode(c, fn)
	}
}

// Files devolve todos os caminhos de arquivo (não-diretório) na árvore.
func (t *Tree) Files() []string {
	var files []string
	t.Walk(func(n *FileNode) {
		if !n.IsDir {
			files = append(files, n.Path)
		}
	})
	return files
}
