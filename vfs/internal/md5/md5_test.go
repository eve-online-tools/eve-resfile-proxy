package md5_test

import (
	cryptomd5 "crypto/md5"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs/internal/md5"
)

func TestParse_roundTrip(t *testing.T) {
	const hex = "faa842f6f3157c8d6cde37b496a43b30"
	got, err := md5.Parse(hex)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.String() != hex {
		t.Fatalf("String() = %q", got.String())
	}
	if got.IsZero() {
		t.Fatal("expected non-zero digest")
	}
}

func TestParse_empty(t *testing.T) {
	got, err := md5.Parse("")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !got.IsZero() {
		t.Fatal("expected zero digest")
	}
}

func TestParse_invalid(t *testing.T) {
	_, err := md5.Parse("not-a-valid-md5")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDigest_equals_md5Sum(t *testing.T) {
	data := []byte("hello resfile")
	sum := cryptomd5.Sum(data)
	digest := md5.Digest(sum)
	if digest.Sum() != sum {
		t.Fatal("Sum() should match crypto/md5.Sum")
	}
}
