package index

import (
	"strings"
	"testing"
)

func TestIconForFileByExtension(t *testing.T) {
	png := string(IconFor("foo.png", false))
	txt := string(IconFor("readme.txt", false))

	if !strings.HasPrefix(png, "data:image/svg+xml;base64,") {
		t.Fatalf("png icon = %q, want data URI prefix", png)
	}
	if png == txt {
		t.Fatal("png and txt icons should differ")
	}
}

func TestIconForDirectory(t *testing.T) {
	dir := string(IconFor("subdir", true))
	file := string(IconFor("readme.txt", false))

	if dir == file {
		t.Fatal("directory and file icons should differ")
	}
	if !strings.HasPrefix(dir, "data:image/svg+xml;base64,") {
		t.Fatalf("dir icon = %q, want data URI prefix", dir)
	}
}

func TestParentIcon(t *testing.T) {
	parent := string(ParentIcon())
	folder := string(IconFor("subdir", true))

	if !strings.HasPrefix(parent, "data:image/svg+xml;base64,") {
		t.Fatalf("parent icon = %q, want data URI prefix", parent)
	}
	if parent == folder {
		t.Fatal("parent and folder icons should differ")
	}
}

func TestIconForUnknownExtension(t *testing.T) {
	unknown := string(IconFor("data.unknown", false))
	fallback := string(iconURI("default_file"))

	if unknown != fallback {
		t.Fatalf("unknown icon = %q, want default %q", unknown, fallback)
	}
}
