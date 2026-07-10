package transform

import (
	"context"
	"fmt"
)

// LoadEngine reads transform rules from path and prepares command and wasm backends.
func LoadEngine(path string) (*Engine, error) {
	cfg, baseDir, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	engine := &Engine{}
	closers := []func(context.Context) error{}

	for _, rule := range cfg.Transforms {
		m, err := newMatcher(rule.matchSpec())
		if err != nil {
			closeAll(context.Background(), closers)
			return nil, fmt.Errorf("rule %q: %w", rule.Name, err)
		}

		var run func(context.Context, Input) ([]byte, error)
		switch {
		case rule.Command != nil:
			cmdCfg := rule.Command.withDefaults()
			run = newCommandRunner(rule.Name, cmdCfg)
		case rule.Wasm != nil:
			wasmCfg := rule.Wasm.withDefaults()
			runner, err := newWasmRunner(baseDir, wasmCfg)
			if err != nil {
				closeAll(context.Background(), closers)
				return nil, fmt.Errorf("rule %q: %w", rule.Name, err)
			}
			closers = append(closers, runner.close)
			run = runner.transform
		}

		engine.rules = append(engine.rules, compiledRule{
			name:    rule.Name,
			matcher: m,
			run:     run,
		})
	}

	return engine, nil
}

func closeAll(ctx context.Context, closers []func(context.Context) error) {
	for _, closeFn := range closers {
		_ = closeFn(ctx)
	}
}
