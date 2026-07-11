package diskcache_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	diskcache "github.com/eve-online-tools/eve-resfile-proxy/cache/disk"
)

func TestCachePath(t *testing.T) {
	c := diskcache.New("/cache")
	got := c.Path("7d/icon64_hash")
	want := filepath.Join("/cache", "7d", "icon64_hash")
	if got != want {
		t.Fatalf("path = %q want %q", got, want)
	}
}

func TestCache_traversalRejected(t *testing.T) {
	c := diskcache.New(t.TempDir())
	ctx := context.Background()

	_, _, err := c.Get(ctx, "../../etc/passwd")
	if err == nil {
		t.Fatal("expected Get error for traversal key")
	}
	if err := c.Store(ctx, "../../etc/passwd", []byte("x")); err == nil {
		t.Fatal("expected Store error for traversal key")
	}
}

func TestCacheGetMiss(t *testing.T) {
	c := diskcache.New(t.TempDir())
	data, found, err := c.Get(context.Background(), "aa/bb")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("expected miss")
	}
	if data != nil {
		t.Fatalf("data = %q", data)
	}
}

func TestCacheStoreGetHit(t *testing.T) {
	root := t.TempDir()
	c := diskcache.New(root)
	ctx := context.Background()

	if err := c.Store(ctx, "icons/icon.png", []byte("cached")); err != nil {
		t.Fatalf("Store: %v", err)
	}

	data, found, err := c.Get(ctx, "icons/icon.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected hit")
	}
	if string(data) != "cached" {
		t.Fatalf("data = %q", data)
	}

	wantPath := filepath.Join(root, "icons", "icon.png")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("stat cached file: %v", err)
	}
}

func TestCacheNilDisabled(t *testing.T) {
	var c *diskcache.Cache
	ctx := context.Background()

	data, found, err := c.Get(ctx, "aa/bb")
	if err != nil || found || data != nil {
		t.Fatalf("Get = (%q, %v, %v)", data, found, err)
	}
	if err := c.Store(ctx, "aa/bb", []byte("x")); err != nil {
		t.Fatalf("Store: %v", err)
	}
}

func TestCacheEmptyRootDisabled(t *testing.T) {
	c := diskcache.New("")
	ctx := context.Background()

	data, found, err := c.Get(ctx, "aa/bb")
	if err != nil || found || data != nil {
		t.Fatalf("Get = (%q, %v, %v)", data, found, err)
	}
	if err := c.Store(ctx, "aa/bb", []byte("x")); err != nil {
		t.Fatalf("Store: %v", err)
	}
}

func TestCacheConcurrent(t *testing.T) {
	c := diskcache.New(t.TempDir())
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("7d/key-%d", n)
			val := []byte{byte(n)}
			if err := c.Store(ctx, key, val); err != nil {
				t.Errorf("Store: %v", err)
			}
			data, found, err := c.Get(ctx, key)
			if err != nil {
				t.Errorf("Get: %v", err)
				return
			}
			if !found || len(data) != 1 || data[0] != byte(n) {
				t.Errorf("Get(%q) = (%q, %v)", key, data, found)
			}
		}(i)
	}
	wg.Wait()
}
