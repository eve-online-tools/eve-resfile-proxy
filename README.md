# eve-resfile-proxy

A local HTTP proxy for EVE Online client resource files (`res:/` paths). Maps bare URL paths to CDN assets via the official resfile indexes.

## Quick start

```bash
go run ./cmd/eve-resfile-proxy
```

By default caching is **disabled** — indexes and assets are fetched from CDN on each run/request. Enable disk caching with `--cache`:

```bash
go run ./cmd/eve-resfile-proxy --cache .cache/eve-resfile-proxy
```

You can also point `--cache` at the EVE Launcher's `ResFiles` directory to reuse assets the client has already downloaded — see [Reusing the EVE client cache](#reusing-the-eve-client-cache).

Then fetch an asset:

```bash
curl -o icon.png 'http://localhost:8080/icons/64/icon64.png'
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:8080` | HTTP listen address |
| `--cache` | *(disabled)* | Cache directory for indexes and assets; omit to disable caching |
| `--build` | *(latest TQ)* | Pin to a specific client build number |
| `--platform` | *(both)* | Load indexes for one platform only: `windows` or `macos` |
| `--index-origin` | `https://binaries.eveonline.com` | Binaries CDN (indexes) |
| `--asset-origin` | `https://resources.eveonline.com` | Resources CDN (assets) |
| `--manifest` | `eveclient_TQ.json` | Client manifest filename for resolving latest build |
| `--refresh` | `false` | Force re-download of cached index files |
| `--no-index` | `false` | Disable directory listing for paths ending in `/` |
| `--transform-config` | *(disabled)* | Path to YAML transform rules file |

## Transforms

Optional file transforms run after assets are loaded from cache or CDN and before response headers (including `ETag`) are computed. The asset disk cache always stores raw CDN bytes; transforms apply on read. When `--cache` is set, transform outputs are also cached on disk under `_transformed/`.

```bash
go run ./cmd/eve-resfile-proxy --cache .cache/eve-resfile-proxy --transform-config transforms.yaml
```

Example `transforms.yaml`:

```yaml
transforms:
  - name: shader-fx
    match:
      path_prefix: "res:/shader/"
      extensions: [".fx"]
    command:
      args: ["/path/to/shader-tool"]
      timeout: 60s
      max_output_bytes: 10485760

  - name: dynamic-overlay
    stable: false
    match:
      extensions: [".json"]
    command:
      args: ["/path/to/non-deterministic-tool"]

  - name: hlsl-patch
    match:
      extensions: [".hlsl"]
    wasm:
      module: "./transforms/hlsl-patch.wasm"
      fuel: 100000000
      max_memory_bytes: 16777216
```

### Stability and transform cache

Rules are **stable** (and disk-cacheable) by default. Set `stable: false` on a rule to always run it fresh — use this for non-deterministic `command` transforms.

Transform disk caching requires `--cache`. Cached entries are keyed by `(platform, ruleName, cdnPath)`; CDN paths are content-addressed, so new asset versions get new cache paths automatically.

Transformed files are cached under `_transformed/`, this directory can be cleared to force regeneration;.

### Match fields

Rules are evaluated in order; the first match wins. All specified match fields must match (`AND`):

| Field | Semantics |
|-------|-----------|
| `stable` | When `false`, skip transform disk cache (default: cacheable) |
| `path_prefix` | `res:/` path starts with prefix |
| `path_glob` | Glob match (e.g. `res:/shader/**/*.fx`) |
| `extensions` | File extension suffix (e.g. `.fx`) |
| `filename` | Exact `res:/` path |

### `command` backend

Runs an external process for each matched asset:

- **stdin:** raw file bytes
- **stdout:** transformed bytes
- **env:** `RES_PATH`, `CDN_PATH`, `EVE_PLATFORM`, `RULE_NAME`

`command` has full host access — only use with trusted configs and scripts.

### `wasm` backend

Runs a WebAssembly module in-process via [wazero](https://wazero.io). Sandboxed (no filesystem, network, or subprocess). See [`examples/transform-wasm/README.md`](examples/transform-wasm/README.md) for the module ABI.

Each rule must specify exactly one of `command` or `wasm`.

## Directory listing

By default, GET requests to paths ending in `/` return an HTML directory index (nginx/apache style) derived from the loaded resfile index. Subdirectories are listed first, then files, sorted alphabetically.

```bash
go run ./cmd/eve-resfile-proxy
curl 'http://localhost:8080/icons/64/'
```

Pass `--no-index` to disable listing; trailing-slash paths then fall through to normal lookup and typically return 404.

Directory rows include file-type icons from [vscode-icons](https://github.com/vscode-icons/vscode-icons) (v12.15.0). Icon assets are licensed under [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/); see [`service/middleware/index/icons/ATTRIBUTION.md`](service/middleware/index/icons/ATTRIBUTION.md).

## Platform lookup

Indexes are loaded per platform at startup. At request time, `?platform=` sets preference with cascade fallback:

| `?platform=` | Search order |
|--------------|--------------|
| *(default)* | windows → macos |
| `windows` | windows → macos |
| `macos` | macos → windows |

Within each platform, the **global** resfile index wins over the **OS-specific** overlay on key collisions.

`--platform` at startup and `?platform=` at request time are independent. If only Windows indexes are loaded, macOS-only paths will not resolve.

## Response headers

Asset responses include:

| Header | Description |
|--------|-------------|
| `Content-Type` | Derived from file extension |
| `Cache-Control` | `public, max-age=3600` |
| `ETag` | SHA-256 of response body |
| `Last-Modified` | Disk cache file mtime when served from cache; omitted on CDN fetch |
| `X-Cache-Status` | `HIT` (disk cache) or `MISS` (CDN fetch) |
| `X-Eve-Build` | Loaded client build number |
| `X-Eve-Platform` | Platform that resolved the asset (`windows` or `macos`) |

Conditional requests are supported via `If-None-Match` and `If-Modified-Since` (304 when validators match).

Health endpoints: `GET /healthz`, `GET /livez` → `200 ok`.

## Architecture

```
cmd/eve-resfile-proxy/     CLI flags, signal handling
service/                   Service lifecycle, index load, HTTP server
  assetcache/              Optional on-disk asset cache
  handler/                 Terminal asset response handler
  middleware/              Request pipeline (heartbeat, resolve, load, conditional, …)
internal/                  Index loader, fetch client, lookup
```

Request pipeline (outer → inner):

```
heartbeat → getonly → index → resolve → load → transform → conditional → handler
```

`service.New` accepts functional options: `WithCache`, `WithFetch`, `WithTransformEngine`, `WithMiddleware`.

## Index reference

### Build indexes

| Platform | URL |
|----------|-----|
| Windows | `https://binaries.eveonline.com/eveonline_<build>.txt` |
| macOS | `https://binaries.eveonline.com/eveonlinemacOS_<build>.txt` |

Build number from `eveclient_TQ.json` unless `--build` is set.

### Resfile index logical paths

| Role | Windows | macOS |
|------|---------|-------|
| Global | `app:/resfileindex.txt` | `app:/EVE.app/Contents/Resources/build/resfileindex.txt` |
| OS-specific | `app:/resfileindex_Windows.txt` | `app:/EVE.app/Contents/Resources/build/resfileindex_macOS.txt` |

## Cache layout

When `--cache` is set, one directory holds both index metadata and asset bytes:

```
<cache>/
  _transformed/               # transformed asset bytes (when --transform-config is set)
    windows/
      shader-fx/
        7d/7d87a0a3a100cf9f_00b8308223bd89d4db39914d2c6488a3
    macos/
      hlsl-patch/
        a9/a9d1721dd5cc6d54_e6bbb2df307e5a9527159a4c971034b5
  <build>/                    # index metadata (per client build)
    windows/
      build-index.txt
      resfileindex-global.txt
      resfileindex-os.txt
      platform-merged.json
    macos/
      ...
    meta.json
  <cdnPath>                   # asset bytes, CDN hash layout
    7d/7d87a0a3a100cf9f_00b8308223bd89d4db39914d2c6488a3
    a9/a9d1721dd5cc6d54_e6bbb2df307e5a9527159a4c971034b5
    ...
```

Asset paths mirror the CDN: resfile index entries map `res:/…` logical paths to `<prefix>/<hash>_<checksum>` files under the cache root (first two hex digits as the subdirectory).

### Reusing the EVE client cache

The EVE Launcher downloads resource files into a shared **ResFiles** directory. Its layout matches the proxy asset cache exactly — each CDN path from the resfile index is a file at `<cache>/<cdnPath>`.

If you have your launcher configured (you can see this in **Settings → EVE Online → Shared cache location**) to e.g. `E:\EVE Online`, you can use `--cache E:\EVE Online\ResFiles\` to re-use the launcher/client cache

Assets present on disk are served as cache `HIT`s (`X-Cache-Status: HIT`); missing assets are fetched from the CDN and written into the same tree, so the launcher and proxy share downloads.

**Notes:**

- `--cache` also stores index metadata under `<cache>/<build>/` and transform outputs under `<cache>/_transformed/`. Build directories (numeric, e.g. `3141592`) sit alongside the hex prefix folders (`00`–`ff`) and the launcher's own entries (such as `bundles/`). This is harmless but means the proxy adds a few small text/JSON files per build.
- Pin `--build` to match your installed client if you want index metadata to align with the game version you have cached.
- Use `--refresh` to force re-download of index files when a new client build ships.

## License

EVE Online and related assets are property of Fenris Creations.

Third-party assets bundled with this project (such as directory listing icons) are licensed separately; see the attribution files alongside those assets.
