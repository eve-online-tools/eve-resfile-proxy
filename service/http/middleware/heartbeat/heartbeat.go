package heartbeat

import (
	"net/http"
	"strconv"
	"strings"
)

var heartbeatContent = []byte("ok")

func Middleware(endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.EqualFold(r.URL.Path, endpoint) {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Content-Length", strconv.Itoa(len(heartbeatContent)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(heartbeatContent)
		})
	}
}
