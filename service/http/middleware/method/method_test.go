package method_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/method"
)

func TestMiddlewareAllowsGetAndHead(t *testing.T) {
	t.Parallel()

	handler := method.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, m := range []string{http.MethodGet, http.MethodHead} {
		req := httptest.NewRequest(m, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d", m, rec.Code)
		}
	}
}

func TestMiddlewareRejectsPost(t *testing.T) {
	t.Parallel()

	handler := method.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}
