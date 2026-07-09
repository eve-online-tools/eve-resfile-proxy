package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CacheMeta struct {
	BuildNumber         string           `json:"buildNumber"`
	LoadedPlatforms     []Platform       `json:"loadedPlatforms"`
	PlatformEntryCounts map[Platform]int `json:"platformEntryCounts,omitempty"`
}

func platformCacheDir(cacheDir, buildNumber string, p Platform) string {
	return filepath.Join(cacheDir, buildNumber, string(p))
}

func buildIndexPath(cacheDir, buildNumber string, p Platform) string {
	return filepath.Join(platformCacheDir(cacheDir, buildNumber, p), "build-index.txt")
}

func resfileGlobalPath(cacheDir, buildNumber string, p Platform) string {
	return filepath.Join(platformCacheDir(cacheDir, buildNumber, p), "resfileindex-global.txt")
}

func resfileOSPath(cacheDir, buildNumber string, p Platform) string {
	return filepath.Join(platformCacheDir(cacheDir, buildNumber, p), "resfileindex-os.txt")
}

func platformMergedPath(cacheDir, buildNumber string, p Platform) string {
	return filepath.Join(platformCacheDir(cacheDir, buildNumber, p), "platform-merged.json")
}

func metaPath(cacheDir, buildNumber string) string {
	return filepath.Join(cacheDir, buildNumber, "meta.json")
}

func readCachedText(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(data), true, nil
}

func writeCachedText(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func readPlatformMerged(path string) (map[string]string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var merged map[string]string
	if err := json.Unmarshal(data, &merged); err != nil {
		return nil, false, fmt.Errorf("decode %s: %w", path, err)
	}
	return merged, true, nil
}

func writePlatformMerged(path string, merged map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readMeta(path string) (CacheMeta, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CacheMeta{}, false, nil
		}
		return CacheMeta{}, false, err
	}

	var meta CacheMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return CacheMeta{}, false, fmt.Errorf("decode %s: %w", path, err)
	}
	return meta, true, nil
}

func writeMeta(path string, meta CacheMeta) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
