package cors_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/cors"
)

func TestMiddlewareSetsHeadersAndCallsNext(t *testing.T) {
	t.Parallel()

	called := false
	handler := cors.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/icon.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin = %q, want *", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(got, "Content-Range") {
		t.Fatalf("expose-headers = %q, missing Content-Range", got)
	}
}

func TestMiddlewareAnswersPreflight(t *testing.T) {
	t.Parallel()

	handler := cors.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called for preflight")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/icon.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin = %q, want *", got)
	}
	// A cross-origin ranged fetch preflights on Range; it must be allowed.
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Range") {
		t.Fatalf("allow-headers = %q, missing Range", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") {
		t.Fatalf("allow-methods = %q, missing GET", got)
	}
}
