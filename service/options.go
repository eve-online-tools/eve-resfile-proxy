package service

import (
	"net/http"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/fetch"
	"github.com/eve-online-tools/eve-resfile-proxy/service/assetcache"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware"
)

type Option func(*options)

type options struct {
	fetch *fetch.Client
	cache *assetcache.Store

	middlewares middleware.MiddlewareChain
}

func WithHttpClient(client *http.Client) Option {
	return func(o *options) {
		o.fetch = fetch.NewFromClient(client)
	}
}

func WithFetch(client *fetch.Client) Option {
	return func(o *options) {
		o.fetch = client
	}
}

func WithCache(cache *assetcache.Store) Option {
	return func(o *options) {
		o.cache = cache
	}
}

func WithMiddleware(middlewares middleware.Middleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, middlewares)
	}
}
