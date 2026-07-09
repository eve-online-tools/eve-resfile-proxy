package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/fetch"
	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service/handler"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/conditional"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/getonly"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/heartbeat"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/load"
	"github.com/eve-online-tools/eve-resfile-proxy/service/middleware/resolve"
)

type Service struct {
	server *http.Server
	index  *index.IndexSet
}

func New(ctx context.Context, cfg Config, opts ...Option) (*Service, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	fetchClient := o.fetch
	if fetchClient == nil {
		fetchClient = fetch.NewClient()
	}

	cfg = cfg.withDefaults()

	indexSet, err := index.Load(ctx, index.LoaderOptions{
		BuildNumber:  cfg.BuildNumber,
		IndexOrigin:  cfg.IndexOrigin,
		CacheDir:     cfg.CacheDir,
		ManifestName: cfg.ManifestName,
		Platform:     cfg.Platform,
		Refresh:      cfg.Refresh,
		Fetch:        fetchClient,
	})
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}

	handler := handler.New(indexSet)

	// Middlewares are applied in-order. Any additional middlewares are appended at the end of the chain.
	middlewares := middleware.MiddlewareChain{
		heartbeat.Middleware("/healthz"),
		heartbeat.Middleware("/livez"),
		getonly.Middleware,
		resolve.Middleware(indexSet),
		load.Middleware(o.cache, fetchClient, cfg.AssetOrigin),
		conditional.Middleware,
	}
	middlewares = append(middlewares, o.middlewares...)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           middlewares.For(handler),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	return &Service{
		server: httpServer,
		index:  indexSet,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	log.Printf(
		"loaded EVE indexes (build %s, platforms %s, windows=%d macos=%d entries)",
		s.index.BuildNumber,
		formatPlatforms(s.index.LoadedPlatforms),
		s.index.EntryCount(index.PlatformWindows),
		s.index.EntryCount(index.PlatformMacOS),
	)

	go func() {
		log.Printf("listening on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	return nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		if closeErr := s.server.Close(); closeErr != nil {
			return fmt.Errorf("shutdown: %w (close: %v)", err, closeErr)
		}
		return err
	}
	return nil
}

func formatPlatforms(platforms []index.Platform) string {
	parts := make([]string, len(platforms))
	for i, p := range platforms {
		parts[i] = string(p)
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, " "))
}
