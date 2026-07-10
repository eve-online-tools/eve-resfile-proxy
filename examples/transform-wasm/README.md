# WASM transform modules

WASM transforms run in-process via [wazero](https://wazero.io). They are suitable for fast, sandboxed byte manipulation. For shader compilers or other heavy tooling, use the `command` backend instead.

## Module ABI

Export a function named `transform` (override with `wasm.export` in YAML):

```text
transform(in_ptr: i32, in_len: i32, out_ptr: i32, out_max: i32) -> i32
```

- **Input** bytes are written by the host at linear memory offset `in_ptr` (`in_len` bytes).
- **Output** must be written starting at `out_ptr`, up to `out_max` bytes.
- **Return** the number of bytes written, or a negative error code.

Return `-1` when `in_len >= out_max` (output buffer too small).

### Optional host imports (`env`)

| Import | Signature | Purpose |
|--------|-----------|---------|
| `get_res_path` | `(buf_ptr i32, max_len i32) -> i32` | Write `RES_PATH` UTF-8 into guest memory; returns length or `-1` if buffer too small |
| `get_cdn_path` | `(buf_ptr i32, max_len i32) -> i32` | Write `CDN_PATH` UTF-8 into guest memory |
| `get_platform` | `() -> i32` | `0` = windows, `1` = macos |

Pass `buf_ptr = 0` to query required length without writing.

## Example

The test module is an identity transform written in [WAT](https://webassembly.github.io/spec/core/text/) (`copy.wat`). Regenerate the binary with:

```bash
go generate ./internal/transform
```

Source: [`internal/transform/testdata/copy.wat`](../../internal/transform/testdata/copy.wat)

For production modules, TinyGo and Rust (`wasm32-unknown-unknown`) are common toolchains — see below.

## YAML

```yaml
transforms:
  - name: hlsl-patch
    match:
      extensions: [".hlsl"]
    wasm:
      module: "./hlsl-patch.wasm"
      export: transform
      max_output_bytes: 1048576
      fuel: 100000000
      max_memory_bytes: 16777216
```

## Building modules

Any toolchain that emits WebAssembly 1.0 with exported memory works. Two common approaches:

### WAT (WebAssembly text)

Readable source checked into git; compile with [`wat2wasm`](https://github.com/WebAssembly/wabt) or via Go:

```bash
go run github.com/wasilibs/go-wabt/cmd/wat2wasm@latest module.wat -o module.wasm
```

This repo uses that pattern for the test copy module (`go generate ./internal/transform`).

### TinyGo

Write the transform in Go and compile to WASM:

```bash
tinygo build -o module.wasm -target=wasm -opt=2 .
```

Use `//go:wasmexport transform` and `unsafe` for linear memory access. Match the `transform(in_ptr, in_len, out_ptr, out_max) -> i32` signature.

Ensure:

1. Linear memory is exported as `memory`
2. `transform` is exported with the signature above
3. Import `env.get_res_path`, `env.get_cdn_path`, and `env.get_platform` if you use metadata (or provide stub imports)

Limits (`fuel`, `max_memory_bytes`, `max_output_bytes`) are enforced by the proxy at runtime.
