# IMAGOR - PROJECT KNOWLEDGE BASE

**Generated:** 2026-01-14
**Commit:** 7c5b1e3
**Branch:** master

## OVERVIEW

High-performance image processing server and Go library using libvips. Thumbor-compatible URL syntax, drop-in replacement with 4-8x speed improvement. Supports HTTP/S3/GCS/File loaders and storages.

## STRUCTURE

```
imagor/
├── cmd/imagor/         # Binary entry point (thin wrapper)
├── config/             # Flag parsing, component assembly
│   ├── awsconfig/      # S3 credentials bridge
│   ├── gcloudconfig/   # GCS credentials bridge
│   └── vipsconfig/     # VIPS processor options
├── processor/
│   └── vipsprocessor/  # libvips image processing (HAS OWN AGENTS.md)
├── loader/
│   ├── httploader/     # Remote URL fetching
│   └── uploadloader/   # POST upload handling
├── storage/
│   ├── filestorage/    # Local filesystem
│   ├── s3storage/      # AWS S3
│   └── gcloudstorage/  # Google Cloud Storage
├── imagorpath/         # Thumbor URL parsing DSL
├── server/             # HTTP lifecycle, middleware
├── fanoutreader/       # Parallel stream reading
├── seekstream/         # Seekable stream buffer
├── metrics/            # Prometheus integration
└── testdata/           # Golden files for regression tests
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add new filter | `processor/vipsprocessor/filter.go` | Register in `processor.go` FilterMap |
| Add storage backend | `storage/` | Implement `Storage` interface |
| Add loader | `loader/` | Implement `Loader` interface |
| Modify URL syntax | `imagorpath/parse.go` | Uses regex, see `paramsRegex` |
| Change HTTP behavior | `server/server.go` | Middleware, CORS, graceful shutdown |
| Add CLI flag | `config/config.go` | Uses `peterbourgon/ff` for flags+env |
| Debug processing | `imagor.go:Do()` | Main orchestration method |

## ARCHITECTURE

### Request Lifecycle
```
Request → ResultStorage(check) → Storage(check) → Loader(fetch)
                                                      ↓
                                              Processor(transform)
                                                      ↓
                                     ResultStorage(save) ← Response
```

### Core Interfaces (imagor.go)
- `Loader`: `Get(r, key) → Blob` — fetch source images
- `Storage`: `Get/Put/Stat/Delete` — cache source OR results
- `Processor`: `Process(ctx, blob, params, loadFunc) → Blob`

### Data Flow
- `Blob` abstraction wraps streams, lazy init, type sniffing
- `fanoutreader` enables parallel reads (save + process simultaneously)
- `singleflight` deduplicates concurrent identical requests

## CONVENTIONS

### Functional Options Pattern (ALWAYS use)
```go
New(options ...Option)
WithLogger(*zap.Logger)
WithTimeout(time.Duration)
```

### Context for Hidden State
- `contextDefer(ctx, fn)` — attach cleanup (closes file handles, VIPS refs)
- Context values pass processor state (rotation, metadata)
- NEVER store request-specific data globally

### Error Handling
- Wrap vips errors via `WrapErr()` → `imagor.Error`
- Use `ErrForward` to delegate to next processor in chain
- Standard errors: `ErrNotFound`, `ErrSignatureMismatch`, `ErrExpired`

### Testing
- Golden files in `testdata/golden/` (arch-specific: `golden_arm64/`)
- Table-driven tests with `testify/assert`
- Mock with `loaderFunc`, `mapStore` closures
- Memory leak detection in vipsprocessor tests

## ANTI-PATTERNS (THIS PROJECT)

| Pattern | Why Forbidden |
|---------|---------------|
| Global state for request data | Use context |
| `panic` in request handlers | Only allowed in startup/init |
| Ignoring `Blob.Close()` | Memory leaks in VIPS |
| Bypassing `CheckResolution()` | Image bomb vulnerability |
| Direct VIPS calls outside processor | Breaks abstraction |

## UNIQUE STYLES

- **POST uploads always unsafe** — hardcoded security constraint
- **Regex-based URL parsing** — thumbor compatibility via `imagorpath`
- **jemalloc in Docker** — `LD_PRELOAD` for memory efficiency
- **libvips source build** — Dockerfile compiles vips 8.18.0
- **Golden file auto-commit** — CI updates regression images

## COMMANDS

```bash
# Development
make dev                    # -debug -imagor-unsafe -upload-loader-enable
make test                   # go test with coverage
make build                  # CGO build to bin/imagor

# Docker
make docker-dev             # Standard build
make docker-magick          # With ImageMagick support
make docker-mozjpeg         # With MozJPEG compression

# Help
./bin/imagor -h             # All CLI flags
./bin/imagor -version       # Current version
```

## NOTES

- **CGO Required**: libvips is C library, needs `CGO_CFLAGS_ALLOW=-Xpreprocessor`
- **Deprecated Flag**: `-http-loader-forward-headers` → use `-http-loader-forward-client-headers`
- **Version**: Currently 1.6.7 (see `imagor.go:25`)
- **Large Files**: `imagor.go` (991 LOC), `process.go` (736 LOC) are complexity hotspots
- **No linter config**: Uses standard `go fmt`/`go vet` only
