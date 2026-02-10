# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Detect orphan files and unused exported functions ([#37](https://github.com/ludo-technologies/jscan/pull/37))
- Detect unused imports and exports in dead code analysis ([#36](https://github.com/ludo-technologies/jscan/pull/36))

### Fixed

- Adjust health score thresholds and add score to text output ([#35](https://github.com/ludo-technologies/jscan/pull/35))

## [0.1.1] - 2026-02-02

### Fixed

- Extract binary to temp dir to avoid overwriting bin/jscan script

## [0.1.0] - 2026-02-02

### Added

- JSON output format
- HTML output format with Lighthouse-style scoring
- Dead Code Service layer
- Application layer with Use Cases
- APTED (Tree Edit Distance) algorithm for clone detection
- MinHash and LSH Index for clone detection
- Clone Detector with Type 1-4 support
- Clone Grouping Strategies
- Module Analyzer for JS/TS import/export analysis
- CBO (Coupling Between Objects) metrics
- Dependency Graph with cycle detection
- DOT format for dependency visualization
- `check` command for CI/CD integration
- `init` command for config file generation
- Progress manager for long-running analysis tasks
- Parallel executor for concurrent task execution
- Default exclude patterns for common directories
- npm package distribution

### Changed

- Default output format to HTML for analyze command

### Fixed

- Clone loss bug and improved determinism in grouping strategies
- Various build and distribution fixes

## [0.1.0-alpha] - 2025-11-27

### Added

- Initial implementation with complexity analysis and dead code detection
- tree-sitter based JavaScript/TypeScript parsing
- CLI with analyze command
- Configuration file support (jscan.config.json)

[Unreleased]: https://github.com/ludo-technologies/jscan/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/ludo-technologies/jscan/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/ludo-technologies/jscan/compare/v0.1.0-alpha...v0.1.0
[0.1.0-alpha]: https://github.com/ludo-technologies/jscan/releases/tag/v0.1.0-alpha
