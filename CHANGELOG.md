# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Bundle [jstall](https://github.com/parttimenerd/jstall) (jstall-minimal.jar) for one-shot JVM inspection
  via `cf java jstall APP_NAME`. Requires Java 17+ locally. Supports all jstall subcommands via `jstall APP --args`.

### Changed
- Improved SSH error messages for better clarity and debugging
- Enhanced documentation and README with better clarity

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
