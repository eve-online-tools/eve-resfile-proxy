package index_test

import (
	"io/fs"
	"testing"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/index"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
)

type listingFileInfo struct {
	name       string
	size       int64
	compressed int64
}

func (i listingFileInfo) Name() string          { return i.name }
func (i listingFileInfo) Size() int64           { return i.size }
func (i listingFileInfo) Mode() fs.FileMode     { return 0 }
func (i listingFileInfo) ModTime() time.Time    { return time.Time{} }
func (i listingFileInfo) IsDir() bool           { return false }
func (i listingFileInfo) Sys() any              { return nil }
func (i listingFileInfo) CompressedSize() int64 { return i.compressed }

var _ vfs.CompressedSizeInfo = listingFileInfo{}

type stubDirEntry struct {
	info fs.FileInfo
}

func (e stubDirEntry) Name() string               { return e.info.Name() }
func (e stubDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e stubDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e stubDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

func TestFormatListingFileSizes(t *testing.T) {
	t.Parallel()

	entry := stubDirEntry{info: listingFileInfo{
		name:       "icon.png",
		size:       4096,
		compressed: 2048,
	}}

	info, err := entry.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	size := index.FormatFileSize(info.Size())
	compression := ""
	if li, ok := info.(vfs.CompressedSizeInfo); ok {
		compression = index.FormatCompressionPercent(info.Size(), li.CompressedSize())
	}

	if size != "4K" {
		t.Fatalf("size = %q, want 4K", size)
	}
	if compression != "50%" {
		t.Fatalf("compression = %q, want 50%%", compression)
	}
}

func TestFormatListingFileSizesWithoutCompressedSize(t *testing.T) {
	t.Parallel()

	entry := stubDirEntry{info: plainFileInfo{name: "raw.bin", size: 100}}

	info, err := entry.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	compression := "-"
	if li, ok := info.(vfs.CompressedSizeInfo); ok {
		compression = index.FormatCompressionPercent(info.Size(), li.CompressedSize())
	}
	if compression != "-" {
		t.Fatalf("compression = %q, want -", compression)
	}
}

type plainFileInfo struct {
	name string
	size int64
	dir  bool
}

func (i plainFileInfo) Name() string       { return i.name }
func (i plainFileInfo) Size() int64        { return i.size }
func (i plainFileInfo) Mode() fs.FileMode  { return 0 }
func (i plainFileInfo) ModTime() time.Time { return time.Time{} }
func (i plainFileInfo) IsDir() bool        { return i.dir }
func (i plainFileInfo) Sys() any           { return nil }
