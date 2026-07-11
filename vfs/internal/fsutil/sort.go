package fsutil

import (
	"io/fs"
	"sort"
)

// SortDirEntries sorts directories before files, then by name.
func SortDirEntries(entries []fs.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		iDir := entries[i].IsDir()
		jDir := entries[j].IsDir()
		if iDir != jDir {
			return iDir
		}
		return entries[i].Name() < entries[j].Name()
	})
}
