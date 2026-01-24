# Setup gh-aw CLI Action

This GitHub Action installs the `gh-aw` CLI extension for a specific version. It supports both release tags and commit SHAs that resolve to releases.

## Features

- ✅ **Version validation**: Ensures the specified version exists as a release
- ✅ **SHA resolution**: Automatically resolves long commit SHAs to their corresponding release tags
- ✅ **Automatic fallback**: Tries `gh extension install` first, falls back to direct download if needed
- ✅ **Cross-platform**: Works on Linux, macOS, Windows, and FreeBSD
- ✅ **Multi-architecture**: Supports amd64, arm64, 386, and arm architectures

## Usage

### Basic Usage (Release Tag)

```yaml
- name: Install gh-aw
  uses: githubnext/gh-aw/actions/setup-cli@main
  with:
    version: v0.37.18
```

### Using Commit SHA

```yaml
- name: Install gh-aw from SHA
  uses: githubnext/gh-aw/actions/setup-cli@main
  with:
    version: 0c77d05a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q  # Must be a long SHA that resolves to a release
```

### Complete Workflow Example

```yaml
name: Test gh-aw

on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
      
      - name: Install gh-aw
        uses: githubnext/gh-aw/actions/setup-cli@main
        with:
          version: v0.37.18
      
      - name: Verify installation
        run: |
          gh aw version
          gh aw --help
```

## Inputs

### `version` (required)

The version of gh-aw to install. Can be either:

- **Release tag**: e.g., `v0.37.18`, `v0.37.0`
- **Long commit SHA**: 40-character hexadecimal SHA that corresponds to a release (e.g., `0c77d05a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q`)

If a commit SHA is provided, the action will automatically resolve it to the corresponding release tag.

## Outputs

### `installed-version`

The version tag that was actually installed (useful when providing a SHA as input).

## How It Works

1. **Version validation**: Checks if the input is a release tag or long SHA
2. **SHA resolution**: If a long SHA is provided, resolves it to the corresponding release tag
3. **Release verification**: Validates that the release exists on GitHub
4. **Primary installation method**: Attempts to install using `gh extension install githubnext/gh-aw`
5. **Fallback method**: If primary method fails, downloads the binary directly from GitHub releases
6. **Verification**: Ensures the installed binary works correctly

## Requirements

- GitHub CLI (`gh`) must be available (pre-installed on GitHub Actions runners)
- `curl` must be available (pre-installed on GitHub Actions runners)

## Error Handling

The action will fail if:

- No version is provided
- The specified release tag doesn't exist
- The specified SHA doesn't resolve to any release
- The binary download fails
- The downloaded binary is not executable or doesn't work

## Platform Support

| OS | Architectures |
|----|---------------|
| Linux | amd64, arm64, 386, arm |
| macOS | amd64, arm64 |
| FreeBSD | amd64, arm64, 386 |
| Windows | amd64, arm64, 386 |

## Examples

### Install Specific Version

```yaml
- uses: githubnext/gh-aw/actions/setup-cli@main
  with:
    version: v0.37.18
```

### Install from SHA and Use Output

```yaml
- name: Install gh-aw
  id: install
  uses: githubnext/gh-aw/actions/setup-cli@main
  with:
    version: ${{ github.sha }}  # Assuming this SHA corresponds to a release

- name: Show installed version
  run: |
    echo "Installed version: ${{ steps.install.outputs.installed-version }}"
```

### Matrix Testing Across Versions

```yaml
jobs:
  test:
    strategy:
      matrix:
        version: [v0.37.18, v0.37.17, v0.37.16]
    runs-on: ubuntu-latest
    steps:
      - uses: githubnext/gh-aw/actions/setup-cli@main
        with:
          version: ${{ matrix.version }}
      
      - name: Test workflow compilation
        run: gh aw compile workflow.md
```

## Troubleshooting

### "Release X does not exist"

Verify the release exists at: https://github.com/githubnext/gh-aw/releases

### "Could not resolve SHA to any release"

The provided commit SHA doesn't correspond to any published release. Only SHAs from release commits can be used.

### "gh extension install failed"

The action automatically falls back to direct download when `gh extension install` fails. Check the action logs for details.

## Development

This action is part of the gh-aw repository. The `install.sh` script is generated during the build process by copying from the root `install-gh-aw.sh` file.

### Building

The installation script is copied during the build process:

```bash
make build  # Copies install-gh-aw.sh to actions/setup-cli/install.sh
```

The generated `install.sh` file is marked as `linguist-generated=true` in `.gitattributes`.

## License

This action is part of the gh-aw project and follows the same license terms.
