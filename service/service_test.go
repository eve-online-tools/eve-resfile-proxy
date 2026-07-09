package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/fetch"
	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service/assetcache"
	"github.com/eve-online-tools/eve-resfile-proxy/service/handler"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/conditional"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/getonly"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/heartbeat"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/load"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/resolve"
)

func TestGracefulShutdown(t *testing.T) {
	svc := newWithHandler("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := svc.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("graceful shutdown: %v", err)
	}
}

func TestHandlerServesAsset(t *testing.T) {
	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-data"))
	}))
	defer assetServer.Close()

	cacheDir := t.TempDir()
	h := testHandler(
		&index.IndexSet{
			BuildNumber: "123",
			PlatformMaps: map[index.Platform]map[string]string{
				index.PlatformWindows: {
					"res:/icons/64/icon.png": "icons/icon.png",
				},
			},
			LoadedPlatforms: []index.Platform{index.PlatformWindows},
		},
		assetcache.New(cacheDir),
		&fetch.Client{
			HTTP:      assetServer.Client(),
			Semaphore: fetch.NewSemaphore(1),
		},
		assetServer.URL,
	)

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("X-Eve-Build") != "123" {
		t.Fatalf("build header = %q", rec.Header().Get("X-Eve-Build"))
	}
	if rec.Header().Get("X-Eve-Platform") != "windows" {
		t.Fatalf("platform header = %q", rec.Header().Get("X-Eve-Platform"))
	}
	if rec.Header().Get("X-Cache-Status") != "MISS" {
		t.Fatalf("first request cache status = %q", rec.Header().Get("X-Cache-Status"))
	}
	if rec.Body.String() != "png-data" {
		t.Fatalf("body = %q", rec.Body.String())
	}

	assetServer.Close()
	req2 := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || rec2.Body.String() != "png-data" {
		t.Fatalf("cache hit failed: status=%d body=%q", rec2.Code, rec2.Body.String())
	}
	if rec2.Header().Get("X-Cache-Status") != "HIT" {
		t.Fatalf("second request cache status = %q", rec2.Header().Get("X-Cache-Status"))
	}
}

func TestHandlerHealthz(t *testing.T) {
	h := testHandler(&index.IndexSet{}, nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("healthz = %d %q", rec.Code, rec.Body.String())
	}
}

func TestHandlerLivez(t *testing.T) {
	h := testHandler(&index.IndexSet{}, nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("livez = %d %q", rec.Code, rec.Body.String())
	}
}

func TestHandlerInvalidPlatform(t *testing.T) {
	h := testHandler(&index.IndexSet{}, nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/icons/icon.png?platform=linux", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerNotFound(t *testing.T) {
	h := testHandler(&index.IndexSet{
		PlatformMaps: map[index.Platform]map[string]string{
			index.PlatformWindows: {},
		},
	}, assetcache.New(t.TempDir()), nil, "")
	req := httptest.NewRequest(http.MethodGet, "/missing.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func testHandler(indexSet *index.IndexSet, cache *assetcache.Store, fetchClient *fetch.Client, assetOrigin string) http.Handler {
	middlewares := middleware.MiddlewareChain{
		heartbeat.Middleware("/healthz"),
		heartbeat.Middleware("/livez"),
		getonly.Middleware,
		resolve.Middleware(indexSet),
		load.Middleware(cache, fetchClient, assetOrigin),
		conditional.Middleware,
	}
	return middlewares.For(handler.New(indexSet))
}

func newWithHandler(addr string, handler http.Handler) *Service {
	return &Service{
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
		},
		index: &index.IndexSet{BuildNumber: "test"},
	}
}
