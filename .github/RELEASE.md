# Release Process Guide

This document describes how to create releases for the Ouroboros DevOps Agent.

## Prerequisites

1. Ensure all tests pass: `make test`
2. Update CHANGELOG.md with release notes
3. Ensure go.mod and go.sum are up to date
4. All changes are committed and pushed to master

## Creating a Release

### Method 1: Tag Push (Recommended)

1. Create and push a new tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. The GitHub Actions workflow will automatically:
   - Build binaries for all platforms (Linux, macOS, Windows)
   - Generate SHA256 checksums
   - Create a GitHub Release
   - Upload all artifacts

### Method 2: Manual Workflow Dispatch

1. Go to: https://github.com/hewenyu/Ouroboros/actions/workflows/release.yml
2. Click "Run workflow"
3. Enter the tag name (e.g., v1.0.0)
4. Click "Run workflow"

## Supported Platforms

The release workflow builds binaries for:

- **Linux**: AMD64, ARM64
- **macOS**: AMD64 (Intel), ARM64 (Apple Silicon)
- **Windows**: AMD64

All binaries are built with:
- CGO enabled (required for SQLite support)
- Version information injected from Git tags
- Build time information
- Optimized with `-s -w` flags (stripped symbols)

## Build Configuration

### GoReleaser Configuration
- File: `.goreleaser.yml`
- Handles multi-platform builds with CGO cross-compilation
- Generates archives (tar.gz for Unix, zip for Windows)
- Creates checksums for verification

### GitHub Actions Workflow
- File: `.github/workflows/release.yml`
- Sets up cross-compilation toolchains
- Installs OSXCross for macOS builds
- Runs GoReleaser to build and release

## Versioning

We follow Semantic Versioning (semver):
- MAJOR version: Incompatible API changes
- MINOR version: New functionality (backward compatible)
- PATCH version: Bug fixes (backward compatible)

Example: v1.2.3
- 1 = Major version
- 2 = Minor version
- 3 = Patch version

## Local Testing

Test the build locally before creating a release:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Test release build (without publishing)
goreleaser release --snapshot --clean --skip=publish

# Check generated artifacts
ls -la dist/
```

## Troubleshooting

### Build Failures

If the release build fails:

1. Check the GitHub Actions logs for specific errors
2. Verify all dependencies are available in go.mod
3. Test local build: `make build`
4. Test cross-compilation locally with GoReleaser

### CGO Cross-Compilation Issues

If CGO builds fail for specific platforms:

1. Verify the cross-compilation toolchain is installed correctly
2. Check the CC and CXX environment variables in .goreleaser.yml
3. Test locally with the specific GOOS/GOARCH combination

### macOS Code Signing

Note: The current workflow does NOT sign macOS binaries. Users may need to:
- Right-click the binary and select "Open" (first time only)
- Or use: `xattr -d com.apple.quarantine devops-agent`

To add code signing, you need:
1. An Apple Developer account
2. Code signing certificates
3. Update .goreleaser.yml with signing configuration

## Release Checklist

- [ ] All tests pass
- [ ] CHANGELOG.md updated
- [ ] Version bumped appropriately
- [ ] All changes committed and pushed
- [ ] Tag created and pushed
- [ ] GitHub Actions workflow completes successfully
- [ ] Release artifacts uploaded correctly
- [ ] Release notes are clear and accurate
- [ ] Download and test at least one binary

## Post-Release

After a successful release:

1. Verify all artifacts are available in the GitHub Release
2. Test download and execution on at least one platform
3. Announce the release (if applicable)
4. Close any related issues/PRs
5. Update documentation if needed
