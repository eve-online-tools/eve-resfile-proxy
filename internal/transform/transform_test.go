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
	if string(out) != "in" {
		t.Fatalf("unexpected output %q", out)
	}

	unchanged, err := engine.Transform(context.Background(), transform.Input{
		ResPath: "res:/icons/icon.png",
		Data:    []byte("raw"),
	})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if string(unchanged) != "raw" {
		t.Fatalf("expected unchanged data, got %q", unchanged)
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
	if string(out) != "ok" {
		t.Fatalf("got %q", out)
	}
}

func TestFirstMatchWins(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "upper.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntr 'a-z' 'A-Z'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	engine := mustLoadEngineFromDir(t, dir, `
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
	if string(out) != "ABC" {
		t.Fatalf("got %q", out)
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
			if _, err := transform.LoadEngine(path); err == nil {
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

	engine := mustLoadEngineFromDir(t, dir, `
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
	if string(out) != "bodysuffix" {
		t.Fatalf("got %q", out)
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

	engine := mustLoadEngineFromDir(t, dir, `
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
	if string(out) != "hello" {
		t.Fatalf("got %q", out)
	}
}

func mustLoadEngine(t *testing.T, yaml string) *transform.Engine {
	t.Helper()
	return mustLoadEngineFromDir(t, t.TempDir(), yaml)
}

func mustLoadEngineFromDir(t *testing.T, dir, yaml string) *transform.Engine {
	t.Helper()
	path := writeConfigInDir(t, dir, yaml)
	engine, err := transform.LoadEngine(path)
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
