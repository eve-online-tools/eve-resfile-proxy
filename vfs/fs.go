package vfs

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/internal/fsutil"
	vfspath "github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

type manifestFS struct {
	prefix   Prefix
	entries  map[string]Entry
	fetcher  Fetcher
	validate bool
}

var _ fs.FS = (*manifestFS)(nil)
var _ fs.ReadDirFS = (*manifestFS)(nil)
var _ fs.GlobFS = (*manifestFS)(nil)
var _ fs.ReadFileFS = (*manifestFS)(nil)
var _ fs.StatFS = (*manifestFS)(nil)

func (f *manifestFS) Open(name string) (fs.File, error) {
	fsPath, err := vfspath.CleanFile(cleanPrefix(name, f.prefix))
	if err != nil {
		return nil, err
	}

	entry, ok := f.entries[fsPath]
	if !ok {
		return nil, fs.ErrNotExist
	}

	if f.fetcher == nil {
		return nil, ErrFetchNotConfigured
	}

	data, err := f.fetcher.FetchEntry(context.Background(), entry)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	if f.validate && !entry.MD5.IsZero() {
		if md5.Sum(data) != entry.MD5.Sum() {
			return nil, &fs.PathError{Op: "open", Path: name, Err: ErrChecksumMismatch}
		}
	}

	return &manifestFile{
		name:  path.Base(fsPath),
		entry: entry,
		data:  data,
	}, nil
}

func (f *manifestFS) Stat(name string) (fs.FileInfo, error) {
	fsPath, err := vfspath.CleanFile(cleanPrefix(name, f.prefix))
	if err != nil {
		return nil, err
	}

	entry, ok := f.entries[fsPath]
	if !ok {
		return nil, fs.ErrNotExist
	}

	base := path.Base(fsPath)
	return manifestFileInfo{name: base, entry: entry}, nil
}

func (f *manifestFS) ReadDir(name string) ([]fs.DirEntry, error) {
	dirPrefix, err := dirPrefixFromName(cleanPrefix(name, f.prefix))
	if err != nil {
		return nil, err
	}

	dirs := map[string]struct{}{}
	files := map[string]Entry{}

	for fsPath, entry := range f.entries {
		if !strings.HasPrefix(fsPath, dirPrefix) {
			continue
		}
		rest := strings.TrimPrefix(fsPath, dirPrefix)
		if rest == "" {
			continue
		}
		if idx := strings.Index(rest, "/"); idx >= 0 {
			dirs[rest[:idx]] = struct{}{}
		} else {
			files[rest] = entry
		}
	}

	if len(dirs) == 0 && len(files) == 0 {
		return nil, fs.ErrNotExist
	}

	entries := make([]fs.DirEntry, 0, len(dirs)+len(files))
	for dirName := range dirs {
		entries = append(entries, manifestDirEntry{info: manifestDirInfo{name: dirName}})
	}
	for fileName, entry := range files {
		entries = append(entries, manifestDirEntry{
			info: manifestFileInfo{name: fileName, entry: entry},
		})
	}

	fsutil.SortDirEntries(entries)

	return entries, nil
}

func (f *manifestFS) ReadFile(name string) ([]byte, error) {

	file, err := f.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck // manifest file from Open
	return io.ReadAll(file)
}

func (f *manifestFS) Glob(pattern string) ([]string, error) {
	if err := vfspath.ValidateGlobPattern(pattern); err != nil {
		return nil, err
	}

	pattern = cleanPrefix(pattern, f.prefix)

	matches := make([]string, 0)
	for fsPath := range f.entries {
		ok, err := doublestar.Match(pattern, fsPath)
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", pattern, err)
		}
		if ok {
			matches = append(matches, fsPath)
		}
	}

	sort.Strings(matches)
	return matches, nil
}

func cleanPrefix(name string, prefix Prefix) string {
	if after, found := strings.CutPrefix(name, fmt.Sprintf("%s:/", prefix)); found {
		return after
	}
	return name
}

func dirPrefixFromName(name string) (string, error) {
	return vfspath.CleanDirPrefix(name)
}
