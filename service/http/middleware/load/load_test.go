package load_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/load"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/request"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/mapfetch"
)

type countingFetcher struct {
	inner vfs.Fetcher
	calls atomic.Int32
}

func (f *countingFetcher) FetchEntry(ctx context.Context, entry vfs.Entry) ([]byte, error) {
	f.calls.Add(1)
	return f.inner.FetchEntry(ctx, entry)
}

func (f *countingFetcher) FetchPath(ctx context.Context, path string) ([]byte, error) {
	f.calls.Add(1)
	return f.inner.FetchPath(ctx, path)
}

func TestMiddlewareHeadDoesNotFetch(t *testing.T) {
	manifest := "res:/icons/64/icon.png,icons/icon.png,d41d8cd98f00b204e9800998ecf8427e,4096,2048\n"
	counter := &countingFetcher{inner: mapfetch.New(map[string][]byte{
		"icons/icon.png": []byte("png-data"),
	})}
	fsys, err := vfs.New([]byte(manifest), counter, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("vfs.New: %v", err)
	}

	handler := load.Middleware(fsys, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, ok := request.ResourceFromContext(r.Context())
		if !ok {
			t.Fatal("resource missing")
		}
		if !res.HasChecksum {
			t.Fatal("expected manifest checksum in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodHead, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if counter.calls.Load() != 0 {
		t.Fatalf("fetch calls = %d, want 0", counter.calls.Load())
	}
}

func TestMiddlewareGetFetchesThroughHandler(t *testing.T) {
	manifest := "res:/icons/64/icon.png,icons/icon.png,d41d8cd98f00b204e9800998ecf8427e,4096,2048\n"
	counter := &countingFetcher{inner: mapfetch.New(map[string][]byte{
		"icons/icon.png": []byte("png-data"),
	})}
	fsys, err := vfs.New([]byte(manifest), counter, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("vfs.New: %v", err)
	}

	handler := load.Middleware(fsys, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, _ := request.ResourceFromContext(r.Context())
		data, err := res.Data()
		if err != nil {
			t.Fatalf("Data: %v", err)
		}
		if string(data) != "png-data" {
			t.Fatalf("data = %q", data)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if counter.calls.Load() == 0 {
		t.Fatal("expected fetch on GET")
	}
}

func TestMiddlewareEmptyChecksumNotSet(t *testing.T) {
	manifest := "res:/icons/64/icon.png,icons/icon.png,,4096,2048\n"
	fsys, err := vfs.New([]byte(manifest), mapfetch.New(map[string][]byte{
		"icons/icon.png": []byte("png-data"),
	}), vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("vfs.New: %v", err)
	}

	handler := load.Middleware(fsys, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, ok := request.ResourceFromContext(r.Context())
		if !ok {
			t.Fatal("resource missing")
		}
		if res.HasChecksum {
			t.Fatal("expected no checksum for empty manifest MD5 column")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodHead, "/icons/64/icon.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
