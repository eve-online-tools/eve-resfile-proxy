package http

import "net/http"

type Middleware func(http.Handler) http.Handler
type MiddlewareChain []Middleware

// Chain composes middleware around an endpoint handler. Middlewares are applied
// in order: the first middleware is outermost, the last is innermost.
func Chain(handler http.Handler, mws ...Middleware) http.Handler {
	h := handler
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func (m MiddlewareChain) For(handler http.Handler) http.Handler {
	return Chain(handler, m...)
}
