---
title: Common Issues
description: Frequently encountered issues when working with GitHub Agentic Workflows and their solutions.
sidebar:
  order: 200
---

Frequently encountered issues, organized by workflow stage and component.

## Installation Issues

### Extension Installation Fails

If `gh extension install github/gh-aw` fails, use the standalone installer (works in Codespaces and restricted networks). Pass a tag as the second argument to pin a version ([releases](https://github.com/github/gh-aw/releases)). Verify with `gh extension list`.

```bash wrap
curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash
curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash -s -- v0.40.0
```

## Organization Policy Issues

### Custom Actions Not Allowed in Enterprise Organizations

**Error:** `The action github/gh-aw/actions/setup@... is not allowed in {ORG} because all actions must be from a repository owned by your enterprise, created by GitHub, or verified in the GitHub Marketplace.`

**Cause:** Enterprise policies restrict which GitHub Actions can be used.

**Solution:** An admin must add `github/gh-aw@*` to the organization's allowed actions, either through Settings → Actions → Policies → "Allow select actions and reusable workflows" ([docs](https://docs.github.com/en/organizations/managing-organization-settings/disabling-or-limiting-github-actions-for-your-organization#allowing-select-actions-and-reusable-workflows-to-run)), or by editing a centralized `policies/actions.yml`:

```yaml
allowed_actions:
  - "actions/*"
  - "github/gh-aw@*"
```

Wait a few minutes for policy propagation, then re-run.

> [!TIP]
> The gh-aw actions are open source at [github.com/github/gh-aw/tree/main/actions](https://github.com/github/gh-aw/tree/main/actions) and pinned to specific SHAs.

## Repository Configuration Issues

### Actions Restrictions Reported During Init

The CLI validates three permission layers. Fix restrictions in Repository Settings → Actions → General:

1. **Actions disabled**: Enable Actions ([docs](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository))
2. **Local-only**: Switch to "Allow all actions" or enable GitHub-created actions ([docs](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository#managing-github-actions-permissions-for-your-repository))
3. **Selective allowlist**: Enable "Allow actions created by GitHub" checkbox ([docs](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository#allowing-select-actions-and-reusable-workflows-to-run))

> [!NOTE]
> Organization policies override repository settings. Contact admins if settings are grayed out.

## Workflow Compilation Issues

### Frontmatter Field Not Taking Effect

If a frontmatter setting appears to be silently ignored, the field name may be misspelled. The compiler does not warn about unknown field names — they are silently discarded.

> [!WARNING]
> Common frontmatter field name mistakes:
>
> | Wrong | Correct |
> |-------|---------|
> | `agent:` | `engine:` |
> | `mcp-servers:` | `tools:` (under which MCP servers are configured) |
> | `tool-sets:` | `toolsets:` (under `tools.github:`) |
> | `allowed_repos:` | `allowed-repos:` (under `tools.github:`) |
> | `timeout:` | `timeout-minutes:` |
>
> Run `gh aw compile --verbose` to confirm which settings were parsed. If your setting is missing from the output, check the [Frontmatter Reference](/gh-aw/reference/frontmatter/) for the correct field name.

### Compilation Failures

Common fixes: validate YAML syntax (indentation and `key: value` spacing), confirm required fields such as `on:`, and check types against the schema with `gh aw compile --verbose`.

If no lock file is generated, fix the reported errors (`gh aw compile 2>&1 | grep -i error`) and confirm `.github/workflows/` is writable. If stale `.lock.yml` files remain after deleting a workflow `.md`, remove them with `gh aw compile --purge`.

## Import and Include Issues

Import paths are relative to the repository root, for example `.github/workflows/shared/tools.md`; verify the expected files with `git status`. A workflow can import only one file from `.github/agents/`. If compilation hangs, check for circular imports and remove the cycle.

## Tool Configuration Issues

### GitHub Tools Not Available

Configure using `toolsets:` ([tools reference](/gh-aw/reference/github-tools/)):

```yaml wrap
tools:
  github:
    toolsets: [repos, issues]
```

### Toolset Missing Expected Tools

Check [GitHub Toolsets](/gh-aw/reference/github-tools/), combine toolsets (`toolsets: [default, actions]`), or inspect with `gh aw mcp inspect <workflow>`.

### MCP Server Connection Failures

Verify package installation, syntax, and environment variables:

```yaml
mcp-servers:
  my-server:
    command: "npx"
    args: ["@myorg/mcp-server"]
    env:
      API_KEY: "${{ secrets.MCP_API_KEY }}"
```

### OpenCode/Crush MCP Tools Not Being Called

When integrating OpenCode-compatible engines (such as `crush`), runs can complete without ever invoking MCP or file tools. Use this `.crush.json`. Port `10004` is the local AWF API proxy port (with `--enable-api-proxy`); `MCP_GATEWAY_PORT` and `MCP_GATEWAY_API_KEY` are expanded from workflow env at runtime (substitute concrete values when running outside a workflow):

```json
{
  "provider": {
    "copilot-proxy": {
      "name": "Copilot Proxy",
      "type": "openai-compatible",
      "baseURL": "http://host.docker.internal:10004",
      "models": ["gpt-4.1", "claude-sonnet-4-6"]
    }
  },
  "model": "copilot-proxy/claude-sonnet-4-6",
  "mcp": {
    "safeoutputs": {
      "type": "http",
      "url": "http://host.docker.internal:${MCP_GATEWAY_PORT}/mcp/safeoutputs",
      "headers": { "Authorization": "${MCP_GATEWAY_API_KEY}" },
      "disabled": false,
      "timeout": 30000
    }
  },
  "agent": {
    "build": {
      "permission": {
        "bash": "allow", "edit": "allow", "read": "allow",
        "glob": "allow", "grep": "allow", "write": "allow",
        "external_directory": "allow"
      }
    }
  }
}
```

Key gotchas:

Crush/OpenCode does not auto-discover MCP servers, so declare an explicit top-level `mcp` block with routed URLs such as `http://host.docker.internal:${MCP_GATEWAY_PORT}/mcp/<server-name>`. Use `agent.build.permission` (singular); `permissions` is silently ignored and leaves tools unavailable. `external_directory` defaults to `ask`, which becomes an implicit deny in non-interactive runs, so set it to `allow` only when you truly need access outside the workspace.

For direct Copilot endpoints (`api.githubcopilot.com`), do **not** append `/v1`. For other OpenAI-compatible providers, use the provider's documented base path so `/chat/completions` is appended correctly. Keep the local proxy URL (`http://host.docker.internal:10004`) unchanged.

When using `--enable-api-proxy`, pass `COPILOT_GITHUB_TOKEN` in the execute step's `env:` so the proxy can authenticate:

```yaml wrap
- name: Execute
  env:
    COPILOT_GITHUB_TOKEN: ${{ steps.copilot-token.outputs.token }}
  run: |
    awf --enable-api-proxy <workflow-args> -- crush run "<prompt>"
```

### Playwright Network Access Denied

Add domains to `network.allowed`:

```yaml wrap
network:
  allowed:
    - github.com
    - "*.github.io"
```

### Cannot Find Module 'playwright'

`Error: Cannot find module 'playwright'` — Playwright is provided as MCP tools, not as an npm package. Use the MCP tools instead of `require('playwright')`:

```javascript
// ❌ Don't: const playwright = require('playwright')
// ✅ Do: use MCP tools
await mcp__playwright__browser_navigate({ url: "https://example.com" });
await mcp__playwright__browser_snapshot();
```

See [Playwright Tool documentation](/gh-aw/reference/tools/#playwright-tool-playwright) for all available tools.

### Playwright MCP Initialization Failure (EOF Error)

`Failed to register tools error="initialize: EOF" name=playwright` — Chromium crashes before tool registration completes due to missing Docker security flags. Upgrade to 0.41.0+ with `gh extension upgrade gh-aw`.

## Permission Issues

### Write Operations Fail

All writes (issues, comments, PR updates) must go through the `safe-outputs` system — declare the types your workflow needs in frontmatter:

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[bot] "
    labels: [automation]
  add-comment:      # no configuration required; uses defaults
  update-issue:     # no configuration required; uses defaults
```

If your operation isn't in the [Safe Outputs reference](/gh-aw/reference/safe-outputs/), it may not be supported yet. See the [Safe Outputs Specification](/gh-aw/specs/safe-outputs-specification/) for the full list.

### Safe Outputs Not Creating Issues

Disable staged mode:

```yaml wrap
safe-outputs:
  staged: false
  create-issue:
    title-prefix: "[bot] "
    labels: [automation]
```

### Project Field Type Errors

GitHub Projects reserves field names like `REPOSITORY`. Use alternatives (`repo`, `source_repository`, `linked_repo`):

```yaml wrap
# ❌ Wrong: repository
# ✅ Correct: repo
safe-outputs:
  update-project:
    fields:
      repo: "myorg/myrepo"
```

Delete conflicting fields in Projects UI and recreate.

## Engine-Specific Issues

- **Copilot CLI not found:** verify compilation succeeded — compiled workflows include CLI installation steps.
- **Model not available:** use the default (`engine: copilot`) or specify an available model (`engine: {id: copilot, model: gpt-4}`).

### Copilot License or Inference Access Issues

If a workflow fails at the Copilot inference step despite a correctly configured `COPILOT_GITHUB_TOKEN` (authentication or quota errors), the PAT owner may lack a valid Copilot license or inference access. Test locally with the [Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli):

```bash
export COPILOT_GITHUB_TOKEN="<your-github-pat>"
copilot -p "write a haiku"
```

If this fails, contact your organization administrator to enable Copilot for the token owner.

> [!NOTE]
> `COPILOT_GITHUB_TOKEN` must belong to a user account with an active Copilot subscription. Org-managed licenses may impose additional restrictions on programmatic API access.

## GitHub Enterprise Server Issues

> [!TIP]
> For a complete walkthrough of setting up and debugging workflows on **GHE Cloud with data residency** (`*.ghe.com`), see [Debugging GHE Cloud with Data Residency](/gh-aw/troubleshooting/debug-ghe/).

### Copilot Engine Prerequisites on GHES

Before running Copilot-based workflows on GHES, verify:

- **Site admin:** GitHub Connect enabled (links GHES to github.com for Copilot cloud services), enterprise-level Copilot licensing activated, and outbound HTTPS allowed to `api.githubcopilot.com` and `api.enterprise.githubcopilot.com`.
- **Enterprise/org admin:** a Copilot seat assigned to the `COPILOT_GITHUB_TOKEN` owner, and the org Copilot policy permits usage.
- **Workflow config:**

  ```aw wrap
  engine:
    id: copilot
    api-target: api.enterprise.githubcopilot.com
  network:
    allowed:
      - defaults
      - api.enterprise.githubcopilot.com
  ```

See [Enterprise API Endpoint](/gh-aw/reference/engines/#enterprise-api-endpoint-api-target) for GHEC/GHES `api-target` values.

### Copilot GHES: Common Error Messages

| Error | Cause | Fix |
|-------|-------|-----|
| `Error loading models: 400 Bad Request` | Enterprise Copilot not licensed or GitHub Connect not enabled | Enable GitHub Connect and enterprise Copilot in site admin settings |
| `403 "unauthorized: not licensed to use Copilot"` | No Copilot seat for PAT owner | Site admin enables Copilot; org admin assigns a seat to the token owner |
| `403 "Resource not accessible by personal access token"` | Wrong token type or missing permissions | Use fine-grained PAT with **Copilot Requests: Read**, or classic PAT with `copilot` scope — see [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token) |
| `Could not resolve to a Repository` | `GH_HOST` not set in custom jobs | Recompile (`gh aw compile`), or set `GH_HOST=github.company.com` explicitly for local CLI commands |
| Firewall blocking `api.<ghes-host>` | Domain not in allowed list | Add to `network.allowed` (see below) |
| `gh aw add-wizard` creates PR on github.com | Not inside a GHES repo clone | Run from within GHES repo, or use `gh aw add` + `gh pr create` |

For firewall issues, add the GHES domain to your workflow's allowed list:

```aw wrap
engine:
  id: copilot
  api-target: api.company.ghe.com
network:
  allowed:
    - defaults
    - company.ghe.com
    - api.company.ghe.com
```

## Context Expression Issues

- **Unauthorized expression:** use only [allowed expressions](/gh-aw/reference/templating/) (`github.event.issue.number`, `github.repository`, `steps.sanitized.outputs.text`). `secrets.*` and `env.*` are disallowed.
- **Sanitized context empty:** `steps.sanitized.outputs.text` requires issue/PR/comment events (`on: issues:`), not `push:` or similar triggers.

## Build and Test Issues

- **Documentation build fails:** clean install (`cd docs && rm -rf node_modules package-lock.json && npm install && npm run build`) and check for malformed frontmatter, MDX syntax errors, or broken links.
- **Tests failing after changes:** run `make fmt && make lint && make test-unit` before iterating.

## Network and Connectivity Issues

### Firewall Denials for Package Registries

Add ecosystem identifiers ([Network Configuration Guide](/gh-aw/guides/network-configuration/)):

```yaml wrap
network:
  allowed:
    - defaults    # Infrastructure
    - python      # PyPI
    - node        # npm
    - containers  # Docker
    - go          # Go modules
```

### Other Network Issues

If URLs appear as `(redacted)`, add the relevant domains to the allowed list ([Network Permissions](/gh-aw/reference/network/)), for example `allowed: [defaults, "api.example.com"]`. If remote imports fail to download, verify both network access (`curl -I https://raw.githubusercontent.com/github/gh-aw/main/README.md`) and authentication (`gh auth status`). For MCP server timeouts, prefer local servers such as `command: "node"` with `args: ["./server.js"]`.

## Cache Issues

If a cache is not restoring, make sure the key pattern matches; caches expire after 7 days, for example `cache: { key: deps-${{ hashFiles('package-lock.json') }}, restore-keys: deps- }`. If cache memory is not persisting, configure the cache-memory MCP server with a key such as `tools.cache-memory.key: memory-${{ github.workflow }}-${{ github.run_id }}`.

## Integrity Filtering Blocking Expected Content

On public repositories, `min-integrity: approved` is applied automatically — restricting agent visibility to content from owners, members, and collaborators. As a result, workflows can't see issues, PRs, or comments from external contributors, and triage workflows don't process community contributions.

To allow all contributors (only safe when the workflow validates input and uses restrictive safe outputs):

```yaml wrap
tools:
  github:
    min-integrity: none
```

Use `min-integrity: unapproved` as a middle ground for community triage workflows. See [Integrity Filtering](/gh-aw/reference/integrity/) for details.

## Workflow Failures and Debugging

### Timeout Errors

GitHub Actions marks the run as `timed_out` when the job exceeds `timeout-minutes` (default: 20 min). The table below maps each engine's error patterns to the right fix; after updating frontmatter, recompile with `gh aw compile`. See [Long Build Times](/gh-aw/reference/sandbox/#long-build-times) for caching strategies and self-hosted runner recommendations.

| Engine | Error Pattern | Fix Setting |
|--------|--------------|-------------|
| All | `The job has exceeded the maximum execution time of N minutes` | `timeout-minutes: N` in frontmatter |
| Claude | `Bash tool timed out after 60 seconds` | `tools: timeout: N` (default: 60s) |
| Claude | `Reached maximum number of turns (N). Stopping.` | `max-turns: N` |
| Codex | `Tool call timed out after 120 seconds` | `tools: timeout: N` (default: 120s) |
| Copilot | *(task incomplete, workflow succeeds)* | `max-continuations: N` |
| Any | `Failed to register tools error="initialize: timeout"` | `tools: startup-timeout: N` |

```yaml wrap
timeout-minutes: 60      # job-level limit
tools:
  timeout: 600           # per-tool-call limit (seconds)
  startup-timeout: 300   # MCP server startup limit (seconds)
max-turns: 30            # Claude: max turns
max-continuations: 5     # Copilot: autopilot continuations
```

### Why Did My Workflow Fail?

Common causes include missing tokens, permission mismatches, network restrictions, disabled tools, and rate limits. The quickest path is usually to give an agent the run URL so it can inspect logs and suggest a fix.

Using Copilot Chat (requires [agentic authoring setup](/gh-aw/guides/agentic-authoring/#configuring-your-repository)):

```text wrap
agentic-workflows debug https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

Using any coding agent (no setup required):

```text wrap
Debug this workflow run using https://raw.githubusercontent.com/github/gh-aw/main/debug.md
The failed workflow run is at https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

For manual investigation, use `gh aw audit <run-id>` and `gh aw logs`, then inspect the generated `.lock.yml`. See the [Debugging Workflows](/gh-aw/troubleshooting/debugging/) guide for a full walkthrough.

### Enable Debug Logging

Enable verbose mode (`--verbose`), set `ACTIONS_STEP_DEBUG = true`, or inspect MCP config (`gh aw mcp inspect`). The `DEBUG` environment variable activates detailed internal logging for any `gh aw` command — output goes to `stderr` and each line shows the namespace (`workflow:compiler`), message, and time since the previous entry. Common namespaces: `cli:compile_command`, `workflow:compiler`, `workflow:expression_extraction`, `parser:frontmatter`. Wildcards match any suffix.

```bash
DEBUG=* gh aw compile                              # all logs
DEBUG=workflow:* gh aw compile my-workflow         # specific package
DEBUG=workflow:*,cli:* gh aw compile my-workflow   # multiple packages
DEBUG=*,-workflow:test gh aw compile my-workflow   # exclude a logger
DEBUG_COLORS=0 DEBUG=* gh aw compile 2>&1 | tee debug.log  # capture to file
```

## Operational Runbooks

See [Workflow Health Monitoring Runbook](https://github.com/github/gh-aw/blob/main/.github/aw/runbooks/workflow-health.md) for diagnosing errors.

## Getting Help

Review [reference docs](/gh-aw/reference/workflow-structure/), search [existing issues](https://github.com/github/gh-aw/issues), or create an issue. See [Error Reference](/gh-aw/troubleshooting/errors/) and [Frontmatter Reference](/gh-aw/reference/frontmatter/).
