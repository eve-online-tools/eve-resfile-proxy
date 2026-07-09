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
heartbeat → getonly → resolve → load → conditional → handler
```

`service.New` accepts functional options: `WithCache`, `WithFetch`, `WithMiddleware`.

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

When `--cache` is set:

```
<cache>/
  <build>/
    windows/
      build-index.txt
      resfileindex-global.txt
      resfileindex-os.txt
      platform-merged.json
    macos/
      ...
    meta.json
  <cdnPath>/          # asset bytes, CDN hash layout
    7d/7d87a0a3...
```

## License

EVE Online and related assets are property of CCP and its licensors. This tool is for development and research use.
