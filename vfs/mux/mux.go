// Package mux overlays multiple fs.FS values at configurable mount paths.
//
// Mount is not safe to call concurrently with reads on the same Mux.
package mux

import (
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs/internal/fsutil"
	vfspath "github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

type mount struct {
	base string
	fsys fs.FS
}

// Mux routes fs operations across mounted filesystems in registration order.
type Mux struct {
	mounts []mount
}

var (
	_ fs.FS         = (*Mux)(nil)
	_ fs.StatFS     = (*Mux)(nil)
	_ fs.ReadDirFS  = (*Mux)(nil)
	_ fs.GlobFS     = (*Mux)(nil)
	_ fs.ReadFileFS = (*Mux)(nil)
)

// NewMux creates a mux. Each optional layer is mounted at "/".
func NewMux(layers ...fs.FS) *Mux {
	m := &Mux{}
	for _, layer := range layers {
		_ = m.Mount("/", layer)
	}
	return m
}

// Mount attaches fsys at base. "" and "/" are root overlays.
// Multiple mounts at the same base overlap; first-registration wins on collision.
func (m *Mux) Mount(base string, fsys fs.FS) error {
	normalized, err := normalizeBase(base)
	if err != nil {
		return err
	}
	m.mounts = append(m.mounts, mount{base: normalized, fsys: fsys})
	return nil
}

func (m *Mux) Open(name string) (fs.File, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}

	for _, mt := range m.mounts {
		rel, ok := mt.relPath(cleaned)
		if !ok {
			continue
		}
		f, err := mt.fsys.Open(rel)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, fs.ErrNotExist
}

func (m *Mux) Stat(name string) (fs.FileInfo, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}

	for _, mt := range m.mounts {
		rel, ok := mt.relPath(cleaned)
		if !ok {
			continue
		}
		sfs, ok := mt.fsys.(fs.StatFS)
		if !ok {
			continue
		}
		info, err := sfs.Stat(rel)
		if err == nil {
			return info, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, fs.ErrNotExist
}

func (m *Mux) ReadFile(name string) ([]byte, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}

	for _, mt := range m.mounts {
		rel, ok := mt.relPath(cleaned)
		if !ok {
			continue
		}
		data, err := fs.ReadFile(mt.fsys, rel)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, fs.ErrNotExist
}

func (m *Mux) ReadDir(name string) ([]fs.DirEntry, error) {
	dir, err := vfspath.CleanDir(name)
	if err != nil {
		return nil, err
	}

	byName := map[string]fs.DirEntry{}
	found := false

	for _, mt := range m.mounts {
		entries, ok, err := mt.readDirAt(dir)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		found = true
		for _, entry := range entries {
			if _, exists := byName[entry.Name()]; !exists {
				byName[entry.Name()] = entry
			}
		}
	}

	if !found {
		return nil, fs.ErrNotExist
	}

	entries := make([]fs.DirEntry, 0, len(byName))
	for _, entry := range byName {
		entries = append(entries, entry)
	}
	fsutil.SortDirEntries(entries)
	return entries, nil
}

func (m *Mux) Glob(pattern string) ([]string, error) {
	if err := vfspath.ValidateGlobPattern(pattern); err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var matches []string

	for _, mt := range m.mounts {
		layerMatches, err := mt.glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range layerMatches {
			if _, exists := seen[match]; exists {
				continue
			}
			seen[match] = struct{}{}
			matches = append(matches, match)
		}
	}

	sort.Strings(matches)
	return matches, nil
}

func (mt mount) relPath(name string) (string, bool) {
	if mt.base == "" {
		return name, true
	}
	if name == mt.base {
		return ".", true
	}
	if strings.HasPrefix(name, mt.base+"/") {
		return strings.TrimPrefix(name, mt.base+"/"), true
	}
	return "", false
}

func (mt mount) readDirAt(dir string) ([]fs.DirEntry, bool, error) {
	if mt.base == "" {
		rdFS, ok := mt.fsys.(fs.ReadDirFS)
		if !ok {
			return nil, false, nil
		}
		entries, err := rdFS.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		return entries, true, nil
	}

	if dir == "." {
		next, _ := vfspath.FirstSegment(mt.base)
		if next == "" {
			return nil, false, nil
		}
		return []fs.DirEntry{dirEntry{name: next}}, true, nil
	}

	if dir == mt.base {
		return mt.readDirChild(".")
	}

	if strings.HasPrefix(mt.base, dir+"/") {
		rest := strings.TrimPrefix(mt.base, dir+"/")
		next, _ := vfspath.FirstSegment(rest)
		if next == "" {
			return nil, false, nil
		}
		return []fs.DirEntry{dirEntry{name: next}}, true, nil
	}

	if strings.HasPrefix(dir, mt.base+"/") {
		rel := strings.TrimPrefix(dir, mt.base+"/")
		return mt.readDirChild(rel)
	}

	return nil, false, nil
}

func (mt mount) readDirChild(rel string) ([]fs.DirEntry, bool, error) {
	rdFS, ok := mt.fsys.(fs.ReadDirFS)
	if !ok {
		return nil, false, nil
	}
	entries, err := rdFS.ReadDir(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return entries, true, nil
}

func (mt mount) glob(pattern string) ([]string, error) {
	gFS, ok := mt.fsys.(fs.GlobFS)
	if !ok {
		return nil, nil
	}

	if mt.base == "" {
		return gFS.Glob(pattern)
	}

	if pattern == mt.base {
		entries, ok, err := mt.readDirChild(".")
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}
		var matches []string
		for _, entry := range entries {
			matches = append(matches, path.Join(mt.base, entry.Name()))
		}
		return matches, nil
	}

	if after, ok := strings.CutPrefix(pattern, mt.base+"/"); ok {
		childMatches, err := gFS.Glob(after)
		if err != nil {
			return nil, err
		}
		matches := make([]string, len(childMatches))
		for i, match := range childMatches {
			matches[i] = path.Join(mt.base, match)
		}
		return matches, nil
	}

	childMatches, err := gFS.Glob(pattern)
	if err != nil {
		return nil, err
	}
	matches := make([]string, 0, len(childMatches))
	for _, match := range childMatches {
		matches = append(matches, path.Join(mt.base, match))
	}
	return matches, nil
}

func normalizeBase(base string) (string, error) {
	if base == "/" || base == "" {
		return "", nil
	}
	if !fs.ValidPath(base) {
		return "", &fs.PathError{Op: "mount", Path: base, Err: fs.ErrInvalid}
	}
	cleaned := path.Clean(base)
	if cleaned == "." || cleaned == ".." {
		return "", &fs.PathError{Op: "mount", Path: base, Err: fs.ErrInvalid}
	}
	for _, seg := range strings.Split(cleaned, "/") {
		if seg == ".." {
			return "", &fs.PathError{Op: "mount", Path: base, Err: fs.ErrInvalid}
		}
	}
	return cleaned, nil
}

type dirEntry struct {
	name string
}

func (d dirEntry) Name() string               { return d.name }
func (d dirEntry) IsDir() bool                { return true }
func (d dirEntry) Type() fs.FileMode          { return fs.ModeDir }
func (d dirEntry) Info() (fs.FileInfo, error) { return dirInfo(d), nil }

type dirInfo struct {
	name string
}

func (d dirInfo) Name() string       { return d.name }
func (d dirInfo) Size() int64        { return 0 }
func (d dirInfo) Mode() fs.FileMode  { return fs.ModeDir }
func (d dirInfo) ModTime() time.Time { return time.Time{} }
func (d dirInfo) IsDir() bool        { return true }
func (d dirInfo) Sys() any           { return nil }
