package key

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	ErrEmptyKey    = errors.New("cache key must not be empty")
	ErrAbsoluteKey = errors.New("cache key must be relative")
	ErrInvalidKey  = errors.New("invalid cache key")
)

// Validate rejects cache keys that could escape a root directory.
func Validate(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if filepath.IsAbs(key) {
		return ErrAbsoluteKey
	}
	if strings.Contains(key, `\`) {
		return fmt.Errorf("%w: backslash", ErrInvalidKey)
	}
	for _, seg := range strings.Split(filepath.ToSlash(key), "/") {
		if seg == ".." {
			return fmt.Errorf("%w: parent segment", ErrInvalidKey)
		}
	}
	return nil
}

// Join validates key and returns an absolute path under root.
func Join(root, key string) (string, error) {
	if err := Validate(key); err != nil {
		return "", err
	}
	joined := filepath.Join(root, filepath.FromSlash(key))
	rel, err := filepath.Rel(root, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: escapes root", ErrInvalidKey)
	}
	return joined, nil
}
