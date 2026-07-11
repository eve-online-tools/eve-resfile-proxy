package transform

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiskCache stores transformed asset bytes under
// <root>/_transformed/<platform>/<ruleName>/<cdnPath>.
type DiskCache struct {
	Root string
}

func (c *DiskCache) Path(platform, ruleName, cdnPath string) string {
	return filepath.Join(c.Root, "_transformed", platform, ruleName, filepath.FromSlash(cdnPath))
}

func (c *DiskCache) Read(platform, ruleName, cdnPath string) ([]byte, bool, error) {
	if c == nil || c.Root == "" {
		return nil, false, nil
	}

	path := c.Path(platform, ruleName, cdnPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

func (c *DiskCache) Write(platform, ruleName, cdnPath string, data []byte) error {
	if c == nil || c.Root == "" {
		return nil
	}

	path := c.Path(platform, ruleName, cdnPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmpPath := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.tmp", filepath.Base(path)))
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
