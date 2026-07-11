package conditional_test

import (
	"crypto/md5"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/conditional"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/request"
)

func preloadedWithChecksum(t *testing.T, fsPath string, data []byte) *request.Resource {
	t.Helper()
	res := request.NewPreloadedResource(fsPath, data)
	res.Checksum = md5.Sum(data)
	res.HasChecksum = true
	return res
}

func TestMiddlewareIfNoneMatchReturns304(t *testing.T) {
	data := []byte("png-data")
	etag := conditional.ETagFor(data)

	handler := conditional.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	ctx := request.WithResource(t.Context(), preloadedWithChecksum(t, "icons/64/icon.png", data))
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("ETag") != etag {
		t.Fatalf("etag = %q", rec.Header().Get("ETag"))
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestMiddlewareIfNoneMatchSkipsDataWhenChecksumPresent(t *testing.T) {
	data := []byte("png-data")
	etag := conditional.ETagForChecksum(md5.Sum(data))

	handler := conditional.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	// No preload: Data would fail if called (nil FS).
	res := &request.Resource{
		FSPath:      "icons/64/icon.png",
		Checksum:    md5.Sum(data),
		HasChecksum: true,
	}
	ctx := request.WithResource(t.Context(), res)
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("ETag") != etag {
		t.Fatalf("etag = %q", rec.Header().Get("ETag"))
	}
}

func TestMiddlewareIfModifiedSinceReturns304(t *testing.T) {
	lastModified := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	handler := conditional.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	res := preloadedWithChecksum(t, "icons/64/icon.png", []byte("png-data"))
	res.LastModified = lastModified
	ctx := request.WithResource(t.Context(), res)
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	req.Header.Set("If-Modified-Since", lastModified.Format(http.TimeFormat))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Last-Modified") != lastModified.Format(http.TimeFormat) {
		t.Fatalf("last-modified = %q", rec.Header().Get("Last-Modified"))
	}
}

func TestMiddlewarePassesThroughWhenStale(t *testing.T) {
	called := false
	handler := conditional.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		res, ok := request.ResourceFromContext(r.Context())
		if !ok {
			t.Fatal("resource missing from context")
		}
		if res.ETag == "" {
			t.Fatal("etag not set")
		}
	}))

	ctx := request.WithResource(t.Context(), preloadedWithChecksum(t, "icons/64/icon.png", []byte("png-data")))
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	req.Header.Set("If-None-Match", `"stale"`)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next was not called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareHeadSkipsETagWithoutIfNoneMatch(t *testing.T) {
	handler := conditional.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, ok := request.ResourceFromContext(r.Context())
		if !ok {
			t.Fatal("resource missing")
		}
		if res.ETag != "" {
			t.Fatalf("etag = %q, want empty on HEAD without If-None-Match", res.ETag)
		}
	}))

	// FS nil and no preload: Data must not be called for ETag
	res := &request.Resource{FSPath: "icons/64/icon.png"}
	ctx := request.WithResource(t.Context(), res)
	req := httptest.NewRequest(http.MethodHead, "/icons/64/icon.png", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
