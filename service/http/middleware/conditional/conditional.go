package conditional

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/load"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/request"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, ok := request.ResourceFromContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		if needsETag(r) {
			if res.HasChecksum {
				res.ETag = ETagForChecksum(res.Checksum)
			} else {
				data, err := res.Data()
				if err != nil {
					load.WriteReadError(w, err)
					return
				}
				res.ETag = ETagFor(data)
			}
		}

		if IsNotModified(r, res.ETag, res.LastModified) {
			WriteNotModified(w, res)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func needsETag(r *http.Request) bool {
	return r.Method == http.MethodGet || r.Header.Get("If-None-Match") != ""
}

func ETagForChecksum(sum [md5.Size]byte) string {
	return fmt.Sprintf(`"%x"`, sum)
}

func ETagFor(data []byte) string {
	return ETagForChecksum(md5.Sum(data))
}

func IsNotModified(r *http.Request, etag string, lastModified time.Time) bool {
	if match := r.Header.Get("If-None-Match"); match != "" {
		if etag != "" && etagMatches(match, etag) {
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

func WriteNotModified(w http.ResponseWriter, res *request.Resource) {
	request.WriteResourceHeaders(w, res)
	w.WriteHeader(http.StatusNotModified)
}
