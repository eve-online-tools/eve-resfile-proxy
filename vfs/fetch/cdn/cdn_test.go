package cdn_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/common/domain"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/cdn"
)

func TestFetch_resourcesOrigin(t *testing.T) {
	const want = "asset-bytes"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/7d/icon64_hash" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(want))
	}))
	t.Cleanup(server.Close)

	fetcher := cdn.New(
		server.Client(),
		cdn.WithDomain(domain.Domain(server.URL)),
	)

	data, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/icon64_hash"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != want {
		t.Fatalf("data = %q, want %q", data, want)
	}
}

func TestFetch_binariesOriginPreset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("index"))
	}))
	t.Cleanup(server.Close)

	fetcher := cdn.New(
		server.Client(),
		cdn.WithDomain(domain.Domain(server.URL)),
	)

	if _, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "res/global.txt"}); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
}

func TestWithBinariesOrigin_constant(t *testing.T) {
	if cdn.BinariesOrigin != "https://binaries.eveonline.com" {
		t.Fatalf("DefaultBinariesOrigin = %q", cdn.BinariesOrigin)
	}
	if cdn.ResourcesOrigin != "https://resources.eveonline.com" {
		t.Fatalf("DefaultResourcesOrigin = %q", cdn.ResourcesOrigin)
	}
}
