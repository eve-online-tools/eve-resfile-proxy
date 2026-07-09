package request

import (
	"context"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
)

type Asset struct {
	ResPath      string
	CDNPath      string
	Platform     index.Platform
	Data         []byte
	CacheStatus  CacheStatus
	ETag         string
	LastModified time.Time
}

type CacheStatus string

const (
	CacheStatusHit  CacheStatus = "HIT"
	CacheStatusMiss CacheStatus = "MISS"
)

type cxAsset struct{}

func WithAsset(ctx context.Context, asset Asset) context.Context {
	return context.WithValue(ctx, cxAsset{}, asset)
}

func AssetFromContext(ctx context.Context) (Asset, bool) {
	asset, ok := ctx.Value(cxAsset{}).(Asset)
	return asset, ok
}
