package transform_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
	"github.com/eve-online-tools/eve-resfile-proxy/internal/transform"
)

//go:generate go run github.com/wasilibs/go-wabt/cmd/wat2wasm@v0.0.0-20240502051220-face6b18f58d -o testdata/copy.wasm testdata/copy.wat

func TestMatcherPathPrefix(t *testing.T) {
	engine := mustLoadEngine(t, `
transforms:
  - name: prefix
    match:
      path_prefix: "res:/shader/"
      extensions: [".fx"]
    command:
      args: ["cat"]
`)

	out, err := engine.Transform(context.Background(), transform.Input{
		ResPath: "res:/shader/foo.fx",
		Data:    []byte("in"),
	})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if string(out.Data) != "in" {
		t.Fatalf("unexpected output %q", out.Data)
	}

	unchanged, err := engine.Transform(context.Background(), transform.Input{
		ResPath: "res:/icons/icon.png",
		Data:    []byte("raw"),
	})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if string(unchanged.Data) != "raw" {
		t.Fatalf("expected unchanged data, got %q", unchanged.Data)
	}
}

func TestMatcherFilename(t *testing.T) {
	engine := mustLoadEngine(t, `
transforms:
  - name: exact
    match:
      filename: "res:/interface/special.yaml"
    command:
      args: ["cat"]
`)

	_, err := engine.Transform(context.Background(), transform.Input{
		ResPath: "res:/interface/other.yaml",
		Data:    []byte("x"),
	})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
}

func TestMatcherPathGlob(t *testing.T) {
	engine := mustLoadEngine(t, `
transforms:
  - name: glob
    match:
      path_glob: "res:/shader/**/*.fx"
    command:
      args: ["cat"]
`)

	out, err := engine.Transform(context.Background(), transform.Input{
		ResPath: "res:/shader/deep/file.fx",
		Data:    []byte("ok"),
	})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if string(out.Data) != "ok" {
		t.Fatalf("got %q", out.Data)
	}
}

func TestFirstMatchWins(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "upper.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntr 'a-z' 'A-Z'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	engine := mustLoadEngineFromDir(t, dir, "", `
transforms:
  - name: first
    match:
      extensions: [".txt"]
    command:
      args: ["`+script+`"]
  - name: second
    match:
      extensions: [".txt"]
    command:
      args: ["false"]
`)

	out, err := engine.Transform(context.Background(), transform.Input{
		ResPath: "res:/foo.txt",
		Data:    []byte("abc"),
	})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if string(out.Data) != "ABC" {
		t.Fatalf("got %q", out.Data)
	}
}

func TestConfigValidation(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "missing backend",
			yaml: `transforms:
  - name: bad
    match:
      extensions: [".fx"]
`,
		},
		{
			name: "both backends",
			yaml: `transforms:
  - name: bad
    match:
      extensions: [".fx"]
    command:
      args: ["true"]
    wasm:
      module: "./x.wasm"
`,
		},
		{
			name: "missing match",
			yaml: `transforms:
  - name: bad
    command:
      args: ["true"]
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfig(t, tc.yaml)
			if _, err := transform.LoadEngine(path, ""); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCommandTransform(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "append.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat; echo -n suffix\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	engine := mustLoadEngineFromDir(t, dir, "", `
transforms:
  - name: append
    match:
      extensions: [".dat"]
    command:
      args: ["`+script+`"]
`)

	out, err := engine.Transform(context.Background(), transform.Input{
		ResPath:  "res:/data/file.dat",
		CDNPath:  "ab/cd",
		Platform: index.PlatformWindows,
		Data:     []byte("body"),
	})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if string(out.Data) != "bodysuffix" {
		t.Fatalf("got %q", out.Data)
	}
}

func TestCommandTransformFailure(t *testing.T) {
	engine := mustLoadEngine(t, `
transforms:
  - name: fail
    match:
      extensions: [".dat"]
    command:
      args: ["false"]
`)

	_, err := engine.Transform(context.Background(), transform.Input{
		ResPath: "res:/data/file.dat",
		Data:    []byte("body"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWasmTransformCopy(t *testing.T) {
	wasmPath := filepath.Join("testdata", "copy.wasm")
	if _, err := os.Stat(wasmPath); err != nil {
		t.Fatalf("missing %s: run go generate ./internal/transform", wasmPath)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "copy.wasm"), mustReadFile(t, wasmPath), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := mustLoadEngineFromDir(t, dir, "", `
transforms:
  - name: copy
    match:
      extensions: [".txt"]
    wasm:
      module: "./copy.wasm"
`)

	out, err := engine.Transform(context.Background(), transform.Input{
		ResPath:  "res:/foo.txt",
		CDNPath:  "aa/bb",
		Platform: index.PlatformMacOS,
		Data:     []byte("hello"),
	})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if string(out.Data) != "hello" {
		t.Fatalf("got %q", out.Data)
	}
}

func TestDiskCacheHitOnSecondTransform(t *testing.T) {
	dir := t.TempDir()
	cacheDir := t.TempDir()
	script := filepath.Join(dir, "upper.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntr 'a-z' 'A-Z'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	engine := mustLoadEngineFromDir(t, dir, cacheDir, `
transforms:
  - name: upper
    match:
      extensions: [".txt"]
    command:
      args: ["`+script+`"]
`)

	in := transform.Input{
		ResPath:  "res:/foo.txt",
		CDNPath:  "aa/bb",
		Platform: index.PlatformWindows,
		Data:     []byte("abc"),
	}

	first, err := engine.Transform(context.Background(), in)
	if err != nil {
		t.Fatalf("first transform: %v", err)
	}
	if first.FromCache {
		t.Fatal("expected cache miss on first transform")
	}
	if string(first.Data) != "ABC" {
		t.Fatalf("first data = %q", first.Data)
	}

	second, err := engine.Transform(context.Background(), in)
	if err != nil {
		t.Fatalf("second transform: %v", err)
	}
	if !second.FromCache {
		t.Fatal("expected cache hit on second transform")
	}
	if string(second.Data) != "ABC" {
		t.Fatalf("second data = %q", second.Data)
	}
}

func TestStableFalseSkipsDiskCache(t *testing.T) {
	dir := t.TempDir()
	cacheDir := t.TempDir()
	countFile := filepath.Join(dir, "count.txt")
	script := filepath.Join(dir, "count.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 1 >> "+countFile+"\ntr 'a-z' 'A-Z'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	engine := mustLoadEngineFromDir(t, dir, cacheDir, `
transforms:
  - name: upper
    stable: false
    match:
      extensions: [".txt"]
    command:
      args: ["`+script+`"]
`)

	in := transform.Input{
		ResPath:  "res:/foo.txt",
		CDNPath:  "aa/bb",
		Platform: index.PlatformWindows,
		Data:     []byte("abc"),
	}

	for i := 0; i < 2; i++ {
		result, err := engine.Transform(context.Background(), in)
		if err != nil {
			t.Fatalf("transform %d: %v", i+1, err)
		}
		if result.FromCache {
			t.Fatalf("transform %d: unexpected cache hit", i+1)
		}
		if string(result.Data) != "ABC" {
			t.Fatalf("transform %d data = %q", i+1, result.Data)
		}
	}

	count, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatalf("read count: %v", err)
	}
	if string(count) != "1\n1\n" {
		t.Fatalf("command invocations = %q", count)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "_transformed")); !os.IsNotExist(err) {
		t.Fatal("expected no transform cache directory")
	}
}

func TestNoDiskCacheWhenCacheRootEmpty(t *testing.T) {
	dir := t.TempDir()
	countFile := filepath.Join(dir, "count.txt")
	script := filepath.Join(dir, "count.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 1 >> "+countFile+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	engine := mustLoadEngineFromDir(t, dir, "", `
transforms:
  - name: cat
    match:
      extensions: [".txt"]
    command:
      args: ["`+script+`"]
`)

	in := transform.Input{
		ResPath:  "res:/foo.txt",
		CDNPath:  "aa/bb",
		Platform: index.PlatformWindows,
		Data:     []byte("abc"),
	}

	for i := 0; i < 2; i++ {
		result, err := engine.Transform(context.Background(), in)
		if err != nil {
			t.Fatalf("transform %d: %v", i+1, err)
		}
		if result.FromCache {
			t.Fatalf("transform %d: unexpected cache hit", i+1)
		}
	}

	count, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatalf("read count: %v", err)
	}
	if string(count) != "1\n1\n" {
		t.Fatalf("command invocations = %q", count)
	}
}

func mustLoadEngine(t *testing.T, yaml string) *transform.Engine {
	t.Helper()
	return mustLoadEngineFromDir(t, t.TempDir(), "", yaml)
}

func mustLoadEngineFromDir(t *testing.T, dir, cacheRoot, yaml string) *transform.Engine {
	t.Helper()
	path := writeConfigInDir(t, dir, yaml)
	engine, err := transform.LoadEngine(path, cacheRoot)
	if err != nil {
		t.Fatalf("load engine: %v", err)
	}
	return engine
}

func writeConfig(t *testing.T, yaml string) string {
	t.Helper()
	return writeConfigInDir(t, t.TempDir(), yaml)
}

func writeConfigInDir(t *testing.T, dir, yaml string) string {
	t.Helper()
	path := filepath.Join(dir, "transforms.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
