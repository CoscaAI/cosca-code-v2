// Package event é o Event Bus do COSCA CODE (spec seção 71) — o sistema
// nervoso do ambiente. Todo acontecimento relevante vira um Event tipado, e
// qualquer componente pode assinar por tipo (ou globalmente) para reagir.
// É o que torna a observabilidade (seção 44) e a extensibilidade nativas, em
// vez de callbacks espalhados e acoplamento direto entre engines.
package event

import (
	"sync"
	"time"
)

// Type identifica o tipo de evento (spec seção 71).
type Type string

const (
	WorkspaceOpened    Type = "workspace.opened"
	WorkspaceClosed    Type = "workspace.closed"
	FileOpened         Type = "file.opened"
	FileClosed         Type = "file.closed"
	FileChanged        Type = "file.changed"
	FileSaved          Type = "file.saved"
	FileCreated        Type = "file.created"
	FileDeleted        Type = "file.deleted"
	EditorChanged      Type = "editor.changed"
	EditorSelectionSet Type = "editor.selection_set"
	BufferModified     Type = "buffer.modified"
	GitChanged         Type = "git.changed"
	AgentStarted       Type = "agent.started"
	AgentCompleted     Type = "agent.completed"
	AgentFailed        Type = "agent.failed"
	AgentToolCalled    Type = "agent.tool_called"
	ToolCompleted      Type = "tool.completed"
	ToolFailed         Type = "tool.failed"
	TestStarted        Type = "test.started"
	TestPassed         Type = "test.passed"
	TestFailed         Type = "test.failed"
	DebugStarted       Type = "debug.started"
	DebugStopped       Type = "debug.stopped"
	BreakpointHit      Type = "debug.breakpoint_hit"
	ErrorDetected      Type = "error.detected"
	DiagnosticsChanged Type = "diagnostics.changed"
	ModelChanged       Type = "model.changed"
	ProviderConnected  Type = "provider.connected"
	ProviderFailed     Type = "provider.failed"
	MemoryRetrieved    Type = "memory.retrieved"
	MemoryStored       Type = "memory.stored"
	TaskCreated        Type = "task.created"
	TaskCompleted      Type = "task.completed"
	TaskFailed         Type = "task.failed"
	CommandExecuted    Type = "command.executed"
	ExtensionLoaded    Type = "extension.loaded"
	TelemetryEmitted   Type = "telemetry.emitted"
	// Engine events (Fase 1 — Cosca Engine integrada ao editor).
	AssetAdded         Type = "asset.added"
	AssetRemoved       Type = "asset.removed"
	GraphValidated     Type = "graph.validated"
	GraphExecuted      Type = "graph.executed"
	ProjectManifestChanged Type = "project.manifest_changed"
)

// Event é a unidade do barramento. Source identifica o componente emissor;
// CorrelationID agrupa eventos de uma mesma unidade de trabalho (uma tarefa,
// uma investigação, um comando).
type Event struct {
	ID            string
	Type          Type
	Timestamp     time.Time
	Source        string
	Payload       any
	CorrelationID string
}

// Handler recebe eventos.
type Handler func(Event)

// subscription é uma assinatura identificada por ID (comparar func values em
// Go é não-confiável — o ID é a identidade estável para unsubscribe).
type subscription struct {
	id uint64
	h  Handler
}

// Bus é um barramento pub/sub tipado e thread-safe. Assinaturas por tipo
// exato e um canal global ("subscribe all") para observabilidade transversal.
// O dispatch é síncrono e na ordem de publicação — handlers devem ser rápidos
// e nunca bloquear (trabalho pesado vai para goroutines próprias).
type Bus struct {
	mu     sync.RWMutex
	nextID uint64
	subs   map[Type][]subscription
	all    []subscription
}

// NewBus cria um barramento vazio.
func NewBus() *Bus {
	return &Bus{subs: make(map[Type][]subscription)}
}

// Subscribe registra um handler para um tipo específico. Retorna um
// cancelador idempotente para remover a assinatura.
func (b *Bus) Subscribe(t Type, h Handler) func() {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	b.subs[t] = append(b.subs[t], subscription{id: id, h: h})
	b.mu.Unlock()
	return func() { b.unsubscribe(t, id) }
}

// SubscribeAll registra um handler que recebe TODOS os eventos.
func (b *Bus) SubscribeAll(h Handler) func() {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	b.all = append(b.all, subscription{id: id, h: h})
	b.mu.Unlock()
	return func() { b.unsubscribeAll(id) }
}

// Publish entrega o evento aos assinantes do tipo e aos globais, na ordem de
// registro. Preenche Timestamp se zero.
func (b *Bus) Publish(ev Event) {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}
	b.mu.RLock()
	handlers := append([]subscription(nil), b.subs[ev.Type]...)
	all := append([]subscription(nil), b.all...)
	b.mu.RUnlock()

	for _, s := range handlers {
		s.h(ev)
	}
	for _, s := range all {
		s.h(ev)
	}
}

// SubscriberCount devolve o total de assinaturas (por tipo + globais).
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	n := len(b.all)
	for _, hs := range b.subs {
		n += len(hs)
	}
	return n
}

func (b *Bus) unsubscribe(t Type, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	hs := b.subs[t]
	for i, s := range hs {
		if s.id == id {
			b.subs[t] = append(hs[:i], hs[i+1:]...)
			return
		}
	}
}

func (b *Bus) unsubscribeAll(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, s := range b.all {
		if s.id == id {
			b.all = append(b.all[:i], b.all[i+1:]...)
			return
		}
	}
}
