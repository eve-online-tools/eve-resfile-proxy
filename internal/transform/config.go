package transform

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultCommandTimeout     = 30 * time.Second
	defaultMaxOutputBytes     = 10 * 1024 * 1024
	defaultWasmExport         = "transform"
	defaultWasmFuel           = uint64(100_000_000)
	defaultWasmMaxMemoryBytes = 16 * 1024 * 1024
	defaultWasmMaxOutputBytes = 1024 * 1024
)

type fileConfig struct {
	Transforms []ruleConfig `yaml:"transforms"`
}

type ruleConfig struct {
	Name    string         `yaml:"name"`
	Stable  *bool          `yaml:"stable,omitempty"`
	Match   matchConfig    `yaml:"match"`
	Command *commandConfig `yaml:"command"`
	Wasm    *wasmConfig    `yaml:"wasm"`
}

type matchConfig struct {
	PathPrefix string   `yaml:"path_prefix"`
	PathGlob   string   `yaml:"path_glob"`
	Extensions []string `yaml:"extensions"`
	Filename   string   `yaml:"filename"`
}

type commandConfig struct {
	Args           []string      `yaml:"args"`
	Timeout        time.Duration `yaml:"timeout"`
	MaxOutputBytes int           `yaml:"max_output_bytes"`
}

type wasmConfig struct {
	Module         string `yaml:"module"`
	Export         string `yaml:"export"`
	MaxOutputBytes int    `yaml:"max_output_bytes"`
	Fuel           uint64 `yaml:"fuel"`
	MaxMemoryBytes int    `yaml:"max_memory_bytes"`
}

// LoadConfig reads and validates a transform rules file.
func LoadConfig(path string) (*fileConfig, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, "", fmt.Errorf("parse yaml: %w", err)
	}

	baseDir := filepath.Dir(path)
	if err := validateConfig(&cfg); err != nil {
		return nil, "", err
	}

	return &cfg, baseDir, nil
}

func validateConfig(cfg *fileConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	seen := make(map[string]struct{}, len(cfg.Transforms))
	for i, rule := range cfg.Transforms {
		if rule.Name == "" {
			return fmt.Errorf("transforms[%d]: name is required", i)
		}
		if _, ok := seen[rule.Name]; ok {
			return fmt.Errorf("transforms[%d]: duplicate rule name %q", i, rule.Name)
		}
		seen[rule.Name] = struct{}{}

		hasCommand := rule.Command != nil
		hasWasm := rule.Wasm != nil
		switch {
		case hasCommand && hasWasm:
			return fmt.Errorf("transforms[%d] %q: specify exactly one of command or wasm", i, rule.Name)
		case !hasCommand && !hasWasm:
			return fmt.Errorf("transforms[%d] %q: specify exactly one of command or wasm", i, rule.Name)
		}

		if rule.Match.PathPrefix == "" && rule.Match.PathGlob == "" && len(rule.Match.Extensions) == 0 && rule.Match.Filename == "" {
			return fmt.Errorf("transforms[%d] %q: match must specify at least one criterion", i, rule.Name)
		}

		if hasCommand {
			if len(rule.Command.Args) == 0 {
				return fmt.Errorf("transforms[%d] %q: command.args is required", i, rule.Name)
			}
		}
		if hasWasm {
			if rule.Wasm.Module == "" {
				return fmt.Errorf("transforms[%d] %q: wasm.module is required", i, rule.Name)
			}
		}
	}

	return nil
}

func (r ruleConfig) isStable() bool {
	if r.Stable == nil {
		return true
	}
	return *r.Stable
}

func (r ruleConfig) matchSpec() matchSpec {
	return matchSpec{
		PathPrefix: r.Match.PathPrefix,
		PathGlob:   r.Match.PathGlob,
		Extensions: r.Match.Extensions,
		Filename:   r.Match.Filename,
	}
}

func (c *commandConfig) withDefaults() commandConfig {
	out := *c
	if out.Timeout == 0 {
		out.Timeout = defaultCommandTimeout
	}
	if out.MaxOutputBytes == 0 {
		out.MaxOutputBytes = defaultMaxOutputBytes
	}
	return out
}

func (w *wasmConfig) withDefaults() wasmConfig {
	out := *w
	if out.Export == "" {
		out.Export = defaultWasmExport
	}
	if out.MaxOutputBytes == 0 {
		out.MaxOutputBytes = defaultWasmMaxOutputBytes
	}
	if out.Fuel == 0 {
		out.Fuel = defaultWasmFuel
	}
	if out.MaxMemoryBytes == 0 {
		out.MaxMemoryBytes = defaultWasmMaxMemoryBytes
	}
	return out
}
