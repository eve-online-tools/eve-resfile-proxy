package cdn

import (
	"net/http"

	"github.com/eve-online-tools/eve-resfile-proxy/common/domain"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
)

const (
	BinariesOrigin  = "https://binaries.eveonline.com"
	ResourcesOrigin = "https://resources.eveonline.com"
)

type options struct {
	domain         domain.Domain
	userAgent      string
	withoutRetries bool
	concurrency    *int
}

type Option func(*options)

func WithDomain(domain domain.Domain) Option {
	return func(o *options) {
		o.domain = domain
	}
}

func WithBinariesDomain() Option {
	return WithDomain(domain.Binaries)
}

func WithResourcesDomain() Option {
	return WithDomain(domain.Resources)
}

func WithUserAgent(userAgent string) Option {
	return func(o *options) {
		o.userAgent = userAgent
	}
}

func WithoutRetries() Option {
	return func(o *options) {
		o.withoutRetries = true
	}
}

func WithConcurrency(n int) Option {
	return func(o *options) {
		o.concurrency = &n
	}
}

func New(client *http.Client, opts ...Option) vfs.Fetcher {
	if client == nil {
		client = http.DefaultClient
	}

	var o options
	for _, opt := range opts {
		opt(&o)
	}

	return newFetcher(o.domain, configureHTTPClient(client, o))
}
