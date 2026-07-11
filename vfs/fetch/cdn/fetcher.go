package cdn

import (
	"context"
	"net/http"

	"github.com/eve-online-tools/eve-resfile-proxy/common/domain"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
)

type fetcher struct {
	domain domain.Domain
	client *http.Client
}

var _ vfs.Fetcher = (*fetcher)(nil)

func newFetcher(domain domain.Domain, client *http.Client) *fetcher {
	return &fetcher{domain: domain, client: client}
}

func (f *fetcher) FetchEntry(ctx context.Context, entry vfs.Entry) ([]byte, error) {
	return f.FetchPath(ctx, entry.CDNPath)
}

func (f *fetcher) FetchPath(ctx context.Context, path string) ([]byte, error) {
	return getBytes(ctx, f.client, f.domain.URL(path))
}
