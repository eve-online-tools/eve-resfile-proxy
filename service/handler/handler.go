package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

var contentTypes = map[string]string{
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".dds":   "image/vnd-ms.dds",
	".mp3":   "audio/mpeg",
	".ogg":   "audio/ogg",
	".wav":   "audio/wav",
	".webm":  "video/webm",
	".mp4":   "video/mp4",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".json":  "application/json",
	".txt":   "text/plain",
}

func New(indexSet *index.IndexSet) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			return
		}

		w.Header().Set("Content-Type", contentTypeForPath(asset.ResPath))
		w.Header().Set("Cache-Control", "public, max-age=3600")
		if asset.ETag != "" {
			w.Header().Set("ETag", asset.ETag)
		}
		if !asset.LastModified.IsZero() {
			w.Header().Set("Last-Modified", asset.LastModified.UTC().Format(http.TimeFormat))
		}
		w.Header().Set("X-Cache-Status", string(asset.CacheStatus))
		w.Header().Set("X-Eve-Build", indexSet.BuildNumber)
		w.Header().Set("X-Eve-Platform", string(asset.Platform))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(asset.Data)
	})
}

func contentTypeForPath(resPath string) string {
	ext := strings.ToLower(filepath.Ext(resPath))
	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}
	return "application/octet-stream"
}
