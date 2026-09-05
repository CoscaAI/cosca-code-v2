// Package memory é a Memory Engine do COSCA CODE (spec seção 46): memória com
// provenance, timestamp, source, confidence e version. Nunca tratada como
// verdade absoluta — é o registro do que o sistema aprendeu sobre o projeto.
package memory

import (
	"sort"
	"sync"
	"time"
)

// Entry é uma memória registrada.
type Entry struct {
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Source     string    `json:"source"`
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"`
	Version    int       `json:"version"`
}

// Store é o repositório de memórias (in-memory, thread-safe).
type Store struct {
	mu      sync.RWMutex
	entries map[string]Entry
}

// NewStore cria uma store vazia.
func NewStore() *Store {
	return &Store{entries: map[string]Entry{}}
}

// Set registra (ou atualiza) uma memória. Versões incrementam por key.
func (s *Store) Set(key, value, source string, confidence float64) Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.entries[key]
	e := Entry{
		Key:        key,
		Value:      value,
		Source:     source,
		Timestamp:  time.Now(),
		Confidence: confidence,
		Version:    prev.Version + 1,
	}
	s.entries[key] = e
	return e
}

// Get devolve uma memória pelo key.
func (s *Store) Get(key string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[key]
	return e, ok
}

// List devolve todas as memórias, ordenadas por key.
func (s *Store) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
