---
title: Enterprise Configuration
description: Configure GitHub Agentic Workflows for GitHub Enterprise Server (GHES) and GitHub Enterprise Cloud (GHEC), including artifact compatibility and CLI setup.
sidebar:
  order: 51
---

# Enterprise Configuration

This page covers configuration options specific to GitHub Enterprise Server (GHES) and GitHub Enterprise Cloud (GHEC) deployments.

## GitHub Enterprise Server (GHES) Compatibility

### Artifact Compatibility Mode

GHES instances running versions that predate `@actions/artifact` v2.0.0 support cannot use `actions/upload-artifact@v4+` or `actions/download-artifact@v4+`. Attempting to run compiled workflows on these instances produces a `GHESNotSupportedError`.

gh-aw includes a GHES compatibility mode that instructs the compiler to emit `upload-artifact@v3.2.2` and `download-artifact@v3.1.0` instead of the latest v4+ versions.

#### Enable via `aw.json` (recommended)

Set `ghes: true` in `.github/workflows/aw.json` to apply GHES compatibility to every workflow compiled in the repository:

```json
{
  "ghes": true
}
```

#### Auto-detection with `gh aw init`

Running `gh aw init` inside a GHES repository automatically detects the deployment and writes `ghes: true` to `.github/workflows/aw.json`. No manual configuration is required.

#### Enable via CLI flag

Pass `--ghes` to `gh aw compile` for a one-off compilation without modifying `aw.json`:

```bash
gh aw compile --ghes my-workflow.md
```

> [!NOTE]
> The `--ghes` flag only affects the current compilation. Use `aw.json` to apply GHES compatibility permanently across all workflows in the repository.

#### Enable per workflow via `features`

Set `features.ghes-artifact-compat: true` in workflow frontmatter to opt a single workflow into artifact-compat mode without changing the repository-wide setting:

```aw wrap
features:
  ghes-artifact-compat: true
```

Equivalently, set the `GH_AW_FEATURES=ghes-artifact-compat` environment variable when invoking `gh aw compile`. When the flag is active the compiler pins `actions/upload-artifact@v3.2.2` and `actions/download-artifact@v3.1.0`; compilation fails fast if the required v3 pin is missing rather than silently emitting an incompatible v4+ reference.

> [!TIP]
> To apply GHES artifact compatibility to every workflow in the repository, use the `--ghes` CLI flag or set `ghes: true` in `aw.json` instead. See [Enable via CLI flag](#enable-via-cli-flag) and [Enable via `aw.json`](#enable-via-awjson-recommended) above.

## GitHub Enterprise Server CLI Setup

For `gh` CLI configuration, host authentication, and `GH_HOST` setup on GHES, see [GitHub Enterprise Server Support](/gh-aw/setup/cli/#github-enterprise-server-support) in the CLI reference.

## Copilot Engine on GHES

For Copilot-specific prerequisites, licensing requirements, and firewall configuration on GHES, see [Copilot Engine Prerequisites on GHES](/gh-aw/troubleshooting/common-issues/#copilot-engine-prerequisites-on-ghes).
