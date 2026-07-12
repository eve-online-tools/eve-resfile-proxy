package alias

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

// FS applies alias rules before delegating to an underlying filesystem.
type FS struct {
	fsys  fs.FS
	paths []compiledPath
	exts  []compiledExtension
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.GlobFS     = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
)

func (f *FS) Open(name string) (fs.File, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}
	target, _, err := f.resolve(cleaned)
	if err != nil {
		return nil, err
	}
	return f.fsys.Open(target)
}

func (f *FS) Stat(name string) (fs.FileInfo, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}
	target, aliased, err := f.resolve(cleaned)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(f.fsys, target)
	if err != nil {
		return nil, err
	}
	if aliased {
		return aliasedFileInfo(path.Base(cleaned), info), nil
	}
	return info, nil
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}
	target, _, err := f.resolve(cleaned)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(f.fsys, target)
}

func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	dir, err := vfspath.CleanDir(name)
	if err != nil {
		return nil, err
	}

	if target, ok := resolvePath(dir, f.paths, f.exts); ok && target != dir {
		entries, err := readDirFS(f.fsys, target)
		if err != nil {
			return nil, err
		}
		return f.mergeReadDir(dir, target, entries)
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

	for _, entry := range virtualPathEntriesForDir(f.fsys, dir, f.paths) {
		if _, exists := byName[entry.Name()]; !exists {
			byName[entry.Name()] = entry
		}
	}

	f.mergeExtensionReadDir(dir, dir, byName)

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

func (f *FS) mergeReadDir(clientDir, underlyingDir string, underlying []fs.DirEntry) ([]fs.DirEntry, error) {
	byName := map[string]fs.DirEntry{}
	for _, entry := range underlying {
		byName[entry.Name()] = entry
	}
	f.mergeExtensionReadDir(clientDir, underlyingDir, byName)

	entries := make([]fs.DirEntry, 0, len(byName))
	for _, entry := range byName {
		entries = append(entries, entry)
	}
	fsutil.SortDirEntries(entries)
	return entries, nil
}

func (f *FS) mergeExtensionReadDir(clientDir, underlyingDir string, byName map[string]fs.DirEntry) {
	hide := map[string]struct{}{}

	for _, rule := range f.exts {
		for name, entry := range byName {
			if entry.IsDir() || !strings.HasSuffix(name, rule.targetExt) {
				continue
			}
			aliasName := swapExtension(name, rule.targetExt, rule.aliasExt)
			clientPath := joinPath(clientDir, aliasName)
			if !rule.matches(clientPath) {
				continue
			}
			if f.realFileExists(clientPath) {
				continue
			}
			hide[name] = struct{}{}
			if _, exists := byName[aliasName]; !exists {
				byName[aliasName] = virtualExtensionFileEntry(f.fsys, aliasName, joinPath(underlyingDir, name))
			}
		}
	}

	for name := range hide {
		delete(byName, name)
	}
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

	gFS := gFSOrNil(f.fsys)
	if gFS != nil {
		direct, err := gFS.Glob(pattern)
		if err != nil {
			return nil, err
		}
		addMatches(direct)
	}

	for _, rule := range f.paths {
		aliasMatches, err := globThroughPathRule(gFS, pattern, rule)
		if err != nil {
			return nil, err
		}
		addMatches(aliasMatches)
	}

	for _, rule := range f.exts {
		aliasMatches, err := globThroughExtensionRule(gFS, pattern, rule, f.paths)
		if err != nil {
			return nil, err
		}
		addMatches(aliasMatches)
	}

	sort.Strings(matches)
	return matches, nil
}

func (f *FS) resolve(name string) (target string, aliased bool, err error) {
	if info, statErr := fs.Stat(f.fsys, name); statErr == nil && !info.IsDir() {
		return name, false, nil
	}
	if target, ok := resolvePath(name, f.paths, f.exts); ok {
		return target, true, nil
	}
	return name, false, nil
}

func (f *FS) realFileExists(name string) bool {
	info, err := fs.Stat(f.fsys, name)
	return err == nil && !info.IsDir()
}

func gFSOrNil(fsys fs.FS) fs.GlobFS {
	gFS, _ := fsys.(fs.GlobFS)
	return gFS
}

func globThroughPathRule(gFS fs.GlobFS, pattern string, rule compiledPath) ([]string, error) {
	if gFS == nil {
		return nil, nil
	}

	var translatedPattern string
	switch {
	case pattern == rule.alias:
		translatedPattern = rule.target
	case strings.HasPrefix(pattern, rule.alias+"/"):
		translatedPattern = path.Join(rule.target, strings.TrimPrefix(pattern, rule.alias+"/"))
	default:
		return nil, nil
	}

	childMatches, err := gFS.Glob(translatedPattern)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0, len(childMatches))
	for _, match := range childMatches {
		matches = append(matches, mapPathMatchToAlias(match, rule))
	}
	return matches, nil
}

func globThroughExtensionRule(gFS fs.GlobFS, pattern string, rule compiledExtension, paths []compiledPath) ([]string, error) {
	if gFS == nil || !strings.HasSuffix(pattern, rule.aliasExt) {
		return nil, nil
	}
	star := strings.Index(pattern, "*")
	if star < 0 {
		return nil, nil
	}

	patternPrefix := pattern[:star]
	testPath := joinPath(strings.TrimSuffix(patternPrefix, "/"), "x"+rule.aliasExt)
	if !rule.matches(testPath) {
		return nil, nil
	}

	underlyingPrefix := patternPrefix
	if trimmed := strings.TrimSuffix(patternPrefix, "/"); trimmed != "" {
		if resolved, ok := resolvePath(trimmed, paths, nil); ok {
			underlyingPrefix = resolved + "/"
		}
	}

	underlyingPattern := strings.TrimSuffix(pattern, rule.aliasExt) + rule.targetExt
	if underlyingPrefix != patternPrefix {
		underlyingPattern = underlyingPrefix + strings.TrimPrefix(strings.TrimSuffix(pattern, rule.aliasExt)+rule.targetExt, patternPrefix)
	}
	childMatches, err := gFS.Glob(underlyingPattern)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0, len(childMatches))
	for _, match := range childMatches {
		client := mapUnderlyingPathToClient(match, paths)
		matches = append(matches, swapExtension(client, rule.targetExt, rule.aliasExt))
	}
	return matches, nil
}

func mapUnderlyingPathToClient(name string, paths []compiledPath) string {
	for _, rule := range paths {
		if mapped := mapPathMatchToAlias(name, rule); mapped != name {
			return mapped
		}
	}
	return name
}

func mapPathMatchToAlias(match string, rule compiledPath) string {
	if match == rule.target {
		return rule.alias
	}
	if strings.HasPrefix(match, rule.target+"/") {
		return path.Join(rule.alias, strings.TrimPrefix(match, rule.target+"/"))
	}
	return match
}

func virtualPathEntriesForDir(fsys fs.FS, dir string, rules []compiledPath) []fs.DirEntry {
	seen := map[string]struct{}{}
	var entries []fs.DirEntry

	for _, rule := range rules {
		childName, isFile := virtualPathChildAt(dir, rule)
		if childName == "" {
			continue
		}
		if _, exists := seen[childName]; exists {
			continue
		}
		seen[childName] = struct{}{}

		if isFile {
			entries = append(entries, virtualExtensionFileEntry(fsys, childName, rule.target))
		} else {
			entries = append(entries, dirEntry{name: childName})
		}
	}

	return entries
}

func virtualExtensionFileEntry(fsys fs.FS, name, target string) fileEntry {
	entry := fileEntry{name: name}
	if info, err := fs.Stat(fsys, target); err == nil && !info.IsDir() {
		entry.info = aliasedFileInfo(name, info)
	}
	return entry
}

func virtualPathChildAt(dir string, rule compiledPath) (name string, isFile bool) {
	aliasPath := rule.alias
	switch dir {
	case ".":
		if rule.dir {
			if strings.Contains(aliasPath, "/") {
				seg, _ := vfspath.FirstSegment(aliasPath)
				return seg, false
			}
			return aliasPath, false
		}
		return aliasPath, true
	default:
		prefix := dir + "/"
		if !strings.HasPrefix(aliasPath, prefix) {
			return "", false
		}
		rest := strings.TrimPrefix(aliasPath, prefix)
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

func joinPath(dir, name string) string {
	if dir == "" || dir == "." {
		return name
	}
	return dir + "/" + name
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
