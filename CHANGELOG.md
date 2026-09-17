# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Bundle [jstall](https://github.com/parttimenerd/jstall) (jstall-minimal.jar) for one-shot JVM inspection via
  `cf java jstall APP_NAME`. Requires Java 17+ locally. Supports all jstall subcommands via `--args`.
- `heap-dump --redact`: zeros primitive arrays (`byte[]`, `char[]`, etc.) in the downloaded dump before saving
  (lean redaction mode), using the bundled [hprof-redact](https://github.com/parttimenerd/hprof-analyzer) binary.
  Supported on Linux (amd64, arm64), macOS (Apple Silicon), and Windows (amd64, arm64).
- `heap-dump --redact-complete`: zeros all primitive arrays and individual primitive fields (complete redaction mode,
  maximum privacy). Mutually exclusive with `--redact`.
- `heap-dump --compress`: saves the local file as `.hprof.gz` instead of decompressing it after transfer
  (requires JDK 17+ on the container). Useful when you want to store or share the compressed dump directly.
- Transparent compressed transfer: on JDK 17+ containers, the plugin always uses `jmap gz=1` to compress
  the dump during SSH transfer (faster on slow connections), then decompresses on the fly so the local file
  is a plain `.hprof`. Use `--compress` to keep the file compressed locally.
- `heap-dump --open`: after downloading (and optionally redacting/compressing) the dump, spins up a temporary local
  HTTP server and opens the [hprof-analyzer](https://parttimenerd.github.io/hprof-analyzer) web app in the default
  browser with the dump pre-loaded. The server serves the file exactly once via a random token URL and shuts down
  automatically after the browser fetches it.
- `heap-dump --open-url <URL>`: override the hprof-analyzer base URL (e.g. a locally running instance). Implies
  `--open`.

### Changed

- Improved SSH error messages for better clarity and debugging

## [4.0.2]

### Fixed

- Fix rare ssh connection issue

## [4.0.1]

### Fixed

- Fix thread-dump command

## [4.0.0]

### Added

- Create a proper test suite
- Profiling and JCMD related features

### Fixed

- Fix many bugs discovered during testing
