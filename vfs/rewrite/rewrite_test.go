package rewrite_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs/rewrite"
)

func TestFS_prefixOpenStatReadFile(t *testing.T) {
	parent := fstest.MapFS{
		"icons/64/icon.png": &fstest.MapFile{Data: []byte("icon-bytes")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "legacy/icons/", To: "icons/"},
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

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "favicon.ico", To: "ui/icons/favicon.ico"},
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

	info, err := fs.Stat(fsys, "favicon.ico")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Name() != "favicon.ico" {
		t.Fatalf("Name() = %q, want favicon.ico", info.Name())
	}
}

func TestFS_longestFromWins(t *testing.T) {
	parent := fstest.MapFS{
		"a/b/data.txt":   &fstest.MapFile{Data: []byte("shallow")},
		"a/b/c/data.txt": &fstest.MapFile{Data: []byte("deep")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "alias/a/b/", To: "a/b/"},
		{From: "alias/a/b/c/", To: "a/b/c/"},
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

	data, err = fs.ReadFile(fsys, "alias/a/b/data.txt")
	if err != nil {
		t.Fatalf("ReadFile shallow: %v", err)
	}
	if string(data) != "shallow" {
		t.Fatalf("shallow data = %q", data)
	}
}

func TestFS_passthrough(t *testing.T) {
	parent := fstest.MapFS{
		"real.txt": &fstest.MapFile{Data: []byte("real")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "alias.txt", To: "real.txt"},
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

func TestFS_emptyRulesReturnsParent(t *testing.T) {
	parent := fstest.MapFS{
		"real.txt": &fstest.MapFile{Data: []byte("real")},
	}

	fsys, err := rewrite.New(parent, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := fsys.(fstest.MapFS); !ok {
		t.Fatal("expected parent fs unchanged")
	}
}

func TestFS_readDirVirtualEntries(t *testing.T) {
	parent := fstest.MapFS{
		"icons/64/icon.png": &fstest.MapFile{Data: []byte("icon")},
		"icons/readme.txt":  &fstest.MapFile{Data: []byte("readme")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "legacy/icons/", To: "icons/"},
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

	legacy, err := fs.ReadDir(fsys, "legacy")
	if err != nil {
		t.Fatalf("ReadDir legacy: %v", err)
	}
	if !containsDirEntry(legacy, "icons") {
		t.Fatalf("legacy entries = %v, want icons dir", entryNames(legacy))
	}

	icons, err := fs.ReadDir(fsys, "legacy/icons")
	if err != nil {
		t.Fatalf("ReadDir legacy/icons: %v", err)
	}
	if !containsDirEntry(icons, "64") || !containsFileEntry(icons, "readme.txt") {
		t.Fatalf("legacy/icons entries = %v", entryNames(icons))
	}
}

func TestFS_singleSegmentDirAliasAtRoot(t *testing.T) {
	parent := fstest.MapFS{
		"ui/foo.txt": &fstest.MapFile{Data: []byte("hello")},
		"ui/bar.txt": &fstest.MapFile{Data: []byte("world")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "ui.base64/", To: "ui/"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	root, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir .: %v", err)
	}
	if !containsDirEntry(root, "ui.base64") {
		t.Fatalf("root entries = %v, want ui.base64 directory", entryNames(root))
	}
	if containsFileEntry(root, "ui.base64") {
		t.Fatal("ui.base64 should be a directory, not a file")
	}

	alias, err := fs.ReadDir(fsys, "ui.base64")
	if err != nil {
		t.Fatalf("ReadDir ui.base64: %v", err)
	}
	if !containsFileEntry(alias, "foo.txt") || !containsFileEntry(alias, "bar.txt") {
		t.Fatalf("ui.base64 entries = %v", entryNames(alias))
	}

	data, err := fs.ReadFile(fsys, "ui.base64/foo.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
}

func TestFS_singleSegmentDirAliasStat(t *testing.T) {
	parent := fstest.MapFS{
		"ui/foo.txt": &fstest.MapFile{Data: []byte("hello")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "ui.base64/", To: "ui/"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	info, err := fs.Stat(fsys, "ui.base64")
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory")
	}
	if info.Name() != "ui.base64" {
		t.Fatalf("Name() = %q, want ui.base64", info.Name())
	}

	fileInfo, err := fs.Stat(fsys, "ui.base64/foo.txt")
	if err != nil {
		t.Fatalf("Stat file: %v", err)
	}
	if fileInfo.Name() != "foo.txt" {
		t.Fatalf("Name() = %q, want foo.txt", fileInfo.Name())
	}
}

func TestFS_nestedVirtualDirAlias(t *testing.T) {
	parent := fstest.MapFS{
		"ui/foo.txt": &fstest.MapFile{Data: []byte("hello")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "examples/ui.md5sum/", To: "ui/"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	root, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir .: %v", err)
	}
	if !containsDirEntry(root, "examples") {
		t.Fatalf("root entries = %v, want examples dir", entryNames(root))
	}

	examples, err := fs.ReadDir(fsys, "examples")
	if err != nil {
		t.Fatalf("ReadDir examples: %v", err)
	}
	if !containsDirEntry(examples, "ui.md5sum") {
		t.Fatalf("examples entries = %v, want ui.md5sum dir", entryNames(examples))
	}

	data, err := fs.ReadFile(fsys, "examples/ui.md5sum/foo.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
}

func TestFS_readDirFileAliasAtRoot(t *testing.T) {
	parent := fstest.MapFS{
		"ui/favicon.ico": &fstest.MapFile{Data: []byte("fav")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "favicon.ico", To: "ui/favicon.ico"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	root, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if !containsFileEntry(root, "favicon.ico") {
		t.Fatalf("root entries = %v, want favicon.ico file", entryNames(root))
	}

	for _, entry := range root {
		if entry.Name() != "favicon.ico" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("Info: %v", err)
		}
		if info.Size() != 3 {
			t.Fatalf("favicon.ico size = %d, want 3", info.Size())
		}
	}
}

func TestFS_globAliasPattern(t *testing.T) {
	parent := fstest.MapFS{
		"icons/a.png": &fstest.MapFile{Data: []byte("a")},
		"icons/b.png": &fstest.MapFile{Data: []byte("b")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "legacy/icons/", To: "icons/"},
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
	for _, path := range want {
		if !containsString(matches, path) {
			t.Fatalf("matches = %v, missing %q", matches, path)
		}
	}
}

func TestFS_globSingleSegmentDirAlias(t *testing.T) {
	parent := fstest.MapFS{
		"ui/foo.txt": &fstest.MapFile{Data: []byte("a")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "ui.base64/", To: "ui/"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	matches, err := fs.Glob(fsys, "ui.base64/*.txt")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) != 1 || matches[0] != "ui.base64/foo.txt" {
		t.Fatalf("matches = %v", matches)
	}
}

func TestFS_globPassthrough(t *testing.T) {
	parent := fstest.MapFS{
		"icons/a.png": &fstest.MapFile{Data: []byte("a")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "legacy/icons/", To: "icons/"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	matches, err := fs.Glob(fsys, "icons/*.png")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) != 1 || matches[0] != "icons/a.png" {
		t.Fatalf("matches = %v", matches)
	}
}

func TestCompileRules_validation(t *testing.T) {
	tests := []struct {
		name  string
		rules []rewrite.Rule
	}{
		{name: "duplicate from", rules: []rewrite.Rule{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
		}},
		{name: "same from and to", rules: []rewrite.Rule{
			{From: "same", To: "same"},
		}},
		{name: "empty from", rules: []rewrite.Rule{
			{From: "", To: "b"},
		}},
		{name: "invalid from", rules: []rewrite.Rule{
			{From: "../escape", To: "b"},
		}},
		{name: "dir from without dir to", rules: []rewrite.Rule{
			{From: "ui.base64/", To: "ui"},
		}},
		{name: "dir to without dir from", rules: []rewrite.Rule{
			{From: "ui.base64", To: "ui/"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := rewrite.New(fstest.MapFS{}, tt.rules)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestFS_underlyingWinsOnReadDirCollision(t *testing.T) {
	parent := fstest.MapFS{
		"legacy/real.txt": &fstest.MapFile{Data: []byte("real")},
	}

	fsys, err := rewrite.New(parent, []rewrite.Rule{
		{From: "legacy/icons/", To: "icons/"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	for _, entry := range entries {
		if entry.Name() == "legacy" && !entry.IsDir() {
			t.Fatalf("expected legacy to remain a directory entry, got file")
		}
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

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestFS_openMissing(t *testing.T) {
	fsys, err := rewrite.New(fstest.MapFS{}, []rewrite.Rule{
		{From: "missing", To: "nowhere"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = fsys.Open("missing")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open err = %v, want ErrNotExist", err)
	}
}
