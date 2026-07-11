package key_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/cache/key"
)

func TestJoin_valid(t *testing.T) {
	root := t.TempDir()
	got, err := key.Join(root, "7d/icon64_hash")
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	want := filepath.Join(root, "7d", "icon64_hash")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestJoin_traversalRejected(t *testing.T) {
	root := t.TempDir()
	_, err := key.Join(root, "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, key.ErrInvalidKey) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidate_empty(t *testing.T) {
	err := key.Validate("")
	if !errors.Is(err, key.ErrEmptyKey) {
		t.Fatalf("err = %v", err)
	}
}
