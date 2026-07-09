package conditional

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func TestMiddlewareIfNoneMatchReturns304(t *testing.T) {
	data := []byte("png-data")
	etag := etagFor(data)

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath: "res:/icons/64/icon.png",
		Data:    data,
	})
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

func TestMiddlewareIfModifiedSinceReturns304(t *testing.T) {
	lastModified := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath:      "res:/icons/64/icon.png",
		Data:         []byte("png-data"),
		LastModified: lastModified,
	})
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
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			t.Fatal("asset missing from context")
		}
		if asset.ETag == "" {
			t.Fatal("etag not set")
		}
	}))

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath: "res:/icons/64/icon.png",
		Data:    []byte("png-data"),
	})
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

func TestMiddlewarePassesThroughWithoutAsset(t *testing.T) {
	called := false
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next was not called")
	}
}
