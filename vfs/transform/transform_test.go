package transform_test

import (
	"context"
	"crypto/md5"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/cache/memory"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/transform"
)

//go:generate go run github.com/wasilibs/go-wabt/cmd/wat2wasm@v0.0.0-20240502051220-face6b18f58d -o testdata/copy.wasm testdata/copy.wat

func TestPassthroughWithoutTransforms(t *testing.T) {
	inner := fstest.MapFS{
		"foo.txt": {Data: []byte("raw")},
	}
	fsys, err := transform.New(inner, nil, nil, transform.Limits{}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "foo.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "raw" {
		t.Fatalf("got %q", data)
	}
}

func TestMatcherPathPrefix(t *testing.T) {
	inner := manifestLikeFS{files: map[string]manifestFile{
		"shader/foo.fx":  {data: []byte("in"), cdnPath: "a/b", size: 2},
		"icons/icon.png": {data: []byte("raw"), cdnPath: "c/d", size: 3},
	}}
	fsys := mustNewFS(t, inner, nil, []transform.Transform{{
		Name: "prefix",
		Match: transform.Match{
			PathPrefix: "shader/",
			Extensions: []string{".fx"},
		},
		Command: &transform.Command{Args: []string{"cat"}},
	}}, "")

	data, err := fs.ReadFile(fsys, "shader/foo.fx")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "in" {
		t.Fatalf("got %q", data)
	}

	raw, err := fs.ReadFile(fsys, "icons/icon.png")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(raw) != "raw" {
		t.Fatalf("got %q", raw)
	}
}

func TestMatcherPathGlob(t *testing.T) {
	inner := manifestLikeFS{files: map[string]manifestFile{
		"shader/deep/file.fx": {data: []byte("ok"), cdnPath: "a/b", size: 2},
	}}
	fsys := mustNewFS(t, inner, nil, []transform.Transform{{
		Name:    "glob",
		Match:   transform.Match{PathGlob: "shader/**/*.fx"},
		Command: &transform.Command{Args: []string{"cat"}},
	}}, "")

	data, err := fs.ReadFile(fsys, "shader/deep/file.fx")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("got %q", data)
	}
}

func TestFirstMatchWins(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "upper.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntr 'a-z' 'A-Z'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	inner := manifestLikeFS{files: map[string]manifestFile{
		"foo.txt": {data: []byte("abc"), cdnPath: "a/b", size: 3},
	}}
	fsys := mustNewFS(t, inner, nil, []transform.Transform{
		{
			Name:    "first",
			Match:   transform.Match{Extensions: []string{".txt"}},
			Command: &transform.Command{Args: []string{script}},
		},
		{
			Name:    "second",
			Match:   transform.Match{Extensions: []string{".txt"}},
			Command: &transform.Command{Args: []string{"false"}},
		},
	}, dir)

	data, err := fs.ReadFile(fsys, "foo.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "ABC" {
		t.Fatalf("got %q", data)
	}
}

func TestConfigValidation(t *testing.T) {
	cases := []struct {
		name       string
		transforms []transform.Transform
	}{
		{
			name: "missing backend",
			transforms: []transform.Transform{{
				Name:  "bad",
				Match: transform.Match{Extensions: []string{".fx"}},
			}},
		},
		{
			name: "both backends",
			transforms: []transform.Transform{{
				Name:    "bad",
				Match:   transform.Match{Extensions: []string{".fx"}},
				Command: &transform.Command{Args: []string{"true"}},
				Wasm:    &transform.Wasm{Module: "./x.wasm"},
			}},
		},
		{
			name: "missing match",
			transforms: []transform.Transform{{
				Name:    "bad",
				Command: &transform.Command{Args: []string{"true"}},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := transform.New(fstest.MapFS{}, nil, tc.transforms, transform.Limits{}, "")
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCommandTransform(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "append.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat; echo -n suffix\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	inner := manifestLikeFS{files: map[string]manifestFile{
		"data/file.dat": {data: []byte("body"), cdnPath: "ab/cd", size: 4},
	}}
	fsys := mustNewFS(t, inner, nil, []transform.Transform{{
		Name:    "append",
		Match:   transform.Match{Extensions: []string{".dat"}},
		Command: &transform.Command{Args: []string{script}},
	}}, dir)

	data, err := fs.ReadFile(fsys, "data/file.dat")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "bodysuffix" {
		t.Fatalf("got %q", data)
	}
}

func TestCommandTransformFailure(t *testing.T) {
	inner := manifestLikeFS{files: map[string]manifestFile{
		"data/file.dat": {data: []byte("body"), cdnPath: "ab/cd", size: 4},
	}}
	fsys := mustNewFS(t, inner, nil, []transform.Transform{{
		Name:    "fail",
		Match:   transform.Match{Extensions: []string{".dat"}},
		Command: &transform.Command{Args: []string{"false"}},
	}}, "")

	_, err := fs.ReadFile(fsys, "data/file.dat")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWasmTransformCopy(t *testing.T) {
	wasmPath := filepath.Join("testdata", "copy.wasm")
	if _, err := os.Stat(wasmPath); err != nil {
		t.Fatalf("missing %s: run go generate ./vfs/transform", wasmPath)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "copy.wasm"), mustReadFile(t, wasmPath), 0o644); err != nil {
		t.Fatal(err)
	}

	inner := manifestLikeFS{files: map[string]manifestFile{
		"foo.txt": {data: []byte("hello"), cdnPath: "aa/bb", size: 5},
	}}
	fsys, err := transform.New(inner, nil, []transform.Transform{{
		Name:  "copy",
		Match: transform.Match{Extensions: []string{".txt"}},
		Wasm:  &transform.Wasm{Module: "./copy.wasm"},
	}}, transform.Limits{MaxOutputBytes: 4096}, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "foo.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("got %q", data)
	}
}

func TestTransformCachesAndHits(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "upper.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntr 'a-z' 'A-Z'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	inner := manifestLikeFS{files: map[string]manifestFile{
		"foo.txt": {data: []byte("abc"), cdnPath: "aa/bb", size: 3},
	}}
	cache := memory.New()
	fsys := mustNewFS(t, inner, cache, []transform.Transform{{
		Name:    "upper",
		Match:   transform.Match{Extensions: []string{".txt"}},
		Command: &transform.Command{Args: []string{script}},
	}}, dir)

	first, err := fs.ReadFile(fsys, "foo.txt")
	if err != nil {
		t.Fatalf("first ReadFile: %v", err)
	}
	if string(first) != "ABC" {
		t.Fatalf("first = %q", first)
	}

	second, err := fs.ReadFile(fsys, "foo.txt")
	if err != nil {
		t.Fatalf("second ReadFile: %v", err)
	}
	if string(second) != "ABC" {
		t.Fatalf("second = %q", second)
	}

	sidecar, ok, err := cache.Get(context.Background(), "_transformed/upper/aa/bb.md5sum")
	if err != nil || !ok {
		t.Fatalf("sidecar Get: ok=%v err=%v", ok, err)
	}
	wantDigest := md5.Sum([]byte("ABC"))
	if string(sidecar) != hexMD5(wantDigest) {
		t.Fatalf("sidecar = %q, want %q", sidecar, hexMD5(wantDigest))
	}
}

func TestStatOnMissRunsTransform(t *testing.T) {
	dir := t.TempDir()
	countFile := filepath.Join(dir, "count.txt")
	script := filepath.Join(dir, "upper.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 1 >> "+countFile+"\ntr 'a-z' 'A-Z'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	inner := manifestLikeFS{files: map[string]manifestFile{
		"foo.txt": {data: []byte("abc"), cdnPath: "aa/bb", size: 3},
	}}
	cache := memory.New()
	fsys := mustNewFS(t, inner, cache, []transform.Transform{{
		Name:    "upper",
		Match:   transform.Match{Extensions: []string{".txt"}},
		Command: &transform.Command{Args: []string{script}},
	}}, dir)

	info, err := fs.Stat(fsys, "foo.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != 3 {
		t.Fatalf("Size() = %d, want 3", info.Size())
	}

	mfi, ok := info.(interface{ MD5() [md5.Size]byte })
	if !ok {
		t.Fatalf("Stat did not return ManifestFileInfo, got %T", info)
	}
	want := md5.Sum([]byte("ABC"))
	if mfi.MD5() != want {
		t.Fatalf("MD5() = %x, want %x", mfi.MD5(), want)
	}

	count, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatalf("read count: %v", err)
	}
	if string(count) != "1\n" {
		t.Fatalf("transform runs = %q, want one invocation", count)
	}
}

func mustNewFS(t *testing.T, inner fs.FS, c *memory.Cache, transforms []transform.Transform, baseDir string) fs.FS {
	t.Helper()
	fsys, err := transform.New(inner, c, transforms, transform.Limits{}, baseDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return fsys
}

type manifestFile struct {
	data    []byte
	cdnPath string
	size    int64
}

type manifestLikeFS struct {
	files map[string]manifestFile
}

func (m manifestLikeFS) Open(name string) (fs.File, error) {
	f, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &mapFile{name: filepath.Base(name), data: f.data, info: manifestFileInfo{name: filepath.Base(name), entry: f}}, nil
}

func (m manifestLikeFS) Stat(name string) (fs.FileInfo, error) {
	f, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return manifestFileInfo{name: filepath.Base(name), entry: f}, nil
}

type manifestFileInfo struct {
	name  string
	entry manifestFile
}

func (i manifestFileInfo) Name() string       { return i.name }
func (i manifestFileInfo) Size() int64        { return i.entry.size }
func (i manifestFileInfo) Mode() fs.FileMode  { return 0o444 }
func (i manifestFileInfo) ModTime() time.Time { return time.Time{} }
func (i manifestFileInfo) IsDir() bool        { return false }
func (i manifestFileInfo) Sys() any {
	return struct {
		CDNPath string
	}{CDNPath: i.entry.cdnPath}
}
func (i manifestFileInfo) GetCDNPath() string { return i.entry.cdnPath }

type mapFile struct {
	name string
	data []byte
	info fs.FileInfo
	off  int64
}

func (f *mapFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *mapFile) Close() error               { return nil }
func (f *mapFile) Read(p []byte) (int, error) {
	if f.off >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.off:])
	f.off += int64(n)
	return n, nil
}

func hexMD5(sum [md5.Size]byte) string {
	const hextable = "0123456789abcdef"
	out := make([]byte, md5.Size*2)
	for i, b := range sum {
		out[i*2] = hextable[b>>4]
		out[i*2+1] = hextable[b&0x0f]
	}
	return string(out)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
