package cors

import "net/http"

// Middleware adds permissive CORS headers so browsers can fetch assets
// cross-origin, and answers preflight OPTIONS requests itself (the method
// middleware would otherwise reject them). The origin is a static "*", so
// responses are origin-independent and no Vary: Origin is needed.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
			// Allow the conditional and range headers a client may send; a
			// cross-origin ranged fetch preflights on these.
			w.Header().Set("Access-Control-Allow-Headers",
				"Range, If-Range, If-None-Match, If-Modified-Since, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Expose the headers a client needs to read on the actual response.
		w.Header().Set("Access-Control-Expose-Headers",
			"ETag, Content-Length, Content-Range, Accept-Ranges, X-Eve-Build, X-Cache-Status")
		next.ServeHTTP(w, r)
	})
}
