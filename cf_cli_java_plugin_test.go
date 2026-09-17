/*
 * Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved.
 * This file is licensed under the Apache Software License, v. 2 except as noted
 * otherwise in the LICENSE file at the root of the repository.
 */

package main

import (
	"testing"
)

const testAppName = "myapp"

func TestParseOptions_Open(t *testing.T) {
	p := &JavaPlugin{}
	opts, _, err := p.parseOptions([]string{cmdHeapDump, testAppName, "--" + flagOpen})
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
	opts, _, err := p.parseOptions([]string{cmdHeapDump, testAppName, "--" + flagOpenURL, "http://localhost:8080"})
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
	_, _, err := p.parseOptions([]string{cmdHeapDump, testAppName, "--" + flagOpen, "--" + flagNoDownload})
	if err == nil {
		t.Fatal("expected error for --open + --no-download, got nil")
	}
}

func TestParseOptions_Open_DryRun_NoError(t *testing.T) {
	p := &JavaPlugin{}
	opts, _, err := p.parseOptions([]string{cmdHeapDump, testAppName, "--" + flagOpen, "--dry-run"})
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
