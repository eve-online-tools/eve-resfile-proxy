package path_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

func TestCleanFile_lowercases(t *testing.T) {
	got, err := path.CleanFile("Icons/Foo.PNG")
	if err != nil {
		t.Fatalf("CleanFile: %v", err)
	}
	if got != "icons/foo.png" {
		t.Fatalf("got %q, want icons/foo.png", got)
	}
}

func TestCleanFile_rejectsDot(t *testing.T) {
	_, err := path.CleanFile(".")
	if err == nil {
		t.Fatal("expected error")
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("err = %v", err)
	}
}

func TestCleanFile_rejectsInvalid(t *testing.T) {
	_, err := path.CleanFile("../escape")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCleanDir_root(t *testing.T) {
	got, err := path.CleanDir(".")
	if err != nil {
		t.Fatalf("CleanDir: %v", err)
	}
	if got != "." {
		t.Fatalf("got %q, want .", got)
	}
}

func TestCleanDirPrefix(t *testing.T) {
	got, err := path.CleanDirPrefix("Icons")
	if err != nil {
		t.Fatalf("CleanDirPrefix: %v", err)
	}
	if got != "icons/" {
		t.Fatalf("got %q, want icons/", got)
	}
}
