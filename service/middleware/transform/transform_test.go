package transform_test

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/internal/transform"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/conditional"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/request"
	transformmw "github.com/eve-online-tools/eve-resfile-proxy/service/middleware/transform"
)

func TestMiddlewareAppliesTransformBeforeConditional(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "upper.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntr 'a-z' 'A-Z'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := writeConfigInDir(t, dir, `
transforms:
  - name: upper
    match:
      extensions: [".txt"]
    command:
      args: ["`+script+`"]
`)
	engine, err := transform.LoadEngine(configPath, "")
	if err != nil {
		t.Fatalf("load engine: %v", err)
	}

	var captured request.Asset
	handler := transformmw.Middleware(engine)(conditional.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok {
			t.Fatal("missing asset")
		}
		captured = asset
		w.WriteHeader(http.StatusOK)
	})))

	body := []byte("abc")
	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	req = req.WithContext(request.WithAsset(req.Context(), request.Asset{
		ResPath:  "res:/file.txt",
		CDNPath:  "aa/bb",
		Platform: index.PlatformWindows,
		Data:     body,
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if string(captured.Data) != "ABC" {
		t.Fatalf("data = %q", captured.Data)
	}

	sum := sha256.Sum256([]byte("ABC"))
	wantETag := fmt.Sprintf(`"%x"`, sum)
	if captured.ETag != wantETag {
		t.Fatalf("etag = %q want %q", captured.ETag, wantETag)
	}
}

func TestMiddlewareNoEnginePassesThrough(t *testing.T) {
	handler := transformmw.Middleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, ok := request.AssetFromContext(r.Context())
		if !ok || string(asset.Data) != "raw" {
			t.Fatalf("asset = %+v", asset)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(request.WithAsset(req.Context(), request.Asset{Data: []byte("raw")}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestMiddlewareDiskCacheShortCircuits304(t *testing.T) {
	dir := t.TempDir()
	cacheDir := t.TempDir()
	countFile := filepath.Join(dir, "count.txt")
	script := filepath.Join(dir, "upper.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncount=$(cat "+countFile+` 2>/dev/null || echo 0)
count=$((count + 1))
echo $count > `+countFile+`
if [ "$count" -gt 1 ]; then exit 1; fi
tr 'a-z' 'A-Z'
`), 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := writeConfigInDir(t, dir, `
transforms:
  - name: upper
    match:
      extensions: [".txt"]
    command:
      args: ["`+script+`"]
`)
	engine, err := transform.LoadEngine(configPath, cacheDir)
	if err != nil {
		t.Fatalf("load engine: %v", err)
	}

	var warmed bool
	handler := transformmw.Middleware(engine)(conditional.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if warmed {
			t.Fatal("handler should not be called on 304")
		}
		warmed = true
		w.WriteHeader(http.StatusOK)
	})))

	asset := request.Asset{
		ResPath:      "res:/file.txt",
		CDNPath:      "aa/bb",
		Platform:     index.PlatformWindows,
		Data:         []byte("abc"),
		CacheStatus:  request.CacheStatusHit,
		LastModified: time.Now(),
	}

	warmReq := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	warmReq = warmReq.WithContext(request.WithAsset(warmReq.Context(), asset))
	warmRec := httptest.NewRecorder()
	handler.ServeHTTP(warmRec, warmReq)
	if warmRec.Code != http.StatusOK {
		t.Fatalf("warm status = %d", warmRec.Code)
	}

	etag := conditional.ETagFor([]byte("ABC"))
	condReq := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	condReq.Header.Set("If-None-Match", etag)
	condReq = condReq.WithContext(request.WithAsset(condReq.Context(), asset))
	condRec := httptest.NewRecorder()
	handler.ServeHTTP(condRec, condReq)

	if condRec.Code != http.StatusNotModified {
		t.Fatalf("status = %d", condRec.Code)
	}
	if condRec.Header().Get("ETag") != etag {
		t.Fatalf("etag = %q", condRec.Header().Get("ETag"))
	}
}

func writeConfigInDir(t *testing.T, dir, yaml string) string {
	t.Helper()
	path := filepath.Join(dir, "transforms.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
