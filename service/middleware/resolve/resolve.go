package resolve

import (
	"net/http"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/internal/lookup"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func Middleware(indexSet *index.IndexSet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pref, err := index.ParsePlatform(r.URL.Query().Get("platform"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			resPath := lookup.FromURLPath(r.URL.Path)
			if resPath == "" {
				http.Error(w, "Resfile not found.", http.StatusNotFound)
				return
			}

			cdnPath, hitPlatform, ok := indexSet.Lookup(resPath, pref)
			if !ok {
				http.Error(w, "Resfile not found.", http.StatusNotFound)
				return
			}

			ctx := request.WithAsset(r.Context(), request.Asset{
				ResPath:  resPath,
				CDNPath:  cdnPath,
				Platform: hitPlatform,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
