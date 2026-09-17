package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServeFileOnce(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "test.hprof")
	if err := os.WriteFile(p, []byte("HEAP_CONTENT"), 0o600); err != nil {
		t.Fatal(err)
	}

	port, urlFile, done, err := serveFileOnce(p)
	if err != nil {
		t.Fatalf("serveFileOnce: %v", err)
	}
	if port <= 0 {
		t.Fatalf("expected positive port, got %d", port)
	}

	// urlFile must have .hprof extension and contain only the token (no path separators)
	if !strings.HasSuffix(urlFile, ".hprof") {
		t.Errorf("urlFile %q does not end with .hprof", urlFile)
	}
	if strings.ContainsAny(urlFile, "/\\") {
		t.Errorf("urlFile %q contains path separators", urlFile)
	}

	url := fmt.Sprintf("http://localhost:%d/%s", port, urlFile)
	resp, err := http.Get(url) //nolint:noctx,gosec
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("body close: %v", closeErr)
		}
	}()

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

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("done channel not closed after successful GET")
	}
}

func TestServeFileOnce_WrongPath404(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "test.hprof")
	if err := os.WriteFile(p, []byte("SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}

	port, _, _, err := serveFileOnce(p)
	if err != nil {
		t.Fatalf("serveFileOnce: %v", err)
	}

	// Any path other than the exact random token must return 404
	for _, badPath := range []string{"/test.hprof", "/", "/other.hprof", "/../../etc/passwd"} {
		url := fmt.Sprintf("http://localhost:%d%s", port, badPath)
		resp, gerr := http.Get(url) //nolint:noctx,gosec
		if gerr != nil {
			continue // server may have shut down already, that's fine
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Errorf("GET %s: expected non-200, got 200", url)
		}
	}
}

func TestServeFileOnce_GzExtension(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "test.hprof.gz")
	if err := os.WriteFile(p, []byte("GZ"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, urlFile, _, err := serveFileOnce(p)
	if err != nil {
		t.Fatalf("serveFileOnce: %v", err)
	}
	if !strings.HasSuffix(urlFile, ".hprof.gz") {
		t.Errorf("urlFile %q does not end with .hprof.gz", urlFile)
	}
}

func TestServeFileOnce_MissingFile(t *testing.T) {
	_, _, _, err := serveFileOnce("/nonexistent/path/dump.hprof")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestServeFileOnce_OptionsPreflight(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "test.hprof")
	if err := os.WriteFile(p, []byte("DATA"), 0o600); err != nil {
		t.Fatal(err)
	}

	port, urlFile, done, err := serveFileOnce(p)
	if err != nil {
		t.Fatalf("serveFileOnce: %v", err)
	}

	url := fmt.Sprintf("http://localhost:%d/%s", port, urlFile)

	// OPTIONS preflight must not trigger shutdown
	req, _ := http.NewRequest(http.MethodOptions, url, nil) //nolint:noctx
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS %s: %v", url, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("OPTIONS: want 204, got %d", resp.StatusCode)
	}
	select {
	case <-done:
		t.Error("done channel closed after OPTIONS — server shut down prematurely")
	default:
	}

	// Subsequent GET must still succeed and close done
	resp2, err := http.Get(url) //nolint:noctx,gosec
	if err != nil {
		t.Fatalf("GET after OPTIONS: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("GET after OPTIONS: want 200, got %d", resp2.StatusCode)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("done channel not closed after GET")
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
		{
			"https://parttimenerd.github.io/hprof-analyzer",
			0,
			"dump.hprof",
			"https://parttimenerd.github.io/hprof-analyzer/?file=http://localhost:PORT/dump.hprof",
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
	url := buildOpenURL("https://example.com/analyzer/", 1234, "dump.hprof")
	if strings.Contains(url, "//?") {
		t.Errorf("double slash before ?: %s", url)
	}
}
