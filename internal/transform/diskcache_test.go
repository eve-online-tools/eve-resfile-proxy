package transform_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/transform"
)

func TestDiskCachePath(t *testing.T) {
	cache := &transform.DiskCache{Root: "/cache"}
	got := cache.Path("windows", "shader-fx", "7d/7d87a0a3_hash")
	want := filepath.Join("/cache", "_transformed", "windows", "shader-fx", "7d", "7d87a0a3_hash")
	if got != want {
		t.Fatalf("path = %q want %q", got, want)
	}
}

func TestDiskCacheReadMiss(t *testing.T) {
	cache := &transform.DiskCache{Root: t.TempDir()}
	data, ok, err := cache.Read("windows", "rule", "aa/bb")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if ok {
		t.Fatal("expected miss")
	}
	if data != nil {
		t.Fatalf("data = %q", data)
	}
}

func TestDiskCacheWriteReadHit(t *testing.T) {
	root := t.TempDir()
	cache := &transform.DiskCache{Root: root}

	if err := cache.Write("macos", "copy", "icons/icon.png", []byte("cached")); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, ok, err := cache.Read("macos", "copy", "icons/icon.png")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !ok {
		t.Fatal("expected hit")
	}
	if string(data) != "cached" {
		t.Fatalf("data = %q", data)
	}

	wantPath := filepath.Join(root, "_transformed", "macos", "copy", "icons", "icon.png")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("stat cached file: %v", err)
	}
}

func TestDiskCacheNilDisabled(t *testing.T) {
	var cache *transform.DiskCache
	data, ok, err := cache.Read("windows", "rule", "aa/bb")
	if err != nil || ok || data != nil {
		t.Fatalf("read = (%q, %v, %v)", data, ok, err)
	}
	if err := cache.Write("windows", "rule", "aa/bb", []byte("x")); err != nil {
		t.Fatalf("write: %v", err)
	}
}
