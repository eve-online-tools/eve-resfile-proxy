package http

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	diskcache "github.com/eve-online-tools/eve-resfile-proxy/cache/disk"
	"github.com/eve-online-tools/eve-resfile-proxy/service/buildnumber"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/handler"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/conditional"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/heartbeat"
	indexmw "github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/index"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/load"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/method"
)

// ServerConfig holds HTTP listener settings.
type ServerConfig struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	IndexListing      bool
}

func (c *ServerConfig) WithDefaults() {
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = 10 * time.Second
	}
}

type Server struct {
	server *http.Server
	logger *slog.Logger
}

func NewServer(
	cfg *ServerConfig,
	fsys fs.FS,
	build *buildnumber.BuildNumber,
	diskCache *diskcache.Cache,
	logger *slog.Logger,
) *Server {
	cfg.WithDefaults()
	if logger == nil {
		logger = slog.Default()
	}

	middlewares := MiddlewareChain{
		heartbeat.Middleware("/healthz"),
		heartbeat.Middleware("/livez"),
		method.Middleware,
		indexmw.Middleware(cfg.IndexListing, fsys, build),
		load.Middleware(fsys, diskCache),
		conditional.Middleware,
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           middlewares.For(handler.Respond(build)),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	return &Server{
		server: httpServer,
		logger: logger,
	}
}

func (s *Server) Start() {
	go func() {
		s.logger.Info("listening", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("http server", "err", err)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		if closeErr := s.server.Close(); closeErr != nil {
			return fmt.Errorf("shutdown: %w (close: %v)", err, closeErr)
		}
		return err
	}
	return nil
}
