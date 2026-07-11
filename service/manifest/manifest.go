package manifest

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/eve-online-tools/eve-resfile-proxy/cache"
	"github.com/eve-online-tools/eve-resfile-proxy/common/domain"
	"github.com/eve-online-tools/eve-resfile-proxy/common/platform"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	cacheFetcher "github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/cache"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/cdn"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/mux"
)

// Manifest loads EVE resfile indexes for a client build and exposes the
// combined tree as an fs.FS.
type Manifest struct {
	activeBuildNumber string

	platforms []platform.Platform
	fullTree  bool

	logger *slog.Logger

	mux     *mux.Mux
	fsys    fs.FS
	cache   cache.Cache
	fetcher map[domain.Domain]vfs.Fetcher
}

var (
	_ fs.FS         = (*Manifest)(nil)
	_ fs.StatFS     = (*Manifest)(nil)
	_ fs.ReadDirFS  = (*Manifest)(nil)
	_ fs.GlobFS     = (*Manifest)(nil)
	_ fs.ReadFileFS = (*Manifest)(nil)
)

func New(c cache.Cache, client *http.Client, platforms []platform.Platform, fullTree bool, logger *slog.Logger) (*Manifest, error) {
	muxFS := &mux.Mux{}
	m := &Manifest{
		activeBuildNumber: "",

		platforms: platforms,
		fullTree:  fullTree,

		logger: logger,

		mux:     muxFS,
		fsys:    muxFS,
		cache:   c,
		fetcher: make(map[domain.Domain]vfs.Fetcher),
	}

	for _, domain := range domain.All {
		fetcher := cdn.New(client, cdn.WithDomain(domain))

		m.fetcher[domain] = fetcher
		if c != nil {
			m.fetcher[domain] = cacheFetcher.New(fetcher, c, cacheFetcher.WithLogger(logger))
		}
	}

	return m, nil
}

func NewFake(logger *slog.Logger, fetcher vfs.Fetcher, platforms []platform.Platform, fullTree bool) *Manifest {
	muxFS := &mux.Mux{}
	return &Manifest{
		platforms: platforms,
		fullTree:  fullTree,
		logger:    logger,
		mux:       muxFS,
		fsys:      muxFS,
		fetcher: map[domain.Domain]vfs.Fetcher{
			domain.Binaries:  fetcher,
			domain.Resources: fetcher,
		},
	}
}

func (m *Manifest) Update(ctx context.Context, buildNumber string) error {
	if buildNumber == m.activeBuildNumber {
		return nil
	}
	return m.Load(ctx, buildNumber)
}

func (m *Manifest) Open(name string) (fs.File, error) {
	return m.fsys.Open(name)
}

func (m *Manifest) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(m.fsys, name)
}

func (m *Manifest) ReadDir(name string) ([]fs.DirEntry, error) {
	if rdFS, ok := m.fsys.(fs.ReadDirFS); ok {
		return rdFS.ReadDir(name)
	}
	return nil, fs.ErrInvalid
}

func (m *Manifest) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(m.fsys, name)
}

func (m *Manifest) Glob(pattern string) ([]string, error) {
	if gFS, ok := m.fsys.(fs.GlobFS); ok {
		return gFS.Glob(pattern)
	}
	return nil, fs.ErrInvalid
}
