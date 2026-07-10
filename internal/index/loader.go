package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type LoaderOptions struct {
	BuildNumber  string
	IndexOrigin  string
	CacheDir     string
	ManifestName string
	Platform     Platform
	Refresh      bool
	Fetch        TextFetcher
}

type TextFetcher interface {
	GetText(ctx context.Context, url string) (string, error)
}

func Load(ctx context.Context, opts LoaderOptions) (*IndexSet, error) {
	fetcher := opts.Fetch
	if fetcher == nil {
		return nil, fmt.Errorf("fetcher is required")
	}

	buildNumber, err := resolveBuildNumber(ctx, fetcher, opts)
	if err != nil {
		return nil, err
	}

	platforms := PlatformsToLoad(opts.Platform)
	if cached, ok, err := tryLoadFromCache(opts.CacheDir, buildNumber, platforms, opts.Refresh); err != nil {
		return nil, err
	} else if ok {
		return cached, nil
	}

	set := &IndexSet{
		BuildNumber:     buildNumber,
		PlatformMaps:    make(map[Platform]map[string]Entry),
		LoadedPlatforms: make([]Platform, 0, len(platforms)),
	}

	entryCounts := make(map[Platform]int)
	var loadErrors []error
	allowDegraded := opts.Platform == "" && len(platforms) > 1

	for _, p := range platforms {
		merged, count, err := loadPlatform(ctx, fetcher, opts, buildNumber, p)
		if err != nil {
			if allowDegraded {
				log.Printf("warning: failed to load %s indexes: %v", p, err)
				loadErrors = append(loadErrors, fmt.Errorf("%s: %w", p, err))
				continue
			}
			return nil, err
		}
		set.PlatformMaps[p] = merged
		set.LoadedPlatforms = append(set.LoadedPlatforms, p)
		entryCounts[p] = count
	}

	if len(set.LoadedPlatforms) == 0 {
		if len(loadErrors) > 0 {
			return nil, loadErrors[0]
		}
		return nil, fmt.Errorf("no platform indexes loaded for build %s", buildNumber)
	}

	meta := CacheMeta{
		BuildNumber:         buildNumber,
		LoadedPlatforms:     set.LoadedPlatforms,
		PlatformEntryCounts: entryCounts,
	}
	if opts.CacheDir != "" {
		if err := writeMeta(metaPath(opts.CacheDir, buildNumber), meta); err != nil {
			return nil, err
		}
	}

	return set, nil
}

func tryLoadFromCache(cacheDir, buildNumber string, platforms []Platform, refresh bool) (*IndexSet, bool, error) {
	if cacheDir == "" || refresh {
		return nil, false, nil
	}

	meta, ok, err := readMeta(metaPath(cacheDir, buildNumber))
	if err != nil || !ok {
		return nil, false, err
	}
	if meta.BuildNumber != buildNumber {
		return nil, false, nil
	}

	set := &IndexSet{
		BuildNumber:     buildNumber,
		PlatformMaps:    make(map[Platform]map[string]Entry),
		LoadedPlatforms: make([]Platform, 0, len(platforms)),
	}

	for _, p := range platforms {
		merged, found, err := readPlatformMerged(platformMergedPath(cacheDir, buildNumber, p))
		if err != nil {
			return nil, false, err
		}
		if !found {
			return nil, false, nil
		}
		set.PlatformMaps[p] = merged
		set.LoadedPlatforms = append(set.LoadedPlatforms, p)
	}

	return set, true, nil
}

func loadPlatform(ctx context.Context, fetcher TextFetcher, opts LoaderOptions, buildNumber string, p Platform) (map[string]Entry, int, error) {
	if opts.CacheDir != "" {
		if err := ensureDir(platformCacheDir(opts.CacheDir, buildNumber, p)); err != nil {
			return nil, 0, err
		}
	}

	buildIndexContent, err := loadBuildIndex(ctx, fetcher, opts, buildNumber, p)
	if err != nil {
		return nil, 0, err
	}

	buildEntries, err := ParseBuildIndex(buildIndexContent)
	if err != nil {
		return nil, 0, err
	}

	paths := PlatformIndexPaths[p]
	globalEntry, ok := FindBuildIndexEntry(buildEntries, paths.Global)
	if !ok {
		return nil, 0, fmt.Errorf("%s not found in %s build index", paths.Global, p)
	}

	globalContent, err := loadResfileCSV(ctx, fetcher, opts, buildNumber, p, globalEntry.CDNPath, resfileGlobalPath(opts.CacheDir, buildNumber, p))
	if err != nil {
		return nil, 0, err
	}

	globalMap, err := ParseResfileIndex(globalContent)
	if err != nil {
		return nil, 0, err
	}

	var osMap map[string]Entry
	if osEntry, ok := FindBuildIndexEntry(buildEntries, paths.OSSpecific); ok {
		osContent, err := loadResfileCSV(ctx, fetcher, opts, buildNumber, p, osEntry.CDNPath, resfileOSPath(opts.CacheDir, buildNumber, p))
		if err != nil {
			return nil, 0, err
		}
		osMap, err = ParseResfileIndex(osContent)
		if err != nil {
			return nil, 0, err
		}
	} else {
		log.Printf("info: %s OS-specific resfile index not found in build index, using global only", p)
	}

	merged := MergeWithinPlatform(osMap, globalMap)
	if opts.CacheDir != "" {
		if err := writePlatformMerged(platformMergedPath(opts.CacheDir, buildNumber, p), merged); err != nil {
			return nil, 0, err
		}
	}

	return merged, len(merged), nil
}

func loadBuildIndex(ctx context.Context, fetcher TextFetcher, opts LoaderOptions, buildNumber string, p Platform) (string, error) {
	cachedPath := buildIndexPath(opts.CacheDir, buildNumber, p)
	if opts.CacheDir != "" && !opts.Refresh {
		if cached, ok, err := readCachedText(cachedPath); err != nil {
			return "", err
		} else if ok {
			return cached, nil
		}
	}

	url := BuildIndexURL(opts.IndexOrigin, buildNumber, p)
	content, err := fetcher.GetText(ctx, url)
	if err != nil {
		return "", fmt.Errorf("load %s build index: %w", p, err)
	}
	if opts.CacheDir != "" {
		if err := writeCachedText(cachedPath, content); err != nil {
			return "", err
		}
	}
	return content, nil
}

func loadResfileCSV(ctx context.Context, fetcher TextFetcher, opts LoaderOptions, buildNumber string, p Platform, cdnPath, cachedPath string) (string, error) {
	if opts.CacheDir != "" && !opts.Refresh {
		if cached, ok, err := readCachedText(cachedPath); err != nil {
			return "", err
		} else if ok {
			return cached, nil
		}
	}

	url := strings.TrimRight(opts.IndexOrigin, "/") + "/" + strings.TrimPrefix(cdnPath, "/")
	content, err := fetcher.GetText(ctx, url)
	if err != nil {
		return "", fmt.Errorf("load %s resfile index from %s: %w", p, cdnPath, err)
	}
	if opts.CacheDir != "" {
		if err := writeCachedText(cachedPath, content); err != nil {
			return "", err
		}
	}
	return content, nil
}

type clientManifest struct {
	BuildNumber string `json:"buildNumber"`
	Build       string `json:"build"`
}

func resolveBuildNumber(ctx context.Context, fetcher TextFetcher, opts LoaderOptions) (string, error) {
	if opts.BuildNumber != "" {
		return opts.BuildNumber, nil
	}

	manifestURL := strings.TrimRight(opts.IndexOrigin, "/") + "/" + opts.ManifestName
	body, err := fetcher.GetText(ctx, manifestURL)
	if err != nil {
		return "", fmt.Errorf("resolve build number: %w", err)
	}

	var manifest clientManifest
	if err := json.Unmarshal([]byte(body), &manifest); err != nil {
		return "", fmt.Errorf("decode manifest %s: %w", manifestURL, err)
	}

	buildNumber := manifest.BuildNumber
	if buildNumber == "" {
		buildNumber = manifest.Build
	}
	if buildNumber == "" {
		return "", fmt.Errorf("missing buildNumber in %s", manifestURL)
	}
	return buildNumber, nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
