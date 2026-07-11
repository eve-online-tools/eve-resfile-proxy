package mapfetch

import (
	"context"
	"fmt"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
)

type fetcher struct {
	files map[string][]byte
}

func New(files map[string][]byte) vfs.Fetcher {
	return &fetcher{files: files}
}

func (f *fetcher) FetchEntry(ctx context.Context, entry vfs.Entry) ([]byte, error) {
	return f.FetchPath(ctx, entry.CDNPath)
}

func (f *fetcher) FetchPath(_ context.Context, path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, fmt.Errorf("fetch: %q not found", path)
	}
	return data, nil
}
