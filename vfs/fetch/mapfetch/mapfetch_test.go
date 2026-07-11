package mapfetch_test

import (
	"context"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/mapfetch"
)

func TestFetch_hit(t *testing.T) {
	want := []byte("png-bytes")
	fetcher := mapfetch.New(map[string][]byte{
		"7d/icon64_hash": want,
	})

	data, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/icon64_hash"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != string(want) {
		t.Fatalf("data = %q, want %q", data, want)
	}
}

func TestFetch_miss(t *testing.T) {
	fetcher := mapfetch.New(map[string][]byte{})

	_, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
}
