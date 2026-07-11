package urlpath_test

import (
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/service/http/urlpath"
)

func TestCleanURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: "/"},
		{in: "/", want: "/"},
		{in: "//", want: "/"},
		{in: "///", want: "/"},
		{in: "/icons//64/", want: "/icons/64/"},
		{in: "//icons//64//", want: "/icons/64/"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := urlpath.CleanURL(tt.in); got != tt.want {
				t.Fatalf("CleanURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestJoinURL(t *testing.T) {
	t.Parallel()

	if got := urlpath.JoinURL("//", "icons"); got != "/icons" {
		t.Fatalf("JoinURL(//, icons) = %q, want /icons", got)
	}
	if got := urlpath.JoinURL("/icons/64/", "foo.png"); got != "/icons/64/foo.png" {
		t.Fatalf("JoinURL = %q, want /icons/64/foo.png", got)
	}
}

func TestToFS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		urlPath string
		want    string
		wantErr bool
	}{
		{name: "file path", urlPath: "/icons/64/icon.png", want: "icons/64/icon.png"},
		{name: "lowercase", urlPath: "/Icons/Foo.PNG", want: "icons/foo.png"},
		{name: "root", urlPath: "/", want: ""},
		{name: "nested", urlPath: "//icons//64//foo.png", want: "icons/64/foo.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := urlpath.ToFS(tt.urlPath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ToFS() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ToFS() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDirFromURL(t *testing.T) {
	t.Parallel()

	got, err := urlpath.DirFromURL("/icons/64/")
	if err != nil {
		t.Fatalf("DirFromURL() error = %v", err)
	}
	if got != "icons/64" {
		t.Fatalf("DirFromURL() = %q, want icons/64", got)
	}

	root, err := urlpath.DirFromURL("/")
	if err != nil {
		t.Fatalf("DirFromURL(/) error = %v", err)
	}
	if root != "." {
		t.Fatalf("DirFromURL(/) = %q, want .", root)
	}

	double, err := urlpath.DirFromURL("//")
	if err != nil {
		t.Fatalf("DirFromURL(//) error = %v", err)
	}
	if double != "." {
		t.Fatalf("DirFromURL(//) = %q, want .", double)
	}
}
