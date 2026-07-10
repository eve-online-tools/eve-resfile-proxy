package handler

import (
	"net/http"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/internal/mimetype"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func New(indexSet *index.IndexSet) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			return
		}

		w.Header().Set("Content-Type", mimetype.ForFilename(asset.ResPath))
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
