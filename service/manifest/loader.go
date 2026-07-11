package manifest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/eve-online-tools/eve-resfile-proxy/common/domain"
	"github.com/eve-online-tools/eve-resfile-proxy/common/platform"
	"github.com/eve-online-tools/eve-resfile-proxy/common/resource"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/mux"
)

var (
	ErrFailedToLoadBinaryIndex      = errors.New("failed to load binary index")
	ErrFailedToLoadResourceManifest = errors.New("failed to load resource manifest")
)

func (m *Manifest) Load(ctx context.Context, buildNumber string) error {
	muxFS := &mux.Mux{}

	fetcher := m.fetcher[domain.Binaries]

	manifests := make(map[platform.Platform]fs.FS)
	m.logger.Info("loading binary indices",
		"buildNumber", buildNumber,
		"platforms", m.platforms,
	)

	for _, platform := range m.platforms {
		path := platform.ManifestPath(buildNumber)
		data, err := fetcher.FetchPath(ctx, path)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToLoadBinaryIndex, err)
		}

		manifest, err := vfs.New(data, fetcher, vfs.WithPrefix(vfs.PrefixApp))
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToLoadBinaryIndex, err)
		}
		manifests[platform] = manifest
		entries, _ := fs.Glob(manifest, "**/*")
		m.logger.Debug("loaded binary index",
			"platform", platform,
			"entries", len(entries),
		)

		if m.fullTree {
			_ = muxFS.Mount(fmt.Sprintf("/app/%s", platform), manifest) //nolint:errcheck // mount paths are fixed
		}
	}

	prefix := "/"
	if m.fullTree {
		prefix = "/res"
	}

	fetcher = m.fetcher[domain.Resources]

	for _, platform := range m.platforms {
		manifest := manifests[platform]
		m.logger.Debug("loading full manifest",
			"platform", platform,
			"path", resource.Full.Path(platform),
		)
		fullManifest, err := m.loadManifestIfAvailable(manifest, resource.Full.Path(platform), fetcher)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToLoadResourceManifest, err)
		}
		if fullManifest != nil {
			_ = muxFS.Mount(prefix, fullManifest) //nolint:errcheck // mount paths are fixed
			entries, _ := fs.Glob(fullManifest, "**/*")

			m.logger.Debug("Loaded full manifest",
				"platform", platform,
				"entries", len(entries),
			)
		}

		osManifest, err := m.loadManifestIfAvailable(manifest, resource.OSSpecific.Path(platform), fetcher)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToLoadResourceManifest, err)
		}
		if osManifest != nil {
			_ = muxFS.Mount(prefix, osManifest) //nolint:errcheck // mount paths are fixed
			entries, _ := fs.Glob(osManifest, "**/*")

			m.logger.Debug("Loaded OS-specific manifest",
				"platform", platform,
				"entries", len(entries),
			)
		}
	}

	entries, _ := fs.Glob(muxFS, "**/*")
	m.logger.Debug("Loaded manifests",
		"entries", len(entries),
	)

	m.mux = muxFS
	m.fsys = muxFS
	m.activeBuildNumber = buildNumber

	return nil
}

func (m *Manifest) loadManifestIfAvailable(manifest fs.FS, fullPath string, fetcher vfs.Fetcher) (fs.FS, error) {
	if _, err := fs.Stat(manifest, fullPath); err == nil {
		handle, err := manifest.Open(fullPath)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToLoadResourceManifest, err)
		}
		defer handle.Close() //nolint:errcheck // in-memory manifest file
		data, err := io.ReadAll(handle)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToLoadResourceManifest, err)
		}

		resourceManifest, err := vfs.New(data, fetcher, vfs.WithPrefix(vfs.PrefixRes))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToLoadResourceManifest, err)
		}
		return resourceManifest, nil
	} else {
		m.logger.Debug("manifest not found",
			"path", fullPath,
			"err", err,
		)
	}
	return nil, nil
}
