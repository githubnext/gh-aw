---
title: Debugging GHE Cloud with Data Residency
description: Step-by-step guide for setting up and debugging agentic workflows on GitHub Enterprise Cloud with data residency (*.ghe.com).
sidebar:
  order: 300
---

This guide walks you through setting up and running agentic workflows on **GitHub Enterprise Cloud with data residency** (`*.ghe.com`). It reflects the configuration needed as of `gh aw` **v0.61.1+** for enterprises using data residency in the EU or other regions.

> [!TIP]
> The one thing you must do differently from github.com is set `api-target` in your workflow frontmatter to `copilot-api.<yourorg>.ghe.com`. Everything else flows from that.

Based on the debugging discussion in [github/gh-aw#18480](https://github.com/github/gh-aw/issues/18480).

## Prerequisites

- A repository on your GHE Cloud data residency instance (e.g., `yourorg.ghe.com`)
- The `gh aw` CLI extension **v0.61.1 or later** (`gh extension install github/gh-aw`)
- Copilot enabled for your enterprise
- The `gh` CLI authenticated with your GHE instance:
  ```bash
  gh auth login --hostname yourorg.ghe.com
  ```

## Setup

### Step 1: Initialize Your Repository

```bash
GH_HOST=yourorg.ghe.com gh aw init
```

### Step 2: Add a Workflow

```bash
GH_HOST=yourorg.ghe.com gh aw add-wizard githubnext/agentics/repo-assist
```

Follow the prompts to configure the workflow for your repository.

### Step 3: Configure the Engine for GHE (Critical)

Open the generated workflow `.md` file (e.g., `.github/workflows/repo-assist.md`) and ensure the `engine` section in the YAML frontmatter includes `api-target` pointing to your enterprise's Copilot API subdomain:

```aw
engine:
  id: "copilot"
  api-target: "copilot-api.yourorg.ghe.com"
```

Replace `yourorg` with your enterprise's slug — the subdomain portion of `yourorg.ghe.com`.

**Why this is required**: On GHE Cloud with data residency, Copilot inference runs on a dedicated subdomain (`copilot-api.yourorg.ghe.com`) rather than the default `api.githubcopilot.com`. Without `api-target`, the AWF api-proxy routes requests to the wrong host, resulting in authentication failures.

See [Enterprise API Endpoint](/gh-aw/reference/engines/#enterprise-api-endpoint-api-target) for full `api-target` documentation.

### Step 4: Compile

```bash
GH_HOST=yourorg.ghe.com gh aw compile repo-assist
```

The compiler (v0.61.1+) will automatically:
- Add your GHE domains (`api.yourorg.ghe.com`, `copilot-api.yourorg.ghe.com`) to the firewall allow-list
- Set `--copilot-api-target` for the AWF api-proxy
- Configure `GH_HOST` so the `gh` CLI targets the correct host

### Step 5: Commit, Push, and Run

```bash
git add .github/workflows/repo-assist.md .github/workflows/repo-assist.lock.yml
git commit -m "Add repo-assist agentic workflow"
git push

# Dispatch the workflow
GH_HOST=yourorg.ghe.com gh workflow run repo-assist.lock.yml --ref main
```

## Troubleshooting

If the workflow fails, start by using the Copilot CLI to help diagnose the issue.

### Debugging with Copilot CLI Locally

The fastest way to diagnose failures is to use the Copilot CLI interactively from your local machine. This lets you confirm Copilot can authenticate against your GHE instance and then use Copilot itself to help debug workflow failures.

1. **Ensure you're authenticated with your GHE instance**:
   ```bash
   GH_HOST=yourorg.ghe.com gh auth status
   ```

2. **Launch the Copilot CLI**:
   ```bash
   GH_HOST=yourorg.ghe.com copilot
   ```

3. **Select the agentic-workflows agent** — when Copilot starts, run `/agent` and choose `agentic-workflows` from the list.

4. **Ask Copilot to run and debug the workflow** — trigger the workflow, wait for it to complete, and then ask Copilot to analyze the results. For example:
   ```
   Run the repo-assist workflow and check if it succeeds.
   If it fails, help me debug the failure.
   ```

Copilot has access to your workflow files, run logs, and the `gh aw audit` tool, so it can inspect failures end-to-end and suggest fixes.

### Common Errors

#### "Authentication failed"

```
Error: Authentication failed
Your GitHub token may be invalid, expired, or lacking the required permissions.
```

**Cause**: The `api-target` is missing or incorrect. The api-proxy is sending Copilot requests to the wrong endpoint.

**Fix**: Verify your `.md` frontmatter has:

```aw
engine:
  id: "copilot"
  api-target: "copilot-api.yourorg.ghe.com"
```

Then recompile with `GH_HOST=yourorg.ghe.com gh aw compile`.

#### "none of the git remotes point to a known GitHub host"

**Cause**: `GH_HOST` is not set. The `gh` CLI doesn't recognize your GHE instance as a GitHub host.

**Fix**: Upgrade to `gh aw` v0.61.1+ and recompile. The compiler now auto-configures `GH_HOST` for GHE instances.

#### "Not Found" during checkout steps

**Cause**: The lock file is trying to access `github.com` repositories with your GHE-scoped token. This can happen with local builds of the compiler that use `actions/checkout` instead of the published `github/gh-aw-actions` action reference.

**Fix**: Always compile with the installed `gh aw` extension rather than a local binary:

```bash
GH_HOST=yourorg.ghe.com gh aw compile <workflow-name>
```

See also [Copilot GHES: Common Error Messages](/gh-aw/troubleshooting/common-issues/#copilot-ghes-common-error-messages) for additional error patterns.

### Advanced: Testing Copilot on the Runner Directly

If you need to verify that Copilot auth works on the Actions runner itself (outside the AWF sandbox), add a temporary diagnostic step to the lock file before the Execute step:

```yaml
- name: Test Copilot CLI directly
  env:
    GH_HOST: yourorg.ghe.com
    GH_TOKEN: ${{ github.token }}
  run: |
    echo "GH_HOST=$GH_HOST"
    echo "GITHUB_SERVER_URL=$GITHUB_SERVER_URL"
    /usr/local/bin/copilot --version
    /usr/local/bin/copilot --prompt "Say hello" --log-level all 2>&1 | head -50
```

If this step succeeds but the Execute step fails, the problem is in the firewall or api-proxy configuration, not in Copilot auth.

### Advanced: Capturing HTTP Traffic

To see exactly which hosts the Copilot CLI contacts, add these environment variables to the Execute step:

```yaml
env:
  NODE_DEBUG: fetch,undici
  UNDICI_DEBUG: full
```

> [!IMPORTANT]
> The Copilot CLI uses Node.js `fetch()`/`undici` internally, not the built-in `http`/`https` modules. Setting `NODE_DEBUG=http,https` will capture nothing. You must use `UNDICI_DEBUG=full`.

The traffic capture reveals the four domains the CLI uses on data residency:

| Domain | Purpose |
|--------|---------|
| `api.yourorg.ghe.com` | REST API, Copilot auth (`/copilot_internal/user`) |
| `copilot-api.yourorg.ghe.com` | Inference, model listing, MCP |
| `copilot-telemetry-service.yourorg.ghe.com` | Telemetry |
| `api.githubcopilot.com` | Shared Copilot services |

### Advanced: Checking Firewall Logs

Download the workflow run artifacts and inspect `sandbox/firewall/logs/access.log`. Each line shows whether a domain was allowed (`TCP_TUNNEL`) or blocked (`DENIED`). Verify that your `yourorg.ghe.com` domains appear as `TCP_TUNNEL`.

## Required Domains Reference

For GHE Cloud with data residency, the following domains must be reachable from inside the AWF sandbox. The compiler adds most of these automatically when `api-target` is set:

| Domain | Auto-added by compiler? | Required for |
|--------|:-----------------------:|-------------|
| `yourorg.ghe.com` | ✅ | Git, web UI |
| `api.yourorg.ghe.com` | ✅ | REST API, Copilot auth |
| `copilot-api.yourorg.ghe.com` | ✅ | Inference, models, MCP |
| `copilot-telemetry-service.yourorg.ghe.com` | ❌ (add manually if needed) | Telemetry |

To add the telemetry domain manually:

```aw
network:
  allowed:
    - defaults
    - copilot-telemetry-service.yourorg.ghe.com
```
