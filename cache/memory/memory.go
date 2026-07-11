package memory

import (
	"context"
	"sync"

	"github.com/eve-online-tools/eve-resfile-proxy/cache"
)

var _ cache.Cache = (*Cache)(nil)

// Cache stores bytes in memory keyed by caller-defined strings.
type Cache struct {
	mu    sync.RWMutex
	items map[string][]byte
}

// New returns an empty in-memory cache.
func New() *Cache {
	return &Cache{items: make(map[string][]byte)}
}

func (c *Cache) Get(_ context.Context, key string) ([]byte, bool, error) {
	if c == nil {
		return nil, false, nil
	}

	c.mu.RLock()
	data, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}

	out := make([]byte, len(data))
	copy(out, data)
	return out, true, nil
}

func (c *Cache) Store(_ context.Context, key string, data []byte) error {
	if c == nil {
		return nil
	}

	cp := make([]byte, len(data))
	copy(cp, data)

	c.mu.Lock()
	c.items[key] = cp
	c.mu.Unlock()
	return nil
}
