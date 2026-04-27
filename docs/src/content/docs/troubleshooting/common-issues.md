---
title: Common Issues
description: Frequently encountered issues when working with GitHub Agentic Workflows and their solutions.
sidebar:
  order: 200
---

This reference documents frequently encountered issues when working with GitHub Agentic Workflows, organized by workflow stage and component.

## Installation Issues

### Extension Installation Fails

If `gh extension install github/gh-aw` fails, use the standalone installer (works in Codespaces and restricted networks):

```bash wrap
curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash
```

For specific versions, pass the tag as an argument ([see releases](https://github.com/github/gh-aw/releases)):

```bash wrap
curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash -s -- v0.40.0
```

Verify with `gh extension list`.

## Organization Policy Issues

### Custom Actions Not Allowed in Enterprise Organizations

**Error Message:**

```text
The action github/gh-aw/actions/setup@a933c835b5e2d12ae4dead665a0fdba420a2d421 is not allowed in {ORG} because all actions must be from a repository owned by your enterprise, created by GitHub, or verified in the GitHub Marketplace.
```

**Cause:** Enterprise policies restrict which GitHub Actions can be used. Workflows use `github/gh-aw/actions/setup` which may not be allowed.

**Solution:** Enterprise administrators must allow `github/gh-aw` in the organization's action policies:

#### Option 1: Allow Specific Repositories (Recommended)

Add `github/gh-aw` to your organization's allowed actions list:

1. Navigate to your organization's settings: `https://github.com/organizations/YOUR_ORG/settings/actions`
2. Under **Policies**, select **Allow select actions and reusable workflows**
3. In the **Allow specified actions and reusable workflows** section, add:
   ```text
   github/gh-aw@*
   ```
4. Save the changes

See GitHub's docs on [managing Actions permissions](https://docs.github.com/en/organizations/managing-organization-settings/disabling-or-limiting-github-actions-for-your-organization#allowing-select-actions-and-reusable-workflows-to-run).

#### Option 2: Configure Organization-Wide Policy File

Add `github/gh-aw@*` to your centralized `policies/actions.yml` and commit to your organization's `.github` repository. See GitHub's docs on [community health files](https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/creating-a-default-community-health-file).

```yaml
allowed_actions:
  - "actions/*"
  - "github/gh-aw@*"
```

#### Verification

Wait a few minutes for policy propagation, then re-run your workflow. If issues persist, verify at `https://github.com/organizations/YOUR_ORG/settings/actions`.

> [!TIP]
> The gh-aw actions are open source at [github.com/github/gh-aw/tree/main/actions](https://github.com/github/gh-aw/tree/main/actions) and pinned to specific SHAs for security.

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

### Workflow Won't Compile

Check YAML frontmatter syntax (indentation, colons with spaces), verify required fields (`on:`), and ensure types match the schema. Use `gh aw compile --verbose` for details.

### Lock File Not Generated

Fix compilation errors (`gh aw compile 2>&1 | grep -i error`) and verify write permissions on `.github/workflows/`.

### Orphaned Lock Files

Remove old `.lock.yml` files with `gh aw compile --purge` after deleting `.md` workflow files.

## Import and Include Issues

### Import File Not Found

Import paths are relative to repository root. Verify with `git status` (e.g., `.github/workflows/shared/tools.md`).

### Multiple Agent Files Error

Import only one `.github/agents/` file per workflow.

### Circular Import Dependencies

Compilation hangs indicate circular imports. Remove circular references.

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

When integrating OpenCode-compatible engines (such as `crush`) in AWF workflows (including smoke tests), runs can complete while never calling MCP tools or file tools.

Use this `.crush.json` structure. Port `10004` is the default local AWF API proxy port (used with `--enable-api-proxy` for OpenCode/Crush-compatible routing), while `MCP_GATEWAY_PORT` is a placeholder for the MCP gateway port.

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
        "bash": "allow",
        "edit": "allow",
        "read": "allow",
        "glob": "allow",
        "grep": "allow",
        "write": "allow",
        "external_directory": "allow"
      }
    }
  }
}
```

`MCP_GATEWAY_PORT` and `MCP_GATEWAY_API_KEY` are placeholders that are expanded from workflow environment variables when AWF renders the config at runtime. When running outside workflow context (such as local development), replace them with concrete values before writing `.crush.json`.

Key gotchas:

- Crush/OpenCode do not auto-discover MCP servers. Add an explicit top-level `mcp` section.
- Use routed gateway URLs: `http://host.docker.internal:${MCP_GATEWAY_PORT}/mcp/<server-name>`.
- ⚠️ Use `agent.build.permission` (singular). Using `permissions` (plural) is silently ignored by OpenCode-compatible config loaders, which leaves tools unavailable even though the run continues.
- In non-interactive mode (such as when running `crush run` in CI or AWF workflows), `external_directory` defaults to `ask`, which becomes an implicit deny without terminal prompts. Set it to `allow` only when the agent must access paths outside its primary workspace, such as `/tmp` or mounted external directories.
- For direct Copilot-compatible endpoints (`api.githubcopilot.com`), do not append `/v1` to the base URL. For other OpenAI-compatible providers, use the provider's expected base path (for example `https://models.inference.ai.azure.com`) so the client can append `/chat/completions` correctly.
- If you route through the local proxy (`http://host.docker.internal:10004`), keep the proxy URL as-is.
- When running through AWF `--enable-api-proxy`, provide `COPILOT_GITHUB_TOKEN` in the same execute step `env:` so the proxy can authenticate.

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

**Error:**

```text
Error: Cannot find module 'playwright'
```

**Cause:** Playwright is provided as MCP tools, not as an npm package.

**Solution:** Use MCP tools instead of `require('playwright')`:

```javascript
// ❌ Don't: const playwright = require('playwright')
// ✅ Do: use MCP tools
await mcp__playwright__browser_navigate({ url: "https://example.com" });
await mcp__playwright__browser_snapshot();
```

See [Playwright Tool documentation](/gh-aw/reference/tools/#playwright-tool-playwright) for all available tools.

### Playwright MCP Initialization Failure (EOF Error)

**Error:**

```text
Failed to register tools error="initialize: EOF" name=playwright
```

**Cause:** Chromium crashes before tool registration completes due to missing Docker security flags.

**Solution:** Upgrade to version 0.41.0+ which includes required Docker flags:

```bash wrap
gh extension upgrade gh-aw
```

## Permission Issues

### Write Operations Fail

Agentic workflows cannot write to GitHub directly. All writes (issues, comments, PR updates)
must go through the `safe-outputs` system, which validates and executes write operations on
behalf of the workflow.

Ensure your workflow frontmatter declares the safe output types it needs:

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[bot] "
    labels: [automation]
  add-comment:      # no configuration required; uses defaults
  update-issue:     # no configuration required; uses defaults
```

If the operation you need is not listed in the [Safe Outputs reference](/gh-aw/reference/safe-outputs/),
it may not be supported yet. See the [Safe Outputs Specification](/gh-aw/reference/safe-outputs-specification/)
for the full list of available output types and their configuration options.

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

### Copilot CLI Not Found

Verify compilation succeeded. Compiled workflows include CLI installation steps.

### Model Not Available

Use default (`engine: copilot`) or specify available model (`engine: {id: copilot, model: gpt-4}`).

### Copilot License or Inference Access Issues

If your workflow fails during the Copilot inference step even though the `COPILOT_GITHUB_TOKEN` secret is configured correctly, the PAT owner's account may not have the necessary Copilot license or inference access.

**Symptoms**: The workflow fails with authentication or quota errors when the Copilot CLI tries to generate a response.

**Diagnosis**: Test locally by installing the [Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) and running:

```bash
export COPILOT_GITHUB_TOKEN="<your-github-pat>"
copilot -p "write a haiku"
```

If this fails, the token owner lacks a valid Copilot license or inference access. Contact your organization administrator to enable it.

> [!NOTE]
> The `COPILOT_GITHUB_TOKEN` must belong to a user account with an active GitHub Copilot subscription. Organization-managed Copilot licenses may have additional restrictions on programmatic API access.

## GitHub Enterprise Server Issues

> [!TIP]
> For a complete walkthrough of setting up and debugging workflows on **GHE Cloud with data residency** (`*.ghe.com`), see [Debugging GHE Cloud with Data Residency](/gh-aw/troubleshooting/debug-ghe/).

### Copilot Engine Prerequisites on GHES

Before running Copilot-based workflows on GHES, verify the following:

**Site admin requirements:**
- **GitHub Connect** must be enabled — it connects GHES to github.com for Copilot cloud services.
- **Copilot licensing** must be purchased and activated at the enterprise level.
- The firewall must allow outbound HTTPS to `api.githubcopilot.com` and `api.enterprise.githubcopilot.com`.

**Enterprise/org admin requirements:**
- Copilot seats must be assigned to the user whose PAT is used as `COPILOT_GITHUB_TOKEN`.
- The organization's Copilot policy must allow Copilot usage.

**Workflow configuration:**

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

### Unauthorized Expression

Use only [allowed expressions](/gh-aw/reference/templating/) (`github.event.issue.number`, `github.repository`, `steps.sanitized.outputs.text`). Disallowed: `secrets.*`, `env.*`.

### Sanitized Context Empty

`steps.sanitized.outputs.text` requires issue/PR/comment events (`on: issues:`), not `push:` or similar triggers.

## Build and Test Issues

### Documentation Build Fails

Clean install and rebuild:

```bash wrap
cd docs
rm -rf node_modules package-lock.json
npm install
npm run build
```

Check for malformed frontmatter, MDX syntax errors, or broken links.

### Tests Failing After Changes

Format and lint before testing:

```bash wrap
make fmt
make lint
make test-unit
```

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

### URLs Appearing as "(redacted)"

Add domains to allowed list ([Network Permissions](/gh-aw/reference/network/)):

```yaml wrap
network:
  allowed:
    - defaults
    - "api.example.com"
```

### Cannot Download Remote Imports

Verify network (`curl -I https://raw.githubusercontent.com/github/gh-aw/main/README.md`) and auth (`gh auth status`).

### MCP Server Connection Timeout

Use local servers (`command: "node"`, `args: ["./server.js"]`).

## Cache Issues

### Cache Not Restoring

Verify key patterns match (caches expire after 7 days):

```yaml wrap
cache:
  key: deps-${{ hashFiles('package-lock.json') }}
  restore-keys: deps-
```

### Cache Memory Not Persisting

Configure cache for memory MCP server:

```yaml wrap
tools:
  cache-memory:
    key: memory-${{ github.workflow }}-${{ github.run_id }}
```

## Integrity Filtering Blocking Expected Content

For public repositories, `min-integrity: approved` is applied automatically, restricting agent visibility to content from owners, members, and collaborators. This is intentional for sensitive operations.

**Symptoms**: Workflows can't see issues/PRs/comments from external contributors, triage workflows don't process community contributions.

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

Common causes: missing tokens, permission mismatches, network restrictions, disabled tools, or rate limits.

**Fastest path**: ask an agent with the run URL — it audits logs, identifies the root cause, and suggests fixes.

Using Copilot Chat (requires [agentic authoring setup](/gh-aw/guides/agentic-authoring/#configuring-your-repository)):

```text wrap
/agent agentic-workflows debug https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

Using any coding agent (no setup required):

```text wrap
Debug this workflow run using https://raw.githubusercontent.com/github/gh-aw/main/debug.md
The failed workflow run is at https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

For manual investigation: `gh aw audit <run-id>`, `gh aw logs`, inspect `.lock.yml`. See the [Debugging Workflows](/gh-aw/troubleshooting/debugging/) guide for a full walkthrough.

### Enable Debug Logging

Enable verbose mode (`--verbose`), set `ACTIONS_STEP_DEBUG = true`, or check MCP config (`gh aw mcp inspect`). The `DEBUG` environment variable activates detailed internal logging for any `gh aw` command:

```bash
DEBUG=* gh aw compile                              # all logs
DEBUG=workflow:* gh aw compile my-workflow         # specific package
DEBUG=workflow:*,cli:* gh aw compile my-workflow   # multiple packages
DEBUG=*,-workflow:test gh aw compile my-workflow   # exclude a logger
DEBUG_COLORS=0 DEBUG=* gh aw compile 2>&1 | tee debug.log  # capture to file
```

Debug output goes to `stderr`. Each log line shows the namespace (`workflow:compiler`), message, and time elapsed since the previous entry. Common namespaces: `cli:compile_command`, `workflow:compiler`, `workflow:expression_extraction`, `parser:frontmatter`. Wildcards match any suffix (`workflow:*`).

## Operational Runbooks

See [Workflow Health Monitoring Runbook](https://github.com/github/gh-aw/blob/main/.github/aw/runbooks/workflow-health.md) for diagnosing errors.

## Getting Help

Review [reference docs](/gh-aw/reference/workflow-structure/), search [existing issues](https://github.com/github/gh-aw/issues), or create an issue. See [Error Reference](/gh-aw/troubleshooting/errors/) and [Frontmatter Reference](/gh-aw/reference/frontmatter/).
