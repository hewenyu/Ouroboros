# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Actions workflow for automated multi-platform releases
- GoReleaser configuration for building with CGO enabled
- Support for Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64)
- Automated release creation with checksums
- Test script for local release builds

### Changed

### Fixed

## [0.1.0] - YYYY-MM-DD

### Added
- Initial release
- GitHub webhook integration with HMAC-SHA256 validation
- Docker SDK integration for container management
- SQLite database with WAL mode
- MCP protocol support (stdio and SSE transports)
- MCP authentication with Bearer tokens
- Single binary deployment with zero dependencies

[Unreleased]: https://github.com/hewenyu/Ouroboros/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/hewenyu/Ouroboros/releases/tag/v0.1.0
