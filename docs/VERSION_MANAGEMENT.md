# Version Management in gh-aw

This document describes how version information is managed in the gh-aw extension.

## Overview

The `gh-aw` extension uses semantic versioning and embeds the version information into the binary at build time using Go's `-ldflags` mechanism.

## Default Version

The default version in `cmd/gh-aw/main.go` is set to `"dev"`:

```go
var (
    version = "dev"
)
```

This default is overridden at build time for official releases.

## Local Development Builds

When building locally using the Makefile, the version is automatically set to the current git description:

```bash
make build
```

This uses:
```makefile
VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"
```

## Release Builds

For official releases published to GitHub, the version is set using a custom build script.

### Release Process

1. When a version tag is pushed (e.g., `v0.22.5`), the release workflow is triggered
2. The workflow uses `cli/gh-extension-precompile@v2` with our custom build script
3. The custom build script (`scripts/build-release.sh`) builds binaries for all platforms
4. Each binary is built with the proper version embedded via ldflags

### Custom Build Script

The `scripts/build-release.sh` script:
- Takes the version tag as a required argument
- Builds binaries for all supported platforms
- Embeds the version using `-ldflags="-s -w -X main.version=${VERSION}"`

Example usage:
```bash
./scripts/build-release.sh v0.22.5
```

### Release Workflow Configuration

In `.github/workflows/release.yml`:

```yaml
- name: Release with gh-extension-precompile
  uses: cli/gh-extension-precompile@v2
  with:
    go_version_file: go.mod
    build_script_override: scripts/build-release.sh
```

## Verification

To check the version of any binary:

```bash
./gh-aw version
```

For release binaries, this will show the actual version tag (e.g., `v0.22.5`) instead of `"dev"`.

## Testing

Tests verify that:
1. The version variable can be overridden at build time (`cmd/gh-aw/version_test.go`)
2. The build script exists and works correctly (`scripts/test-build-release.sh`)
3. Release binaries contain the proper version, not "dev"

Run tests:
```bash
make test
```

## Troubleshooting

If a release binary shows version `"dev"`:
1. Check that the release workflow used `build_script_override: scripts/build-release.sh`
2. Verify the build script was executed and received the version argument
3. Check the GitHub Actions logs for the release workflow
