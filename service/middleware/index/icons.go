package index

import (
	"embed"
	"encoding/base64"
	"html/template"
	"path/filepath"
	"strings"
)

//go:embed icons/*.svg
var iconFS embed.FS

var (
	iconURIs   map[string]template.URL
	extIconKey map[string]string
)

func init() {
	iconURIs = make(map[string]template.URL)

	entries, err := iconFS.ReadDir("icons")
	if err != nil {
		panic("index icons: read icons dir: " + err.Error())
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".svg") {
			continue
		}

		data, err := iconFS.ReadFile("icons/" + entry.Name())
		if err != nil {
			panic("index icons: read " + entry.Name() + ": " + err.Error())
		}

		key := strings.TrimSuffix(entry.Name(), ".svg")
		iconURIs[key] = svgDataURI(data)
	}

	extIconKey = map[string]string{
		".png":    "file_type_image",
		".jpg":    "file_type_image",
		".jpeg":   "file_type_image",
		".gif":    "file_type_image",
		".webp":   "file_type_image",
		".ico":    "file_type_image",
		".dds":    "file_type_image",
		".svg":    "file_type_image",
		".json":   "file_type_json",
		".txt":    "default_file",
		".mp3":    "file_type_audio",
		".ogg":    "file_type_audio",
		".wav":    "file_type_audio",
		".mp4":    "file_type_flash",
		".webm":   "file_type_flash",
		".woff":   "file_type_font",
		".woff2":  "file_type_font",
		".ttf":    "file_type_font",
		".shader": "file_type_shaderlab",
	}
}

func svgDataURI(data []byte) template.URL {
	encoded := base64.StdEncoding.EncodeToString(data)
	return template.URL("data:image/svg+xml;base64," + encoded)
}

func iconURI(key string) template.URL {
	if uri, ok := iconURIs[key]; ok {
		return uri
	}
	return iconURIs["default_file"]
}

// IconFor returns the embedded SVG data URI for a listing entry.
func IconFor(name string, isDir bool) template.URL {
	if isDir {
		return iconURI("default_folder")
	}

	ext := strings.ToLower(filepath.Ext(name))
	if key, ok := extIconKey[ext]; ok {
		return iconURI(key)
	}

	return iconURI("default_file")
}

// ParentIcon returns the icon for the parent ("../") directory row.
func ParentIcon() template.URL {
	return iconURI("default_root_folder")
}
