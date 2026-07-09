package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func TestHandlerWritesResponse(t *testing.T) {
	h := New(&index.IndexSet{BuildNumber: "123"})

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath:     "res:/icons/64/icon.png",
		CDNPath:     "icons/icon.png",
		Platform:    index.PlatformWindows,
		Data:        []byte("png-data"),
		CacheStatus: request.CacheStatusHit,
		ETag:        `"placeholder"`,
	})
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=3600" {
		t.Fatalf("cache-control = %q", rec.Header().Get("Cache-Control"))
	}
	if rec.Header().Get("X-Eve-Build") != "123" {
		t.Fatalf("build header = %q", rec.Header().Get("X-Eve-Build"))
	}
	if rec.Header().Get("X-Eve-Platform") != "windows" {
		t.Fatalf("platform header = %q", rec.Header().Get("X-Eve-Platform"))
	}
	if rec.Header().Get("X-Cache-Status") != "HIT" {
		t.Fatalf("cache status header = %q", rec.Header().Get("X-Cache-Status"))
	}
	if rec.Body.String() != "png-data" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandlerUnknownExtension(t *testing.T) {
	h := New(&index.IndexSet{})

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath: "res:/data/file.bin",
		Data:    []byte("data"),
	})
	req := httptest.NewRequest(http.MethodGet, "/data/file.bin", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
}
