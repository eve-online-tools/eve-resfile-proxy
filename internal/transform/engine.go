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

// Result is the output of a transform, including whether it came from disk cache.
type Result struct {
	Data      []byte
	FromCache bool
}

// Engine applies the first matching transform rule to an asset.
type Engine struct {
	rules     []compiledRule
	diskCache *DiskCache
}

type compiledRule struct {
	name    string
	stable  bool
	matcher *matcher
	run     func(ctx context.Context, in Input) ([]byte, error)
}

// Transform returns transformed bytes when a rule matches, or the original data unchanged.
func (e *Engine) Transform(ctx context.Context, in Input) (Result, error) {
	if e == nil {
		return Result{Data: in.Data}, nil
	}

	for _, rule := range e.rules {
		if !rule.matcher.matches(in.ResPath) {
			continue
		}

		if e.diskCache != nil && rule.stable {
			if data, ok, err := e.diskCache.Read(string(in.Platform), rule.name, in.CDNPath); err != nil {
				return Result{}, fmt.Errorf("transform %q: read cache: %w", rule.name, err)
			} else if ok {
				return Result{Data: data, FromCache: true}, nil
			}
		}

		out, err := rule.run(ctx, in)
		if err != nil {
			return Result{}, fmt.Errorf("transform %q: %w", rule.name, err)
		}

		if e.diskCache != nil && rule.stable {
			_ = e.diskCache.Write(string(in.Platform), rule.name, in.CDNPath, out)
		}

		return Result{Data: out}, nil
	}

	return Result{Data: in.Data}, nil
}
