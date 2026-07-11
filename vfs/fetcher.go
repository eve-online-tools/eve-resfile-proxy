package vfs

import (
	"context"
	"errors"
)

// Fetcher retrieves raw bytes from the CDN.
type Fetcher interface {
	FetchEntry(ctx context.Context, entry Entry) ([]byte, error)
	FetchPath(ctx context.Context, path string) ([]byte, error)
}

// ErrFetchNotConfigured is returned by Open when the path exists in the manifest
// but no fetcher was configured.
var ErrFetchNotConfigured = errors.New("fetcher not configured")

// ErrChecksumMismatch is returned when fetched bytes do not match the manifest MD5.
var ErrChecksumMismatch = errors.New("checksum mismatch")
