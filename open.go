/*
 * Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved.
 * This file is licensed under the Apache Software License, v. 2 except as noted
 * otherwise in the LICENSE file at the root of the repository.
 */

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// serveFileOnce starts a local HTTP server on a random port that serves path
// exactly once. The file is exposed under a random token path (e.g. /a3f9c2.hprof)
// so the local filename is never leaked and only the holder of the URL can fetch it.
// Returns the bound port, the randomised URL path segment, and a channel that
// closes when the first GET request completes.
func serveFileOnce(path string) (port int, urlFile string, done <-chan struct{}, err error) {
	if _, err = os.Stat(path); err != nil {
		return 0, "", nil, fmt.Errorf("file not found: %w", err)
	}

	// Build a random token + preserve only the file extension (.hprof or .hprof.gz).
	var tokenBytes [8]byte
	if _, err = rand.Read(tokenBytes[:]); err != nil {
		return 0, "", nil, fmt.Errorf("could not generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes[:])
	ext := extHprof
	if strings.HasSuffix(path, extHprofGz) {
		ext = extHprofGz
	}
	urlFile = token + ext

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, "", nil, fmt.Errorf("could not bind local port: %w", err)
	}
	port = ln.Addr().(*net.TCPAddr).Port

	doneCh := make(chan struct{})
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Register only the exact random path — any other request gets 404.
	mux.HandleFunc("/"+urlFile, func(w http.ResponseWriter, r *http.Request) {
		f, ferr := os.Open(path) //nolint:gosec // path comes from plugin internals, not user input
		if ferr != nil {
			http.Error(w, "file unavailable", http.StatusInternalServerError)
			return
		}
		defer func() { _ = f.Close() }()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, f)
		// Use a fresh context: r.Context() is canceled when the handler returns,
		// but Shutdown must outlive the request.
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:contextcheck
		go func(ctx context.Context, cancel context.CancelFunc) {                               //nolint:contextcheck
			defer cancel()
			close(doneCh)
			_ = srv.Shutdown(ctx)
		}(shutdownCtx, shutdownCancel)
	})

	go func() { _ = srv.Serve(ln) }()

	return port, urlFile, doneCh, nil
}

// buildOpenURL constructs the hprof-analyzer URL with the ?file= parameter.
// base is the analyzer base URL (trailing slash optional).
// Use port=0 to produce a PORT placeholder (for dry-run output).
func buildOpenURL(base string, port int, filename string) string {
	base = strings.TrimRight(base, "/")
	portStr := fmt.Sprintf("%d", port)
	if port == 0 {
		portStr = "PORT"
	}
	return fmt.Sprintf("%s/?file=http://localhost:%s/%s", base, portStr, filename)
}

// openBrowser opens url in the system default browser.
// If launch fails, prints the URL to stdout so the user can open it manually.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case osWindows:
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("Opening: %s\n", url)
	}
}
