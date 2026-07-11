package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eve-online-tools/eve-resfile-proxy/common/platform"
	"github.com/eve-online-tools/eve-resfile-proxy/service"
	svchttp "github.com/eve-online-tools/eve-resfile-proxy/service/http"
)

const defaultServer = "tranquility"

type config struct {
	service.Config
	Debug         bool
	noIndex       bool
	platformsFlag string
}

func defaultConfig() *config {
	return &config{
		Config: service.Config{
			ServerName: defaultServer,
			ServerConfig: svchttp.ServerConfig{
				Addr:         ":8080",
				IndexListing: true,
			},
		},
	}
}

func parseConfig() (*config, error) {
	cfg := defaultConfig()

	configPath := configPathFromArgs(os.Args[1:])
	if configPath != "" {
		if err := loadConfigFile(cfg, configPath); err != nil {
			return nil, err
		}
	}

	flag.StringVar(&cfg.ServerName, "server", cfg.ServerName, "EVE server name (e.g. tranquility, singularity)")
	flag.StringVar(&cfg.BuildNumber, "build", cfg.BuildNumber, "Pin to a specific client build; omit for latest")
	flag.StringVar(&cfg.platformsFlag, "platforms", cfg.platformsFlag, "Comma-separated platforms to load (e.g. windows, macOS); default all available")
	flag.StringVar(&cfg.CacheDir, "cache", cfg.CacheDir, "Cache directory for indexes and assets; omit to disable caching")
	flag.BoolVar(&cfg.Debug, "debug", cfg.Debug, "Enable debug logging")
	flag.BoolVar(&cfg.FullTree, "full-tree", cfg.FullTree, "Expose full app and res filesystem trees")
	flag.StringVar(&cfg.ServerConfig.Addr, "addr", cfg.ServerConfig.Addr, "HTTP listen address")
	flag.BoolVar(&cfg.noIndex, "no-index", cfg.noIndex, "Disable directory listing for paths ending in /")
	flag.String("config", "", "Path to YAML config file")

	flag.Parse()

	if cfg.ServerName == "" {
		return nil, fmt.Errorf("server must not be empty")
	}

	if cfg.platformsFlag != "" {
		platforms, err := platform.ParseList(cfg.platformsFlag)
		if err != nil {
			return nil, fmt.Errorf("platforms: %w", err)
		}
		cfg.Platforms = platforms
	}

	if cfg.noIndex {
		cfg.ServerConfig.IndexListing = false
	}

	return cfg, nil
}
