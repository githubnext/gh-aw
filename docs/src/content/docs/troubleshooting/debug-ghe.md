---
title: Debugging GHE Cloud with Data Residency
description: Step-by-step guide for setting up and debugging agentic workflows on GitHub Enterprise Cloud with data residency (*.ghe.com).
sidebar:
  order: 300
---

This guide walks you through setting up and running agentic workflows on **GitHub Enterprise Cloud with data residency** (`*.ghe.com`). It reflects the configuration needed as of `gh aw` **v0.61.1+** for enterprises using data residency in the EU or other regions.

For an existing failed run, start with the [debug.md bootstrap](https://github.com/github/gh-aw/blob/main/debug.md)
and [shared debugging strategy](https://github.com/github/gh-aw/blob/main/.github/aw/debug-agentic-workflow.md),
using the full `https://yourorg.ghe.com/OWNER/REPO/actions/runs/RUN_ID` URL.
Those instructions own setup, evidence collection, review, and live-execution
gates; this page adds enterprise-specific configuration, not a separate debug loop.
The historical minimum version below is not a recommended current version.

> [!TIP]
> The one thing you must do differently from github.com is set `api-target` in your workflow frontmatter to `copilot-api.<yourorg>.ghe.com`. Everything else flows from that.

Based on the debugging discussion in [github/gh-aw#18480](https://github.com/github/gh-aw/issues/18480).

## Prerequisites

You need a repository on your GHE Cloud data residency instance (for example, `yourorg.ghe.com`), `gh aw` **v0.61.1+** (`gh extension install github/gh-aw`), Copilot enabled for your enterprise, and `gh` authenticated against your GHE host:

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

The compiler (v0.61.1+) automatically adds your GHE domains (`api.yourorg.ghe.com`, `copilot-api.yourorg.ghe.com`) to the firewall allow-list, sets `--copilot-api-target` for the AWF api-proxy, and configures `GH_HOST` so the `gh` CLI targets the correct host.

### Step 5: Publish and Test Within the Shared Gates

Publish source and generated lock together to an authorized feature branch.
For a debugging test, follow the shared strategy's security review,
`gh aw compile repo-assist --dry-run`, and explicit live-authorization gates
before dispatching the exact reviewed branch:

```bash
GH_HOST=yourorg.ghe.com gh aw run repo-assist --ref REVIEWED_BRANCH
```

Compilation alone does not authorize execution. A general debugging request does
not authorize this command, and explicit no-dispatch rules take precedence.

## Troubleshooting

If the workflow fails, start with the Copilot CLI on your local machine. First confirm authentication:

```bash
GH_HOST=yourorg.ghe.com gh auth status
```

Then launch Copilot:

```bash
GH_HOST=yourorg.ghe.com copilot
```

Invoke the `agentic-workflows` skill and provide the existing failed run, for example:

```text
Debug this workflow run using https://raw.githubusercontent.com/github/gh-aw/main/debug.md

Run URL: https://yourorg.ghe.com/OWNER/REPO/actions/runs/RUN_ID
```

Copilot can inspect your workflow files, existing run logs, and `gh aw audit`
output without dispatching another run. Never send enterprise credentials to
the public raw-content host used for gh-aw guidance.

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

First compare existing installation, authentication, firewall, and api-proxy
logs to identify the failing boundary. Do not hand-edit generated locks or run
Copilot outside AWF to bypass isolation. If component reproduction is necessary,
use the shared strategy's reviewed, isolated test bindings and fixtures; leave
hosted credentials and runner behavior unverified until an authorized live test.

### Advanced: Capturing HTTP Traffic

Prefer existing firewall and api-proxy artifacts. If additional HTTP diagnostics
are necessary, review a separate diagnostic source change under the shared
strategy before executing or publishing it. Do not hand-edit the Execute step.
Copilot's Node.js runtime supports these diagnostic environment variables:

```yaml
env:
  NODE_DEBUG: fetch,undici
  UNDICI_DEBUG: full
```

> [!IMPORTANT]
> The Copilot CLI uses Node.js `fetch()`/`undici` internally, not the built-in `http`/`https` modules. Setting `NODE_DEBUG=http,https` will capture nothing. You must use `UNDICI_DEBUG=full`.

Treat captured headers, bodies, and authorization URLs as sensitive; keep raw
captures private and redact before reuse. Traffic may include these domains on
data residency:

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
