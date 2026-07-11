//go:build livetests

// This test should only run explicitly, as it crosses network boundaries.
// Run it with `go test -tags livetests ./...`
package vfs_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/cdn"
)

func TestLive_manifestChainAndGlobPNG(t *testing.T) {
	ctx := context.Background()
	httpClient := &http.Client{Timeout: 2 * time.Minute}

	binaries, err := cdn.New(httpClient, cdn.WithBinariesOrigin())
	if err != nil {
		t.Fatalf("binaries fetcher: %v", err)
	}

	build, err := resolveTQBuild(ctx, httpClient)
	if err != nil {
		t.Fatalf("resolve build: %v", err)
	}
	t.Logf("build: %s", build)

	buildIndexURL := fmt.Sprintf("%s/eveonline_%s.txt", cdn.BinariesOrigin, build)
	buildIndexBytes, err := getBytes(ctx, httpClient, buildIndexURL)
	if err != nil {
		t.Fatalf("fetch build index: %v", err)
	}
	t.Logf("build index: %d bytes from %s", len(buildIndexBytes), buildIndexURL)

	appFS, err := vfs.NewFS(buildIndexBytes, binaries, vfs.WithPrefix(vfs.PrefixApp))
	if err != nil {
		t.Fatalf("app NewFS: %v", err)
	}

	if _, err := fs.Stat(appFS, "resfileindex.txt"); err != nil {
		t.Fatalf("Stat resfileindex.txt: %v", err)
	}

	f, err := appFS.Open("resfileindex.txt")
	if err != nil {
		t.Fatalf("Open resfileindex.txt: %v", err)
	}
	resfileIndexBytes, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil {
		t.Fatalf("Read resfileindex.txt: %v", err)
	}
	t.Logf("resfileindex.txt: %d bytes", len(resfileIndexBytes))

	resFS, err := vfs.NewFS(resfileIndexBytes, nil, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("res NewFS: %v", err)
	}

	matches, err := fs.Glob(resFS, "*.png")
	if err != nil {
		t.Fatalf("Glob *.png: %v", err)
	}
	t.Logf("Glob *.png: %d matches", len(matches))
	if len(matches) > 0 {
		limit := len(matches)
		if limit > 5 {
			limit = 5
		}
		t.Logf("sample: %v", matches[:limit])
	}

	recursive, err := fs.Glob(resFS, "**/*.png")
	if err != nil {
		t.Fatalf("Glob **/*.png: %v", err)
	}
	t.Logf("Glob **/*.png: %d matches", len(recursive))

	if len(resfileIndexBytes) == 0 {
		t.Fatal("resfileindex.txt was empty")
	}
	if len(recursive) == 0 {
		t.Fatal("expected at least one .png in resfile index")
	}
}

type clientManifest struct {
	BuildNumber string `json:"buildNumber"`
	Build       string `json:"build"`
}

func getBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func resolveTQBuild(ctx context.Context, client *http.Client) (string, error) {
	body, err := getBytes(ctx, client, cdn.BinariesOrigin+"/eveclient_TQ.json")
	if err != nil {
		return "", err
	}
	var manifest clientManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return "", err
	}
	build := manifest.BuildNumber
	if build == "" {
		build = manifest.Build
	}
	build = strings.TrimSpace(build)
	if build == "" {
		return "", fmt.Errorf("no build number in eveclient_TQ.json")
	}
	return build, nil
}
