package assetcache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Store struct {
	Root string
}

func New(root string) *Store {
	return &Store{Root: root}
}

func (s *Store) Path(cdnPath string) string {
	return filepath.Join(s.Root, filepath.FromSlash(cdnPath))
}

func (s *Store) Read(cdnPath string) ([]byte, bool, error) {
	path := s.Path(cdnPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

func (s *Store) ModTime(cdnPath string) (time.Time, bool, error) {
	info, err := os.Stat(s.Path(cdnPath))
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return info.ModTime(), true, nil
}

func (s *Store) Write(cdnPath string, data []byte) error {
	path := s.Path(cdnPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmpPath := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.tmp", filepath.Base(path)))
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
