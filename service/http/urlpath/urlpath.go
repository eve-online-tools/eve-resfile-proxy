package urlpath

import (
	"path"
	"strings"

	vfspath "github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

// CleanURL returns a normalized HTTP URL path with a single leading slash.
// Duplicate slashes and dot segments are collapsed. A trailing slash on the
// input is preserved (except for the root path "/").
func CleanURL(urlPath string) string {
	trailing := strings.HasSuffix(urlPath, "/")
	if urlPath == "" {
		return "/"
	}
	cleaned := path.Clean(urlPath)
	if cleaned == "." {
		return "/"
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if trailing && cleaned != "/" && !strings.HasSuffix(cleaned, "/") {
		cleaned += "/"
	}
	return cleaned
}

// JoinURL joins base and name into a normalized absolute URL path.
func JoinURL(base, name string) string {
	base = CleanURL(base)
	if base == "/" {
		return "/" + name
	}
	return strings.TrimSuffix(base, "/") + "/" + name
}

// ToFS converts an HTTP URL path to a vfs filesystem path.
// Returns empty string for root or invalid paths.
func ToFS(urlPath string) (string, error) {
	cleaned := CleanURL(urlPath)
	if cleaned == "/" {
		return "", nil
	}
	return vfspath.CleanFile(strings.TrimPrefix(cleaned, "/"))
}

// DirFromURL returns the vfs directory path for a trailing-slash URL path.
func DirFromURL(urlPath string) (string, error) {
	cleaned := CleanURL(urlPath)
	trimmed := strings.TrimSuffix(cleaned, "/")
	if trimmed == "" || trimmed == "/" {
		return ".", nil
	}
	return vfspath.CleanDir(strings.TrimPrefix(trimmed, "/"))
}
