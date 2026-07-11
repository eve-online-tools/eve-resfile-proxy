package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/common/platform"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/rewrite"
	vfstransform "github.com/eve-online-tools/eve-resfile-proxy/vfs/transform"
	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	ServerName      string                   `yaml:"server"`
	BuildNumber     string                   `yaml:"build"`
	Platforms       []string                 `yaml:"platforms"`
	CacheDir        string                   `yaml:"cache"`
	Debug           *bool                    `yaml:"debug"`
	FullTree        *bool                    `yaml:"full_tree"`
	Addr            string                   `yaml:"addr"`
	NoIndex         *bool                    `yaml:"no_index"`
	IndexListing    *bool                    `yaml:"index_listing"`
	Rewrites        []rewrite.Rule           `yaml:"rewrites"`
	TransformLimits vfstransform.Limits      `yaml:"transform_limits"`
	Transforms      []vfstransform.Transform `yaml:"transforms"`
}

func configPathFromArgs(args []string) string {
	for i, arg := range args {
		switch {
		case arg == "--config", arg == "-config":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(arg, "--config="):
			return strings.TrimPrefix(arg, "--config=")
		}
	}
	return ""
}

func loadConfigFile(cfg *config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var fileCfg fileConfig
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	cfg.ConfigDir = filepath.Dir(path)
	return fileCfg.applyTo(cfg)
}

func (fc *fileConfig) applyTo(cfg *config) error {
	if fc.ServerName != "" {
		cfg.ServerName = fc.ServerName
	}
	if fc.BuildNumber != "" {
		cfg.BuildNumber = fc.BuildNumber
	}
	if len(fc.Platforms) > 0 {
		platforms, err := parsePlatforms(fc.Platforms)
		if err != nil {
			return fmt.Errorf("platforms: %w", err)
		}
		cfg.Platforms = platforms
	}
	if fc.CacheDir != "" {
		cfg.CacheDir = fc.CacheDir
	}
	if fc.Debug != nil {
		cfg.Debug = *fc.Debug
	}
	if fc.FullTree != nil {
		cfg.FullTree = *fc.FullTree
	}
	if fc.Addr != "" {
		cfg.ServerConfig.Addr = fc.Addr
	}
	if fc.NoIndex != nil {
		cfg.noIndex = *fc.NoIndex
		cfg.ServerConfig.IndexListing = !*fc.NoIndex
	}
	if fc.IndexListing != nil {
		cfg.ServerConfig.IndexListing = *fc.IndexListing
		cfg.noIndex = !*fc.IndexListing
	}
	if len(fc.Rewrites) > 0 {
		cfg.Rewrites = fc.Rewrites
	}
	if fc.TransformLimits != (vfstransform.Limits{}) {
		cfg.TransformLimits = fc.TransformLimits
	}
	if len(fc.Transforms) > 0 {
		cfg.Transforms = fc.Transforms
	}
	return nil
}

func parsePlatforms(names []string) ([]platform.Platform, error) {
	platforms := make([]platform.Platform, 0, len(names))
	for _, name := range names {
		p, err := platform.Parse(name)
		if err != nil {
			return nil, err
		}
		platforms = append(platforms, p)
	}
	return platforms, nil
}
