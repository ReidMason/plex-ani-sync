package server

import (
	"sync"
	"time"
)

const defaultUsernameTTL = 5 * time.Minute

type usernameEntry struct {
	name    string
	expires time.Time
}

type usernameCache struct {
	clock   func() time.Time
	ttl     time.Duration
	mu      sync.Mutex
	entries map[int64]usernameEntry
}

func newUsernameCache(clock func() time.Time, ttl time.Duration) *usernameCache {
	return &usernameCache{clock: clock, ttl: ttl, entries: map[int64]usernameEntry{}}
}

func (c *usernameCache) fresh(userID int64) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[userID]
	return e.name, ok && c.clock().Before(e.expires)
}

func (c *usernameCache) stale(userID int64) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[userID]
	return e.name, ok
}

func (c *usernameCache) store(userID int64, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userID] = usernameEntry{name: name, expires: c.clock().Add(c.ttl)}
}

func (c *usernameCache) evict(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userID)
}
