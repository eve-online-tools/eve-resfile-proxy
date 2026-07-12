package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadConfigFileAliases(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
aliases:
  - alias: legacy/icons
    target: icons
  - alias: favicon.ico
    target: ui/icons/favicon.ico
  - alias: .webm
    target: .png
    match:
      path_prefix: ui/icons/
`)

	cfg := defaultConfig()
	if err := loadConfigFile(cfg, path); err != nil {
		t.Fatalf("loadConfigFile() error = %v", err)
	}

	if len(cfg.Aliases) != 3 {
		t.Fatalf("len(Aliases) = %d, want 3", len(cfg.Aliases))
	}
	if cfg.Aliases[0].Alias != "legacy/icons" || cfg.Aliases[0].Target != "icons" {
		t.Fatalf("Aliases[0] = %+v", cfg.Aliases[0])
	}
	if cfg.Aliases[1].Alias != "favicon.ico" || cfg.Aliases[1].Target != "ui/icons/favicon.ico" {
		t.Fatalf("Aliases[1] = %+v", cfg.Aliases[1])
	}
	if cfg.Aliases[2].Match == nil || cfg.Aliases[2].Alias != ".webm" {
		t.Fatalf("Aliases[2] = %+v", cfg.Aliases[2])
	}
}

func TestLoadConfigFileRejectsLegacyAliasKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contents string
		wantErr  string
	}{
		{
			name: "from renamed",
			contents: `
aliases:
  - from: a
    target: b
`,
			wantErr: "from was renamed to alias",
		},
		{
			name: "to renamed",
			contents: `
aliases:
  - alias: a
    to: b
`,
			wantErr: "to was renamed to target",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := writeConfigFile(t, tc.contents)
			cfg := defaultConfig()
			if err := loadConfigFile(cfg, path); err == nil {
				t.Fatal("expected error for legacy alias keys")
			} else if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestLoadConfigFileRejectsLegacyRewrites(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
rewrites:
  - from: a
    to: b
`)

	cfg := defaultConfig()
	if err := loadConfigFile(cfg, path); err == nil {
		t.Fatal("expected error for legacy rewrites key")
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
