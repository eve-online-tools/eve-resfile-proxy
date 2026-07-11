package cache

import (
	"context"
	"log/slog"

	"github.com/eve-online-tools/eve-resfile-proxy/cache"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
)

type Option func(*fetcher)

// WithLogger logs cache store failures at warn level.
func WithLogger(logger *slog.Logger) Option {
	return func(f *fetcher) {
		f.logger = logger
	}
}

type fetcher struct {
	inner  vfs.Fetcher
	cache  cache.Cache
	logger *slog.Logger
}

var _ vfs.Fetcher = (*fetcher)(nil)

// New returns a Fetcher that reads through cache before delegating to inner.
// When c is nil, inner is returned unchanged.
func New(inner vfs.Fetcher, c cache.Cache, opts ...Option) vfs.Fetcher {
	if c == nil {
		return inner
	}
	f := &fetcher{inner: inner, cache: c}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *fetcher) FetchEntry(ctx context.Context, entry vfs.Entry) ([]byte, error) {
	return f.FetchPath(ctx, entry.CDNPath)
}

func (f *fetcher) FetchPath(ctx context.Context, path string) ([]byte, error) {
	if data, ok, err := f.cache.Get(ctx, path); err != nil || ok {
		return data, err
	}

	data, err := f.inner.FetchPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := f.cache.Store(ctx, path, data); err != nil && f.logger != nil {
		f.logger.Warn("cache store failed", "cdn_path", path, "err", err)
	}
	return data, nil
}
