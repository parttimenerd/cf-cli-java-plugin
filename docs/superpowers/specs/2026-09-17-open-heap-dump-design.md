# Design: `--open` flag for heap-dump

**Date:** 2026-09-17  
**Branch:** heap-dump-compress-redact  
**Status:** Approved

## Summary

Add `--open` to the `heap-dump` command. After the final local file is saved (including any redaction or compression), the plugin spins up a temporary single-serve HTTP server, then opens the browser to the hprof-analyzer web app with a `?file=` URL pointing at that server. The server shuts down after the browser fetches the file once.

The hprof-analyzer web app (`parttimenerd/hprof-analyzer`) is being updated in parallel to support `?file=URL` loading.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--open` | bool | false | After download, open the heap dump in the hprof-analyzer web app |
| `--open-url` | string | `https://parttimenerd.github.io/hprof-analyzer` | Base URL of the hprof-analyzer web instance to open |

**Validation rules:**
- `--open` is only valid with `heap-dump`; all other commands error at parse time
- `--open-url` implies `--open`
- `--open` + `--no-download` errors at parse time (no local file to serve)
- `--open` is compatible with `--redact`, `--redact-complete`, `--compress` — always opens the final output file

## Options struct changes

```go
// in Options struct
Open    bool
OpenURL string
```

`OpenURL` defaults to `"https://parttimenerd.github.io/hprof-analyzer"` always (set at options-parse time, not conditionally on `--open`), so `--open-url` alone works without requiring `--open` to be explicitly set.

## Execution flow

After `localFileFullPath` is finalized (post-download, post-redact/compress):

1. Bind `net/http` listener on `:0` (OS assigns free port)
2. Register a single handler that:
   - Sets `Access-Control-Allow-Origin: *` (cross-origin fetch from the web app)
   - Sets `Content-Type: application/octet-stream`
   - Streams the file
   - Signals a `done` channel after the response is written
3. Open browser to `{OpenURL}/?file=http://localhost:{PORT}/{basename}`
   - macOS: `open <url>`
   - Linux: `xdg-open <url>`
   - Windows: `start <url>`
4. Block until `done` fires, then shut down the server and return

## New file: `open.go`

Two functions:

```go
// serveFileOnce binds a random port, serves path exactly once, returns the
// port and a channel that closes when the request completes.
func serveFileOnce(path string) (port int, done <-chan struct{}, err error)

// openBrowser opens url in the default system browser.
// If the browser cannot be launched, it prints the URL to stdout instead.
func openBrowser(url string) error
```

## Dry-run behaviour

Print the URL that would be opened with a `PORT` placeholder instead of a real port:

```
Would open: https://parttimenerd.github.io/hprof-analyzer/?file=http://localhost:PORT/sapmachine21-heapdump-abc123.hprof
```

## Error handling

| Scenario | Behaviour |
|----------|-----------|
| Port binding fails | Return error, abort command |
| Browser launch fails | Print URL to stdout, continue blocking for fetch |
| User Ctrl+Cs before fetch | Normal signal handling; plugin exits, server stops |
| `--open` + `--no-download` | Error at parse time |
| `--open` on non-heap-dump command | Error at parse time |

## Dependencies

No new external dependencies. Uses `net/http`, `os/exec`, `runtime` from stdlib.
