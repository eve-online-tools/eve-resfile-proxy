package handler

import (
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

		contentLength, data, err := res.ContentLength(r.Method == http.MethodGet)
		if err != nil {
			load.WriteReadError(w, err)
			return
		}

		w.Header().Set("Content-Type", mimetype.ForFilename(res.FSPath))
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
		request.WriteResourceHeaders(w, res)
		buildNumber := ""
		if build != nil {
			buildNumber = build.Get()
		}
		w.Header().Set("X-Eve-Build", buildNumber)
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	})
}
