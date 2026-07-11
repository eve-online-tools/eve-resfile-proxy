package vfs_test

import (
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/mapfetch"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/mux"
)

func TestNewFS_res_Stat(t *testing.T) {
	manifest := "" +
		"res:/icons/64/icon64.png,7d/icon64_hash," + manifestMD5 + ",1024,512\n" +
		"res:/readme.txt,aa/readme_hash," + manifestMD5 + ",42,21\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	info, err := fs.Stat(fsys, "icons/64/icon64.png")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != 1024 {
		t.Fatalf("size = %d, want 1024", info.Size())
	}
	if info.IsDir() {
		t.Fatal("expected file, got directory")
	}
}

func TestNewFS_app_prefix(t *testing.T) {
	manifest := "app:/resfileindex.txt,res/global.txt," + manifestMD5 + ",2048,1024\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("app"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	info, err := fs.Stat(fsys, "resfileindex.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != 2048 {
		t.Fatalf("size = %d, want 2048", info.Size())
	}
}

func TestNewFS_ReadDir(t *testing.T) {
	manifest := "" +
		"res:/icons/64/icon64.png,7d/icon64_hash\n" +
		"res:/icons/64/icon32.png,7d/icon32_hash\n" +
		"res:/icons/readme.txt,aa/readme_hash\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	root, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir .: %v", err)
	}
	if len(root) != 1 || root[0].Name() != "icons" || !root[0].IsDir() {
		t.Fatalf("root = %#v, want single icons/ directory", root)
	}

	icons64, err := fs.ReadDir(fsys, "icons/64")
	if err != nil {
		t.Fatalf("ReadDir icons/64: %v", err)
	}
	if len(icons64) != 2 {
		t.Fatalf("icons/64 entries = %d, want 2", len(icons64))
	}
	if icons64[0].Name() != "icon32.png" || icons64[1].Name() != "icon64.png" {
		t.Fatalf("icons/64 order = [%s, %s]", icons64[0].Name(), icons64[1].Name())
	}
}

func TestNewFS_ErrNotExist(t *testing.T) {
	manifest := "res:/a.png,aa/a_hash\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	if _, err := fs.Stat(fsys, "missing.png"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat missing: %v", err)
	}
	if _, err := fs.Stat(fsys, "../escape.png"); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("Stat ..: %v", err)
	}
}

func TestNewFS_OpenWithoutFetcher(t *testing.T) {
	manifest := "res:/a.png,aa/a_hash\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	_, err = fsys.Open("a.png")
	if !errors.Is(err, vfs.ErrFetchNotConfigured) {
		t.Fatalf("Open: %v, want ErrFetchNotConfigured", err)
	}
}

func TestNewFS_OpenWithMapfetch(t *testing.T) {
	const cdnPath = "aa/a_hash"
	want := []byte("png-bytes")
	manifest := "res:/a.png," + cdnPath + "\n"

	fsys, err := vfs.New([]byte(manifest), mapfetch.New(map[string][]byte{
		cdnPath: want,
	}), vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	f, err := fsys.Open("a.png")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = f.Close() }()

	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("data = %q, want %q", got, want)
	}
}

func TestNewFS_Open_checksumMismatchIgnoredByDefault(t *testing.T) {
	const cdnPath = "aa/a_hash"
	manifest := "res:/a.png," + cdnPath + "," + manifestMD5 + "\n"

	fsys, err := vfs.New([]byte(manifest), mapfetch.New(map[string][]byte{
		cdnPath: []byte("wrong-bytes"),
	}), vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	if _, err := fsys.Open("a.png"); err != nil {
		t.Fatalf("Open: %v, want success without validation", err)
	}
}

func TestNewFS_Open_checksumMismatchWithValidate(t *testing.T) {
	const cdnPath = "aa/a_hash"
	manifest := "res:/a.png," + cdnPath + "," + manifestMD5 + "\n"

	fsys, err := vfs.New([]byte(manifest), mapfetch.New(map[string][]byte{
		cdnPath: []byte("wrong-bytes"),
	}), vfs.WithPrefix("res"), vfs.WithValidate())
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	_, err = fsys.Open("a.png")
	if !errors.Is(err, vfs.ErrChecksumMismatch) {
		t.Fatalf("Open: %v, want ErrChecksumMismatch", err)
	}
}

func TestNewFS_Open_emptyMD5WithValidate(t *testing.T) {
	const cdnPath = "aa/a_hash"
	manifest := "res:/a.png," + cdnPath + "\n"

	fsys, err := vfs.New([]byte(manifest), mapfetch.New(map[string][]byte{
		cdnPath: []byte("any-bytes"),
	}), vfs.WithPrefix("res"), vfs.WithValidate())
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	if _, err := fsys.Open("a.png"); err != nil {
		t.Fatalf("Open: %v, want success when MD5 column empty", err)
	}
}

func TestNewFS_resLowercase(t *testing.T) {
	manifest := "res:/Icons/Foo.PNG,aa/foo_hash\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	if _, err := fs.Stat(fsys, "icons/foo.png"); err != nil {
		t.Fatalf("Stat lowercase path: %v", err)
	}
}

func TestNewFS_requiresPrefix(t *testing.T) {
	_, err := vfs.New([]byte("res:/a.png,aa/a\n"), nil)
	if err == nil {
		t.Fatal("expected error without WithPrefix")
	}
}

func TestGlob(t *testing.T) {
	manifest := "" +
		"res:/root.png,aa/root\n" +
		"res:/icons/64/icon64.png,7d/icon64\n" +
		"res:/icons/64/icon32.png,7d/icon32\n" +
		"res:/icons/readme.txt,aa/readme\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	matches, err := fs.Glob(fsys, "*.png")
	if err != nil {
		t.Fatalf("Glob *.png: %v", err)
	}
	if len(matches) != 1 || matches[0] != "root.png" {
		t.Fatalf("Glob *.png = %v", matches)
	}

	matches, err = fs.Glob(fsys, "icons/**/*.png")
	if err != nil {
		t.Fatalf("Glob icons/**/*.png: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("Glob icons/**/*.png = %v", matches)
	}
}

func TestNestedManifest(t *testing.T) {
	appManifest := "app:/resfileindex.txt,res/global.txt," + manifestMD5 + "\n"
	fsApp, err := vfs.New([]byte(appManifest), nil, vfs.WithPrefix("app"))
	if err != nil {
		t.Fatalf("NewFS app: %v", err)
	}

	if _, err := fs.Stat(fsApp, "resfileindex.txt"); err != nil {
		t.Fatalf("Stat resfileindex.txt: %v", err)
	}

	resManifest := "" +
		"res:/shared.png,global/shared.png," + manifestMD5A + "\n" +
		"res:/win-only.png,win/only.png," + manifestMD5B + "\n"

	fsWin, err := vfs.New([]byte(resManifest), nil, vfs.WithPrefix("res"))
	if err != nil {
		t.Fatalf("NewFS win: %v", err)
	}

	combined := mux.NewMux(fsWin)

	if _, err := fs.Stat(combined, "shared.png"); err != nil {
		t.Fatalf("Stat shared.png: %v", err)
	}
	if _, err := fs.Stat(combined, "win-only.png"); err != nil {
		t.Fatalf("Stat win-only.png: %v", err)
	}
}
