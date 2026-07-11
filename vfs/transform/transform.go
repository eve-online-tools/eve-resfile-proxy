package transform

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/cache"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	vfspath "github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

// CDNPathProvider exposes the CDN storage path for a manifest-backed file.
type CDNPathProvider interface {
	GetCDNPath() string
}

type pathCache interface {
	Path(key string) string
}

// FS applies transform rules before returning file contents.
type FS struct {
	fsys  fs.FS
	rules []compiledTransform
	cache cache.Cache
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.GlobFS     = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
)

// New wraps fsys with transforms. Returns fsys unchanged when transforms is empty.
func New(fsys fs.FS, c cache.Cache, transforms []Transform, limits Limits, baseDir string) (fs.FS, error) {
	rules, err := compileTransforms(transforms, limits, baseDir)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return fsys, nil
	}
	return &FS{fsys: fsys, rules: rules, cache: c}, nil
}

func (f *FS) Open(name string) (fs.File, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}

	resolved, innerInfo, err := f.resolve(context.Background(), cleaned, true)
	if err != nil {
		return nil, err
	}
	if !resolved.matched {
		return f.fsys.Open(cleaned)
	}

	return &bytesFile{
		name: path.Base(cleaned),
		info: transformFileInfo{inner: innerInfo, size: int64(len(resolved.data)), digest: resolved.digest},
		data: resolved.data,
	}, nil
}

func (f *FS) Stat(name string) (fs.FileInfo, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}

	if _, ok := matchingRule(f.rules, cleaned); !ok {
		return fs.Stat(f.fsys, cleaned)
	}

	resolved, innerInfo, err := f.resolve(context.Background(), cleaned, false)
	if err != nil {
		return nil, err
	}
	if !resolved.matched {
		return fs.Stat(f.fsys, cleaned)
	}

	size := resolved.size
	if size == 0 && len(resolved.data) > 0 {
		size = int64(len(resolved.data))
	}

	return transformFileInfo{
		inner:  innerInfo,
		size:   size,
		digest: resolved.digest,
	}, nil
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	cleaned, err := vfspath.CleanFile(name)
	if err != nil {
		return nil, err
	}

	resolved, _, err := f.resolve(context.Background(), cleaned, true)
	if err != nil {
		return nil, err
	}
	if !resolved.matched {
		return fs.ReadFile(f.fsys, cleaned)
	}
	return resolved.data, nil
}

func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if rdFS, ok := f.fsys.(fs.ReadDirFS); ok {
		return rdFS.ReadDir(name)
	}
	return nil, fs.ErrInvalid
}

func (f *FS) Glob(pattern string) ([]string, error) {
	if gFS, ok := f.fsys.(fs.GlobFS); ok {
		return gFS.Glob(pattern)
	}
	return nil, fs.ErrInvalid
}

type resolved struct {
	matched bool
	data    []byte
	size    int64
	digest  [md5.Size]byte
}

func (f *FS) resolve(ctx context.Context, name string, needData bool) (resolved, fs.FileInfo, error) {
	ruleName, ok := matchingRule(f.rules, name)
	if !ok {
		return resolved{}, nil, nil
	}

	innerInfo, err := fs.Stat(f.fsys, name)
	if err != nil {
		return resolved{}, nil, err
	}
	if innerInfo.IsDir() {
		return resolved{}, innerInfo, nil
	}

	cdnPath, ok := cdnPathFromInfo(innerInfo)
	if !ok {
		return resolved{}, innerInfo, nil
	}

	dataKey := cacheKey(ruleName, cdnPath)
	digestKey := dataKey + ".md5sum"

	if f.cache != nil {
		if hit, err := f.loadFromCache(ctx, dataKey, digestKey, needData); err != nil {
			return resolved{}, innerInfo, err
		} else if hit != nil {
			hit.matched = true
			return *hit, innerInfo, nil
		}
	}

	raw, err := fs.ReadFile(f.fsys, name)
	if err != nil {
		return resolved{}, innerInfo, err
	}

	out, err := runTransform(f.rules, ctx, name, assetInput{
		ResPath: name,
		CDNPath: cdnPath,
		Data:    raw,
	})
	if err != nil {
		return resolved{}, innerInfo, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	digest := md5.Sum(out)
	if f.cache != nil {
		if err := f.storeInCache(ctx, dataKey, digestKey, out, digest); err != nil {
			return resolved{}, innerInfo, err
		}
	}

	return resolved{
		matched: true,
		data:    out,
		size:    int64(len(out)),
		digest:  digest,
	}, innerInfo, nil
}

func (f *FS) loadFromCache(ctx context.Context, dataKey, digestKey string, needData bool) (*resolved, error) {
	if !needData {
		if pc, ok := f.cache.(pathCache); ok {
			if hit, err := f.loadMetadataFromDisk(pc.Path(dataKey), dataKey, digestKey, ctx); err != nil {
				return nil, err
			} else if hit != nil {
				return hit, nil
			}
		}
	}

	data, dataOK, err := f.cache.Get(ctx, dataKey)
	if err != nil {
		return nil, err
	}
	if !dataOK {
		return nil, nil
	}

	digest, err := f.loadDigest(ctx, digestKey, data)
	if err != nil {
		return nil, err
	}

	if !needData {
		return &resolved{
			matched: true,
			size:    int64(len(data)),
			digest:  digest,
		}, nil
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	return &resolved{
		matched: true,
		data:    cp,
		size:    int64(len(cp)),
		digest:  digest,
	}, nil
}

func (f *FS) loadMetadataFromDisk(dataPath, dataKey, digestKey string, ctx context.Context) (*resolved, error) {
	st, err := os.Stat(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if st.IsDir() {
		return nil, nil
	}

	digestBytes, digestOK, err := f.cache.Get(ctx, digestKey)
	if err != nil {
		return nil, err
	}
	if !digestOK {
		data, dataOK, err := f.cache.Get(ctx, dataKey)
		if err != nil || !dataOK {
			return nil, err
		}
		digest, err := f.backfillDigest(ctx, digestKey, data)
		if err != nil {
			return nil, err
		}
		return &resolved{matched: true, size: st.Size(), digest: digest}, nil
	}

	digest, err := parseDigest(digestBytes)
	if err != nil {
		return nil, err
	}
	return &resolved{matched: true, size: st.Size(), digest: digest}, nil
}

func (f *FS) loadDigest(ctx context.Context, digestKey string, data []byte) ([md5.Size]byte, error) {
	digestBytes, digestOK, err := f.cache.Get(ctx, digestKey)
	if err != nil {
		return [md5.Size]byte{}, err
	}
	if digestOK {
		return parseDigest(digestBytes)
	}
	return f.backfillDigest(ctx, digestKey, data)
}

func (f *FS) backfillDigest(ctx context.Context, digestKey string, data []byte) ([md5.Size]byte, error) {
	digest := md5.Sum(data)
	if err := f.cache.Store(ctx, digestKey, []byte(hex.EncodeToString(digest[:]))); err != nil {
		return [md5.Size]byte{}, err
	}
	return digest, nil
}

func (f *FS) storeInCache(ctx context.Context, dataKey, digestKey string, data []byte, digest [md5.Size]byte) error {
	if err := f.cache.Store(ctx, dataKey, data); err != nil {
		return err
	}
	return f.cache.Store(ctx, digestKey, []byte(hex.EncodeToString(digest[:])))
}

func cacheKey(ruleName, cdnPath string) string {
	return path.Join("_transformed", ruleName, cdnPath)
}

func cdnPathFromInfo(info fs.FileInfo) (string, bool) {
	if info == nil {
		return "", false
	}
	if p, ok := info.(CDNPathProvider); ok {
		if cdn := p.GetCDNPath(); cdn != "" {
			return cdn, true
		}
	}
	if sys := info.Sys(); sys != nil {
		if p, ok := sys.(CDNPathProvider); ok {
			if cdn := p.GetCDNPath(); cdn != "" {
				return cdn, true
			}
		}
		if entry, ok := sys.(vfs.Entry); ok && entry.CDNPath != "" {
			return entry.CDNPath, true
		}
	}
	return "", false
}

func parseDigest(data []byte) ([md5.Size]byte, error) {
	hexDigest := strings.TrimSpace(string(data))
	if len(hexDigest) != md5.Size*2 {
		return [md5.Size]byte{}, fmt.Errorf("invalid md5 sidecar length %d", len(hexDigest))
	}
	var out [md5.Size]byte
	if _, err := hex.Decode(out[:], []byte(hexDigest)); err != nil {
		return [md5.Size]byte{}, fmt.Errorf("decode md5 sidecar: %w", err)
	}
	return out, nil
}

type transformFileInfo struct {
	inner  fs.FileInfo
	size   int64
	digest [md5.Size]byte
}

var _ vfs.ManifestFileInfo = transformFileInfo{}

func (i transformFileInfo) Name() string       { return i.inner.Name() }
func (i transformFileInfo) Size() int64        { return i.size }
func (i transformFileInfo) Mode() fs.FileMode  { return i.inner.Mode() }
func (i transformFileInfo) ModTime() time.Time { return i.inner.ModTime() }
func (i transformFileInfo) IsDir() bool        { return i.inner.IsDir() }
func (i transformFileInfo) Sys() any           { return i.inner.Sys() }
func (i transformFileInfo) MD5() [md5.Size]byte {
	return i.digest
}

type bytesFile struct {
	name string
	info transformFileInfo
	data []byte
	off  int64
}

var _ fs.File = (*bytesFile)(nil)

func (f *bytesFile) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f *bytesFile) Read(p []byte) (int, error) {
	if f.off >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.off:])
	f.off += int64(n)
	return n, nil
}

func (f *bytesFile) Close() error {
	return nil
}
