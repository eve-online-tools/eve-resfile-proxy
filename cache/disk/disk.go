package diskcache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eve-online-tools/eve-resfile-proxy/cache"
	cachekey "github.com/eve-online-tools/eve-resfile-proxy/cache/key"
)

var _ cache.Cache = (*Cache)(nil)

// Cache stores bytes on disk under Root, keyed by slash-separated paths.
type Cache struct {
	Root string
}

// New returns a disk-backed cache rooted at dir.
func New(root string) *Cache {
	return &Cache{Root: root}
}

func (c *Cache) resolvedPath(key string) (string, error) {
	if c == nil || c.Root == "" {
		return "", nil
	}
	return cachekey.Join(c.Root, key)
}

// Path returns the filesystem path for key. Invalid keys return an empty string.
func (c *Cache) Path(key string) string {
	path, err := c.resolvedPath(key)
	if err != nil {
		return ""
	}
	return path
}

func (c *Cache) Get(_ context.Context, key string) ([]byte, bool, error) {
	if c == nil || c.Root == "" {
		return nil, false, nil
	}

	path, err := c.resolvedPath(key)
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

func (c *Cache) Store(_ context.Context, key string, data []byte) error {
	if c == nil || c.Root == "" {
		return nil
	}

	path, err := c.resolvedPath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmpPath := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.tmp", filepath.Base(path)))
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
