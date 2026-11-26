package cache

import (
	"context"
	"sync"
	"time"
)

// Cache stores opaque values by key with an expiry. Implementations must be safe for concurrent use.
type Cache interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type entry struct {
	value   []byte
	expires time.Time
}

// Memory is an in-process Cache; expired entries are dropped when read.
type Memory struct {
	clock   func() time.Time
	mu      sync.Mutex
	entries map[string]entry
}

func NewMemory(clock func() time.Time) *Memory {
	return &Memory{clock: clock, entries: map[string]entry{}}
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[key]
	if ok && !m.clock().Before(e.expires) {
		delete(m.entries, key)
		return nil, false, nil
	}
	return e.value, ok, nil
}

func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[key] = entry{value: value, expires: m.clock().Add(ttl)}
	return nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
	return nil
}
