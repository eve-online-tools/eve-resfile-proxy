// Package path provides shared path normalization for vfs layers.
package path

import (
	"errors"
	"io/fs"
	"path"
	"strings"
)

// CleanFile normalizes a file path: ValidPath, Clean, lowercase, reject ".".
func CleanFile(name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	cleaned := strings.ToLower(path.Clean(name))
	if cleaned == "." {
		return "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	return cleaned, nil
}

// CleanDir normalizes a directory path for ReadDir.
func CleanDir(name string) (string, error) {
	if name == "" || name == "." {
		return ".", nil
	}
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	cleaned := strings.ToLower(path.Clean(name))
	if cleaned == "." {
		return ".", nil
	}
	return cleaned, nil
}

// CleanDirPrefix returns the directory prefix form used for manifest ReadDir.
func CleanDirPrefix(name string) (string, error) {
	if name == "" || name == "." {
		return "", nil
	}
	cleaned, err := CleanDir(name)
	if err != nil {
		return "", err
	}
	if cleaned == "." {
		return "", nil
	}
	return cleaned + "/", nil
}

// FirstSegment splits p at the first slash. tail is empty when p has no slash.
func FirstSegment(p string) (head, tail string) {
	if p == "" {
		return "", ""
	}
	if idx := strings.Index(p, "/"); idx >= 0 {
		return p[:idx], p[idx+1:]
	}
	return p, ""
}

// ValidateGlobPattern rejects glob patterns containing backslashes.
func ValidateGlobPattern(pattern string) error {
	if strings.Contains(pattern, "\\") {
		return errors.New("backslashes in glob pattern are not supported")
	}
	return nil
}
