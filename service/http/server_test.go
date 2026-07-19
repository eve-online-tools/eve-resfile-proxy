package http_test

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	diskcache "github.com/eve-online-tools/eve-resfile-proxy/cache/disk"
	"github.com/eve-online-tools/eve-resfile-proxy/service/buildnumber"
	svchttp "github.com/eve-online-tools/eve-resfile-proxy/service/http"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/handler"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/conditional"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/cors"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/heartbeat"
	indexmw "github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/load"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/method"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/mapfetch"
)

const testManifestMD5 = "d41d8cd98f00b204e9800998ecf8427e"

func testBuildNumber(value string) *buildnumber.BuildNumber {
	b := &buildnumber.BuildNumber{}
	b.Set(value)
	return b
}

func testHandler(t *testing.T, fsys fs.FS, cacheDir string) http.Handler {
	t.Helper()
	return testHandlerWithBuild(t, fsys, cacheDir, testBuildNumber("123"))
}

func testHandlerWithBuild(t *testing.T, fsys fs.FS, cacheDir string, build *buildnumber.BuildNumber) http.Handler {
	t.Helper()

	var disk *diskcache.Cache
	if cacheDir != "" {
		disk = diskcache.New(cacheDir)
	}

	chain := svchttp.MiddlewareChain{
		heartbeat.Middleware("/healthz"),
		heartbeat.Middleware("/livez"),
		method.Middleware,
		indexmw.Middleware(true, fsys, build),
		load.Middleware(fsys, disk),
		conditional.Middleware,
	}
	return chain.For(handler.Respond(build))
}

func testFS(t *testing.T) fs.FS {
	t.Helper()

	manifest := "res:/icons/64/icon.png,icons/icon.png," + testManifestMD5 + ",8,4\n"
	fsys, err := vfs.New([]byte(manifest), mapfetch.New(map[string][]byte{
		"icons/icon.png": []byte("png-data"),
	}), vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("vfs.New: %v", err)
	}
	return fsys
}

func TestHandlerServesAsset(t *testing.T) {
	h := testHandler(t, testFS(t), "")

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
	if rec.Header().Get("X-Cache-Status") != "MISS" {
		t.Fatalf("cache status = %q", rec.Header().Get("X-Cache-Status"))
	}
	if rec.Body.String() != "png-data" {
		t.Fatalf("body = %q", rec.Body.String())
	}
	if rec.Header().Get("Content-Length") != "8" {
		t.Fatalf("content-length = %q, want 8", rec.Header().Get("Content-Length"))
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatal("missing ETag")
	}
}

func TestHandlerHeadReturnsNoBody(t *testing.T) {
	h := testHandler(t, testFS(t), "")

	req := httptest.NewRequest(http.MethodHead, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Content-Length") != "8" {
		t.Fatalf("content-length = %q, want 8", rec.Header().Get("Content-Length"))
	}
}

func TestHandlerServesFromCache(t *testing.T) {
	cacheDir := t.TempDir()
	cdnPath := "icons/icon.png"
	cacheFile := filepath.Join(cacheDir, cdnPath)
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("cached-png"), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	h := testHandler(t, testFS(t), cacheDir)

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Cache-Status") != "HIT" {
		t.Fatalf("cache status = %q", rec.Header().Get("X-Cache-Status"))
	}
	if rec.Header().Get("Last-Modified") == "" {
		t.Fatal("missing Last-Modified on cache hit")
	}
}

func TestHealthz(t *testing.T) {
	h := testHandler(t, testFS(t), "")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q", rec.Body.String())
	}
	if rec.Header().Get("Content-Length") != "2" {
		t.Fatalf("content-length = %q, want 2", rec.Header().Get("Content-Length"))
	}
}

func TestDirectoryListing(t *testing.T) {
	manifest := "res:/icons/64/icon.png,icons/icon.png," + testManifestMD5 + ",4096,2048\n" +
		"res:/icons/32/icon.png,icons/icon32.png," + testManifestMD5 + ",2048,1024\n"
	fsys, err := vfs.New([]byte(manifest), mapfetch.New(nil), vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("vfs.New: %v", err)
	}

	h := testHandler(t, fsys, "")
	req := httptest.NewRequest(http.MethodGet, "/icons/64/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "icon.png") {
		t.Fatalf("listing body missing icon.png: %s", rec.Body.String())
	}
}

func TestDirectoryListingDoubleSlashRoot(t *testing.T) {
	manifest := "res:/icons/64/icon.png,icons/icon.png," + testManifestMD5 + ",4096,2048\n"
	fsys, err := vfs.New([]byte(manifest), mapfetch.New(nil), vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("vfs.New: %v", err)
	}

	h := testHandler(t, fsys, "")
	req := httptest.NewRequest(http.MethodGet, "//", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `href="//`) {
		t.Fatalf("listing contains protocol-relative href: %s", body)
	}
	if !strings.Contains(body, `href="/icons/"`) {
		t.Fatalf("listing missing normalized /icons/ link: %s", body)
	}
}

func TestBuildNumberUpdates(t *testing.T) {
	build := &buildnumber.BuildNumber{}
	build.Set("123")

	h := testHandlerWithBuild(t, testFS(t), "", build)

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Eve-Build") != "123" {
		t.Fatalf("initial build header = %q, want 123", rec.Header().Get("X-Eve-Build"))
	}

	build.Set("456")

	req = httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Eve-Build") != "456" {
		t.Fatalf("updated build header = %q, want 456", rec.Header().Get("X-Eve-Build"))
	}

	manifest := "res:/icons/64/icon.png,icons/icon.png," + testManifestMD5 + ",4096,2048\n"
	fsys, err := vfs.New([]byte(manifest), mapfetch.New(nil), vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("vfs.New: %v", err)
	}
	h = testHandlerWithBuild(t, fsys, "", build)

	req = httptest.NewRequest(http.MethodGet, "/icons/64/", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "Client build 456") {
		t.Fatalf("listing footer missing updated build: %s", rec.Body.String())
	}
}

func TestHandlerRangeRequest(t *testing.T) {
	h := testHandler(t, testFS(t), "")

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	req.Header.Set("Range", "bytes=0-3")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206; body=%s", rec.Code, rec.Body.String())
	}
	if cr := rec.Header().Get("Content-Range"); cr != "bytes 0-3/8" {
		t.Fatalf("content-range = %q, want bytes 0-3/8", cr)
	}
	if rec.Body.String() != "png-" {
		t.Fatalf("body = %q, want png-", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestHandlerRangeUnsatisfiable(t *testing.T) {
	h := testHandler(t, testFS(t), "")

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	req.Header.Set("Range", "bytes=100-200")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("status = %d, want 416", rec.Code)
	}
}

func TestHandlerAdvertisesAcceptRanges(t *testing.T) {
	h := testHandler(t, testFS(t), "")

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ar := rec.Header().Get("Accept-Ranges"); ar != "bytes" {
		t.Fatalf("accept-ranges = %q, want bytes", ar)
	}
}

func TestHandlerHeadAdvertisesAcceptRanges(t *testing.T) {
	h := testHandler(t, testFS(t), "")

	req := httptest.NewRequest(http.MethodHead, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if ar := rec.Header().Get("Accept-Ranges"); ar != "bytes" {
		t.Fatalf("accept-ranges = %q, want bytes", ar)
	}
}

func TestHandlerIfRange(t *testing.T) {
	h := testHandler(t, testFS(t), "")

	// Obtain the current ETag.
	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	// Matching If-Range → the range is served (206).
	req = httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	req.Header.Set("Range", "bytes=0-3")
	req.Header.Set("If-Range", etag)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("matching If-Range status = %d, want 206", rec.Code)
	}

	// Stale If-Range → the full body is served (200).
	req = httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	req.Header.Set("Range", "bytes=0-3")
	req.Header.Set("If-Range", `"stale"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stale If-Range status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "png-data" {
		t.Fatalf("stale If-Range body = %q, want png-data", rec.Body.String())
	}
}

func TestCORSHeadersOnResponse(t *testing.T) {
	h := cors.Middleware(testHandler(t, testFS(t), ""))

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin = %q, want *", got)
	}
}

func TestCORSHeadersOnNotFound(t *testing.T) {
	h := cors.Middleware(testHandler(t, testFS(t), ""))

	req := httptest.NewRequest(http.MethodGet, "/does/not/exist.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin = %q, want * on error response", got)
	}
}

func TestGracefulShutdown(t *testing.T) {
	cfg := svchttp.ServerConfig{Addr: "127.0.0.1:0"}
	srv := svchttp.NewServer(&cfg, testFS(t), nil, nil, nil)
	srv.Start()

	time.Sleep(50 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("graceful shutdown: %v", err)
	}
}
