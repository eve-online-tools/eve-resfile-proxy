# eve-resfile-proxy

A local HTTP proxy for EVE Online client resource files. Maps URL paths to CDN assets via the official resfile indexes.

## Quick start

```bash
go run ./cmd/eve-resfile-proxy --config example-config.yaml
```

Or with flags only:

```bash
go run ./cmd/eve-resfile-proxy
```

By default caching is **disabled** — indexes and assets are fetched from CDN on each run/request. Enable disk caching with `--cache` or a `cache:` entry in the config file:

```bash
go run ./cmd/eve-resfile-proxy --cache .cache/eve-resfile-proxy
```

You can also point `--cache` at the EVE Launcher's `ResFiles` directory to reuse assets the client has already downloaded — see [Reusing the EVE client cache](#reusing-the-eve-client-cache).

Then fetch an asset:

```bash
curl -o icon.png 'http://localhost:8080/icons/64/icon64.png'
```

## Configuration

Settings can come from a YAML config file, CLI flags, or both. When both are used, flags override values loaded from the file.

```bash
go run ./cmd/eve-resfile-proxy --config example-config.yaml
```

Example `example-config.yaml`:

```yaml
server: tranquility
addr: ":8080"

# Optional: pin to a specific client build (omit for latest TQ build).
# build: "1234567"

# Optional: load indexes for specific platforms only.
# platforms:
#   - windows
#   - macos

cache: .cache

# debug: false
# no_index: false

# Optional: expose the full app and res trees under /app/ and /res/.
# full_tree: false

aliases:
  - alias: favicon.ico
    target: ui/texture/icons/icons111_07.png

transform_limits:
  max_output_bytes: 134217728   # 128 MiB (default)
  max_memory_bytes: 134217728   # WASM guest memory cap (default)
  fuel: 100000000               # WASM execution budget (default)

transforms: []
```

Relative paths in config (for example WASM module paths) resolve against the config file directory.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | *(none)* | Path to YAML config file |
| `--server` | `tranquility` | EVE server name (`tranquility`, `singularity`, …) |
| `--addr` | `:8080` | HTTP listen address |
| `--cache` | *(disabled)* | Cache directory for indexes and assets |
| `--build` | *(latest)* | Pin to a specific client build number |
| `--platforms` | *(all available)* | Comma-separated platforms to load (`windows`, `macOS`) |
| `--full-tree` | `false` | Expose full app and res filesystem trees |
| `--debug` | `false` | Enable debug logging |
| `--no-index` | `false` | Disable directory listing for paths ending in `/` |
| `--no-cors` | `false` | Disable CORS headers (enabled by default) |

## URL paths

By default, resource files are served from the URL root:

```bash
curl 'http://localhost:8080/icons/64/icon64.png'
```

With `full_tree: true`, the binary distribution is mounted under `/app/<platform>/` and resource files under `/res/`:

```bash
curl 'http://localhost:8080/res/icons/64/icon64.png'
curl 'http://localhost:8080/app/windows/resfileindex.txt'
```

Aliases and transform match paths use the same logical paths exposed by the HTTP server (without a leading slash).

## Aliases

Optional path and extension aliases are applied before transforms. Path aliases map a virtual path onto an underlying path (longest matching `alias` wins). Extension aliases swap file extensions within a `match` scope (same fields as transforms).

**File alias** — single path, no `match`, no trailing slash:

```yaml
aliases:
  - alias: favicon.ico
    target: ui/texture/icons/icons111_07.png
```

**Directory alias** — trailing slash required on **both** `alias` and `target`:

```yaml
aliases:
  - alias: ui.base64/
    target: ui/
  - alias: legacy/icons/
    target: icons/
```

**Extension alias** — include `match`; `alias` and `target` are extensions:

```yaml
aliases:
  - alias: .webm
    target: .png
    match:
      path_prefix: ui/textures/icons/

  - alias: .webm
    target: .png
    match:
      path_prefix: ui/
      recursive: true

  - alias: .webm
    target: .png
    match: {}   # any path, any depth
```

Extension rules stack with path aliases (for example, alias `graphics/effect.vulkan/` with target `graphics/effect.dx11/` and swap `.gr2` to `.cmf` under the vulkan path). List order in YAML does not affect execution.

Virtual intermediate directories are created as needed for path aliases.

Aliases appear in directory listings and participate in glob matching, so aliased paths behave like real files in the index.

## Transforms

Optional file transforms run when assets are read from the underlying filesystem, after cache/CDN fetch and before response headers (including `ETag`) are computed. The asset disk cache always stores raw CDN bytes; transforms apply on read. When caching is enabled, transform outputs are also cached on disk under `_transformed/`.

```yaml
transform_limits:
  max_output_bytes: 10485760
  max_memory_bytes: 16777216
  fuel: 100000000

transforms:
  - name: shader-fx
    match:
      path_prefix: shader/
      extensions: [".fx"]
    command:
      args: ["/path/to/shader-tool"]
      timeout: 60s

  - name: hlsl-patch
    match:
      extensions: [".hlsl"]
    wasm:
      module: "./transforms/hlsl-patch.wasm"
      export: transform
```

### Transform cache

Transform disk caching requires a cache directory. Cached entries are keyed by `(ruleName, cdnPath)`; CDN paths are content-addressed, so new asset versions get new cache paths automatically.

Transformed files are stored under `_transformed/<ruleName>/<cdnPath>` with an adjacent `.md5sum` sidecar. Delete `_transformed/` to force regeneration.

### Match fields

Rules are evaluated in order; the first match wins. All specified match fields must match (`AND`):

| Field | Semantics |
|-------|-----------|
| `path_prefix` | Path starts with prefix |
| `path_glob` | Glob match (e.g. `shader/**/*.fx`) |
| `extensions` | File extension suffix (e.g. `.fx`) |
| `filename` | Exact path |

### Global limits (`transform_limits`)

| Field | Default | Description |
|-------|---------|-------------|
| `max_output_bytes` | 128 MiB | Maximum transform output size (command and WASM) |
| `max_memory_bytes` | 128 MiB | Maximum WASM guest memory |
| `fuel` | 100000000 | WASM execution budget; maps to a wall-clock timeout on the transform call |

### `command` backend

Runs an external process for each matched asset:

- **stdin:** raw file bytes
- **stdout:** transformed bytes
- **env:** `RES_PATH`, `CDN_PATH`, `RULE_NAME`

`command` has full host access — only use with trusted configs and scripts.

### `wasm` backend

Runs a WebAssembly module in-process via [wazero](https://wazero.io). Sandboxed (no filesystem, network, or subprocess). Module paths are relative to the config file directory unless absolute.

Guest ABI (see also [`vfs/transform/wasm_abi.go`](vfs/transform/wasm_abi.go)):

- Export `transform(in_ptr, in_len, out_ptr, out_max) -> i32`
- Optional imports `env.get_res_path` and `env.get_cdn_path`

The fuel-derived timeout applies only to the WASM `transform` call itself, not module instantiation or memory setup.

Each rule must specify exactly one of `command` or `wasm`.

## Directory listing

By default, GET requests to paths ending in `/` return an HTML directory index (nginx/apache style) derived from the loaded resfile index. Subdirectories are listed first, then files, sorted alphabetically.

```bash
go run ./cmd/eve-resfile-proxy --config example-config.yaml
curl 'http://localhost:8080/icons/64/'
```

Pass `--no-index` or set `no_index: true` to disable listing; trailing-slash paths then fall through to normal lookup and typically return 404.

Directory rows include file-type icons from [vscode-icons](https://github.com/vscode-icons/vscode-icons) (v12.15.0). Icon assets are licensed under [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/); see [`service/http/middleware/index/icons/ATTRIBUTION.md`](service/http/middleware/index/icons/ATTRIBUTION.md).

## CORS

By default, responses include permissive CORS headers (`Access-Control-Allow-Origin: *`) so assets can be fetched cross-origin from a browser, and `OPTIONS` preflight requests are answered directly. Pass `--no-cors` or set `cors: false` to disable.

## Range requests

Responses advertise `Accept-Ranges: bytes` and honor HTTP `Range` requests (partial `206`, `416` for unsatisfiable ranges, and `If-Range`). This is handled by the standard library, so conditional and multipart-range semantics come for free.

Note: ranges are served from the fully-resolved asset held in memory (assets are checksum-verified and may be transformed as a whole), so a ranged request still fetches the complete asset from the CDN — it saves proxy→client bytes, not proxy→CDN bytes.

## Platform loading

Indexes are loaded per platform at startup. The set of platforms is the intersection of:

1. Platforms available for the selected server/build (from `eveclient_*.json`)
2. Platforms configured via `platforms:` / `--platforms` (if set)

For each loaded platform, the global resfile index is merged with the OS-specific overlay; earlier mounts win on key collisions.

If only Windows indexes are loaded, macOS-only paths will not resolve.

## Build updates

When no build is pinned (`build` omitted / `--build` unset), the service polls for a new client build every five minutes and reloads indexes automatically. Pin `build` to disable auto-updates and keep a fixed client version.

## Response headers

Asset responses include:

| Header | Description |
|--------|-------------|
| `Content-Type` | Derived from file extension |
| `Cache-Control` | `public, max-age=3600` |
| `ETag` | MD5 of response body |
| `Last-Modified` | Disk cache file mtime when served from cache; omitted on CDN fetch |
| `X-Cache-Status` | `HIT` (disk cache) or `MISS` (CDN fetch) |
| `X-Eve-Build` | Loaded client build number |

Conditional requests are supported via `If-None-Match` and `If-Modified-Since` (304 when validators match).

Health endpoints: `GET /healthz`, `GET /livez` → `200 ok`.

## Architecture

```
cmd/eve-resfile-proxy/     CLI, YAML config, signal handling
service/                   Service lifecycle, manifest load, HTTP server
  manifest/                Index loading and platform mux
  http/                    HTTP server, middleware, handlers
cache/                     Cache interface and disk backend
vfs/                       Virtual filesystem layers
  fetch/                   CDN fetch and cache-through
  mux/                     Multi-mount filesystem overlay
  alias/                   Path and extension alias rules
  transform/               Read-time file transforms
```

Filesystem layers (inner → outer):

```
manifest (CDN-backed resfile index)
  → alias (optional path and extension aliases)
  → transform (optional read-time transforms)
```

HTTP middleware pipeline (outer → inner):

```
heartbeat (/healthz, /livez)
  → method (GET/HEAD only)
  → index (HTML directory listing for trailing-slash paths)
  → load (map URL to vfs path, stat, cache HIT/MISS)
  → conditional (ETag, 304 Not Modified)
  → handler.Respond (read bytes, write asset response)
```

Aliases and transforms run inside the vfs when `load` / `handler` read from the filesystem — they are not separate HTTP middleware.

## Index reference

### Build indexes

| Platform | URL |
|----------|-----|
| Windows | `https://binaries.eveonline.com/eveonline_<build>.txt` |
| macOS | `https://binaries.eveonline.com/eveonlinemacOS_<build>.txt` |

Build number comes from `eveclient_TQ.json` (or `eveclient_SISI.json` for Singularity) unless `build` is set.

### Resfile index logical paths

| Role | Windows | macOS |
|------|---------|-------|
| Global | `app:/resfileindex.txt` | `app:/EVE.app/Contents/Resources/build/resfileindex.txt` |
| OS-specific | `app:/resfileindex_Windows.txt` | `app:/EVE.app/Contents/Resources/build/resfileindex_macOS.txt` |

## Cache layout

When caching is enabled, one directory holds fetched bytes keyed by CDN path:

```
<cache>/
  _transformed/               # transformed asset bytes (when transforms are configured)
    shader-fx/
      7d/7d87a0a3a100cf9f_00b8308223bd89d4db39914d2c6488a3
      7d/7d87a0a3a100cf9f_00b8308223bd89d4db39914d2c6488a3.md5sum
  eveonline_<build>.txt       # cached build index (example)
  <cdnPath>                   # asset bytes, CDN hash layout
    7d/7d87a0a3a100cf9f_00b8308223bd89d4db39914d2c6488a3
    a9/a9d1721dd5cc6d54_e6bbb2df307e5a9527159a4c971034b5
    ...
```

Asset paths mirror the CDN: resfile index entries map logical paths to `<prefix>/<hash>_<checksum>` files under the cache root (first two hex digits as the subdirectory).

### Reusing the EVE client cache

The EVE Launcher downloads resource files into a shared **ResFiles** directory. Its layout matches the proxy asset cache exactly — each CDN path from the resfile index is a file at `<cache>/<cdnPath>`.

If your launcher is configured (see **Settings → EVE Online → Shared cache location**) to e.g. `E:\EVE Online`, you can use `--cache "E:\EVE Online\ResFiles"` to reuse the launcher/client cache.

Assets present on disk are served as cache `HIT`s (`X-Cache-Status: HIT`); missing assets are fetched from the CDN and written into the same tree, so the launcher and proxy share downloads.

**Notes:**

- Caching also stores fetched index files (for example `eveonline_<build>.txt`) and transform outputs under `_transformed/`. These sit alongside the launcher's hex prefix folders (`00`–`ff`) and entries such as `bundles/`.
- Pin `build` to match your installed client if you want indexes to align with the game version you have cached.

## License

EVE Online and related assets are property of Fenris Creations.

Third-party assets bundled with this project (such as directory listing icons) are licensed separately; see the attribution files alongside those assets.
