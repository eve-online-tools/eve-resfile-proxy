package handler

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/eve-online-tools/eve-resfile-proxy/common/mimetype"
	"github.com/eve-online-tools/eve-resfile-proxy/service/buildnumber"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/load"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/request"
)

func Respond(build *buildnumber.BuildNumber) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, ok := request.ResourceFromContext(r.Context())
		if !ok {
			return
		}

		w.Header().Set("Content-Type", mimetype.ForFilename(res.FSPath))
		w.Header().Set("Cache-Control", "public, max-age=3600")
		request.WriteResourceHeaders(w, res)
		buildNumber := ""
		if build != nil {
			buildNumber = build.Get()
		}
		w.Header().Set("X-Eve-Build", buildNumber)

		if r.Method == http.MethodGet {
			data, err := res.Data()
			if err != nil {
				load.WriteReadError(w, err)
				return
			}
			// ServeContent handles Content-Length, Accept-Ranges, range
			// requests (206/416), and If-Range; it respects the Content-Type
			// we set above. Assets are whole-file in memory, so serving from a
			// bytes.Reader adds no cost.
			http.ServeContent(w, r, res.FSPath, res.LastModified, bytes.NewReader(data))
			return
		}

		// HEAD: report length from the (transform-aware) resource size without
		// loading the body, and advertise range support.
		contentLength, _, err := res.ContentLength(false)
		if err != nil {
			load.WriteReadError(w, err)
			return
		}
		w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
	})
}
