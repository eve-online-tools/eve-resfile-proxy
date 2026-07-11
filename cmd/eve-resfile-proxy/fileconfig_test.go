package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/common/platform"
)

func TestLoadConfigFile(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
server: singularity
build: "123456"
platforms:
  - windows
  - macOS
cache: /tmp/cache
debug: true
full_tree: true
addr: ":9090"
no_index: true
`)

	cfg := defaultConfig()
	if err := loadConfigFile(cfg, path); err != nil {
		t.Fatalf("loadConfigFile() error = %v", err)
	}

	if cfg.ServerName != "singularity" {
		t.Fatalf("ServerName = %q, want singularity", cfg.ServerName)
	}
	if cfg.BuildNumber != "123456" {
		t.Fatalf("BuildNumber = %q, want 123456", cfg.BuildNumber)
	}
	if len(cfg.Platforms) != 2 || cfg.Platforms[0] != platform.Windows || cfg.Platforms[1] != platform.Mac {
		t.Fatalf("Platforms = %v, want [windows macOS]", cfg.Platforms)
	}
	if cfg.CacheDir != "/tmp/cache" {
		t.Fatalf("CacheDir = %q, want /tmp/cache", cfg.CacheDir)
	}
	if !cfg.Debug {
		t.Fatal("Debug = false, want true")
	}
	if !cfg.FullTree {
		t.Fatal("FullTree = false, want true")
	}
	if cfg.ServerConfig.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", cfg.ServerConfig.Addr)
	}
	if !cfg.noIndex {
		t.Fatal("noIndex = false, want true")
	}
	if cfg.ServerConfig.IndexListing {
		t.Fatal("IndexListing = true, want false")
	}
}

func TestLoadConfigFilePartial(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
cache: /var/cache/eve
`)

	cfg := defaultConfig()
	if err := loadConfigFile(cfg, path); err != nil {
		t.Fatalf("loadConfigFile() error = %v", err)
	}

	if cfg.ServerName != defaultServer {
		t.Fatalf("ServerName = %q, want default %q", cfg.ServerName, defaultServer)
	}
	if cfg.CacheDir != "/var/cache/eve" {
		t.Fatalf("CacheDir = %q, want /var/cache/eve", cfg.CacheDir)
	}
}

func TestLoadConfigFileRewrites(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
rewrites:
  - from: legacy/icons
    to: icons
  - from: favicon.ico
    to: ui/icons/favicon.ico
`)

	cfg := defaultConfig()
	if err := loadConfigFile(cfg, path); err != nil {
		t.Fatalf("loadConfigFile() error = %v", err)
	}

	if len(cfg.Rewrites) != 2 {
		t.Fatalf("len(Rewrites) = %d, want 2", len(cfg.Rewrites))
	}
	if cfg.Rewrites[0].From != "legacy/icons" || cfg.Rewrites[0].To != "icons" {
		t.Fatalf("Rewrites[0] = %+v", cfg.Rewrites[0])
	}
	if cfg.Rewrites[1].From != "favicon.ico" || cfg.Rewrites[1].To != "ui/icons/favicon.ico" {
		t.Fatalf("Rewrites[1] = %+v", cfg.Rewrites[1])
	}
}

func TestLoadConfigFileTransforms(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
transform_limits:
  max_output_bytes: 67108864
  fuel: 50000000
transforms:
  - name: upper
    match:
      extensions: [".txt"]
    command:
      args: ["cat"]
`)

	cfg := defaultConfig()
	if err := loadConfigFile(cfg, path); err != nil {
		t.Fatalf("loadConfigFile() error = %v", err)
	}
	if len(cfg.Transforms) != 1 {
		t.Fatalf("len(Transforms) = %d, want 1", len(cfg.Transforms))
	}
	if cfg.Transforms[0].Name != "upper" {
		t.Fatalf("Transforms[0].Name = %q, want upper", cfg.Transforms[0].Name)
	}
	if cfg.TransformLimits.MaxOutputBytes != 67108864 {
		t.Fatalf("TransformLimits.MaxOutputBytes = %d", cfg.TransformLimits.MaxOutputBytes)
	}
	if cfg.TransformLimits.Fuel != 50000000 {
		t.Fatalf("TransformLimits.Fuel = %d", cfg.TransformLimits.Fuel)
	}
	if cfg.ConfigDir != filepath.Dir(path) {
		t.Fatalf("ConfigDir = %q, want %q", cfg.ConfigDir, filepath.Dir(path))
	}
}

func TestConfigPathFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "long form",
			args: []string{"--config", "config.yaml", "--server", "foo"},
			want: "config.yaml",
		},
		{
			name: "equals form",
			args: []string{"--server=foo", "--config=/etc/eve.yaml"},
			want: "/etc/eve.yaml",
		},
		{
			name: "missing",
			args: []string{"--server", "foo"},
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := configPathFromArgs(tc.args); got != tc.want {
				t.Fatalf("configPathFromArgs() = %q, want %q", got, tc.want)
			}
		})
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return path
}
