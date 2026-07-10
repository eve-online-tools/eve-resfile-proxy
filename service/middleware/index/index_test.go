package index

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	resindex "github.com/eve-online-tools/eve-resfile-proxy/internal/index"
)

func testIndexSet() *resindex.IndexSet {
	return &resindex.IndexSet{
		BuildNumber:     "1234567",
		LoadedPlatforms: []resindex.Platform{resindex.PlatformWindows},
		PlatformMaps: map[resindex.Platform]map[string]resindex.Entry{
			resindex.PlatformWindows: {
				"res:/foo/z.txt":     {CDNPath: "z.txt", Size: 100, CompressedSize: 50},
				"res:/foo/a.txt":     {CDNPath: "a.txt", Size: 4096, CompressedSize: 2048},
				"res:/foo/sub/x.png": {CDNPath: "sub/x.png"},
			},
		},
	}
}

func TestMiddlewareDisabledPassesThrough(t *testing.T) {
	called := false
	handler := Middleware(false, testIndexSet())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next should be called when disabled")
	}
}

func TestMiddlewareNonTrailingSlashPassesThrough(t *testing.T) {
	called := false
	handler := Middleware(true, testIndexSet())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo/a.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next should be called for non-trailing-slash paths")
	}
}

func TestMiddlewareNoMatchPassesThrough(t *testing.T) {
	called := false
	handler := Middleware(true, testIndexSet())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next should be called when directory has no entries")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareInvalidPlatform(t *testing.T) {
	handler := Middleware(true, testIndexSet())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo/?platform=linux", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareRendersListing(t *testing.T) {
	handler := Middleware(true, testIndexSet())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}

	body := rec.Body.String()
	for _, want := range []string{
		`href="/foo/sub/"`,
		`href="/foo/a.txt"`,
		`href="/foo/z.txt"`,
		">Parent Directory</a>",
		"<td>folder</td>",
		"<td>text/plain</td>",
		"Client build 1234567",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}

	subPos := strings.Index(body, ">sub/</a>")
	aPos := strings.Index(body, ">a.txt</a>")
	if subPos < 0 || aPos < 0 || subPos > aPos {
		t.Fatal("expected sub/ before files in listing")
	}
}

func TestMiddlewareRootListing(t *testing.T) {
	indexSet := &resindex.IndexSet{
		BuildNumber:     "999",
		LoadedPlatforms: []resindex.Platform{resindex.PlatformWindows},
		PlatformMaps: map[resindex.Platform]map[string]resindex.Entry{
			resindex.PlatformWindows: {
				"res:/foo/a.txt": {CDNPath: "a.txt"},
			},
		},
	}

	handler := Middleware(true, indexSet)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, ">Parent Directory</a>") {
		t.Fatal("root listing should not include parent link")
	}
	if !strings.Contains(body, `href="/foo/"`) {
		t.Fatal("body missing root child link")
	}
}

func TestMiddlewarePreservesPlatformQuery(t *testing.T) {
	handler := Middleware(true, testIndexSet())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo/?platform=windows", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `href="/foo/sub/?platform=windows"`) {
		t.Fatalf("body missing platform query on child link:\n%s", body)
	}
}
