package vfs

import (
	"io"
	"io/fs"
)

type manifestFile struct {
	name  string
	entry Entry
	data  []byte
	off   int64
}

var _ fs.File = (*manifestFile)(nil)

func (f *manifestFile) Stat() (fs.FileInfo, error) {
	return manifestFileInfo{name: f.name, entry: f.entry}, nil
}

func (f *manifestFile) Read(p []byte) (int, error) {
	if f.off >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.off:])
	f.off += int64(n)
	return n, nil
}

func (f *manifestFile) Close() error {
	return nil
}
