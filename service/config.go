package service

import (
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
)

const defaultReadHeaderTimeout = 10 * time.Second

type Config struct {
	Addr              string
	ReadHeaderTimeout time.Duration

	CacheDir     string
	BuildNumber  string
	IndexOrigin  string
	AssetOrigin  string
	ManifestName string
	Platform     index.Platform
	Refresh      bool
	IndexListing bool

	TransformConfig string
}

func (c Config) withDefaults() Config {
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = defaultReadHeaderTimeout
	}
	return c
}
