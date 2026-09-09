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
  Supported on Linux, macOS (Apple Silicon), and Windows.
- `heap-dump --redact-complete`: zeros all primitive arrays and individual primitive fields (complete redaction mode,
  maximum privacy). Mutually exclusive with `--redact`.
- `heap-dump --compress`: saves the dump as `.hprof.gz` by transferring it gzip-compressed over SSH (requires JDK 17+
  on the container). Prints a warning and falls back to uncompressed on older JDKs.
- Transparent compressed transfer: on JDK 17+ containers, the plugin automatically uses `jmap gz=1` to reduce
  transfer size even without `--compress`, decompressing on the fly so the local file is always a plain `.hprof`.

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
