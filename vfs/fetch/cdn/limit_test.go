package cdn

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadLimitedBody_withinLimit(t *testing.T) {
	data, err := readLimitedBody(bytes.NewReader([]byte("hello")), 10)
	if err != nil {
		t.Fatalf("readLimitedBody: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
}

func TestReadLimitedBody_exceedsLimit(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 11)
	_, err := readLimitedBody(bytes.NewReader(body), 10)
	if err == nil {
		t.Fatal("expected error for oversized body")
	}
	if !strings.Contains(err.Error(), "exceeds max size") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadLimitedBody_atLimit(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 10)
	data, err := readLimitedBody(bytes.NewReader(body), 10)
	if err != nil {
		t.Fatalf("readLimitedBody: %v", err)
	}
	if len(data) != 10 {
		t.Fatalf("len = %d, want 10", len(data))
	}
}
