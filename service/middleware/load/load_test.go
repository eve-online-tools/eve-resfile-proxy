package load

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/fetch"
	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service/assetcache"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func TestMiddlewareCacheHit(t *testing.T) {
	cacheDir := t.TempDir()
	cache := assetcache.New(cacheDir)
	if err := cache.Write("icons/icon.png", []byte("cached")); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	handler := Middleware(cache, nil, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			t.Fatal("asset missing from context")
		}
		if string(asset.Data) != "cached" {
			t.Fatalf("data = %q", asset.Data)
		}
		if asset.CacheStatus != request.CacheStatusHit {
			t.Fatalf("cache status = %q", asset.CacheStatus)
		}
	}))

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath:  "res:/icons/64/icon.png",
		CDNPath:  "icons/icon.png",
		Platform: index.PlatformWindows,
	})
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestMiddlewareCacheMissFetches(t *testing.T) {
	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fetched"))
	}))
	defer assetServer.Close()

	cache := assetcache.New(t.TempDir())
	client := &fetch.Client{
		HTTP:      assetServer.Client(),
		Semaphore: fetch.NewSemaphore(1),
	}

	handler := Middleware(cache, client, assetServer.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			t.Fatal("asset missing from context")
		}
		if string(asset.Data) != "fetched" {
			t.Fatalf("data = %q", asset.Data)
		}
		if asset.CacheStatus != request.CacheStatusMiss {
			t.Fatalf("cache status = %q", asset.CacheStatus)
		}
	}))

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath:  "res:/icons/64/icon.png",
		CDNPath:  "icons/icon.png",
		Platform: index.PlatformWindows,
	})
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestMiddlewareNoCacheFetches(t *testing.T) {
	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fetched"))
	}))
	defer assetServer.Close()

	client := &fetch.Client{
		HTTP:      assetServer.Client(),
		Semaphore: fetch.NewSemaphore(1),
	}

	handler := Middleware(nil, client, assetServer.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			t.Fatal("asset missing from context")
		}
		if string(asset.Data) != "fetched" {
			t.Fatalf("data = %q", asset.Data)
		}
		if asset.CacheStatus != request.CacheStatusMiss {
			t.Fatalf("cache status = %q", asset.CacheStatus)
		}
	}))

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath:  "res:/icons/64/icon.png",
		CDNPath:  "icons/icon.png",
		Platform: index.PlatformWindows,
	})
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestMiddlewareFetchError(t *testing.T) {
	cache := assetcache.New(t.TempDir())
	client := &fetch.Client{
		HTTP:      http.DefaultClient,
		Semaphore: fetch.NewSemaphore(1),
	}

	handler := Middleware(cache, client, "http://127.0.0.1:1")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	ctx := request.WithAsset(t.Context(), request.Asset{
		ResPath:  "res:/icons/64/icon.png",
		CDNPath:  "icons/icon.png",
		Platform: index.PlatformWindows,
	})
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
