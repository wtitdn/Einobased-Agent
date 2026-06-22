package store

import (
	"context"
	"sync"
)

type InMemoryStore struct {
	mu sync.Mutex
	m  map[string][]byte
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{m: make(map[string][]byte)}
}

func (s *InMemoryStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.m[id]
	return data, ok, nil
}

func (s *InMemoryStore) Set(_ context.Context, id string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = data
	return nil
}

func (s *InMemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
	return nil
}

func NewStore() *InMemoryStore {
	store := NewInMemoryStore()
	return store
}
