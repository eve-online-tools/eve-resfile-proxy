package vfs

import (
	"crypto/md5"
	"io/fs"
	"time"
)

// ManifestFileInfo is implemented by fs.FileInfo values for files backed by
// manifest entries (from Stat or ReadDir on a manifest fs.FS).
type ManifestFileInfo interface {
	fs.FileInfo
	// MD5 returns the manifest checksum. Compare with crypto/md5.Sum(data).
	MD5() [md5.Size]byte
}

// CompressedSizeInfo is implemented by fs.FileInfo values that expose CDN
// compressed size for directory listings. VFS layers may implement this to
// surface listing metadata without exposing manifest entries.
type CompressedSizeInfo interface {
	CompressedSize() int64
}

// manifestFileInfo implements fs.FileInfo for manifest entries.
type manifestFileInfo struct {
	name  string
	entry Entry
}

var _ ManifestFileInfo = manifestFileInfo{}
var _ CompressedSizeInfo = manifestFileInfo{}

func (i manifestFileInfo) Name() string          { return i.name }
func (i manifestFileInfo) Size() int64           { return i.entry.Size }
func (i manifestFileInfo) Mode() fs.FileMode     { return 0o444 }
func (i manifestFileInfo) ModTime() time.Time    { return time.Time{} }
func (i manifestFileInfo) IsDir() bool           { return false }
func (i manifestFileInfo) Sys() any              { return i.entry }
func (i manifestFileInfo) MD5() [md5.Size]byte   { return i.entry.MD5.Sum() }
func (i manifestFileInfo) CompressedSize() int64 { return i.entry.CompressedSize }

// manifestDirInfo implements fs.FileInfo for synthetic directories.
type manifestDirInfo struct {
	name string
}

var _ fs.FileInfo = &manifestDirInfo{}

func (i manifestDirInfo) Name() string       { return i.name }
func (i manifestDirInfo) Size() int64        { return 0 }
func (i manifestDirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (i manifestDirInfo) ModTime() time.Time { return time.Time{} }
func (i manifestDirInfo) IsDir() bool        { return true }
func (i manifestDirInfo) Sys() any           { return nil }

// manifestDirEntry implements fs.DirEntry for ReadDir results.
type manifestDirEntry struct {
	info fs.FileInfo
}

var _ fs.DirEntry = &manifestDirEntry{}

func (e manifestDirEntry) Name() string               { return e.info.Name() }
func (e manifestDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e manifestDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e manifestDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }
