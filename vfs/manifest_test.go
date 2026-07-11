package vfs_test

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
)

func TestNewFS_prefixValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []vfs.Option
		wantErr string
	}{
		{
			name:    "missing prefix",
			opts:    nil,
			wantErr: "prefix is required",
		},
		{
			name:    "invalid prefix",
			opts:    []vfs.Option{vfs.WithPrefix("foo")},
			wantErr: `invalid prefix "foo"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := vfs.New([]byte("res:/a.png,aa/a\n"), nil, tt.opts...)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewFS_validEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest string
		prefix   vfs.Prefix
		path     string
		wantSize int64
	}{
		{
			name:     "res full columns",
			manifest: "res:/icons/64/icon.png,icons/icon_hash," + manifestMD5 + ",4096,2048,0644\n",
			prefix:   vfs.PrefixRes,
			path:     "icons/64/icon.png",
			wantSize: 4096,
		},
		{
			name:     "res minimal columns",
			manifest: "res:/a.png,aa/a_hash\n",
			prefix:   vfs.PrefixRes,
			path:     "a.png",
			wantSize: 0,
		},
		{
			name:     "res lowercase normalization",
			manifest: "res:/Icons/Foo.PNG,aa/foo_hash," + manifestMD5 + ",128,64\n",
			prefix:   vfs.PrefixRes,
			path:     "icons/foo.png",
			wantSize: 128,
		},
		{
			name:     "app build index",
			manifest: "app:/resfileindex.txt,res/global.txt," + manifestMD5 + ",2048,1024\n",
			prefix:   vfs.PrefixApp,
			path:     "resfileindex.txt",
			wantSize: 2048,
		},
		{
			name: "app macos path",
			manifest: "app:/EVE.app/Contents/Resources/build/resfileindex.txt," +
				"res/global.txt," + manifestMD5 + ",8192,4096\n",
			prefix:   vfs.PrefixApp,
			path:     "eve.app/contents/resources/build/resfileindex.txt",
			wantSize: 8192,
		},
		{
			name:     "whitespace trimmed fields",
			manifest: "  res:/trimmed.png , aa/trimmed_hash , " + manifestMD5 + " , 99 , 50 \n",
			prefix:   vfs.PrefixRes,
			path:     "trimmed.png",
			wantSize: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fsys, err := vfs.New([]byte(tt.manifest), nil, vfs.WithPrefix(tt.prefix))
			if err != nil {
				t.Fatalf("NewFS: %v", err)
			}

			info, err := fs.Stat(fsys, tt.path)
			if err != nil {
				t.Fatalf("Stat(%q): %v", tt.path, err)
			}
			if info.Size() != tt.wantSize {
				t.Fatalf("Size() = %d, want %d", info.Size(), tt.wantSize)
			}
			if info.IsDir() {
				t.Fatal("expected file")
			}
		})
	}
}

func TestNewFS_skippedLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		manifest      string
		prefix        vfs.Prefix
		present       []string
		absent        []string
		wantReadCount int
	}{
		{
			name: "wrong prefix and empty lines",
			manifest: "" +
				"\n" +
				"res:/good.png,aa/good\n" +
				"app:/wrong-prefix.txt,aa/wrong\n" +
				"only-one-column\n" +
				"   \n",
			prefix:        vfs.PrefixRes,
			present:       []string{"good.png"},
			absent:        []string{"wrong-prefix.txt"},
			wantReadCount: 1,
		},
		{
			name: "non matching scheme",
			manifest: "" +
				"other:/file.txt,other/file.txt," + manifestMD5 + "\n" +
				"res:/kept.png,aa/kept\n",
			prefix:        vfs.PrefixRes,
			present:       []string{"kept.png"},
			absent:        []string{"file.txt"},
			wantReadCount: 1,
		},
		{
			name: "bare namespace without path",
			manifest: "" +
				"res:/,aa/root\n" +
				"res:/valid.png,aa/valid\n",
			prefix:        vfs.PrefixRes,
			present:       []string{"valid.png"},
			wantReadCount: 1,
		},
		{
			name: "app prefix filter",
			manifest: "" +
				"res:/skip.png,aa/skip\n" +
				"app:/resfileindex.txt,res/index," + manifestMD5 + "\n",
			prefix:        vfs.PrefixApp,
			present:       []string{"resfileindex.txt"},
			absent:        []string{"skip.png"},
			wantReadCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fsys, err := vfs.New([]byte(tt.manifest), nil, vfs.WithPrefix(tt.prefix))
			if err != nil {
				t.Fatalf("NewFS: %v", err)
			}

			for _, path := range tt.present {
				if _, err := fs.Stat(fsys, path); err != nil {
					t.Fatalf("Stat(%q): %v", path, err)
				}
			}
			for _, path := range tt.absent {
				if _, err := fs.Stat(fsys, path); !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("Stat(%q) = %v, want ErrNotExist", path, err)
				}
			}

			entries, err := fs.ReadDir(fsys, ".")
			if err != nil {
				t.Fatalf("ReadDir: %v", err)
			}
			if got := countFiles(entries); got != tt.wantReadCount {
				t.Fatalf("root file count = %d, want %d", got, tt.wantReadCount)
			}
		})
	}
}

func TestNewFS_parseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest string
		prefix   vfs.Prefix
		wantErr  string
	}{
		{
			name:     "invalid size",
			manifest: "res:/bad.png,aa/bad," + manifestMD5 + ",notanumber\n",
			prefix:   vfs.PrefixRes,
			wantErr:  "invalid size",
		},
		{
			name:     "invalid compressed size",
			manifest: "res:/bad.png,aa/bad," + manifestMD5 + ",100,notanumber\n",
			prefix:   vfs.PrefixRes,
			wantErr:  "invalid compressed size",
		},
		{
			name:     "empty logical path",
			manifest: ",aa/bad," + manifestMD5 + "\n",
			prefix:   vfs.PrefixRes,
			wantErr:  "filename and cdn filename are required",
		},
		{
			name:     "empty cdn path",
			manifest: "res:/bad.png,," + manifestMD5 + "\n",
			prefix:   vfs.PrefixRes,
			wantErr:  "filename and cdn filename are required",
		},
		{
			name: "error after valid row does not partial apply",
			manifest: "" +
				"res:/good.png,aa/good\n" +
				"res:/bad.png,aa/bad," + manifestMD5 + ",notanumber\n",
			prefix:  vfs.PrefixRes,
			wantErr: "invalid size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := vfs.New([]byte(tt.manifest), nil, vfs.WithPrefix(tt.prefix))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewFS_duplicatePaths(t *testing.T) {
	t.Parallel()

	manifest := "" +
		"res:/dup.png,aa/first," + manifestMD5A + ",100\n" +
		"res:/dup.png,aa/second," + manifestMD5B + ",200\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	info, err := fs.Stat(fsys, "dup.png")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != 100 {
		t.Fatalf("Size() = %d, want first row to win (100)", info.Size())
	}
}

func TestNewFS_emptyManifest(t *testing.T) {
	t.Parallel()

	fsys, err := vfs.New([]byte(""), nil, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	if _, err := fs.ReadDir(fsys, "."); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadDir empty manifest: %v", err)
	}
}

func TestNewFS_optionalColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest string
		wantSize int64
	}{
		{
			name:     "checksum only",
			manifest: "res:/a.png,aa/a," + manifestMD5 + "\n",
			wantSize: 0,
		},
		{
			name:     "size without compressed",
			manifest: "res:/b.png,aa/b," + manifestMD5 + ",512\n",
			wantSize: 512,
		},
		{
			name:     "blank size column skipped",
			manifest: "res:/c.png,aa/c," + manifestMD5 + ",,256\n",
			wantSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fsys, err := vfs.New([]byte(tt.manifest), nil, vfs.WithPrefix(vfs.PrefixRes))
			if err != nil {
				t.Fatalf("NewFS: %v", err)
			}

			base := strings.TrimPrefix(strings.Split(strings.TrimSpace(tt.manifest), ",")[0], "res:/")
			info, err := fs.Stat(fsys, base)
			if err != nil {
				t.Fatalf("Stat(%q): %v", base, err)
			}
			if info.Size() != tt.wantSize {
				t.Fatalf("Size() = %d, want %d", info.Size(), tt.wantSize)
			}
		})
	}
}

func countFiles(entries []fs.DirEntry) int {
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}
	return n
}
