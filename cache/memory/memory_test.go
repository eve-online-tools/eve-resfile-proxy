package memory_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/cache/memory"
)

func TestCacheGetMiss(t *testing.T) {
	c := memory.New()
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
	c := memory.New()
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
}

func TestCacheNilDisabled(t *testing.T) {
	var c *memory.Cache
	ctx := context.Background()

	data, found, err := c.Get(ctx, "aa/bb")
	if err != nil || found || data != nil {
		t.Fatalf("Get = (%q, %v, %v)", data, found, err)
	}
	if err := c.Store(ctx, "aa/bb", []byte("x")); err != nil {
		t.Fatalf("Store: %v", err)
	}
}

func TestCacheCopiesStoredBytes(t *testing.T) {
	c := memory.New()
	ctx := context.Background()
	val := []byte("cached")

	if err := c.Store(ctx, "key", val); err != nil {
		t.Fatalf("Store: %v", err)
	}
	val[0] = 'X'

	data, found, err := c.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(data) != "cached" {
		t.Fatalf("Get = (%q, %v)", data, found)
	}
}

func TestCacheCopiesReturnedBytes(t *testing.T) {
	c := memory.New()
	ctx := context.Background()

	if err := c.Store(ctx, "key", []byte("cached")); err != nil {
		t.Fatalf("Store: %v", err)
	}

	data, found, err := c.Get(ctx, "key")
	if err != nil || !found {
		t.Fatalf("Get = (%q, %v, %v)", data, found, err)
	}
	data[0] = 'X'

	again, found, err := c.Get(ctx, "key")
	if err != nil || !found || string(again) != "cached" {
		t.Fatalf("Get again = (%q, %v, %v)", again, found, err)
	}
}

func TestCacheConcurrent(t *testing.T) {
	c := memory.New()
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
