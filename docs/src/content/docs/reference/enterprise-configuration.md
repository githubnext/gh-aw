---
title: Enterprise Configuration
description: Configure GitHub Agentic Workflows for GitHub Enterprise Server (GHES) and GitHub Enterprise Cloud (GHEC), including GHES compatibility mode and CLI setup.
sidebar:
  order: 51
---

# Enterprise Configuration

This page covers configuration options specific to GitHub Enterprise Server (GHES) and GitHub Enterprise Cloud (GHEC) deployments.

## GitHub Enterprise Server (GHES) Compatibility

### GHES Compatibility Mode

GHES instances running versions that predate `@actions/artifact` v2.0.0 support cannot use `actions/upload-artifact@v4+` or `actions/download-artifact@v4+`. Attempting to run compiled workflows on these instances produces a `GHESNotSupportedError`.

gh-aw includes a GHES compatibility mode toggle (`aw.json` `ghes` or `gh aw compile --ghes`) for GHES releases that require the v3 artifact backend. Compatibility mode emits `upload-artifact@v3.2.2` and `download-artifact@v3.1.0`; default GitHub.com compilation continues to use the latest artifact actions.

This compatibility path supports GHES 3.21.x and earlier when the workflow runs on Actions Runner 2.327.1 or later, which is required by the Node.js 24 runtime used by these pinned artifact actions. For later GHES releases, keep compatibility mode enabled until your instance supports the v4 artifact backend.

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

### Reusable-workflow repository identity

GHES does not expose the `job.workflow_*` context properties. Non-inlined reusable workflows instead resolve their host repository and exact checkout revision from the runner's OIDC `job_workflow_ref` and `job_workflow_sha` claims. This requires a GHES release supporting both claims and `id-token: write` in the agentic workflow:

```aw
permissions:
  contents: read
  id-token: write
```

The compiler forwards this explicit permission to each job that resolves the host repository, including pre-activation and conclusion jobs. The calling workflow must also grant `id-token: write`; reusable workflows cannot elevate the caller's permissions. An explicit job-level denial remains effective.

If the runner cannot provide both reusable-workflow claims, resolution fails closed rather than substituting the caller's repository, workflow ref, or checkout. Inlined imports use the caller checkout and do not require this identity fallback. Their omitted dispatch ref retains PR-head resolution for `pull_request` and PR `issue_comment` events.

Host identity resolution does not enable unsupported cross-repository artifact publication or prove end-to-end GHES runtime compatibility. Artifact-backend and publication limitations still apply.

## GitHub Enterprise Server CLI Setup

For `gh` CLI configuration, host authentication, and `GH_HOST` setup on GHES, see [GitHub Enterprise Server Support](/gh-aw/setup/cli/#github-enterprise-server-support) in the CLI reference.

## Copilot Engine on GHES

For Copilot-specific prerequisites, licensing requirements, and firewall configuration on GHES, see [Copilot Engine Prerequisites on GHES](/gh-aw/troubleshooting/common-issues/#copilot-engine-prerequisites-on-ghes).
