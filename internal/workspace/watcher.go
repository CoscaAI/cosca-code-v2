package workspace

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/CoscaAI/cosca-code/internal/event"
)

// fileState é o rastro observável de um arquivo (para detectar mudanças).
type fileState struct {
	modTime time.Time
	size    int64
	isDir   bool
}

// Watcher observa o diretório root e emite eventos file.created/changed/deleted
// no Event Bus. Implementação por polling (portátil, sem dependência externa);
// a interface isola o mecanismo, permitindo trocar por fsnotify depois sem
// tocar nos consumidores.
type Watcher struct {
	root     string
	bus      *event.Bus
	interval time.Duration
	mu       sync.Mutex
	snapshot map[string]fileState
	stop     chan struct{}
	done     chan struct{}
}

// NewWatcher cria um watcher (não iniciado) para root.
func NewWatcher(root string, bus *event.Bus) *Watcher {
	return &Watcher{
		root:     root,
		bus:      bus,
		interval: 2 * time.Second,
		snapshot: map[string]fileState{},
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start faz o snapshot inicial e inicia o loop de observação em goroutine.
func (w *Watcher) Start() {
	w.snapshot = w.scan()
	go w.loop()
}

// SetInterval ajusta o intervalo de polling (útil em testes e tuning).
func (w *Watcher) SetInterval(d time.Duration) { w.interval = d }

// Stop encerra o loop e aguarda a goroutine terminar.
func (w *Watcher) Stop() {
	close(w.stop)
	<-w.done
}

func (w *Watcher) loop() {
	defer close(w.done)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.diff(w.scan())
		}
	}
}

// scan varre o root e devolve o mapa path→estado.
func (w *Watcher) scan() map[string]fileState {
	out := map[string]fileState{}
	_ = filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if defaultIgnores[info.Name()] && path != w.root {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(w.root, path)
		if err != nil {
			return nil
		}
		out[rel] = fileState{modTime: info.ModTime(), size: info.Size()}
		return nil
	})
	return out
}

// diff compara o snapshot novo com o anterior e emite eventos.
func (w *Watcher) diff(next map[string]fileState) {
	w.mu.Lock()
	prev := w.snapshot
	w.snapshot = next
	w.mu.Unlock()

	for path, st := range next {
		old, ok := prev[path]
		if !ok {
			w.bus.Publish(event.Event{Type: event.FileCreated, Source: "workspace", Payload: path})
			continue
		}
		if !old.modTime.Equal(st.modTime) || old.size != st.size {
			w.bus.Publish(event.Event{Type: event.FileChanged, Source: "workspace", Payload: path})
		}
	}
	for path := range prev {
		if _, ok := next[path]; !ok {
			w.bus.Publish(event.Event{Type: event.FileDeleted, Source: "workspace", Payload: path})
		}
	}
}
