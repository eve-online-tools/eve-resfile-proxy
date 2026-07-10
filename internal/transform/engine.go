package transform

import (
	"context"
	"fmt"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
)

// Input is passed to each transform backend for a matched asset.
type Input struct {
	ResPath  string
	CDNPath  string
	Platform index.Platform
	Data     []byte
}

// Engine applies the first matching transform rule to an asset.
type Engine struct {
	rules []compiledRule
}

type compiledRule struct {
	name    string
	matcher *matcher
	run     func(ctx context.Context, in Input) ([]byte, error)
}

// Transform returns transformed bytes when a rule matches, or the original data unchanged.
func (e *Engine) Transform(ctx context.Context, in Input) ([]byte, error) {
	if e == nil {
		return in.Data, nil
	}

	for _, rule := range e.rules {
		if !rule.matcher.matches(in.ResPath) {
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
