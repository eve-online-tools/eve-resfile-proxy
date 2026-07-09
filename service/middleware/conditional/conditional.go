package conditional

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		asset.ETag = etagFor(asset.Data)
		ctx := request.WithAsset(r.Context(), asset)

		if isNotModified(r, asset.ETag, asset.LastModified) {
			writeNotModified(w, asset)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func etagFor(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf(`"%x"`, sum)
}

func isNotModified(r *http.Request, etag string, lastModified time.Time) bool {
	if match := r.Header.Get("If-None-Match"); match != "" {
		if etagMatches(match, etag) {
			return true
		}
	}

	ifModifiedSince := r.Header.Get("If-Modified-Since")
	if ifModifiedSince != "" && !lastModified.IsZero() {
		if t, err := http.ParseTime(ifModifiedSince); err == nil {
			last := lastModified.UTC().Truncate(time.Second)
			since := t.UTC().Truncate(time.Second)
			if !last.After(since) {
				return true
			}
		}
	}

	return false
}

func etagMatches(header, etag string) bool {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "*" || part == etag {
			return true
		}
	}
	return false
}

func writeNotModified(w http.ResponseWriter, asset request.Asset) {
	if asset.ETag != "" {
		w.Header().Set("ETag", asset.ETag)
	}
	if !asset.LastModified.IsZero() {
		w.Header().Set("Last-Modified", asset.LastModified.UTC().Format(http.TimeFormat))
	}
	w.Header().Set("X-Cache-Status", string(asset.CacheStatus))
	w.WriteHeader(http.StatusNotModified)
}
