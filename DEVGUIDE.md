# Developer Guide

## Development Environment Setup

### 1. Clone and Setup Repository

```bash
# Clone the repository
git clone https://github.com/githubnext/gh-aw.git
cd gh-aw
```

### 2. Install Development Dependencies

```bash
# Install basic Go dependencies
make deps

# For full development (including linter)
make deps-dev
```

### 3. Build and Verify Development Environment

```bash
# Verify GitHub CLI is authenticated
gh auth status

# Run all tests to ensure everything works
make test

# Check code formatting
make fmt-check

# Run linter (may require golangci-lint installation)
make lint

# Build and test the binary
make build
./gh-aw --help
```

### 4. Install the Extension Locally for Testing

```bash
# Install the local version of gh-aw extension
make install

# Verify installation
gh aw --help
```

## Testing

### Test Structure

The project has comprehensive testing at multiple levels:

#### Unit Tests
```bash
# Run specific package tests
go test ./pkg/cli -v
go test ./pkg/parser -v  
go test ./pkg/workflow -v

# Run all unit tests
make test
```

#### End-to-End Tests
```bash
# Comprehensive test validation
make test-script
```

### Adding New Tests

1. **Unit tests**: Add to `pkg/*/package_test.go`
2. **Follow existing patterns**: Look at current tests for structure

## Debugging and Troubleshooting

### Common Development Issues

#### Build Failures
```bash
# Clean and rebuild
make clean
make deps-dev  # Use deps-dev for full development dependencies
make build
```

#### Test Failures
```bash
# Run specific test with verbose output
go test ./pkg/cli -v -run TestSpecificFunction

# Check test dependencies
go mod verify
go mod tidy
```

#### Linter Issues
```bash
# Fix formatting issues
make fmt

# Address linter warnings
make lint
```

### Development Tips

1. **Use verbose testing**: `go test -v` for detailed output
2. **Run tests frequently**: Ensure changes don't break existing functionality
3. **Check formatting**: Run `make fmt` before committing
4. **Validate thoroughly**: Use `go run test_validation.go` before pull requests

## Release Process

### Prerequisites for Releases

Before creating a release, ensure you have:

- **Maintainer access** to the GitHub repository
- **Push permissions** to create tags
- **Write access** to GitHub releases
- **All tests passing** on the main branch

### Release Types

The project uses semantic versioning (semver):
- **Major** (v2.0.0): Breaking API changes, incompatible updates
- **Minor** (v1.1.0): New features, backward compatible
- **Patch** (v1.0.1): Bug fixes, backward compatible

### Official Release Process

Releases are **automatically handled by GitHub Actions** when you create a git tag. The process is:

#### 1. Prepare for Release

```bash
# Ensure you're on the main branch with latest changes
git checkout main
git pull origin main

# Run all tests to ensure stability
make test
make lint
make fmt-check

# Test build locally
make build-all
```

#### 2. Create and Push Release Tag

For patch releases (bug fixes), you can use the automated make target:

```bash
# Automated patch release - finds current version and increments patch number
make patch-release

# Automated patch release - finds current version and increments minor number
make minor-release
```

Or create the tag manually:

```bash
# Create a new tag following semantic versioning
# Replace x.y.z with the actual version number
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag to trigger the release workflow
git push origin v1.0.0
```

#### 3. Automated Release Process

When you push a tag matching `v*.*.*`, GitHub Actions automatically:

1. **Runs tests** to ensure code quality
2. **Builds cross-platform binaries** using `gh-extension-precompile`
3. **Creates GitHub release** with:
   - Pre-compiled binaries for Linux (amd64, arm64)
   - Pre-compiled binaries for macOS (amd64, arm64) 
   - Pre-compiled binaries for Windows (amd64)
   - Automatic changelog generation

#### 4. Verify Release

After the GitHub Actions workflow completes:

```bash
# Check the release was created successfully
gh release list

# Remove any existing extension
gh extension remove gh-aw || true

# Test installation as a GitHub CLI extension
gh extension install githubnext/gh-aw@v1.0.0
gh aw --help
```

### Release Workflow Details

The release is orchestrated by `.github/workflows/release.yml` which:

- **Triggers on**: Git tags matching `v*.*.*` pattern or manual workflow dispatch
- **Runs on**: Ubuntu latest with Go version from `go.mod`
- **Permissions**: Contents (write), packages (write), ID token (write)
- **Artifacts**: Cross-platform binaries, Docker images, checksums

### Rollback Process

If a release has critical issues:

1. **Immediate**: Delete the problematic release from GitHub
   ```bash
   gh release delete v1.0.0 --yes
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

2. **Long-term**: Create a new release with fixes

### Current Release Infrastructure Status

The project has a complete automated release system in place:

- ✅ **GitHub Actions workflow** (`.github/workflows/release.yml`)
- ✅ **Cross-platform binary builds** via `gh-extension-precompile`
- ✅ **Semantic versioning** with git tags

The release system is **production-ready** and uses GitHub's official `gh-extension-precompile` action, which is the recommended approach for GitHub CLI extensions.

### Release Notes and Changelog

Release notes are automatically generated from:
- **Commit messages** between releases
- **Pull request titles** and descriptions
- **Conventional commit format** is recommended for better changelog generation

To improve changelog quality, use conventional commit messages:
```bash
git commit -m "feat: add new workflow command"
git commit -m "fix: resolve path handling on Windows"
git commit -m "docs: update installation instructions"
```

### Version Management

- **Version information** is automatically injected at build time
- **Current version** comes from git tags (`git describe --tags`)
- **No manual version files** need to be updated
- **Build metadata** includes commit hash and build date

## Updating the .github/workflows files

We dogfood the workflows from githubnext/agentics in this repo:
```
./gh-aw add daily-dependency-updates -r githubnext/agentics  --force
./gh-aw add daily-plan -r githubnext/agentics  --force
./gh-aw add daily-qa -r githubnext/agentics  --force
./gh-aw add daily-team-status -r githubnext/agentics  --force
./gh-aw add issue-triage -r githubnext/agentics  --force
./gh-aw add update-docs -r githubnext/agentics  --force
./gh-aw add weekly-research -r githubnext/agentics  --force
```
