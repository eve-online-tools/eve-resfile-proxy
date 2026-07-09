package resolve

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
)

func TestMiddlewareInvalidPlatform(t *testing.T) {
	handler := Middleware(&index.IndexSet{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/icons/icon.png?platform=linux", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareNotFoundPath(t *testing.T) {
	handler := Middleware(&index.IndexSet{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareIndexMiss(t *testing.T) {
	handler := Middleware(&index.IndexSet{
		PlatformMaps: map[index.Platform]map[string]string{
			index.PlatformWindows: {},
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareSetsContext(t *testing.T) {
	indexSet := &index.IndexSet{
		PlatformMaps: map[index.Platform]map[string]string{
			index.PlatformWindows: {
				"res:/icons/64/icon.png": "icons/icon.png",
			},
		},
	}

	handler := Middleware(indexSet)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			t.Fatal("asset missing from context")
		}
		if asset.ResPath != "res:/icons/64/icon.png" {
			t.Fatalf("resPath = %q", asset.ResPath)
		}
		if asset.CDNPath != "icons/icon.png" {
			t.Fatalf("cdnPath = %q", asset.CDNPath)
		}
		if asset.Platform != index.PlatformWindows {
			t.Fatalf("platform = %q", asset.Platform)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
