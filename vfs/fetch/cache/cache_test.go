package cache_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/cache/memory"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	fetchcache "github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/cache"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/mapfetch"
)

func TestNew_nilCachePassthrough(t *testing.T) {
	inner := mapfetch.New(map[string][]byte{"7d/icon": []byte("data")})
	got := fetchcache.New(inner, nil)

	if got != inner {
		t.Fatal("expected inner fetcher unchanged when cache is nil")
	}
}

func TestFetch_cacheHitSkipsInner(t *testing.T) {
	const cdnPath = "7d/icon64_hash"
	store := memory.New()
	ctx := context.Background()
	if err := store.Store(ctx, cdnPath, []byte("cached")); err != nil {
		t.Fatalf("Store: %v", err)
	}

	inner := mapfetch.New(map[string][]byte{cdnPath: []byte("remote")})
	fetcher := fetchcache.New(inner, store)

	data, err := fetcher.FetchEntry(ctx, vfs.Entry{CDNPath: cdnPath})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != "cached" {
		t.Fatalf("data = %q, want cached bytes", data)
	}
}

func TestFetch_cacheMissDelegatesAndStores(t *testing.T) {
	const cdnPath = "7d/icon64_hash"
	const remote = "remote-bytes"

	store := memory.New()
	inner := mapfetch.New(map[string][]byte{cdnPath: []byte(remote)})
	fetcher := fetchcache.New(inner, store)

	ctx := context.Background()
	data, err := fetcher.FetchEntry(ctx, vfs.Entry{CDNPath: cdnPath})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != remote {
		t.Fatalf("data = %q, want %q", data, remote)
	}

	cached, found, err := store.Get(ctx, cdnPath)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected cache entry after miss")
	}
	if string(cached) != remote {
		t.Fatalf("cached = %q, want %q", cached, remote)
	}
}

func TestFetch_storeFailureLogsAndReturnsData(t *testing.T) {
	const cdnPath = "7d/icon64_hash"
	const remote = "remote-bytes"

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	inner := mapfetch.New(map[string][]byte{cdnPath: []byte(remote)})
	fetcher := fetchcache.New(inner, errStore{err: errors.New("disk full")}, fetchcache.WithLogger(logger))

	data, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: cdnPath})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != remote {
		t.Fatalf("data = %q, want %q", data, remote)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("cache store failed")) {
		t.Fatalf("log = %q, want cache store warning", logBuf.Bytes())
	}
}

func TestFetch_innerErrorPropagates(t *testing.T) {
	store := memory.New()
	inner := errFetcher{err: errors.New("fetch failed")}
	fetcher := fetchcache.New(inner, store)

	_, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/missing"})
	if err == nil {
		t.Fatal("expected error from inner fetcher")
	}
	if err.Error() != "fetch failed" {
		t.Fatalf("err = %v", err)
	}

	_, found, err := store.Get(context.Background(), "7d/missing")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("expected no cache write on inner error")
	}
}

type errFetcher struct {
	err error
}

func (f errFetcher) FetchEntry(ctx context.Context, entry vfs.Entry) ([]byte, error) {
	return f.FetchPath(ctx, entry.CDNPath)
}

func (f errFetcher) FetchPath(context.Context, string) ([]byte, error) {
	return nil, f.err
}

type errStore struct {
	err error
}

func (s errStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (s errStore) Store(context.Context, string, []byte) error {
	return s.err
}
