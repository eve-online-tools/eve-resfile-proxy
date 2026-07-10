package index

import (
	"path"
	"sort"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/lookup"
)

type ListingEntry struct {
	Name           string
	IsDir          bool
	Size           int64
	CompressedSize int64
}

// DirPrefixFromURL converts a trailing-slash URL path to a lowercase res:/ prefix.
func DirPrefixFromURL(urlPath string) string {
	cleaned := path.Clean("/" + strings.TrimPrefix(urlPath, "/"))
	if cleaned == "/" || cleaned == "." {
		return "res:/"
	}
	rel := strings.TrimPrefix(cleaned, "/")
	return lookup.ResPathKey("res:/" + rel + "/")
}

func (s *IndexSet) ListDirectory(dirPrefix string, pref Platform) ([]ListingEntry, bool) {
	platforms := platformsForListing(s, pref)

	dirs := map[string]struct{}{}
	files := map[string]Entry{}

	for _, p := range platforms {
		m, loaded := s.PlatformMaps[p]
		if !loaded {
			continue
		}
		for resPath, entry := range m {
			if !strings.HasPrefix(resPath, dirPrefix) {
				continue
			}
			rest := strings.TrimPrefix(resPath, dirPrefix)
			if rest == "" {
				continue
			}
			if idx := strings.Index(rest, "/"); idx >= 0 {
				dirs[rest[:idx]] = struct{}{}
			} else {
				mergeListingFile(files, rest, entry)
			}
		}
	}

	if len(dirs) == 0 && len(files) == 0 {
		return nil, false
	}

	entries := make([]ListingEntry, 0, len(dirs)+len(files))
	for name := range dirs {
		entries = append(entries, ListingEntry{Name: name, IsDir: true})
	}
	for name, entry := range files {
		entries = append(entries, ListingEntry{
			Name:           name,
			Size:           entry.Size,
			CompressedSize: entry.CompressedSize,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, true
}

func mergeListingFile(files map[string]Entry, name string, entry Entry) {
	existing, ok := files[name]
	if !ok {
		files[name] = entry
		return
	}
	if existing.Size == 0 && entry.Size > 0 {
		existing.Size = entry.Size
	}
	if existing.CompressedSize == 0 && entry.CompressedSize > 0 {
		existing.CompressedSize = entry.CompressedSize
	}
	files[name] = existing
}

func platformsForListing(s *IndexSet, pref Platform) []Platform {
	if pref != "" {
		return []Platform{pref}
	}
	return s.LoadedPlatforms
}
