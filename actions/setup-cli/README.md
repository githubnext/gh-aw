# Setup gh-aw CLI Action

This GitHub Action installs the `gh-aw` CLI extension for a specific version using release tags.

## Features

- ✅ **Version validation**: Ensures the specified version exists as a release
- ✅ **Automatic fallback**: Tries `gh extension install` first, falls back to direct download if needed
- ✅ **Cross-platform**: Works on Linux, macOS, Windows, and FreeBSD
- ✅ **Multi-architecture**: Supports amd64, arm64, 386, and arm architectures

## Usage

### Basic Usage

```yaml
- name: Install gh-aw
  uses: githubnext/gh-aw/actions/setup-cli@main
  with:
    version: v0.37.18
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

The version of gh-aw to install. Must be a release tag.

- **Release tag**: e.g., `v0.37.18`, `v0.37.0`

## Outputs

### `installed-version`

The version tag that was actually installed.

## How It Works

1. **Version validation**: Validates the input is a valid release tag
2. **Release verification**: Validates that the release exists on GitHub
3. **Primary installation method**: Attempts to install using `gh extension install githubnext/gh-aw`
4. **Fallback method**: If primary method fails, downloads the binary directly from GitHub releases
5. **Verification**: Ensures the installed binary works correctly

## Requirements

- GitHub CLI (`gh`) must be available (pre-installed on GitHub Actions runners)
- `curl` must be available (pre-installed on GitHub Actions runners)

## Error Handling

The action will fail if:

- No version is provided
- The specified release tag doesn't exist
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

### Use Output

```yaml
- name: Install gh-aw
  id: install
  uses: githubnext/gh-aw/actions/setup-cli@main
  with:
    version: v0.37.18

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

### "Release X does not exist"

Verify the release exists at: https://github.com/githubnext/gh-aw/releases

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
