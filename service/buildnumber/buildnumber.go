package buildnumber

import "sync"

// BuildNumber holds the active client build number for HTTP responses.
// It is safe for concurrent reads and writes.
type BuildNumber struct {
	mu     sync.RWMutex
	number string
}

func (b *BuildNumber) Set(n string) {
	b.mu.Lock()
	b.number = n
	b.mu.Unlock()
}

func (b *BuildNumber) Get() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.number
}
