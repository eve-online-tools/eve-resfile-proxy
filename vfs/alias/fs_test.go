package alias_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs/alias"
)

func pathAlias(aliasPath, target string) alias.Alias {
	return alias.Alias{Alias: aliasPath, Target: target}
}

func extAlias(aliasExt, targetExt, prefix string, recursive *bool) alias.Alias {
	return alias.Alias{
		Alias:  aliasExt,
		Target: targetExt,
		Match: &alias.Match{
			PathPrefix: prefix,
			Recursive:  recursive,
		},
	}
}

func boolPtr(v bool) *bool { return &v }

func TestFS_prefixOpenStatReadFile(t *testing.T) {
	parent := fstest.MapFS{
		"icons/64/icon.png": &fstest.MapFile{Data: []byte("icon-bytes")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		pathAlias("legacy/icons/", "icons/"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "legacy/icons/64/icon.png")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "icon-bytes" {
		t.Fatalf("data = %q", data)
	}

	info, err := fs.Stat(fsys, "legacy/icons/64/icon.png")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.IsDir() {
		t.Fatal("expected file")
	}
	if info.Name() != "icon.png" {
		t.Fatalf("Name() = %q, want icon.png", info.Name())
	}
}

func TestFS_exactFileAlias(t *testing.T) {
	parent := fstest.MapFS{
		"ui/icons/favicon.ico": &fstest.MapFile{Data: []byte("fav")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		pathAlias("favicon.ico", "ui/icons/favicon.ico"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "favicon.ico")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "fav" {
		t.Fatalf("data = %q", data)
	}
}

func TestFS_longestAliasWins(t *testing.T) {
	parent := fstest.MapFS{
		"a/b/data.txt":   &fstest.MapFile{Data: []byte("shallow")},
		"a/b/c/data.txt": &fstest.MapFile{Data: []byte("deep")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		pathAlias("alias/a/b/", "a/b/"),
		pathAlias("alias/a/b/c/", "a/b/c/"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "alias/a/b/c/data.txt")
	if err != nil {
		t.Fatalf("ReadFile deep: %v", err)
	}
	if string(data) != "deep" {
		t.Fatalf("deep data = %q", data)
	}
}

func TestFS_passthrough(t *testing.T) {
	parent := fstest.MapFS{
		"real.txt": &fstest.MapFile{Data: []byte("real")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		pathAlias("alias.txt", "real.txt"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "real.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "real" {
		t.Fatalf("data = %q", data)
	}
}

func TestFS_emptyAliasesReturnsParent(t *testing.T) {
	parent := fstest.MapFS{
		"real.txt": &fstest.MapFile{Data: []byte("real")},
	}

	fsys, err := alias.New(parent, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := fsys.(fstest.MapFS); !ok {
		t.Fatal("expected parent fs unchanged")
	}
}

func TestFS_extensionRead(t *testing.T) {
	parent := fstest.MapFS{
		"ui/icons/foo.png": &fstest.MapFile{Data: []byte("png-bytes")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		extAlias(".webm", ".png", "ui/icons/", nil),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "ui/icons/foo.webm")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "png-bytes" {
		t.Fatalf("data = %q", data)
	}

	info, err := fs.Stat(fsys, "ui/icons/foo.webm")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Name() != "foo.webm" {
		t.Fatalf("Name() = %q", info.Name())
	}
}

func TestFS_extensionStackWithPath(t *testing.T) {
	parent := fstest.MapFS{
		"graphics/effect.dx11/model.cmf": &fstest.MapFile{Data: []byte("cmf")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		pathAlias("graphics/effect.vulkan/", "graphics/effect.dx11/"),
		extAlias(".gr2", ".cmf", "graphics/effect.vulkan/", nil),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "graphics/effect.vulkan/model.gr2")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "cmf" {
		t.Fatalf("data = %q", data)
	}
}

func TestFS_extensionCollisionRealFileWins(t *testing.T) {
	parent := fstest.MapFS{
		"ui/icons/foo.webm": &fstest.MapFile{Data: []byte("real-webm")},
		"ui/icons/foo.png":  &fstest.MapFile{Data: []byte("png")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		extAlias(".webm", ".png", "ui/icons/", nil),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := fs.ReadFile(fsys, "ui/icons/foo.webm")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "real-webm" {
		t.Fatalf("data = %q, want real-webm", data)
	}
}

func TestFS_extensionReadDirVirtual(t *testing.T) {
	parent := fstest.MapFS{
		"ui/icons/foo.png": &fstest.MapFile{Data: []byte("png")},
		"ui/icons/bar.txt": &fstest.MapFile{Data: []byte("txt")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		extAlias(".webm", ".png", "ui/icons/", nil),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	entries, err := fs.ReadDir(fsys, "ui/icons")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if !containsFileEntry(entries, "foo.webm") {
		t.Fatalf("entries = %v, want foo.webm", entryNames(entries))
	}
	if containsFileEntry(entries, "foo.png") {
		t.Fatalf("entries = %v, should not contain foo.png", entryNames(entries))
	}
	if !containsFileEntry(entries, "bar.txt") {
		t.Fatalf("entries = %v, want bar.txt", entryNames(entries))
	}
}

func TestFS_extensionReadDirCollisionShowsBoth(t *testing.T) {
	parent := fstest.MapFS{
		"ui/icons/foo.webm": &fstest.MapFile{Data: []byte("webm")},
		"ui/icons/foo.png":  &fstest.MapFile{Data: []byte("png")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		extAlias(".webm", ".png", "ui/icons/", nil),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	entries, err := fs.ReadDir(fsys, "ui/icons")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if !containsFileEntry(entries, "foo.webm") || !containsFileEntry(entries, "foo.png") {
		t.Fatalf("entries = %v, want both foo.webm and foo.png", entryNames(entries))
	}
}

func TestFS_readDirVirtualEntries(t *testing.T) {
	parent := fstest.MapFS{
		"icons/64/icon.png": &fstest.MapFile{Data: []byte("icon")},
		"icons/readme.txt":  &fstest.MapFile{Data: []byte("readme")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		pathAlias("legacy/icons/", "icons/"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	root, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir .: %v", err)
	}
	if !containsDirEntry(root, "legacy") {
		t.Fatalf("root entries = %v, want legacy dir", entryNames(root))
	}
}

func TestFS_singleSegmentDirAliasAtRoot(t *testing.T) {
	parent := fstest.MapFS{
		"ui/foo.txt": &fstest.MapFile{Data: []byte("hello")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		pathAlias("ui.base64/", "ui/"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	root, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir .: %v", err)
	}
	if !containsDirEntry(root, "ui.base64") {
		t.Fatalf("root entries = %v", entryNames(root))
	}
}

func TestFS_globAliasPattern(t *testing.T) {
	parent := fstest.MapFS{
		"icons/a.png": &fstest.MapFile{Data: []byte("a")},
		"icons/b.png": &fstest.MapFile{Data: []byte("b")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		pathAlias("legacy/icons/", "icons/"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	matches, err := fs.Glob(fsys, "legacy/icons/*.png")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	want := []string{"legacy/icons/a.png", "legacy/icons/b.png"}
	if len(matches) != len(want) {
		t.Fatalf("matches = %v, want %v", matches, want)
	}
}

func TestFS_globExtension(t *testing.T) {
	parent := fstest.MapFS{
		"ui/icons/a.png": &fstest.MapFile{Data: []byte("a")},
	}

	fsys, err := alias.New(parent, []alias.Alias{
		extAlias(".webm", ".png", "ui/icons/", nil),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	matches, err := fs.Glob(fsys, "ui/icons/*.webm")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) != 1 || matches[0] != "ui/icons/a.webm" {
		t.Fatalf("matches = %v", matches)
	}
}

func TestFS_openMissing(t *testing.T) {
	fsys, err := alias.New(fstest.MapFS{}, []alias.Alias{
		pathAlias("missing", "nowhere"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = fsys.Open("missing")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open err = %v, want ErrNotExist", err)
	}
}

func TestFS_extensionMissingTarget(t *testing.T) {
	fsys, err := alias.New(fstest.MapFS{}, []alias.Alias{
		extAlias(".webm", ".png", "ui/", boolPtr(true)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = fs.ReadFile(fsys, "ui/foo.webm")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadFile err = %v, want ErrNotExist", err)
	}
}

func containsDirEntry(entries []fs.DirEntry, name string) bool {
	for _, entry := range entries {
		if entry.Name() == name && entry.IsDir() {
			return true
		}
	}
	return false
}

func containsFileEntry(entries []fs.DirEntry, name string) bool {
	for _, entry := range entries {
		if entry.Name() == name && !entry.IsDir() {
			return true
		}
	}
	return false
}

func entryNames(entries []fs.DirEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}
