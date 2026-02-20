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

Configure using `toolsets:` ([tools reference](/gh-aw/reference/tools/)):

```yaml wrap
tools:
  github:
    toolsets: [repos, issues]
```

### Toolset Missing Expected Tools

Check [GitHub Toolsets](/gh-aw/reference/tools/#github-toolsets), combine toolsets (`toolsets: [default, actions]`), or inspect with `gh aw mcp inspect <workflow>`.

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

### Playwright Network Access Denied

Add domains to `allowed_domains`:

```yaml wrap
tools:
  playwright:
    allowed_domains: ["github.com", "*.github.io"]
```

### Cannot Find Module 'playwright'

**Error:**

```text
Error: Cannot find module 'playwright'
```

**Cause:** The agent tried to `require('playwright')` but Playwright is provided through MCP tools, not as an npm package.

**Solution:** Use MCP Playwright tools:

```javascript
// ❌ INCORRECT - This won't work
const playwright = require('playwright');
const browser = await playwright.chromium.launch();

// ✅ CORRECT - Use MCP Playwright tools
// Example: Navigate and take screenshot
await mcp__playwright__browser_navigate({
  url: "https://example.com"
});

await mcp__playwright__browser_snapshot();

// Example: Execute custom Playwright code
await mcp__playwright__browser_run_code({
  code: `async (page) => {
    await page.setViewportSize({ width: 390, height: 844 });
    const title = await page.title();
    return { title, url: page.url() };
  }`
});
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

Use safe outputs or request new safe output types.

### Safe Outputs Not Creating Issues

Disable staged mode:

```yaml wrap
safe-outputs:
  staged: false
  create-issue:
    title-prefix: "[bot] "
    labels: [automation]
```

### Token Permission Errors

Grant permissions or use a custom token:

```yaml wrap
permissions:
  contents: write
  issues: write

# Alternative: custom token
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}
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

**Diagnosis**: Verify that the account associated with the `COPILOT_GITHUB_TOKEN` can successfully run inference by testing it locally.

1. Install the Copilot CLI locally by following the [GitHub Copilot CLI documentation](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli).

2. Export the token as an environment variable:

   ```bash
   export COPILOT_GITHUB_TOKEN="<your-github-pat>"
   ```

3. Run a simple inference test:

   ```bash
   copilot -p "write a haiku"
   ```

If this command fails, the account associated with the token does not have a valid Copilot license or inference access. Contact your organization administrator to verify that the token owner has an active Copilot subscription with inference enabled.

> [!NOTE]
> The `COPILOT_GITHUB_TOKEN` must belong to a user account with an active GitHub Copilot subscription. Organization-managed Copilot licenses may have additional restrictions on programmatic API access.

## Context Expression Issues

### Unauthorized Expression

Use only [allowed expressions](/gh-aw/reference/templating/) (`github.event.issue.number`, `github.repository`, `needs.activation.outputs.text`). Disallowed: `secrets.*`, `env.*`.

### Sanitized Context Empty

`needs.activation.outputs.text` requires issue/PR/comment events (`on: issues:`), not `push:` or similar triggers.

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

## GitHub Lockdown Mode Blocking Expected Content

Lockdown mode filters public repository content to show only items from users with push access.

### Symptoms

Workflows can't see issues/PRs/comments from external contributors, status reports miss activity, triage workflows don't process community contributions.

### Cause

Lockdown is enabled by default for public repositories to protect against untrusted input.

### Solution

**Option 1: Keep Lockdown Enabled (Recommended)**

For sensitive operations (code generation, repository updates, web access), use separate workflows, manual triggers, or approval stages.

**Option 2: Disable Lockdown (Safe Public Workflows Only)**

Disable only if your workflow validates input, uses restrictive safe outputs, and doesn't access secrets:

```yaml wrap
tools:
  github:
    lockdown: false
```

Safe use cases: issue triage, spam detection, public dashboards with permission verification.

See [Lockdown Mode](/gh-aw/reference/lockdown-mode/) for details.

## Workflow Failures and Debugging

### Why Did My Workflow Fail?

Common causes: missing tokens, permission mismatches, network restrictions, disabled tools, or rate limits. Use `gh aw audit <run-id>` to investigate.

### How Do I Debug a Failing Workflow?

Check logs (`gh aw logs`), audit run (`gh aw audit <run-id>`), inspect `.lock.yml`, use Copilot Chat (`/agent agentic-workflows debug`), or watch compilation (`gh aw compile --watch`).

### Debugging Strategies

Enable verbose mode (`--verbose`), set `ACTIONS_STEP_DEBUG = true`, check MCP config (`gh aw mcp inspect`), and review logs.

## Operational Runbooks

See [Workflow Health Monitoring Runbook](https://github.com/github/gh-aw/blob/main/.github/aw/runbooks/workflow-health.md) for diagnosing errors.

## Getting Help

Review [reference docs](/gh-aw/reference/workflow-structure/), search [existing issues](https://github.com/github/gh-aw/issues), or create an issue. See [Error Reference](/gh-aw/troubleshooting/errors/) and [Frontmatter Reference](/gh-aw/reference/frontmatter/).
