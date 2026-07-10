package index

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type mapFetcher struct {
	files map[string]string
}

func (m mapFetcher) GetText(ctx context.Context, url string) (string, error) {
	if content, ok := m.files[url]; ok {
		return content, nil
	}
	return "", fmt.Errorf("not found: %s", url)
}

func TestLoadWindowsPlatform(t *testing.T) {
	origin := "https://binaries.test"
	build := "999"
	fetcher := mapFetcher{files: map[string]string{
		fmt.Sprintf("%s/eveonline_%s.txt", origin, build): "" +
			"app:/resfileindex.txt,res/global.txt,hash\n" +
			"app:/resfileindex_Windows.txt,res/windows.txt,hash\n",
		origin + "/res/global.txt": "" +
			"res:/shared.png,global/shared.png,hash\n" +
			"res:/global-only.png,global/only.png,hash\n",
		origin + "/res/windows.txt": "res:/shared.png,win/shared.png,hash\n",
	}}

	set, err := Load(context.Background(), LoaderOptions{
		BuildNumber: build,
		IndexOrigin: origin,
		CacheDir:    t.TempDir(),
		Platform:    PlatformWindows,
		Fetch:       fetcher,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(set.LoadedPlatforms) != 1 || set.LoadedPlatforms[0] != PlatformWindows {
		t.Fatalf("platforms = %v", set.LoadedPlatforms)
	}
	cdn, _, ok := set.Lookup("res:/shared.png", "")
	if !ok || cdn != "global/shared.png" {
		t.Fatalf("shared = %q ok=%v", cdn, ok)
	}
	cdn, _, ok = set.Lookup("res:/global-only.png", "")
	if !ok || cdn != "global/only.png" {
		t.Fatalf("global-only = %q ok=%v", cdn, ok)
	}
}

func TestLoadMacOSPlatform(t *testing.T) {
	origin := "https://binaries.test"
	build := "999"
	globalPath := "app:/EVE.app/Contents/Resources/build/resfileindex.txt"
	osPath := "app:/EVE.app/Contents/Resources/build/resfileindex_macOS.txt"
	fetcher := mapFetcher{files: map[string]string{
		fmt.Sprintf("%s/eveonlinemacOS_%s.txt", origin, build): "" +
			globalPath + ",res/global.txt,hash\n" +
			osPath + ",res/macos.txt,hash\n",
		origin + "/res/global.txt": "res:/shared.png,global/shared.png,hash\n",
		origin + "/res/macos.txt":  "res:/mac-only.png,mac/only.png,hash\n",
	}}

	set, err := Load(context.Background(), LoaderOptions{
		BuildNumber: build,
		IndexOrigin: origin,
		CacheDir:    t.TempDir(),
		Platform:    PlatformMacOS,
		Fetch:       fetcher,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cdn, _, ok := set.Lookup("res:/mac-only.png", "")
	if !ok || cdn != "mac/only.png" {
		t.Fatalf("mac-only = %q ok=%v", cdn, ok)
	}
}

func TestLoadUsesCacheFastPath(t *testing.T) {
	cacheDir := t.TempDir()
	build := "999"
	merged := map[string]Entry{"res:/a.png": {CDNPath: "a.png"}}
	if err := writePlatformMerged(platformMergedPath(cacheDir, build, PlatformWindows), merged); err != nil {
		t.Fatalf("write merged: %v", err)
	}
	if err := writeMeta(metaPath(cacheDir, build), CacheMeta{
		BuildNumber:     build,
		LoadedPlatforms: []Platform{PlatformWindows},
	}); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	set, err := Load(context.Background(), LoaderOptions{
		BuildNumber: build,
		IndexOrigin: "https://binaries.test",
		CacheDir:    cacheDir,
		Platform:    PlatformWindows,
		Fetch: mapFetcher{files: map[string]string{
			"https://binaries.test/should-not-be-called": "fail",
		}},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cdn, _, ok := set.Lookup("res:/a.png", "")
	if !ok || cdn != "a.png" {
		t.Fatalf("lookup = %q ok=%v", cdn, ok)
	}
}

func TestLoadDegradedWhenMacOSMissing(t *testing.T) {
	origin := "https://binaries.test"
	build := "999"
	fetcher := mapFetcher{files: map[string]string{
		fmt.Sprintf("%s/eveonline_%s.txt", origin, build): "" +
			"app:/resfileindex.txt,res/global.txt,hash\n",
		origin + "/res/global.txt": "res:/win.png,win.png,hash\n",
	}}

	set, err := Load(context.Background(), LoaderOptions{
		BuildNumber: build,
		IndexOrigin: origin,
		CacheDir:    t.TempDir(),
		Fetch:       fetcher,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(set.LoadedPlatforms) != 1 {
		t.Fatalf("platforms = %v", set.LoadedPlatforms)
	}
}

func TestLoadFailsWhenPinnedPlatformMissing(t *testing.T) {
	origin := "https://binaries.test"
	build := "999"
	_, err := Load(context.Background(), LoaderOptions{
		BuildNumber: build,
		IndexOrigin: origin,
		CacheDir:    t.TempDir(),
		Platform:    PlatformMacOS,
		Fetch:       mapFetcher{files: map[string]string{}},
	})
	if err == nil || !strings.Contains(err.Error(), "macos") {
		t.Fatalf("expected macos error, got %v", err)
	}
}
