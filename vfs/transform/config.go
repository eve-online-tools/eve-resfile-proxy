// Package transform wraps fs.FS and applies configurable transforms on read.
package transform

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultCommandTimeout     = 30 * time.Second
	defaultMaxOutputBytes     = 128 << 20 // 128 MiB
	defaultWasmMaxMemoryBytes = 128 << 20
	defaultWasmFuel           = 100_000_000
	defaultWasmExport         = "transform"
)

// Limits configures global caps for all transforms.
type Limits struct {
	MaxOutputBytes int    `yaml:"max_output_bytes"`
	MaxMemoryBytes int    `yaml:"max_memory_bytes"`
	Fuel           uint64 `yaml:"fuel"`
}

func (l Limits) withDefaults() Limits {
	out := l
	if out.MaxOutputBytes == 0 {
		out.MaxOutputBytes = defaultMaxOutputBytes
	}
	if out.MaxMemoryBytes == 0 {
		out.MaxMemoryBytes = defaultWasmMaxMemoryBytes
	}
	if out.Fuel == 0 {
		out.Fuel = defaultWasmFuel
	}
	return out
}

// Transform configures one transform applied to matching vfs paths.
type Transform struct {
	Name    string   `yaml:"name"`
	Match   Match    `yaml:"match"`
	Command *Command `yaml:"command"`
	Wasm    *Wasm    `yaml:"wasm"`
}

// Match selects paths a transform applies to. All specified criteria must match.
type Match struct {
	PathPrefix string   `yaml:"path_prefix"`
	PathGlob   string   `yaml:"path_glob"`
	Extensions []string `yaml:"extensions"`
	Filename   string   `yaml:"filename"`
}

// Command runs an external process for the transform.
type Command struct {
	Args    []string      `yaml:"args"`
	Timeout time.Duration `yaml:"timeout"`
}

// Wasm runs a WebAssembly module for the transform.
type Wasm struct {
	Module string `yaml:"module"`
	Export string `yaml:"export"`
}

type assetInput struct {
	ResPath string
	CDNPath string
	Data    []byte
}

type compiledTransform struct {
	name    string
	matcher *matcher
	run     func(ctx context.Context, in assetInput) ([]byte, error)
}

type compiledCommand struct {
	args           []string
	timeout        time.Duration
	maxOutputBytes int
}

type compiledWasm struct {
	module         string
	export         string
	maxOutputBytes int
	maxMemoryBytes int
	fuel           uint64
}

func compileTransforms(transforms []Transform, limits Limits, baseDir string) ([]compiledTransform, error) {
	if len(transforms) == 0 {
		return nil, nil
	}
	if err := validateTransforms(transforms); err != nil {
		return nil, err
	}

	limits = limits.withDefaults()
	compiled := make([]compiledTransform, 0, len(transforms))
	closers := []func(context.Context) error{}

	for _, tr := range transforms {
		m, err := newMatcher(matchSpec{
			PathPrefix: tr.Match.PathPrefix,
			PathGlob:   tr.Match.PathGlob,
			Extensions: tr.Match.Extensions,
			Filename:   tr.Match.Filename,
		})
		if err != nil {
			closeAll(context.Background(), closers)
			return nil, fmt.Errorf("transform %q: %w", tr.Name, err)
		}

		var run func(context.Context, assetInput) ([]byte, error)
		switch {
		case tr.Command != nil:
			cmdCfg := compileCommand(*tr.Command, limits.MaxOutputBytes)
			run = newCommandRunner(tr.Name, cmdCfg)
		case tr.Wasm != nil:
			wasmCfg := compileWasm(*tr.Wasm, limits)
			runner, err := newWasmRunner(baseDir, wasmCfg)
			if err != nil {
				closeAll(context.Background(), closers)
				return nil, fmt.Errorf("transform %q: %w", tr.Name, err)
			}
			closers = append(closers, runner.close)
			run = runner.transform
		}

		compiled = append(compiled, compiledTransform{
			name:    tr.Name,
			matcher: m,
			run:     run,
		})
	}

	return compiled, nil
}

func validateTransforms(transforms []Transform) error {
	seen := make(map[string]struct{}, len(transforms))
	for i, tr := range transforms {
		if tr.Name == "" {
			return fmt.Errorf("transforms[%d]: name is required", i)
		}
		if _, ok := seen[tr.Name]; ok {
			return fmt.Errorf("transforms[%d]: duplicate transform name %q", i, tr.Name)
		}
		seen[tr.Name] = struct{}{}

		hasCommand := tr.Command != nil
		hasWasm := tr.Wasm != nil
		switch {
		case hasCommand && hasWasm:
			return fmt.Errorf("transforms[%d] %q: specify exactly one of command or wasm", i, tr.Name)
		case !hasCommand && !hasWasm:
			return fmt.Errorf("transforms[%d] %q: specify exactly one of command or wasm", i, tr.Name)
		}

		if tr.Match.PathPrefix == "" && tr.Match.PathGlob == "" && len(tr.Match.Extensions) == 0 && tr.Match.Filename == "" {
			return fmt.Errorf("transforms[%d] %q: match must specify at least one criterion", i, tr.Name)
		}

		if hasCommand {
			if len(tr.Command.Args) == 0 {
				return fmt.Errorf("transforms[%d] %q: command.args is required", i, tr.Name)
			}
		}
		if hasWasm {
			if tr.Wasm.Module == "" {
				return fmt.Errorf("transforms[%d] %q: wasm.module is required", i, tr.Name)
			}
		}
	}
	return nil
}

func compileCommand(c Command, maxOutputBytes int) compiledCommand {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = defaultCommandTimeout
	}
	return compiledCommand{
		args:           c.Args,
		timeout:        timeout,
		maxOutputBytes: maxOutputBytes,
	}
}

func compileWasm(w Wasm, limits Limits) compiledWasm {
	export := w.Export
	if export == "" {
		export = defaultWasmExport
	}
	return compiledWasm{
		module:         w.Module,
		export:         export,
		maxOutputBytes: limits.MaxOutputBytes,
		maxMemoryBytes: limits.MaxMemoryBytes,
		fuel:           limits.Fuel,
	}
}

func closeAll(ctx context.Context, closers []func(context.Context) error) {
	for _, closeFn := range closers {
		_ = closeFn(ctx)
	}
}

func matchingRule(rules []compiledTransform, path string) (string, bool) {
	for _, rule := range rules {
		if rule.matcher.matches(path) {
			return rule.name, true
		}
	}
	return "", false
}

func runTransform(rules []compiledTransform, ctx context.Context, path string, in assetInput) ([]byte, error) {
	for _, rule := range rules {
		if !rule.matcher.matches(path) {
			continue
		}
		out, err := rule.run(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("transform %q: %w", rule.name, err)
		}
		return out, nil
	}
	return in.Data, nil
}
