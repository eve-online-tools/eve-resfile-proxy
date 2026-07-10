package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service"
)

const (
	defaultAddr         = ":8080"
	defaultIndexOrigin  = "https://binaries.eveonline.com"
	defaultAssetOrigin  = "https://resources.eveonline.com"
	defaultManifestName = "eveclient_TQ.json"
)

type config struct {
	Addr            string
	CacheDir        string
	BuildNumber     string
	Platform        index.Platform
	IndexOrigin     string
	AssetOrigin     string
	ManifestName    string
	Refresh         bool
	IndexListing    bool
	TransformConfig string
}

func parseConfig() (*config, error) {
	cfg := &config{}
	var platform string

	var noIndex bool

	flag.StringVar(&cfg.Addr, "addr", defaultAddr, "HTTP listen address")
	flag.StringVar(&cfg.CacheDir, "cache", "", "Cache directory for indexes and assets; omit to disable caching")
	flag.StringVar(&cfg.BuildNumber, "build", "", "Pin to a specific client build; omit for latest TQ")
	flag.StringVar(&platform, "platform", "", "Load indexes for one platform only: windows or macos")
	flag.StringVar(&cfg.IndexOrigin, "index-origin", defaultIndexOrigin, "EVE binaries CDN origin")
	flag.StringVar(&cfg.AssetOrigin, "asset-origin", defaultAssetOrigin, "EVE resources CDN origin")
	flag.StringVar(&cfg.ManifestName, "manifest", defaultManifestName, "Client manifest filename")
	flag.BoolVar(&cfg.Refresh, "refresh", false, "Force re-download of index files")
	flag.BoolVar(&noIndex, "no-index", false, "Disable directory listing for paths ending in /")
	flag.StringVar(&cfg.TransformConfig, "transform-config", "", "Path to YAML transform rules file")

	flag.Parse()

	cfg.IndexListing = !noIndex

	parsed, err := index.ParsePlatform(platform)
	if err != nil {
		return nil, err
	}
	cfg.Platform = parsed

	if cfg.IndexOrigin == "" {
		return nil, fmt.Errorf("index-origin must not be empty")
	}
	if cfg.AssetOrigin == "" {
		return nil, fmt.Errorf("asset-origin must not be empty")
	}

	return cfg, nil
}

func (c *config) serviceConfig() service.Config {
	return service.Config{
		Addr:              c.Addr,
		ReadHeaderTimeout: 10 * time.Second,
		CacheDir:          c.CacheDir,
		BuildNumber:       c.BuildNumber,
		IndexOrigin:       c.IndexOrigin,
		AssetOrigin:       c.AssetOrigin,
		ManifestName:      c.ManifestName,
		Platform:          c.Platform,
		Refresh:           c.Refresh,
		IndexListing:      c.IndexListing,
		TransformConfig:   c.TransformConfig,
	}
}
