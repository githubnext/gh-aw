---
title: GitHub Tokens
description: Comprehensive reference for all GitHub tokens used in gh-aw, including authentication, token precedence, and security best practices
sidebar:
  order: 650
---

GitHub Agentic Workflows authenticate using multiple tokens depending on the operation. This reference explains which token to use, when it's required, and how precedence works across different operations.

## Quick start: tokens you actually configure

GitHub Actions always provides `GITHUB_TOKEN` for you automatically.
For GitHub Agentic Workflows, you only need to create a few **optional** secrets in your own repo:

| When you need this…                                  | Secret to create                       | Notes |
|------------------------------------------------------|----------------------------------------|-------|
| Cross-repo Project Ops / remote GitHub tools         | `GH_AW_GITHUB_TOKEN`                   | PAT or app token with cross-repo access. |
| Copilot workflows (CLI, engine, agent tasks, etc.)   | `COPILOT_GITHUB_TOKEN`                 | Needs Copilot Requests permission and repo access. |
| Assigning agents/bots to issues or pull requests     | `GH_AW_AGENT_TOKEN`                    | Used by `assign-to-agent` and Copilot assignee/reviewer flows. |
| Any GitHub Projects v2 operations                    | `GH_AW_PROJECT_GITHUB_TOKEN`           | **Required** for `update-project`. Default `GITHUB_TOKEN` cannot access Projects v2 API. |
| Isolating MCP server permissions (advanced optional) | `GH_AW_GITHUB_MCP_SERVER_TOKEN`        | Only if you want MCP to use a different token than other jobs. |

Create these as **repository secrets in *your* repo**. The easiest way is to use the GitHub Agentic Workflows CLI:

```bash
# Current repository
gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_PAT"
gh aw secrets set COPILOT_GITHUB_TOKEN --value "YOUR_COPILOT_PAT"
gh aw secrets set GH_AW_AGENT_TOKEN --value "YOUR_AGENT_PAT"
gh aw secrets set GH_AW_PROJECT_GITHUB_TOKEN --value "YOUR_PROJECT_PAT"
```

After these are set, gh-aw will automatically pick the right token for each operation; you should not need per-workflow PATs in most cases.

### CLI helpers for tokens and secrets

- `gh aw secrets bootstrap` – checks which recommended token secrets (like `GH_AW_GITHUB_TOKEN`, `COPILOT_GITHUB_TOKEN`) exist in a repository and prints suggested scopes plus copy‑pasteable `gh aw secrets set` commands.
- `gh aw init --tokens --engine <engine>` – runs token checks as part of repository initialization for a specific engine (`copilot`, `claude`, `codex`).
- `gh aw secrets set <NAME>` – creates or updates a repository secret. Values can come from `--value`, `--value-from-env`, or stdin (for example, `echo "PAT" | gh aw secrets set NAME`).

### Security and scopes (least privilege)

- Use `permissions:` at the workflow or job level so `GITHUB_TOKEN` only has what that workflow needs (for example, read contents and write PRs, but nothing else):

```yaml
permissions:
  contents: read
  pull-requests: write
```

- When creating each PAT/App token above, grant access **only** to the repos and scopes required for its scenario (cross-repo Project Ops, Copilot, agents, or MCP) and nothing more.
- Only expose powerful secrets to the jobs that need them by scoping them to `env:` at the job or step level, not globally:

```yaml
jobs:
  project-ops:
    env:
      GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
```

- For very sensitive tokens, prefer GitHub Environments or organization-level secrets with required reviewers so only trusted workflows can use them.

## Token Overview

| Token | Type | Purpose | User Configurable |
|-------|------|---------|-------------------|
| `GITHUB_TOKEN` | Auto-provided | Default Actions token for current repository | No (auto-provided) |
| `GH_AW_GITHUB_TOKEN` | PAT | Enhanced token for cross-repo and remote GitHub tools | **Yes** (required for cross-repo) |
| `GH_AW_PROJECT_GITHUB_TOKEN` | PAT | Required token for GitHub Projects v2 operations | **Yes** (required for Projects v2) |
| `GH_AW_GITHUB_MCP_SERVER_TOKEN` | PAT | Custom token specifically for GitHub MCP server | **Yes** (optional override) |
| `COPILOT_GITHUB_TOKEN` | PAT | Copilot authentication (recommended) | **Yes** (required for Copilot) |
| `GH_AW_AGENT_TOKEN` | PAT | Agent assignment operations | **Yes** (required for agent ops) |
| `GITHUB_MCP_SERVER_TOKEN` | Auto-set | Automatically configured by compiler | No (auto-configured) |

## `GITHUB_TOKEN` (Default)

**Type**: Automatically provided by GitHub Actions

GitHub Actions automatically provides this token with scoped access to the current repository. It's used as a fallback when no custom token is configured.

**Capabilities**:

- Read and write access to current repository
- Default permissions based on workflow `permissions:` configuration
- No cost or setup required

**Limitations**:

- Cannot access other repositories
- Cannot trigger workflows via GitHub API
- Cannot assign bots (Copilot) to issues or PRs
- Cannot authenticate with Copilot engine
- Not supported for remote GitHub MCP server mode

**When to use**: Simple workflows that only need to interact with the current repository (comments, labels, issues in the same repo).

## `GH_AW_GITHUB_TOKEN` (Enhanced PAT)

**Type**: Personal Access Token (user must configure)

A fine-grained or classic Personal Access Token providing enhanced capabilities beyond `GITHUB_TOKEN`. This is the primary token for workflows that need cross-repository access or remote GitHub tools.

**Required for**:

- Cross-repository operations (accessing other repos)
- Remote GitHub tools mode (faster startup without Docker)
- Codex engine operations with GitHub MCP
- Any operation that needs to access multiple repositories

**Setup**:

1. Create a [fine-grained PAT](https://github.com/settings/personal-access-tokens/new) with:
   - Repository access: Select specific repos or "All repositories"
   - Permissions:
     - Contents: Read (minimum) or Read+Write (for PRs)
     - Issues: Read+Write (for issue operations)
     - Pull requests: Read+Write (for PR operations)

2. Add to repository secrets:

```bash wrap
gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_PAT"
```

**Token precedence** (highest to lowest):

1. Per-output `github-token:` (safe output level)
2. Global `safe-outputs.github-token:` (all outputs)
3. Workflow-level `github-token:` frontmatter (top-level)
4. Default fallback: `${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`

**Example**:

```yaml wrap
github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}  # Workflow-level

safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}  # Global override
  create-issue:
    target-repo: "org/other-repo"
    github-token: ${{ secrets.SPECIFIC_PAT }}  # Per-output override (highest priority)
```

## `GH_AW_GITHUB_MCP_SERVER_TOKEN` (GitHub MCP Server)

**Type**: Personal Access Token (optional override)

A specialized token for the GitHub MCP server that takes precedence over the standard token fallback chain. Use this when you want to provide different permissions specifically for GitHub MCP server operations versus other workflow operations.

**When to use**:

- You need different permission levels for MCP server vs. other operations
- You want to isolate MCP server authentication from general workflow authentication
- You're using remote GitHub MCP mode and need a token with specific scopes

**Setup**:

```bash wrap
gh aw secrets set GH_AW_GITHUB_MCP_SERVER_TOKEN --value "YOUR_PAT"
```

**Token precedence** for GitHub MCP server (highest to lowest):

1. Custom token (tool-level `github-token`)
2. Top-level `github-token` (frontmatter)
3. `${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}`
4. `${{ secrets.GH_AW_GITHUB_TOKEN }}`
5. `${{ secrets.GITHUB_TOKEN }}`

The compiler automatically sets the `GITHUB_MCP_SERVER_TOKEN` environment variable using this precedence:

```yaml wrap
env:
  GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
```

This token is passed to the GitHub MCP server as:

- `GITHUB_PERSONAL_ACCESS_TOKEN` environment variable (local/Docker mode)
- `Authorization: Bearer` header (remote mode)

:::note
In most cases, you don't need to set this token separately. Use `GH_AW_GITHUB_TOKEN` instead, which works for both general operations and GitHub MCP server.
:::

## `GH_AW_PROJECT_GITHUB_TOKEN` (GitHub Projects v2)

**Type**: Personal Access Token (required for Projects v2 operations)

A specialized token for GitHub Projects v2 operations used by the [`update-project`](/gh-aw/reference/safe-outputs/#project-board-updates-update-project) safe output. **Required** because the default `GITHUB_TOKEN` cannot access the GitHub Projects v2 GraphQL API.

**When to use**:

- **Always required** for any Projects v2 operations (creating, updating, or reading project boards)
- The default `GITHUB_TOKEN` cannot create or manage ProjectV2 objects via GraphQL
- You want to isolate Projects permissions from other workflow operations

**Setup**:

The required token type depends on whether you're working with **user-owned** or **organization-owned** Projects:

**For User-owned Projects (v2)**:

You **must** use a **classic PAT** with the `project` scope. Fine-grained PATs do **not** work with user-owned Projects.

1. Create a [classic PAT](https://github.com/settings/tokens/new) with scopes:
   - `project` (required for user Projects)
   - `repo` (required if accessing private repositories)

**For Organization-owned Projects (v2)**:

You can use either a classic PAT or a fine-grained PAT:

1. **Option A**: Create a **classic PAT** with `project` and `read:org` scopes:
   - `project` (required)
   - `read:org` (required for org Projects)
   - `repo` (required if accessing private repositories)

2. **Option B (recommended)**: Create a [fine-grained PAT](https://github.com/settings/personal-access-tokens/new) with:
   - **Repository access**: Select specific repos that will use the workflow
   - **Repository permissions**:
     - Contents: Read
     - Issues: Read (if needed for issue-triggered workflows)
     - Pull requests: Read (if needed for PR-triggered workflows)
   - **Organization permissions** (must be explicitly granted):
     - Projects: Read & Write (required for updating org Projects)
   - **Important**: You must explicitly grant organization access during token creation

3. **Option C**: Use a GitHub App with Projects: Read+Write permission

After creating your token, add it to repository secrets:

```bash wrap
gh aw secrets set GH_AW_PROJECT_GITHUB_TOKEN --value "YOUR_PROJECT_PAT"
```

**Token precedence** for `update-project` safe output (highest to lowest):

1. Per-output `github-token:` (`safe-outputs.update-project.github-token`)
2. Top-level `github-token:` (frontmatter)
3. `${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}`
4. `${{ secrets.GITHUB_TOKEN }}` (lowest)

The compiler automatically sets the `GH_AW_PROJECT_GITHUB_TOKEN` environment variable in the update-project job using this precedence. This environment variable is used by the JavaScript implementation to provide helpful error messages when project operations fail.

**Example configuration**:

```yaml wrap
---
# Option 1: Use GH_AW_PROJECT_GITHUB_TOKEN secret (recommended for org Projects)
# Just create the secret - no workflow config needed
---

# Option 2: Explicitly configure at safe-output level
safe-outputs:
  update-project:
    github-token: ${{ secrets.CUSTOM_PROJECT_TOKEN }}

# Option 3: Organization projects with GitHub tools integration
tools:
  github:
    toolsets: [default, projects]
    github-token: ${{ secrets.ORG_PROJECT_WRITE }}
safe-outputs:
  update-project:
    github-token: ${{ secrets.ORG_PROJECT_WRITE }}
```

**For organization-owned projects**, the complete configuration should include both the GitHub tools and safe outputs using the same token with appropriate permissions.

:::note[Default behavior]
By default, `update-project` is **update-only**: it will not create projects. If a project doesn't exist, the job fails with instructions to create it manually.

**Important**: The default `GITHUB_TOKEN` **cannot** be used for Projects v2 operations. You **must** configure `GH_AW_PROJECT_GITHUB_TOKEN` or provide a custom token via `safe-outputs.update-project.github-token`. 

**GitHub Projects v2 PAT Requirements**:
- **User-owned Projects**: Require a **classic PAT** with the `project` scope (plus `repo` if accessing private repos). Fine-grained PATs do **not** work with user-owned Projects.
- **Organization-owned Projects**: Can use either a classic PAT with `project` + `read:org` scopes, **or** a fine-grained PAT with:
  - Repository access to specific repositories
  - Repository permissions: Contents: Read, Issues: Read, Pull requests: Read (as needed)
  - Organization permissions: Projects: Read & Write
  - Explicit organization access granted during token creation
- **GitHub App**: Works for both user and org Projects with Projects: Read+Write permission.

To opt-in to creating projects, the agent must include `create_if_missing: true` in its output, and the token must have sufficient permissions to create projects in the organization.
:::

:::tip[When to use vs GH_AW_GITHUB_TOKEN]
- Use `GH_AW_PROJECT_GITHUB_TOKEN` when you need **Projects-specific permissions** separate from other operations
- Use `GH_AW_GITHUB_TOKEN` as the top-level token if it already has Projects permissions and you don't need isolation
- The precedence chain allows the top-level token to be used if `GH_AW_PROJECT_GITHUB_TOKEN` isn't set
:::

## `COPILOT_GITHUB_TOKEN` (Copilot Authentication)

**Type**: Personal Access Token (user must configure)

The recommended token for all Copilot-related operations including the Copilot engine, agent task creation, and bot assignments.

**Required for**:

- `engine: copilot` workflows
- `create-agent-task:` safe outputs
- Assigning `copilot` as issue assignee
- Adding `copilot` as PR reviewer

**Setup**:

1. Create a [PAT](https://github.com/settings/personal-access-tokens/new) with:
   - Resource owner: Your user account (not organization)
   - Repository access: "Public repositories" or specific repos
   - Permissions: "Copilot Requests" (required)

2. Add to repository secrets:

```bash wrap
gh aw secrets set COPILOT_GITHUB_TOKEN --value "YOUR_COPILOT_PAT"
```

**Copilot token precedence** (highest to lowest):

1. Per-output `github-token:`
2. Global `safe-outputs.github-token:`
3. Workflow-level `github-token:`
4. `${{ secrets.COPILOT_GITHUB_TOKEN }}`
5. `${{ secrets.GH_AW_GITHUB_TOKEN }}` (legacy, deprecated)

:::caution[Legacy Tokens Removed]
The `COPILOT_CLI_TOKEN` and `GH_AW_COPILOT_TOKEN` secret names are **no longer supported** as of v0.26+. Use `COPILOT_GITHUB_TOKEN` instead.
:::

:::caution
The default `GITHUB_TOKEN` is **not** included in the Copilot fallback chain because it lacks the "Copilot Requests" permission required for Copilot operations.
:::

## `GH_AW_AGENT_TOKEN` (Agent Assignment)

**Type**: Personal Access Token (user must configure)

Specialized token for `assign-to-agent:` safe outputs that assign GitHub Copilot agents to issues or pull requests.

**Required for**:

- `assign-to-agent:` safe outputs
- Programmatic agent assignment operations

**Required permissions**:

- Actions: Write
- Contents: Write
- Issues: Write
- Pull requests: Write

**Setup**:

1. Create a [fine-grained PAT](https://github.com/settings/personal-access-tokens/new) with the above permissions
2. Add to repository secrets:

```bash wrap
gh aw secrets set GH_AW_AGENT_TOKEN --value "YOUR_AGENT_PAT"
```

**Token precedence** (highest to lowest):

1. Per-output `github-token:`
2. Global `safe-outputs.github-token:`
3. Workflow-level `github-token:`
4. `${{ secrets.GH_AW_AGENT_TOKEN }}` (no further fallback)

:::caution
Unlike other tokens, `GH_AW_AGENT_TOKEN` does **not** fall back to `GH_AW_GITHUB_TOKEN`. You must explicitly configure this token for agent assignment operations.
:::

## `GITHUB_MCP_SERVER_TOKEN` (Auto-configured)

**Type**: Automatically set by the compiler (do not configure manually)

This environment variable is automatically set by gh-aw based on your GitHub tools configuration and token precedence. You should never need to set this manually.

**How it works**:
When you configure GitHub tools in your workflow, the compiler automatically generates the appropriate token configuration:

```yaml wrap
tools:
  github:
    mode: remote
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
```

The compiler sets:

```yaml wrap
env:
  GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
```

:::note
This is an internal implementation detail. Configure tokens using `GH_AW_GITHUB_TOKEN`, `GH_AW_GITHUB_MCP_SERVER_TOKEN`, or workflow-level `github-token` instead.
:::

## Token Configuration Patterns

### Per-Output vs Global vs Workflow-Level

You can configure tokens at three levels with different precedence:

```yaml wrap
# Workflow-level (applies to all operations by default)
github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}

safe-outputs:
  # Global safe-outputs level (overrides workflow-level for all outputs)
  github-token: ${{ secrets.GLOBAL_PAT }}
  
  create-issue:
    # Per-output level (highest priority)
    github-token: ${{ secrets.ISSUE_PAT }}
    target-repo: "org/other-repo"
  
  create-pull-request:
    # Automatically uses Copilot token chain when copilot is reviewer
    reviewers: copilot
```

### Cross-Repository Operations

Cross-repository operations always require `GH_AW_GITHUB_TOKEN` or a custom PAT with access to the target repositories:

```yaml wrap
github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}

safe-outputs:
  create-issue:
    target-repo: "org/tracking-repo"  # Requires PAT with access to org/tracking-repo
  
  add-comment:
    target-repo: "org/another-repo"  # Requires PAT with access to org/another-repo
```

### Remote GitHub Tools Mode

Remote mode requires a PAT because the default `GITHUB_TOKEN` is not supported:

```yaml wrap
github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}  # Required for remote mode

tools:
  github:
    mode: remote  # Faster startup, no Docker required
    toolsets: [default]
```

### Copilot Operations

Copilot operations require a PAT with "Copilot Requests" permission:

```yaml wrap
engine: copilot

# Option 1: Configure COPILOT_GITHUB_TOKEN secret (recommended)
# No workflow configuration needed - automatically used

# Option 2: Explicitly configure in workflow
github-token: ${{ secrets.COPILOT_GITHUB_TOKEN }}
```

## GitHub App Tokens

GitHub App installation tokens provide enhanced security with short-lived, automatically-revoked credentials. This is the recommended approach for production workflows.

**Benefits**:

- **On-demand minting**: Tokens created at job start, minimizing exposure window
- **Short-lived**: Tokens automatically revoked at job end (even on failure)
- **Automatic permissions**: Compiler calculates required permissions based on safe outputs
- **Audit trail**: All actions logged under the GitHub App identity
- **No PAT rotation**: Eliminates need for manual token rotation

**Setup**:

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    owner: "my-org"                    # Optional: defaults to current repo owner
    repositories: ["repo1", "repo2"]   # Optional: defaults to current repo only
  
  create-issue:
    # No github-token needed - uses App token automatically
  
  create-pull-request:
    # Permissions computed based on safe output types
```

**Permission mapping**:

- `create-issue:` → Issues: Write
- `create-pull-request:` → Contents: Write, Pull requests: Write
- `add-comment:` → Issues: Write
- `add-labels:` → Issues: Write
- `update-issue:` → Issues: Write
- `create-agent-task:` → Actions: Write, Contents: Write

**Configuration inheritance**:
App configuration can be imported from shared workflows. Local configuration takes precedence:

```yaml wrap
imports:
  - shared/common-app.md  # Defines app: config

safe-outputs:
  app:
    repositories: ["repo3"]  # Overrides imported config
  create-issue:
```

## Token Selection Guide

Use this guide to choose the right token for your workflow:

| Scenario | Recommended Token | Alternative |
|----------|------------------|-------------|
| Single repository, basic operations | `GITHUB_TOKEN` (default) | None needed |
| Cross-repository operations | `GH_AW_GITHUB_TOKEN` | GitHub App |
| Copilot engine workflows | `COPILOT_GITHUB_TOKEN` | None |
| Remote GitHub MCP mode | `GH_AW_GITHUB_TOKEN` | GitHub App |
| Agent assignments | `GH_AW_AGENT_TOKEN` | `GH_AW_GITHUB_TOKEN` with elevated permissions |
| GitHub Projects v2 operations | `GH_AW_PROJECT_GITHUB_TOKEN` | `GH_AW_GITHUB_TOKEN` with Projects permissions |
| Production workflows | GitHub App | `GH_AW_GITHUB_TOKEN` with fine-grained PAT |
| Custom MCP server permissions | `GH_AW_GITHUB_MCP_SERVER_TOKEN` | Use `GH_AW_GITHUB_TOKEN` |

## Security Best Practices

### Principle of Least Privilege

Always use the minimal `permissions:` in your workflow and let safe outputs handle API access with tokens:

```yaml wrap
permissions:
  contents: read  # Minimal workflow permissions

safe-outputs:
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
  create-issue:  # Token handles API authentication, not workflow permissions
```

### Token Scoping

Scope different operations with different tokens when they need different permission levels:

```yaml wrap
safe-outputs:
  create-issue:
    github-token: ${{ secrets.READ_WRITE_PAT }}
    target-repo: "org/public-issues"
  
  create-pull-request:
    github-token: ${{ secrets.LIMITED_PAT }}
    target-repo: "org/code-repo"
```

### Prefer GitHub Apps

Use GitHub Apps for production workflows whenever possible:

- Better security (short-lived tokens)
- Better auditability (app identity in logs)
- No credential rotation needed
- Automatic permission management

### PAT Best Practices

When using Personal Access Tokens:

1. **Use fine-grained PATs** over classic PATs
2. **Set short expiration periods** (90 days or less)
3. **Implement rotation schedules** before expiration
4. **Limit repository access** to only what's needed
5. **Use separate tokens** for different permission levels
6. **Monitor token usage** in organization audit logs

### Avoid Common Pitfalls

**Don't**: Hardcode tokens in workflows

```yaml wrap
github-token: "ghp_xxxxxxxxxxxx"  # ❌ Never do this
```

**Do**: Use secrets

```yaml wrap
github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}  # ✅ Correct
```

**Don't**: Use overly permissive tokens

```yaml wrap
# ❌ Classic PAT with full repo access for simple issue creation
github-token: ${{ secrets.ADMIN_TOKEN }}
```

**Do**: Use appropriately scoped tokens

```yaml wrap
# ✅ Fine-grained PAT with Issues: Write only
github-token: ${{ secrets.ISSUE_TOKEN }}
```

## Common Workflow Examples

### Basic Single-Repository Workflow

Uses default `GITHUB_TOKEN` - no configuration needed:

```yaml wrap
---
engine: copilot
---

Analyze this issue and add a comment with recommendations.

---
safe-outputs:
  add-comment:
    body: "Analysis complete"
```

### Cross-Repository Issue Tracking

Requires `GH_AW_GITHUB_TOKEN` with access to target repository:

```yaml wrap
---
github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
---

Create tracking issue in the central repo.

---
safe-outputs:
  create-issue:
    target-repo: "org/tracking-repo"
    title: "Track progress on {{ github.repository }}"
```

### Copilot Agent Workflow

Requires `COPILOT_GITHUB_TOKEN` for Copilot operations:

```yaml wrap
---
engine: copilot
---

Review this PR and provide feedback.

---
safe-outputs:
  create-agent-task:
    title: "Review PR #{{ github.event.pull_request.number }}"
```

### Multi-Repository with Different Permissions

Uses different tokens for different permission levels:

```yaml wrap
---
github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
---

Coordinate across multiple repositories.

---
safe-outputs:
  create-issue:
    target-repo: "org/public-tracker"
    github-token: ${{ secrets.PUBLIC_PAT }}
  
  create-pull-request:
    target-repo: "org/private-code"
    github-token: ${{ secrets.PRIVATE_PAT }}
```

### Production Workflow with GitHub App

Most secure option using GitHub App tokens:

```yaml wrap
---
safe-outputs:
  app:
    app-id: ${{ vars.PRODUCTION_APP_ID }}
    private-key: ${{ secrets.PRODUCTION_APP_KEY }}
    repositories: ["repo1", "repo2", "repo3"]
---

Automated production workflow with enhanced security.

---
safe-outputs:
  create-issue:
  create-pull-request:
```

## Troubleshooting

### "Resource not accessible by integration"

**Cause**: The token being used lacks required permissions for the operation.

**Solutions**:

1. **For cross-repository operations**: Configure `GH_AW_GITHUB_TOKEN` with access to the target repository:

  ```bash wrap
  gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_PAT"
  ```

2. **For bot operations**: Use `COPILOT_GITHUB_TOKEN` with "Copilot Requests" permission:

  ```bash wrap
  gh aw secrets set COPILOT_GITHUB_TOKEN --value "YOUR_COPILOT_PAT"
  ```

3. **Check token permissions**: Verify your PAT has the required scopes:
   - Issues: Read+Write for issue operations
   - Pull requests: Read+Write for PR operations
   - Contents: Read+Write for code/PR creation
   - Copilot Requests: Required for Copilot operations

### Copilot Authentication Failures

**Error**: "Failed to create agent task" or "Cannot assign copilot to issue"

**Cause**: Missing or invalid `COPILOT_GITHUB_TOKEN`.

**Solution**:

1. Verify the secret exists:

   ```bash wrap
   gh secret list -a actions
   ```

2. Ensure PAT has correct configuration:
   - Resource owner: Your user account (not org)
   - Repository access: Appropriate repos
   - Permission: "Copilot Requests" (required)

3. Set or update the secret:

  ```bash wrap
  gh aw secrets set COPILOT_GITHUB_TOKEN --value "YOUR_PAT"
  ```

### Remote GitHub Tools Failures

**Error**: "Remote mode requires authentication"

**Cause**: Remote GitHub MCP mode does not support default `GITHUB_TOKEN`.

**Solution**: Configure `GH_AW_GITHUB_TOKEN` or use local mode:

**Option 1**: Add token (recommended for remote mode):

```bash wrap
gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_PAT"
```

**Option 2**: Switch to local mode (uses Docker):

```yaml wrap
tools:
  github:
    mode: local  # No PAT required, but slower startup
```

### Agent Assignment Failures

**Error**: "Failed to assign agent" or "Cannot assign copilot as reviewer"

**Cause**: Token lacks elevated permissions required for agent operations.

**Solution**: Configure `GH_AW_AGENT_TOKEN` with required permissions:

1. Create PAT with permissions:
   - Actions: Write
   - Contents: Write
   - Issues: Write
   - Pull requests: Write

2. Add to secrets:

  ```bash wrap
  gh aw secrets set GH_AW_AGENT_TOKEN --value "YOUR_AGENT_PAT"
  ```

### Token Not Being Used

**Symptom**: You've configured a token but the workflow still uses the default `GITHUB_TOKEN`.

**Cause**: Token precedence not configured correctly.

**Solution**: Check token precedence levels:

1. **Per-output** (highest priority):

   ```yaml wrap
   safe-outputs:
     create-issue:
       github-token: ${{ secrets.MY_TOKEN }}
   ```

2. **Global safe-outputs**:

   ```yaml wrap
   safe-outputs:
     github-token: ${{ secrets.MY_TOKEN }}
     create-issue:
   ```

3. **Workflow-level** (frontmatter):

   ```yaml wrap
   ---
   github-token: ${{ secrets.MY_TOKEN }}
   ---
   ```

### Multiple Workflows, Same Token

**Question**: Can I use the same PAT across multiple workflows?

**Answer**: Yes, but consider security implications:

**Good practice**: Use one general-purpose PAT for multiple workflows with similar permissions:

```bash wrap
gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_PAT"
```

**Better practice**: Use different tokens for different permission levels:

```bash wrap
gh aw secrets set READ_ONLY_TOKEN --value "YOUR_READ_PAT"
gh aw secrets set READ_WRITE_TOKEN --value "YOUR_WRITE_PAT"
```

**Best practice**: Use GitHub Apps with automatic permission management.

### Token Expiration

**Symptom**: Workflows suddenly fail with authentication errors after working previously.

**Cause**: PAT has expired.

**Solution**:

1. Check PAT expiration in [GitHub settings](https://github.com/settings/tokens)
2. Regenerate or create new PAT
3. Update secret:

  ```bash wrap
  gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_NEW_PAT"
  ```

4. Consider shorter expiration periods with regular rotation schedule

### Organization Policies

**Error**: "Personal Access Token creation is restricted"

**Cause**: Organization has restricted PAT creation.

**Solution**: Work with your organization admin to:

1. Request exemption for specific repositories
2. Use organization-wide GitHub App instead
3. Request pre-approved fine-grained PAT with required scopes

## Quick Reference

### Token Setup Commands

```bash wrap
# Enhanced GitHub token (most common, current repository)
gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_PAT"

# Copilot operations
gh aw secrets set COPILOT_GITHUB_TOKEN --value "YOUR_COPILOT_PAT"

# Agent assignments
gh aw secrets set GH_AW_AGENT_TOKEN --value "YOUR_AGENT_PAT"

# GitHub Projects v2 operations (optional)
gh aw secrets set GH_AW_PROJECT_GITHUB_TOKEN --value "YOUR_PROJECT_PAT"

# Custom MCP server token (optional)
gh aw secrets set GH_AW_GITHUB_MCP_SERVER_TOKEN --value "YOUR_PAT"

# List configured secrets (GitHub CLI)
gh secret list -a actions
```

### Token Precedence Summary

**For safe outputs** (create-issue, create-pr, add-comment, etc.):

1. Per-output `github-token:` (highest)
2. Global `safe-outputs.github-token:`
3. Workflow-level `github-token:`
4. `secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN` (lowest)

**For Copilot operations** (create-agent-task, copilot assignee/reviewer):

1. Per-output `github-token:` (highest)
2. Global `safe-outputs.github-token:`
3. Workflow-level `github-token:`
4. `secrets.COPILOT_GITHUB_TOKEN`
5. `secrets.GH_AW_GITHUB_TOKEN` (lowest)

:::caution[Legacy Tokens Removed]
The `COPILOT_CLI_TOKEN` and `GH_AW_COPILOT_TOKEN` secret names are **no longer supported** as of v0.26+. Use `COPILOT_GITHUB_TOKEN` instead.
:::

**For update-project safe output** (GitHub Projects v2):

1. Per-output `github-token:` (`safe-outputs.update-project.github-token`) (highest)
2. Workflow-level `github-token:`
3. `secrets.GH_AW_PROJECT_GITHUB_TOKEN`
4. `secrets.GITHUB_TOKEN` (lowest)

**For GitHub MCP server**:

1. Tool-level `github-token` (highest)
2. Workflow-level `github-token:`
3. `secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN`
4. `secrets.GH_AW_GITHUB_TOKEN`
5. `secrets.GITHUB_TOKEN` (lowest)

### Required PAT Permissions

| Operation Type | Required Permissions |
|---------------|---------------------|
| Cross-repository read | Contents: Read |
| Cross-repository issues | Issues: Read+Write, Contents: Read |
| Cross-repository PRs | Pull requests: Read+Write, Contents: Read+Write |
| Copilot operations | Copilot Requests (special permission) |
| Agent assignments | Actions: Write, Contents: Write, Issues: Write, Pull requests: Write |
| GitHub Projects v2 | Projects: Read+Write (org-level for org Projects) |
| Remote GitHub MCP | Contents: Read (minimum), adjust based on toolsets |

### Migration from Legacy Tokens

| Old Token Name | New Token Name | Status |
|---------------|----------------|--------|
| `GH_AW_COPILOT_TOKEN` | `COPILOT_GITHUB_TOKEN` | **Removed in v0.26+** |
| `COPILOT_CLI_TOKEN` | `COPILOT_GITHUB_TOKEN` | **Removed in v0.26+** |
| - | `GH_AW_GITHUB_MCP_SERVER_TOKEN` | Optional, new in v0.23+ |

:::caution[Legacy Tokens Removed]
The `COPILOT_CLI_TOKEN` and `GH_AW_COPILOT_TOKEN` secrets are **no longer supported** as of v0.26+. Workflows using these tokens will fail. Please migrate to `COPILOT_GITHUB_TOKEN`:

```bash wrap
# Add new secret
gh aw secrets set COPILOT_GITHUB_TOKEN --value "<your-github-pat>"
```

:::

To migrate from legacy tokens:

```bash wrap
# Remove old secrets (if present)
gh secret remove GH_AW_COPILOT_TOKEN -a actions
gh secret remove COPILOT_CLI_TOKEN -a actions

# Add new secret
gh aw secrets set COPILOT_GITHUB_TOKEN --value "YOUR_PAT"
```

## Related Documentation

- [Engines](/gh-aw/reference/engines/) - Engine-specific authentication
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output token configuration
- [Tools](/gh-aw/reference/tools/) - Tool authentication and modes
- [Permissions](/gh-aw/reference/permissions/) - Permission model overview
