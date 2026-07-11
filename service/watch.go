package service

import (
	"context"
	"time"
)

const buildCheckInterval = 5 * time.Minute

func (s *Service) watchBuild(ctx context.Context) {
	ticker := time.NewTicker(buildCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkForBuildUpdate(ctx)
		}
	}
}

func (s *Service) checkForBuildUpdate(ctx context.Context) {
	buildNumber, clientBuild, err := s.resolveBuild(ctx)
	if err != nil {
		s.logger.Warn("build check failed", "err", err)
		return
	}

	current := s.build.Get()
	if buildNumber == current {
		return
	}

	s.logger.Info("client build changed, reloading manifest",
		"from", current,
		"to", buildNumber,
	)

	if err := s.manifest.Update(ctx, buildNumber); err != nil {
		s.logger.Error("manifest reload failed", "buildNumber", buildNumber, "err", err)
		return
	}

	s.build.Set(buildNumber)
	s.clientBuild = clientBuild
}
