package transform

import (
	"net/http"

	xform "github.com/eve-online-tools/eve-resfile-proxy/internal/transform"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/conditional"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func Middleware(engine *xform.Engine) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if engine == nil {
				next.ServeHTTP(w, r)
				return
			}

			asset, ok := request.AssetFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			result, err := engine.Transform(r.Context(), xform.Input{
				ResPath:  asset.ResPath,
				CDNPath:  asset.CDNPath,
				Platform: asset.Platform,
				Data:     asset.Data,
			})
			if err != nil {
				http.Error(w, "Transform failed.", http.StatusInternalServerError)
				return
			}

			if result.FromCache {
				asset.ETag = conditional.ETagFor(result.Data)
				if conditional.IsNotModified(r, asset.ETag, asset.LastModified) {
					conditional.WriteNotModified(w, asset)
					return
				}
			}

			asset.Data = result.Data
			next.ServeHTTP(w, r.WithContext(request.WithAsset(r.Context(), asset)))
		})
	}
}
