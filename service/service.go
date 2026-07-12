package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	stdhttp "net/http"

	"github.com/eve-online-tools/eve-resfile-proxy/cache"
	diskcache "github.com/eve-online-tools/eve-resfile-proxy/cache/disk"
	"github.com/eve-online-tools/eve-resfile-proxy/common/platform"
	"github.com/eve-online-tools/eve-resfile-proxy/service/buildnumber"
	"github.com/eve-online-tools/eve-resfile-proxy/service/clientbuild"
	svchttp "github.com/eve-online-tools/eve-resfile-proxy/service/http"
	"github.com/eve-online-tools/eve-resfile-proxy/service/manifest"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/alias"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/transform"
)

var (
	ErrFailedToLoadServerInformation = errors.New("failed to load server information")
	ErrProtectedBuild                = errors.New("client build is protected")
	ErrNoMatchingPlatforms           = errors.New("no configured platforms are available for this client build")
)

type Service struct {
	config *Config
	logger *slog.Logger
	client *stdhttp.Client

	manifest *manifest.Manifest
	fsys     fs.FS

	cache cache.Cache

	build       *buildnumber.BuildNumber
	clientBuild *clientbuild.ClientBuild

	httpServer  *svchttp.Server
	watchCancel context.CancelFunc
}

func New(cfg *Config, logger *slog.Logger) (*Service, error) {
	var cache cache.Cache
	if cfg.CacheDir != "" {
		cache = diskcache.New(cfg.CacheDir)
	}

	return &Service{
		config: cfg,
		logger: logger,
		client: defaultHTTPClient,
		cache:  cache,
	}, nil
}

func (s *Service) resolveBuild(ctx context.Context) (string, *clientbuild.ClientBuild, error) {
	clientBuild, err := clientbuild.LoadClientBuild(ctx, s.client, s.config.ServerName)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrFailedToLoadServerInformation, err)
	}
	if clientBuild.Protected {
		return "", nil, fmt.Errorf("%w: %s", ErrProtectedBuild, s.config.ServerName)
	}

	buildNumber := s.config.BuildNumber
	if buildNumber == "" {
		buildNumber = clientBuild.BuildNumber
	}
	s.logger.Debug("loaded server information",
		"serverName", s.config.ServerName,
		"protected", clientBuild.Protected,
		"localBuildNumber", buildNumber,
		"remoteBuildNumber", clientBuild.BuildNumber,
	)

	return buildNumber, clientBuild, nil
}

func (s *Service) Start(ctx context.Context) error {
	var err error

	buildNumber, clientBuild, err := s.resolveBuild(ctx)
	if err != nil {
		return err
	}
	s.build = &buildnumber.BuildNumber{}
	s.build.Set(buildNumber)
	s.clientBuild = clientBuild

	sets := [][]platform.Platform{clientBuild.Platforms}
	if len(s.config.Platforms) > 0 {
		sets = append(sets, s.config.Platforms)
	}
	platforms := platform.Intersect(sets...)
	if len(platforms) == 0 {
		return ErrNoMatchingPlatforms
	}

	s.manifest, err = manifest.New(s.cache, s.client, platforms, s.config.FullTree, s.logger)
	if err != nil {
		return err
	}

	if err := s.manifest.Load(ctx, buildNumber); err != nil {
		return err
	}

	// alias.New does not wrap if no aliases are configured
	fsys, err := alias.New(s.manifest, s.config.Aliases)
	if err != nil {
		return err
	}

	// transform.New does not wrap if no transforms are configured
	fsys, err = transform.New(fsys, s.cache, s.config.Transforms, s.config.TransformLimits, s.config.ConfigDir)
	if err != nil {
		return err
	}

	s.fsys = fsys

	var diskCache *diskcache.Cache
	if dc, ok := s.cache.(*diskcache.Cache); ok {
		diskCache = dc
	}

	s.httpServer = svchttp.NewServer(&s.config.ServerConfig, s.fsys, s.build, diskCache, s.logger)
	s.httpServer.Start()

	if s.config.BuildNumber == "" {
		watchCtx, cancel := context.WithCancel(context.Background())
		s.watchCancel = cancel
		go s.watchBuild(watchCtx)
	}

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if s.watchCancel != nil {
		s.watchCancel()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
