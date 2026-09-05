package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/CoscaAI/cosca-code/internal/event"
	"github.com/CoscaAI/cosca-code/internal/gitengine"
)

// Workspace é um projeto aberto: um root + detecção + árvore + watcher + git.
// É o "backend isolado" que a UI consome via contratos — nunca o filesystem.
type Workspace struct {
	root    string
	info    ProjectInfo
	tree    *Tree
	bus     *event.Bus
	watcher *Watcher
	git     *gitengine.Engine
	mu      sync.RWMutex
}

// Open abre um workspace: valida o root, detecta o projeto, escaneia a árvore
// e inicia o watcher (emitindo workspace.opened no bus).
func Open(root string, bus *event.Bus) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("root inacessível: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q não é um diretório", abs)
	}

	w := &Workspace{
		root:    abs,
		info:    Detect(abs),
		tree:    ScanTree(abs),
		bus:     bus,
		watcher: NewWatcher(abs, bus),
	}
	// Git Engine é opcional: abre se for um repositório (nunca falha o Open).
	if g, err := gitengine.Open(abs); err == nil {
		w.git = g
	}
	w.watcher.Start()

	if bus != nil {
		bus.Publish(event.Event{
			Type:   event.WorkspaceOpened,
			Source: "workspace",
			Payload: map[string]any{
				"root":     abs,
				"language": w.info.Language,
			},
		})
	}
	return w, nil
}

// Root devolve o caminho absoluto do root.
func (w *Workspace) Root() string { return w.root }

// Info devolve o ProjectInfo detectado.
func (w *Workspace) Info() ProjectInfo { return w.info }

// Tree devolve a árvore de arquivos.
func (w *Workspace) Tree() *Tree { return w.tree }

// Files devolve todos os caminhos de arquivo no workspace.
func (w *Workspace) Files() []string { return w.tree.Files() }

// Git devolve o Git Engine, ou nil se o root não for um repositório git.
func (w *Workspace) Git() *gitengine.Engine { return w.git }

// Close encerra o watcher e emite workspace.closed.
func (w *Workspace) Close() {
	w.watcher.Stop()
	if w.bus != nil {
		w.bus.Publish(event.Event{Type: event.WorkspaceClosed, Source: "workspace", Payload: w.root})
	}
}
