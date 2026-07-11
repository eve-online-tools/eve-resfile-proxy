package mux_test

import (
	"errors"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/mapfetch"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/mux"
)

const (
	manifestMD5  = "d41d8cd98f00b204e9800998ecf8427e"
	manifestMD5A = "01010101010101010101010101010101"
	manifestMD5B = "02020202020202020202020202020202"
)

func TestMux_disjoint(t *testing.T) {
	win := "res:/win-only.png,win/only.png," + manifestMD5A + ",100\n"
	mac := "res:/mac-only.png,mac/only.png," + manifestMD5B + ",200\n"

	fsWin, err := vfs.New([]byte(win), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS win: %v", err)
	}
	fsMac, err := vfs.New([]byte(mac), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS mac: %v", err)
	}

	combined := mux.NewMux(fsWin, fsMac)

	winInfo, err := fs.Stat(combined, "win-only.png")
	if err != nil {
		t.Fatalf("Stat win-only.png: %v", err)
	}
	if winInfo.Size() != 100 {
		t.Fatalf("win size = %d", winInfo.Size())
	}

	macInfo, err := fs.Stat(combined, "mac-only.png")
	if err != nil {
		t.Fatalf("Stat mac-only.png: %v", err)
	}
	if macInfo.Size() != 200 {
		t.Fatalf("mac size = %d", macInfo.Size())
	}
}

func TestMux_firstMatch(t *testing.T) {
	first := "res:/shared.png,first/shared.png," + manifestMD5A + ",111\n"
	second := "res:/shared.png,second/shared.png," + manifestMD5B + ",222\n"

	fsFirst, err := vfs.New([]byte(first), mapfetch.New(map[string][]byte{
		"first/shared.png": []byte("first-bytes"),
	}), vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS first: %v", err)
	}
	fsSecond, err := vfs.New([]byte(second), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS second: %v", err)
	}

	combined := mux.NewMux(fsFirst, fsSecond)

	info, err := fs.Stat(combined, "shared.png")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != 111 {
		t.Fatalf("size = %d, want first layer (111)", info.Size())
	}

	f, err := combined.Open("shared.png")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "first-bytes" {
		t.Fatalf("data = %q, want first layer bytes", data)
	}
}

func TestMux_ReadDir(t *testing.T) {
	win := "res:/win-only.png,win/only.png," + manifestMD5A + "\n"
	mac := "res:/mac-only.png,mac/only.png," + manifestMD5B + "\n"

	fsWin, err := vfs.New([]byte(win), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS win: %v", err)
	}
	fsMac, err := vfs.New([]byte(mac), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS mac: %v", err)
	}

	combined := mux.NewMux(fsWin, fsMac)

	entries, err := fs.ReadDir(combined, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
}

func TestMux_withMapFS(t *testing.T) {
	mapLayer := fstest.MapFS{
		"extra.txt": &fstest.MapFile{Data: []byte("hello")},
	}
	manifest := "res:/indexed.png,aa/indexed.png," + manifestMD5 + "\n"
	indexLayer, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	combined := mux.NewMux(indexLayer, mapLayer)

	if _, err := fs.Stat(combined, "indexed.png"); err != nil {
		t.Fatalf("Stat indexed.png: %v", err)
	}
	if _, err := fs.Stat(combined, "extra.txt"); err != nil {
		t.Fatalf("Stat extra.txt: %v", err)
	}
}

func TestMux_empty(t *testing.T) {
	m := mux.NewMux()
	_, err := m.Open("missing.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open err = %v, want ErrNotExist", err)
	}
}

func TestMux_single(t *testing.T) {
	manifest := "res:/a.png,aa/a.png," + manifestMD5 + "\n"
	layer, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	combined := mux.NewMux(layer)
	if _, err := fs.Stat(combined, "a.png"); err != nil {
		t.Fatalf("Stat: %v", err)
	}
}

func TestMux_Glob(t *testing.T) {
	win := "res:/win.png,win/win.png," + manifestMD5A + "\n"
	mac := "res:/mac.png,mac/mac.png," + manifestMD5B + "\n"

	fsWin, err := vfs.New([]byte(win), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS win: %v", err)
	}
	fsMac, err := vfs.New([]byte(mac), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS mac: %v", err)
	}

	combined := mux.NewMux(fsWin, fsMac)

	matches, err := fs.Glob(combined, "*.png")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("matches = %v", matches)
	}
}

func TestMux_subpathOpen(t *testing.T) {
	child := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("hello")},
	}
	m := mux.NewMux()
	if err := m.Mount("foo", child); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	f, err := m.Open("foo/a.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
}

func TestMux_subpathReadDirRoot(t *testing.T) {
	child := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("hello")},
	}
	m := mux.NewMux()
	if err := m.Mount("foo", child); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	entries, err := fs.ReadDir(m, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "foo" || !entries[0].IsDir() {
		t.Fatalf("entries = %+v, want single foo directory", entries)
	}
}

func TestMux_registrationOrderRootBeforeSubpath(t *testing.T) {
	root := fstest.MapFS{
		"foo/x.txt": &fstest.MapFile{Data: []byte("root")},
	}
	child := fstest.MapFS{
		"x.txt": &fstest.MapFile{Data: []byte("child")},
	}

	m := mux.NewMux(root)
	if err := m.Mount("foo", child); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	data, err := m.ReadFile("foo/x.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "root" {
		t.Fatalf("data = %q, want root layer (earlier registration)", data)
	}
}

func TestMux_samePathOverlap(t *testing.T) {
	first := fstest.MapFS{
		"shared.txt": &fstest.MapFile{Data: []byte("first")},
		"only-a.txt": &fstest.MapFile{Data: []byte("a")},
	}
	second := fstest.MapFS{
		"shared.txt": &fstest.MapFile{Data: []byte("second")},
		"only-b.txt": &fstest.MapFile{Data: []byte("b")},
	}

	m := mux.NewMux()
	if err := m.Mount("foo", first); err != nil {
		t.Fatalf("Mount first: %v", err)
	}
	if err := m.Mount("foo", second); err != nil {
		t.Fatalf("Mount second: %v", err)
	}

	entries, err := fs.ReadDir(m, "foo")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}

	data, err := m.ReadFile("foo/shared.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("data = %q, want first mount", data)
	}
}

func TestMux_mountInvalidPath(t *testing.T) {
	m := mux.NewMux()
	err := m.Mount("..", fstest.MapFS{})
	if err == nil {
		t.Fatal("expected error for invalid mount path")
	}
	if !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("err = %v, want fs.ErrInvalid", err)
	}
}

func TestMux_duplicateMountBaseAllowed(t *testing.T) {
	m := mux.NewMux()
	if err := m.Mount("foo", fstest.MapFS{"a.txt": &fstest.MapFile{Data: []byte("a")}}); err != nil {
		t.Fatalf("first Mount: %v", err)
	}
	if err := m.Mount("foo", fstest.MapFS{"b.txt": &fstest.MapFile{Data: []byte("b")}}); err != nil {
		t.Fatalf("second Mount: %v", err)
	}
}
