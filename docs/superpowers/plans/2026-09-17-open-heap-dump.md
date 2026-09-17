# `--open` Flag for heap-dump Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--open` and `--open-url` flags to `heap-dump` that spin up a single-serve local HTTP server and open the hprof-analyzer web app with `?file=http://localhost:PORT/dump.hprof` after the final local file is saved.

**Architecture:** A new `open.go` file provides two functions — `serveFileOnce` (binds `:0`, serves the file once, returns port + done channel) and `openBrowser` (cross-platform browser launch). The main plugin wiring in `cf_cli_java_plugin.go` adds two flags, two `Options` fields, parse-time validation, and a call to the open logic after `localFileFullPath` is finalized.

**Tech Stack:** Go stdlib (`net/http`, `os/exec`, `runtime`, `path/filepath`). No new dependencies.

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `open.go` | **Create** | `serveFileOnce` + `openBrowser` |
| `cf_cli_java_plugin.go` | **Modify** | Flag constants, `Options` fields, `flagDefinitions` entries, `parseOptions` wiring + validation, post-download `--open` call |

---

### Task 1: `open.go` — `serveFileOnce` and `openBrowser`

**Files:**
- Create: `open.go`

- [ ] **Step 1: Write the failing test**

Create `open_test.go`:

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeFileOnce(t *testing.T) {
	// Create a temp file with known content
	tmp := t.TempDir()
	p := filepath.Join(tmp, "test.hprof")
	if err := os.WriteFile(p, []byte("HEAP_CONTENT"), 0o600); err != nil {
		t.Fatal(err)
	}

	port, done, err := serveFileOnce(p)
	if err != nil {
		t.Fatalf("serveFileOnce: %v", err)
	}
	if port <= 0 {
		t.Fatalf("expected positive port, got %d", port)
	}

	url := fmt.Sprintf("http://localhost:%d/test.hprof", port)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("Content-Type: want application/octet-stream, got %s", ct)
	}
	if acao := resp.Header.Get("Access-Control-Allow-Origin"); acao != "*" {
		t.Errorf("Access-Control-Allow-Origin: want *, got %s", acao)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "HEAP_CONTENT" {
		t.Errorf("body: want HEAP_CONTENT, got %s", body)
	}

	// done channel must close after the request completes
	select {
	case <-done:
	default:
		t.Error("done channel not closed after successful GET")
	}
}

func TestServeFileOnce_MissingFile(t *testing.T) {
	_, _, err := serveFileOnce("/nonexistent/path/dump.hprof")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestBuildOpenURL(t *testing.T) {
	cases := []struct {
		base     string
		port     int
		filename string
		want     string
	}{
		{
			"https://parttimenerd.github.io/hprof-analyzer",
			54321,
			"myapp-heapdump-abc.hprof",
			"https://parttimenerd.github.io/hprof-analyzer/?file=http://localhost:54321/myapp-heapdump-abc.hprof",
		},
		{
			"https://parttimenerd.github.io/hprof-analyzer/",
			9000,
			"dump.hprof.gz",
			"https://parttimenerd.github.io/hprof-analyzer/?file=http://localhost:9000/dump.hprof.gz",
		},
	}
	for _, tc := range cases {
		got := buildOpenURL(tc.base, tc.port, tc.filename)
		if got != tc.want {
			t.Errorf("buildOpenURL(%q, %d, %q)\n  want %q\n   got %q", tc.base, tc.port, tc.filename, tc.want, got)
		}
	}
}

func TestBuildOpenURL_TrailingSlash(t *testing.T) {
	// base with trailing slash must not produce double slash before ?
	url := buildOpenURL("https://example.com/analyzer/", 1234, "dump.hprof")
	if strings.Contains(url, "//?" ) {
		t.Errorf("double slash before ?: %s", url)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test -run 'TestServeFileOnce|TestBuildOpenURL' ./... 2>&1
```

Expected: `undefined: serveFileOnce`, `undefined: buildOpenURL`

- [ ] **Step 3: Implement `open.go`**

```go
/*
 * Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved.
 * This file is licensed under the Apache Software License, v. 2 except as noted
 * otherwise in the LICENSE file at the root of the repository.
 */

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// serveFileOnce starts a local HTTP server on a random port that serves path
// exactly once. It returns the bound port and a channel that is closed when
// the first GET request completes. The server shuts itself down after serving.
// Returns an error if the file does not exist or the port cannot be bound.
func serveFileOnce(path string) (int, <-chan struct{}, error) {
	if _, err := os.Stat(path); err != nil {
		return 0, nil, fmt.Errorf("file not found: %w", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, fmt.Errorf("could not bind local port: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	done := make(chan struct{})
	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/"+filepath.Base(path), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, path)
		go func() {
			close(done)
			_ = srv.Shutdown(context.Background())
		}()
	})

	go func() { _ = srv.Serve(ln) }()

	return port, done, nil
}

// buildOpenURL constructs the hprof-analyzer URL with the ?file= parameter.
// base is the analyzer base URL (trailing slash optional).
func buildOpenURL(base string, port int, filename string) string {
	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/?file=http://localhost:%d/%s", base, port, filename)
}

// openBrowser opens url in the system default browser.
// If launch fails, it prints the URL to stdout so the user can open it manually.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case osWindows:
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("Opening: %s\n", url)
		return nil
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -run 'TestServeFileOnce|TestBuildOpenURL' -v ./... 2>&1
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add open.go open_test.go
git commit -m "feat: add serveFileOnce and openBrowser helpers for --open"
```

---

### Task 2: Flag constants, Options fields, flagDefinitions

**Files:**
- Modify: `cf_cli_java_plugin.go`

- [ ] **Step 1: Write the failing test**

Add to `cf_cli_java_plugin_test.go` (or create it if it doesn't exist — check with `ls *_test.go`):

```go
func TestParseOptions_Open(t *testing.T) {
	p := &JavaPlugin{}

	// --open sets Open=true and OpenURL to default
	opts, _, err := p.parseOptions([]string{"heap-dump", "myapp", "--open"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Open {
		t.Error("expected Open=true")
	}
	if opts.OpenURL != defaultOpenURL {
		t.Errorf("expected OpenURL=%q, got %q", defaultOpenURL, opts.OpenURL)
	}
}

func TestParseOptions_OpenURL_ImpliesOpen(t *testing.T) {
	p := &JavaPlugin{}
	opts, _, err := p.parseOptions([]string{"heap-dump", "myapp", "--open-url", "http://localhost:8080"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Open {
		t.Error("--open-url should imply Open=true")
	}
	if opts.OpenURL != "http://localhost:8080" {
		t.Errorf("unexpected OpenURL: %q", opts.OpenURL)
	}
}

func TestParseOptions_Open_NoDownload_Error(t *testing.T) {
	p := &JavaPlugin{}
	_, _, err := p.parseOptions([]string{"heap-dump", "myapp", "--open", "--no-download"})
	if err == nil {
		t.Fatal("expected error for --open + --no-download, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test -run 'TestParseOptions_Open' -v ./... 2>&1
```

Expected: `undefined: defaultOpenURL` or field `Open` not found in `Options`

- [ ] **Step 3: Add constants, Options fields, flagDefinitions entries**

In `cf_cli_java_plugin.go`:

**Add constants** (after `flagCompress = "compress"` on line ~42):

```go
flagOpen        = "open"
flagOpenURL     = "open-url"
defaultOpenURL  = "https://parttimenerd.github.io/hprof-analyzer"
```

**Add fields to `Options` struct** (after `Compress bool` on line ~218):

```go
Open    bool
OpenURL string
```

**Add flag definitions** (append after the `flagCompress` entry in `flagDefinitions`, before the closing `}`):

```go
{
    Name:  flagOpen,
    Usage: "open the heap dump in the hprof-analyzer web app after downloading",
    Type:  typeBool,
},
{
    Name:  flagOpenURL,
    Usage: "base URL of the hprof-analyzer instance to open (implies --open)",
    Type:  typeString,
},
```

- [ ] **Step 4: Wire up in `parseOptions`**

In the `options := &Options{...}` block (after `Compress:` line ~382), add:

```go
Open:    commandFlags.IsSet(flagOpen) || commandFlags.IsSet(flagOpenURL),
OpenURL: func() string {
    if u := commandFlags.String(flagOpenURL); u != "" {
        return u
    }
    return defaultOpenURL
}(),
```

Add validation after the existing `Redact && RedactComplete` check (after line ~389):

```go
if options.Open && options.NoDownload {
    return nil, nil, &InvalidUsageError{
        message: "Error: flag '--open' requires a local file and cannot be used with '--no-download'",
    }
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test -run 'TestParseOptions_Open' -v ./... 2>&1
```

Expected: all PASS

- [ ] **Step 6: Verify build is clean**

```bash
go build ./... 2>&1
```

Expected: no output (success)

- [ ] **Step 7: Commit**

```bash
git add cf_cli_java_plugin.go cf_cli_java_plugin_test.go
git commit -m "feat: add --open and --open-url flags to Options and flagDefinitions"
```

---

### Task 3: Call open logic after file is finalized

**Files:**
- Modify: `cf_cli_java_plugin.go` (the post-download section, ~lines 1278-1297)

- [ ] **Step 1: Write the failing test**

Add to `cf_cli_java_plugin_test.go`:

```go
func TestParseOptions_Open_DryRun_NoError(t *testing.T) {
	// --open + --dry-run is valid (no server started, just prints URL)
	p := &JavaPlugin{}
	opts, _, err := p.parseOptions([]string{"heap-dump", "myapp", "--open", "--dry-run"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Open {
		t.Error("expected Open=true")
	}
	if !opts.DryRun {
		t.Error("expected DryRun=true")
	}
}
```

- [ ] **Step 2: Run to verify it passes immediately** (it's a parse-only test, should pass after Task 2)

```bash
go test -run 'TestParseOptions_Open_DryRun_NoError' -v ./... 2>&1
```

Expected: PASS

- [ ] **Step 3: Add the open call after file finalization**

Locate the block in `cf_cli_java_plugin.go` that ends with redaction (around line 1296). The current structure is:

```go
fmt.Println(utils.ToSentenceCase(command.FileLabel) + " file saved to: " + localFileFullPath)

if command.Name == cmdHeapDump && (options.Redact || options.RedactComplete) {
    // ... redaction sets finalPath ...
    fmt.Println("Redacted heap dump saved to: " + finalPath)
}
```

Replace that block with:

```go
fmt.Println(utils.ToSentenceCase(command.FileLabel) + " file saved to: " + localFileFullPath)

finalLocalPath := localFileFullPath
if command.Name == cmdHeapDump && (options.Redact || options.RedactComplete) {
    mode := "lean"
    if options.RedactComplete {
        mode = "complete"
    }
    redactBin, rerr := ensureHprofRedact()
    if rerr != nil {
        return "", fmt.Errorf("hprof-redact unavailable: %w", rerr)
    }
    localIsGz := strings.HasSuffix(localFileFullPath, ".hprof.gz")
    finalPath, rerr := pipeHeapDumpThroughRedact(redactBin, localFileFullPath, mode, options.Compress || localIsGz)
    if rerr != nil {
        return "", fmt.Errorf("redaction failed (unredacted file at %s): %w", localFileFullPath, rerr)
    }
    fmt.Println("Redacted heap dump saved to: " + finalPath)
    finalLocalPath = finalPath
}

if command.Name == cmdHeapDump && options.Open {
    if options.DryRun {
        fmt.Printf("Would open: %s\n", buildOpenURL(options.OpenURL, 0, filepath.Base(finalLocalPath)))
    } else {
        port, done, serveErr := serveFileOnce(finalLocalPath)
        if serveErr != nil {
            return "", fmt.Errorf("could not start local file server: %w", serveErr)
        }
        openURL := buildOpenURL(options.OpenURL, port, filepath.Base(finalLocalPath))
        fmt.Printf("Opening heap dump in browser: %s\n", openURL)
        _ = openBrowser(openURL)
        <-done
    }
}
```

Note: the dry-run `PORT` placeholder uses `0` — `buildOpenURL` with port `0` produces `localhost:0` which clearly signals a placeholder. Update `buildOpenURL` in `open.go` to produce `PORT` when port is 0:

```go
func buildOpenURL(base string, port int, filename string) string {
	base = strings.TrimRight(base, "/")
	portStr := fmt.Sprintf("%d", port)
	if port == 0 {
		portStr = "PORT"
	}
	return fmt.Sprintf("%s/?file=http://localhost:%s/%s", base, portStr, filename)
}
```

Also update the test in `open_test.go` for `buildOpenURL` — add a dry-run case:

```go
{
    "https://parttimenerd.github.io/hprof-analyzer",
    0,
    "dump.hprof",
    "https://parttimenerd.github.io/hprof-analyzer/?file=http://localhost:PORT/dump.hprof",
},
```

Also add `"path/filepath"` to the import in `cf_cli_java_plugin.go` if not already present.

- [ ] **Step 4: Verify build is clean**

```bash
go build ./... 2>&1
```

Expected: no output

- [ ] **Step 5: Run all tests**

```bash
go test -v ./... 2>&1
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add cf_cli_java_plugin.go open.go open_test.go
git commit -m "feat: invoke serveFileOnce+openBrowser after heap-dump download for --open"
```

---

### Task 4: Build, install, and smoke-test against a live CF app

**Files:** none (verification only)

- [ ] **Step 1: Build and reinstall the plugin**

```bash
go build -o build/cf-cli-java-plugin . && cf install-plugin -f build/cf-cli-java-plugin
```

Expected: `Plugin java X.X.X successfully installed.`

- [ ] **Step 2: Dry-run smoke test**

```bash
cf java heap-dump sapmachine21 --open --dry-run
```

Expected output contains:
```
Would open: https://parttimenerd.github.io/hprof-analyzer/?file=http://localhost:PORT/sapmachine21-heapdump-<uuid>.hprof
```

- [ ] **Step 3: Dry-run with custom URL**

```bash
cf java heap-dump sapmachine21 --open-url http://localhost:8080 --dry-run
```

Expected: URL starts with `http://localhost:8080/?file=http://localhost:PORT/`

- [ ] **Step 4: Dry-run `--open` + `--no-download` errors**

```bash
cf java heap-dump sapmachine21 --open --no-download --dry-run 2>&1; echo "EXIT: $?"
```

Expected: error message `'--open' requires a local file and cannot be used with '--no-download'`, exit 1

- [ ] **Step 5: Dry-run `--open` + `--compress`**

```bash
cf java heap-dump sapmachine21 --open --compress --dry-run
```

Expected: URL path ends with `.hprof.gz`

- [ ] **Step 6: Dry-run `--open` + `--redact`**

```bash
cf java heap-dump sapmachine21 --open --redact --dry-run
```

Expected: URL path ends with `-redacted.hprof`

- [ ] **Step 7: Commit smoke-test confirmation (no code change needed)**

If all above pass with no code changes required:
```bash
git commit --allow-empty -m "test: smoke-tested --open flag against live CF app"
```

---

## Self-Review Notes

- **Spec coverage:** All requirements covered — `--open`, `--open-url`, `--open-url` implies `--open`, `--open` + `--no-download` error, compatible with `--redact`/`--compress`, dry-run prints placeholder URL, serve-once lifecycle, cross-platform `openBrowser`.
- **Placeholder scan:** All code blocks are complete and runnable.
- **Type consistency:** `serveFileOnce` returns `(int, <-chan struct{}, error)` in Task 1 and is called identically in Task 3. `buildOpenURL(base string, port int, filename string) string` is defined in Task 1 and used the same way in Task 3.
- **`filepath` import:** Task 3 notes to add `"path/filepath"` — check if already imported before adding.
- **`finalLocalPath` variable:** Introduced in Task 3 to track the post-redact path; the existing `localFileFullPath` variable is preserved unchanged for the error-path retry message at line ~1303.
