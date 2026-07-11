// Package rewrite wraps fs.FS and applies configurable path aliases.
package rewrite

import (
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/internal/fsutil"
	vfspath "github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

// FS applies rewrite rules before delegating to an underlying filesystem.
type FS struct {
	fsys  fs.FS
	rules []compiledRule
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.GlobFS     = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
)

// New wraps fsys with rewrite rules. Returns fsys unchanged when rules is empty.
func New(fsys fs.FS, rules []Rule) (fs.FS, error) {
	compiled, err := compileRules(rules)
	if err != nil {
		return nil, err
	}
	if len(compiled) == 0 {
		return fsys, nil
	}
	return &FS{fsys: fsys, rules: compiled}, nil
}

func (f *FS) Open(name string) (fs.File, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}
	if target, ok := translatePath(cleaned, f.rules); ok {
		return f.fsys.Open(target)
	}
	return f.fsys.Open(cleaned)
}

func (f *FS) Stat(name string) (fs.FileInfo, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}
	if target, ok := translatePath(cleaned, f.rules); ok {
		return fs.Stat(f.fsys, target)
	}
	return fs.Stat(f.fsys, cleaned)
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}
	target := cleaned
	if rewritten, ok := translatePath(cleaned, f.rules); ok {
		target = rewritten
	}

	return fs.ReadFile(f.fsys, target)
}

func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	dir, err := vfspath.CleanDir(name)
	if err != nil {
		return nil, err
	}

	if target, ok := translatePath(dir, f.rules); ok {
		return readDirFS(f.fsys, target)
	}

	byName := map[string]fs.DirEntry{}

	if rdFS, ok := f.fsys.(fs.ReadDirFS); ok {
		entries, err := rdFS.ReadDir(dir)
		if err == nil {
			for _, entry := range entries {
				byName[entry.Name()] = entry
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	for _, entry := range virtualEntriesForDir(f.fsys, dir, f.rules) {
		if _, exists := byName[entry.Name()]; !exists {
			byName[entry.Name()] = entry
		}
	}

	if len(byName) == 0 {
		return nil, fs.ErrNotExist
	}

	entries := make([]fs.DirEntry, 0, len(byName))
	for _, entry := range byName {
		entries = append(entries, entry)
	}
	fsutil.SortDirEntries(entries)
	return entries, nil
}

func (f *FS) Glob(pattern string) ([]string, error) {
	if err := vfspath.ValidateGlobPattern(pattern); err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var matches []string

	addMatches := func(items []string) {
		for _, item := range items {
			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			matches = append(matches, item)
		}
	}

	if gFS, ok := f.fsys.(fs.GlobFS); ok {
		direct, err := gFS.Glob(pattern)
		if err != nil {
			return nil, err
		}
		addMatches(direct)
	}

	for _, rule := range f.rules {
		aliasMatches, err := globThroughRule(gFSOrNil(f.fsys), pattern, rule)
		if err != nil {
			return nil, err
		}
		addMatches(aliasMatches)
	}

	sort.Strings(matches)
	return matches, nil
}

func gFSOrNil(fsys fs.FS) fs.GlobFS {
	gFS, _ := fsys.(fs.GlobFS)
	return gFS
}

func globThroughRule(gFS fs.GlobFS, pattern string, rule compiledRule) ([]string, error) {
	if gFS == nil {
		return nil, nil
	}

	var translatedPattern string
	switch {
	case pattern == rule.from:
		translatedPattern = rule.to
	case strings.HasPrefix(pattern, rule.from+"/"):
		translatedPattern = path.Join(rule.to, strings.TrimPrefix(pattern, rule.from+"/"))
	default:
		return nil, nil
	}

	childMatches, err := gFS.Glob(translatedPattern)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0, len(childMatches))
	for _, match := range childMatches {
		matches = append(matches, mapMatchToAlias(match, rule))
	}
	return matches, nil
}

func mapMatchToAlias(match string, rule compiledRule) string {
	if match == rule.to {
		return rule.from
	}
	if strings.HasPrefix(match, rule.to+"/") {
		return path.Join(rule.from, strings.TrimPrefix(match, rule.to+"/"))
	}
	return match
}

func virtualEntriesForDir(fsys fs.FS, dir string, rules []compiledRule) []fs.DirEntry {
	seen := map[string]struct{}{}
	var entries []fs.DirEntry

	for _, rule := range rules {
		childName, isFile := virtualChildAt(dir, rule.from)
		if childName == "" {
			continue
		}
		if _, exists := seen[childName]; exists {
			continue
		}
		seen[childName] = struct{}{}

		if isFile {
			entries = append(entries, virtualFileEntry(fsys, childName, rule.to))
		} else {
			entries = append(entries, dirEntry{name: childName})
		}
	}

	return entries
}

func virtualFileEntry(fsys fs.FS, name, target string) fileEntry {
	entry := fileEntry{name: name}
	if info, err := fs.Stat(fsys, target); err == nil && !info.IsDir() {
		entry.info = aliasedFileInfo(name, info)
	}
	return entry
}

func virtualChildAt(dir, from string) (name string, isFile bool) {
	switch dir {
	case ".":
		if strings.Contains(from, "/") {
			seg, _ := vfspath.FirstSegment(from)
			return seg, false
		}
		return from, true
	default:
		prefix := dir + "/"
		if !strings.HasPrefix(from, prefix) {
			return "", false
		}
		rest := strings.TrimPrefix(from, prefix)
		seg, _ := vfspath.FirstSegment(rest)
		if seg == "" {
			return "", false
		}
		return seg, false
	}
}

func readDirFS(fsys fs.FS, name string) ([]fs.DirEntry, error) {
	if rdFS, ok := fsys.(fs.ReadDirFS); ok {
		return rdFS.ReadDir(name)
	}
	return nil, fs.ErrNotExist
}

type dirEntry struct {
	name string
}

func (d dirEntry) Name() string               { return d.name }
func (d dirEntry) IsDir() bool                { return true }
func (d dirEntry) Type() fs.FileMode          { return fs.ModeDir }
func (d dirEntry) Info() (fs.FileInfo, error) { return dirInfo(d), nil }

type fileEntry struct {
	name string
	info fs.FileInfo
}

func (f fileEntry) Name() string { return f.name }
func (f fileEntry) IsDir() bool  { return false }
func (f fileEntry) Type() fs.FileMode {
	if f.info != nil {
		return f.info.Mode().Type()
	}
	return 0
}

func (f fileEntry) Info() (fs.FileInfo, error) {
	if f.info != nil {
		return f.info, nil
	}
	return fileInfo{name: f.name}, nil
}

type dirInfo struct {
	name string
}

func (d dirInfo) Name() string       { return d.name }
func (d dirInfo) Size() int64        { return 0 }
func (d dirInfo) Mode() fs.FileMode  { return fs.ModeDir }
func (d dirInfo) ModTime() time.Time { return time.Time{} }
func (d dirInfo) IsDir() bool        { return true }
func (d dirInfo) Sys() any           { return nil }

type fileInfo struct {
	name string
}

func (f fileInfo) Name() string       { return f.name }
func (f fileInfo) Size() int64        { return 0 }
func (f fileInfo) Mode() fs.FileMode  { return 0 }
func (f fileInfo) ModTime() time.Time { return time.Time{} }
func (f fileInfo) IsDir() bool        { return false }
func (f fileInfo) Sys() any           { return nil }

type namedFileInfo struct {
	name  string
	inner fs.FileInfo
}

var _ vfs.CompressedSizeInfo = (*namedFileInfo)(nil)

func aliasedFileInfo(name string, inner fs.FileInfo) fs.FileInfo {
	return &namedFileInfo{name: name, inner: inner}
}

func (n *namedFileInfo) Name() string       { return n.name }
func (n *namedFileInfo) Size() int64        { return n.inner.Size() }
func (n *namedFileInfo) Mode() fs.FileMode  { return n.inner.Mode() }
func (n *namedFileInfo) ModTime() time.Time { return n.inner.ModTime() }
func (n *namedFileInfo) IsDir() bool        { return n.inner.IsDir() }
func (n *namedFileInfo) Sys() any           { return n.inner.Sys() }

func (n *namedFileInfo) CompressedSize() int64 {
	if li, ok := n.inner.(vfs.CompressedSizeInfo); ok {
		return li.CompressedSize()
	}
	return 0
}
