package load

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	diskcache "github.com/eve-online-tools/eve-resfile-proxy/cache/disk"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/request"
	"github.com/eve-online-tools/eve-resfile-proxy/service/http/urlpath"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
)

func Middleware(fsys fs.FS, disk *diskcache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fsPath, err := urlpath.ToFS(r.URL.Path)
			if err != nil {
				http.Error(w, "Resfile not found.", http.StatusNotFound)
				return
			}
			if fsPath == "" {
				http.Error(w, "Resfile not found.", http.StatusNotFound)
				return
			}

			info, err := fs.Stat(fsys, fsPath)
			if errors.Is(err, fs.ErrNotExist) {
				http.Error(w, "Resfile not found.", http.StatusNotFound)
				return
			}
			if err != nil {
				writeStatError(w, err)
				return
			}
			if info.IsDir() {
				next.ServeHTTP(w, r)
				return
			}

			res := &request.Resource{
				FSPath: fsPath,
				FS:     fsys,
			}
			if mi, ok := info.(vfs.ManifestFileInfo); ok {
				if sum := mi.MD5(); sum != ([md5.Size]byte{}) {
					res.Checksum = sum
					res.HasChecksum = true
				}
			}
			res.Size = info.Size()

			if disk != nil {
				if cdnPath := cdnPathFromInfo(info); cdnPath != "" {
					if hit, mod, err := cacheHit(disk, cdnPath); err != nil {
						http.Error(w, "Failed to read cache.", http.StatusInternalServerError)
						return
					} else if hit {
						res.CacheStatus = request.CacheStatusHit
						res.LastModified = mod
					}
				}
			}
			if res.CacheStatus == "" {
				res.CacheStatus = request.CacheStatusMiss
			}

			ctx := request.WithResource(r.Context(), res)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func cdnPathFromInfo(info fs.FileInfo) string {
	if mi, ok := info.(vfs.ManifestFileInfo); ok {
		if entry, ok := mi.Sys().(vfs.Entry); ok {
			return entry.CDNPath
		}
	}
	return ""
}

func cacheHit(disk *diskcache.Cache, cdnPath string) (bool, time.Time, error) {
	path := disk.Path(cdnPath)
	if path == "" {
		return false, time.Time{}, nil
	}
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, err
	}
	return true, st.ModTime(), nil
}

func writeStatError(w http.ResponseWriter, err error) {
	if isFetchError(err) {
		http.Error(w, fmt.Sprintf("Failed to fetch asset: %v", unwrapErr(err)), http.StatusBadGateway)
		return
	}
	http.Error(w, "Resfile not found.", http.StatusNotFound)
}

// WriteReadError maps vfs read failures to HTTP error responses.
func WriteReadError(w http.ResponseWriter, err error) {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		if errors.Is(pathErr.Err, vfs.ErrChecksumMismatch) {
			http.Error(w, "Checksum mismatch.", http.StatusInternalServerError)
			return
		}
		if errors.Is(pathErr.Err, vfs.ErrFetchNotConfigured) {
			http.Error(w, "Failed to fetch asset: fetcher not configured", http.StatusBadGateway)
			return
		}
	}
	if isFetchError(err) {
		http.Error(w, fmt.Sprintf("Failed to fetch asset: %v", unwrapErr(err)), http.StatusBadGateway)
		return
	}
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, "Resfile not found.", http.StatusNotFound)
		return
	}
	http.Error(w, fmt.Sprintf("Failed to fetch asset: %v", err), http.StatusBadGateway)
}

func isFetchError(err error) bool {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	msg := err.Error()
	return strings.Contains(msg, "fetch") || strings.Contains(msg, "http")
}

func unwrapErr(err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}
