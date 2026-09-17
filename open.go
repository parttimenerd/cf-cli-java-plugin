/*
 * Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved.
 * This file is licensed under the Apache Software License, v. 2 except as noted
 * otherwise in the LICENSE file at the root of the repository.
 */

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	mux.HandleFunc("/"+filepath.Base(path), func(w http.ResponseWriter, r *http.Request) {
		// Open the specific file directly rather than using http.ServeFile, which
		// follows path cleaning and redirects and could expose other files.
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
		// Intentionally not using r.Context(): the request context is canceled as
		// soon as the handler returns, but Shutdown must outlive the request.
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:contextcheck
		go func(ctx context.Context, cancel context.CancelFunc) {                               //nolint:contextcheck
			defer cancel()
			close(done)
			_ = srv.Shutdown(ctx)
		}(shutdownCtx, shutdownCancel)
	})

	go func() { _ = srv.Serve(ln) }()

	return port, done, nil
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
