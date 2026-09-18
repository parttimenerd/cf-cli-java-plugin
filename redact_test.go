package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// makeFakeBin writes a shell script that exits with the given code and returns its path.
func makeFakeBin(t *testing.T, exitCode int) string {
	t.Helper()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "fake-redact")
	script := fmt.Sprintf("#!/bin/sh\nexit %d\n", exitCode)
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil { //nolint:gosec // test binary must be executable
		t.Fatalf("write fake bin: %v", err)
	}
	return bin
}

func TestPipeHeapDumpThroughRedact_ErrorDeletesPartial(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "dump.hprof")
	if err := os.WriteFile(src, []byte("FAKE"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Create a partial output file to simulate hprof-redact having started writing
	partial := filepath.Join(tmp, "dump-redacted.hprof")
	if err := os.WriteFile(partial, []byte("PARTIAL"), 0o600); err != nil {
		t.Fatal(err)
	}

	failBin := makeFakeBin(t, 1)
	_, err := pipeHeapDumpThroughRedact(failBin, src, "lean", false, false)
	if err == nil {
		t.Fatal("expected error from failing redact binary, got nil")
	}

	if _, statErr := os.Stat(partial); !os.IsNotExist(statErr) {
		t.Error("partial redacted file should have been deleted on error, but still exists")
	}
	// source must still be present (we only delete source on success)
	if _, statErr := os.Stat(src); statErr != nil {
		t.Errorf("source file unexpectedly removed on error: %v", statErr)
	}
}

func TestPipeHeapDumpThroughRedact_KeepOnError(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "dump.hprof")
	if err := os.WriteFile(src, []byte("FAKE"), 0o600); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(tmp, "dump-redacted.hprof")
	if err := os.WriteFile(partial, []byte("PARTIAL"), 0o600); err != nil {
		t.Fatal(err)
	}

	failBin := makeFakeBin(t, 1)
	_, err := pipeHeapDumpThroughRedact(failBin, src, "lean", false, true)
	if err == nil {
		t.Fatal("expected error from failing redact binary, got nil")
	}

	if _, statErr := os.Stat(partial); statErr != nil {
		t.Error("partial file should have been kept with keepOnError=true, but is gone")
	}
}

func TestPipeHeapDumpThroughRedact_HappyPath(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "dump.hprof")
	if err := os.WriteFile(src, []byte("HEAP"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Fake binary: copy input to output and exit 0
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "fake-redact")
	script := "#!/bin/sh\ncp \"$1\" \"$2\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil { //nolint:gosec // test binary must be executable
		t.Fatal(err)
	}

	out, err := pipeHeapDumpThroughRedact(bin, src, "lean", false, false)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	want := filepath.Join(tmp, "dump-redacted.hprof")
	if out != want {
		t.Errorf("output path: want %q, got %q", want, out)
	}
	if _, statErr := os.Stat(out); statErr != nil {
		t.Errorf("output file missing: %v", statErr)
	}
	// source must be deleted on success
	if _, statErr := os.Stat(src); !os.IsNotExist(statErr) {
		t.Error("source file should have been deleted after successful redaction")
	}
}
