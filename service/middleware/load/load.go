package load

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/fetch"
	"github.com/eve-online-tools/eve-resfile-proxy/service/assetcache"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func Middleware(cache *assetcache.Store, client *fetch.Client, assetOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			asset, ok := request.AssetFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			var data []byte
			found := false
			var cacheStatus request.CacheStatus

			if cache != nil {
				var err error
				data, found, err = cache.Read(asset.CDNPath)
				if err != nil {
					http.Error(w, "Failed to read cache.", http.StatusInternalServerError)
					return
				}
				if found {
					cacheStatus = request.CacheStatusHit
					if modTime, ok, err := cache.ModTime(asset.CDNPath); err != nil {
						http.Error(w, "Failed to read cache.", http.StatusInternalServerError)
						return
					} else if ok {
						asset.LastModified = modTime
					}
				}
			}

			if !found {
				url := strings.TrimRight(assetOrigin, "/") + "/" + strings.TrimPrefix(asset.CDNPath, "/")
				var err error
				data, err = client.GetBytes(r.Context(), url)
				if err != nil {
					http.Error(w, fmt.Sprintf("Failed to fetch asset: %v", err), http.StatusBadGateway)
					return
				}
				cacheStatus = request.CacheStatusMiss
				if cache != nil {
					_ = cache.Write(asset.CDNPath, data)
				}
			}

			asset.Data = data
			asset.CacheStatus = cacheStatus
			next.ServeHTTP(w, r.WithContext(request.WithAsset(r.Context(), asset)))
		})
	}
}
