package request

import (
	"context"
	"crypto/md5"
	"io/fs"
	"sync"
	"sync/atomic"
	"time"
)

type CacheStatus string

const (
	CacheStatusHit  CacheStatus = "HIT"
	CacheStatusMiss CacheStatus = "MISS"
)

// Resource describes a resolved file without eagerly loading its bytes.
type Resource struct {
	FSPath       string
	FS           fs.FS
	Checksum     [md5.Size]byte
	HasChecksum  bool
	Size         int64
	CacheStatus  CacheStatus
	LastModified time.Time
	ETag         string

	once     sync.Once
	data     []byte
	err      error
	onceDone atomic.Bool

	preload []byte // when set, Data returns preload without touching FS
}

// NewPreloadedResource returns a resource with fixed bytes. Intended for tests.
func NewPreloadedResource(fsPath string, data []byte) *Resource {
	return &Resource{
		FSPath:  fsPath,
		Size:    int64(len(data)),
		preload: data,
	}
}

// Data returns file bytes, loading from FS on first call.
func (r *Resource) Data() ([]byte, error) {
	if r.preload != nil {
		return r.preload, nil
	}
	r.once.Do(func() {
		defer r.onceDone.Store(true)
		if r.FS == nil {
			return
		}
		r.data, r.err = fs.ReadFile(r.FS, r.FSPath)
	})
	return r.data, r.err
}

// ContentLength returns the response body length. When loadBody is true, bytes are
// loaded if not already cached and returned for writing.
func (r *Resource) ContentLength(loadBody bool) (int64, []byte, error) {
	if loadBody {
		data, err := r.Data()
		return int64(len(data)), data, err
	}
	if r.preload != nil {
		return int64(len(r.preload)), nil, nil
	}
	if r.onceDone.Load() && r.err == nil {
		return int64(len(r.data)), nil, nil
	}
	return r.Size, nil, nil
}

type cxResource struct{}

func WithResource(ctx context.Context, res *Resource) context.Context {
	return context.WithValue(ctx, cxResource{}, res)
}

func ResourceFromContext(ctx context.Context) (*Resource, bool) {
	res, ok := ctx.Value(cxResource{}).(*Resource)
	return res, ok
}
