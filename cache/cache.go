package cache

import "context"

// Cache stores opaque byte blobs keyed by caller-defined strings.
// Implementations must be safe for concurrent use.
type Cache interface {
	Get(ctx context.Context, key string) (data []byte, found bool, err error)
	Store(ctx context.Context, key string, data []byte) error
}
