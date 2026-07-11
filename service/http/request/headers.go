package request

import (
	"net/http"
)

// WriteResourceHeaders sets ETag, Last-Modified, and X-Cache-Status from res.
func WriteResourceHeaders(w http.ResponseWriter, res *Resource) {
	if res.ETag != "" {
		w.Header().Set("ETag", res.ETag)
	}
	if !res.LastModified.IsZero() {
		w.Header().Set("Last-Modified", res.LastModified.UTC().Format(http.TimeFormat))
	}
	w.Header().Set("X-Cache-Status", string(res.CacheStatus))
}
