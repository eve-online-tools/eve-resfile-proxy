package lookup

import (
	"path"
	"strings"
)

const resPrefix = "res:/"

// ResPathKey lowercases a res:/ path for index lookup.
func ResPathKey(resPath string) string {
	return strings.ToLower(resPath)
}

// FromURLPath converts an HTTP path to a lowercase res:/ lookup key.
func FromURLPath(urlPath string) string {
	cleaned := path.Clean("/" + strings.TrimPrefix(urlPath, "/"))
	if cleaned == "/" || cleaned == "." {
		return ""
	}
	return ResPathKey(resPrefix + strings.TrimPrefix(cleaned, "/"))
}
