# Install GitHub Copilot CLI Action

A secure GitHub Action for installing GitHub Copilot CLI with checksum verification.

## Features

- 🔒 **Secure**: Downloads from official GitHub releases
- ✅ **Verified**: Optional SHA256 checksum verification
- 📌 **Pinned**: Version pinning for reproducible builds
- 🌍 **Cross-platform**: Supports Linux and macOS on x64 and ARM64

## Usage

### Basic Usage

```yaml
- name: Install GitHub Copilot CLI
  uses: ./actions/install-copilot-cli
  with:
    version: '0.0.369'
```

### With Checksum Verification (Recommended)

```yaml
- name: Install GitHub Copilot CLI
  uses: ./actions/install-copilot-cli
  with:
    version: '0.0.369'
    checksum: 'YOUR_SHA256_CHECKSUM_HERE'
```

### Finding Checksums

To find the checksum for a specific version:

1. Visit the [Copilot CLI releases page](https://github.com/github/copilot-cli/releases)
2. Download the appropriate file for your platform (e.g., `copilot-linux-x64.tar.gz`)
3. Calculate the checksum: `sha256sum copilot-linux-x64.tar.gz`

Alternatively, run the action without a checksum once, and it will output the checksum in the logs.

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `version` | Copilot CLI version (e.g., "0.0.369" or "v0.0.369") | Yes | `0.0.369` |
| `checksum` | SHA256 checksum for verification | No | `''` |

## Outputs

| Output | Description |
|--------|-------------|
| `installed-version` | The version of Copilot CLI that was installed |

## Security

This action improves security over the previous `curl | sudo bash` pattern by:

1. **Direct Downloads**: Downloads binaries directly from GitHub releases instead of executing remote scripts
2. **Checksum Verification**: Optionally verifies file integrity with SHA256 checksums
3. **Version Pinning**: Explicitly pins to specific versions for reproducible builds
4. **Transparent Installation**: All installation steps are visible in the action definition

### Note on Sudo Usage

The action still requires `sudo` privileges to install to `/usr/local/bin` (a system directory). This is necessary for the binary to be available system-wide. However, this is significantly more secure than the previous pattern because:

- The downloaded binary is from a trusted source (official GitHub releases)
- Checksums can be verified before installation
- No remote script is executed with elevated privileges
- All installation steps are explicit and auditable

Future improvements could include:
- Optional user-space installation (e.g., `~/.local/bin`)
- Support for containerized installations that don't require sudo

### Checksum Verification

While checksum verification is optional, it is **strongly recommended** for production use. When checksums are not provided, the action will:
- Display a warning in the workflow logs
- Show the calculated checksum for easy addition to your workflow
- Highlight the security implications of skipping verification

## Migration Guide

### Before (Old Pattern)

```yaml
- name: Install GitHub Copilot CLI
  run: |
    export VERSION=0.0.369 && curl -fsSL https://gh.io/copilot-install | sudo bash
    copilot --version
```

### After (New Pattern)

```yaml
- name: Install GitHub Copilot CLI
  uses: ./actions/install-copilot-cli
  with:
    version: '0.0.369'
```

## License

This action is part of the gh-aw project and is licensed under the same terms.
