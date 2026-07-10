package mimetype

import (
	"path/filepath"
	"strings"
)

// Extension map derived from TQ resfile indexes (windows + macos, build 3430261).
// Unlisted extensions still fall back to application/octet-stream.
var byExtension = map[string]string{
	// Images
	".dds":  "image/vnd-ms.dds",
	".gif":  "image/gif",
	".ico":  "image/x-icon",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".tga":  "image/x-tga",
	".webp": "image/webp",

	// Audio / video
	".mp3":  "audio/mpeg",
	".mp4":  "video/mp4",
	".ogg":  "audio/ogg",
	".wav":  "audio/wav",
	".webm": "video/webm",

	// Fonts
	".otf":   "font/otf",
	".ttf":   "font/ttf",
	".woff":  "font/woff",
	".woff2": "font/woff2",

	// Text / structured
	".css":  "text/css",
	".json": "application/json",
	".py":   "text/x-python",
	".txt":  "text/plain",
	".xml":  "application/xml",
	".yaml": "text/yaml",

	// Other
	".apb":       "application/octet-stream",
	".base":      "application/octet-stream",
	".black":     "application/octet-stream",
	".bnk":       "application/octet-stream",
	".color":     "application/octet-stream",
	".face":      "application/octet-stream",
	".fsdbinary": "application/octet-stream",
	".gr2":       "application/octet-stream",
	".gsf":       "application/octet-stream",
	".info":      "application/octet-stream",
	".pathdata":  "application/octet-stream",
	".pickle":    "application/octet-stream",
	".pose":      "application/octet-stream",
	".proj":      "application/octet-stream",
	".prs":       "application/octet-stream",
	".region":    "application/octet-stream",
	".schema":    "application/octet-stream",
	".sm_depth":  "application/octet-stream",
	".sm_hi":     "application/octet-stream",
	".sm_lo":     "application/octet-stream",
	".srt":       "application/octet-stream",
	".static":    "application/octet-stream",
	".tri":       "application/octet-stream",
	".trif":      "application/octet-stream",
	".type":      "application/octet-stream",
	".vta":       "application/octet-stream",
	".wem":       "application/octet-stream",
}

// ForFilename returns the MIME type for a file name based on its extension.
func ForFilename(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if contentType, ok := byExtension[ext]; ok {
		return contentType
	}
	return "application/octet-stream"
}
