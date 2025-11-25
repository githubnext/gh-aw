---
title: GitHub Tokens
description: Comprehensive reference for all GitHub tokens used in gh-aw, including authentication, token precedence, and security best practices
sidebar:
  order: 650
---

GitHub Agentic Workflows use multiple tokens for authentication across different operations. Understanding which token to use—and how token precedence works—ensures your workflows operate securely and effectively.

## Token Overview

| Token | Purpose | Required For |
|-------|---------|--------------|
| `GITHUB_TOKEN` | Default GitHub Actions token | Basic workflow operations |
| `GH_AW_GITHUB_TOKEN` | Enhanced PAT for gh-aw operations | Safe outputs, cross-repo operations |
| `COPILOT_GITHUB_TOKEN` | Copilot CLI authentication | Copilot engine, agent operations |
| `GITHUB_MCP_SERVER_TOKEN` | GitHub MCP Server authentication | GitHub tools (local/remote mode) |
| `GH_AW_AGENT_TOKEN` | Agent assignment operations | Assigning Copilot to issues |

## GitHub Actions Token (`GITHUB_TOKEN`)

The `GITHUB_TOKEN` is automatically provided by GitHub Actions for every workflow run. It provides scoped access to the repository where the workflow runs.

### Capabilities

- Read/write access to the current repository (based on permissions)
- Cannot access other repositories
- Cannot trigger other workflows
- Cannot assign Copilot bots to issues

### Limitations for Agentic Workflows

The default `GITHUB_TOKEN` is insufficient for several gh-aw operations:

- **Copilot CLI operations**: Requires user-scoped token with Copilot permissions
- **Cross-repository operations**: Cannot access other repositories
- **Bot assignments**: Cannot assign Copilot agents or add bot reviewers
- **Remote GitHub MCP mode**: Not supported

### Usage

The default token is used as a fallback when no custom token is configured:

```yaml wrap
safe-outputs:
  create-issue:  # Uses GITHUB_TOKEN if no custom token specified
```

## gh-aw GitHub Token (`GH_AW_GITHUB_TOKEN`)

The `GH_AW_GITHUB_TOKEN` is a Personal Access Token (PAT) that provides enhanced capabilities for gh-aw operations.

### When Required

- Cross-repository operations (e.g., `target-repo:` in safe outputs)
- GitHub tools remote mode
- Any operation that exceeds default `GITHUB_TOKEN` capabilities

### Setup

```bash wrap
gh secret set GH_AW_GITHUB_TOKEN -a actions --body "YOUR_PAT"
```

### Token Precedence

For standard safe output operations, tokens are resolved in this order:

1. Per-output `github-token:` configuration
2. Global `safe-outputs.github-token:` configuration
3. Workflow-level `github-token:` frontmatter
4. `${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`

```yaml wrap
# Token precedence example
github-token: ${{ secrets.WORKFLOW_PAT }}  # Level 3

safe-outputs:
  github-token: ${{ secrets.SAFE_OUTPUT_PAT }}  # Level 2
  create-issue:
    github-token: ${{ secrets.ISSUE_PAT }}  # Level 1 (highest priority)
  add-comment:  # Uses Level 2 token
```

## Copilot CLI Token (`COPILOT_GITHUB_TOKEN`)

The `COPILOT_GITHUB_TOKEN` authenticates with the GitHub Copilot CLI and is required for the Copilot engine.

### Creating the Token

1. Visit <https://github.com/settings/personal-access-tokens/new>
2. Select your **user account** as resource owner (not an organization)
3. Choose **"Public repositories"** under repository access
4. Add the **"Copilot Requests"** permission
5. Generate and save the token

### Setup

```bash wrap
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "YOUR_COPILOT_PAT"
```

### Token Precedence for Copilot Operations

Copilot-related operations (agent tasks, bot assignments, bot reviewers) use a separate precedence chain:

1. Per-output `github-token:` configuration
2. Global `safe-outputs.github-token:` configuration
3. Workflow-level `github-token:` frontmatter
4. `${{ secrets.COPILOT_GITHUB_TOKEN }}`
5. `${{ secrets.COPILOT_CLI_TOKEN }}` (alternative)
6. `${{ secrets.GH_AW_COPILOT_TOKEN }}` (legacy)
7. `${{ secrets.GH_AW_GITHUB_TOKEN }}` (legacy fallback)

:::note[Backward Compatibility]
Legacy token names `COPILOT_CLI_TOKEN`, `GH_AW_COPILOT_TOKEN`, and `GH_AW_GITHUB_TOKEN` remain supported. For new workflows, use `COPILOT_GITHUB_TOKEN`.
:::

### Important Limitation

The default `GITHUB_TOKEN` is **not** included in the Copilot token chain because it lacks permissions for:

- Creating agent tasks
- Assigning issues to bots
- Adding bots as PR reviewers

## GitHub MCP Server Token (`GITHUB_MCP_SERVER_TOKEN`)

The `GITHUB_MCP_SERVER_TOKEN` authenticates with the GitHub MCP Server, which provides GitHub API tools to AI engines.

### How It's Set

This token is set automatically by gh-aw during workflow execution based on your GitHub tools configuration:

```yaml wrap
tools:
  github:
    github-token: ${{ secrets.CUSTOM_PAT }}  # Custom token
```

If no custom token is specified, the effective token is resolved using the standard precedence (custom → workflow-level → default fallback).

### Local vs Remote Mode

**Local Mode** (Docker-based): The token is passed as `GITHUB_PERSONAL_ACCESS_TOKEN` environment variable to the Docker container.

**Remote Mode** (hosted): The token is passed via `Authorization: Bearer` header to the hosted MCP server.

```yaml wrap
# Local mode (default)
tools:
  github:
    mode: local  # Token passed via env var

# Remote mode
tools:
  github:
    mode: remote  # Token passed via HTTP header
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}  # Required for remote mode
```

:::caution
Remote mode requires `GH_AW_GITHUB_TOKEN` or a custom PAT. The default `GITHUB_TOKEN` is not supported.
:::

## Agent Assignment Token (`GH_AW_AGENT_TOKEN`)

The `GH_AW_AGENT_TOKEN` is specifically for assigning Copilot agents to issues and related operations.

### When Required

- `assign-to-agent:` safe output
- Assigning `copilot` to issues via `assignees: copilot`
- Adding `copilot` as a reviewer

### Token Precedence

1. Per-output `github-token:` configuration
2. Global `safe-outputs.github-token:` configuration
3. Workflow-level `github-token:` frontmatter
4. `${{ secrets.GH_AW_AGENT_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}`

### Setup

```bash wrap
gh secret set GH_AW_AGENT_TOKEN -a actions --body "YOUR_AGENT_PAT"
```

### Required Permissions

The token needs the following permissions:
- `actions: write`
- `contents: write`
- `issues: write`
- `pull-requests: write`

## Add Reviewer Token

Adding reviewers to pull requests uses the standard safe output token precedence. However, when adding `copilot` as a reviewer, the Copilot token chain is used instead.

### Standard Reviewers

```yaml wrap
safe-outputs:
  add-reviewer:
    reviewers: [user1, user2]
    github-token: ${{ secrets.REVIEWER_PAT }}  # Standard token chain
```

### Copilot as Reviewer

```yaml wrap
safe-outputs:
  create-pull-request:
    reviewers: copilot  # Uses Copilot token chain
```

When `copilot` is specified as a reviewer, gh-aw automatically uses the Copilot token precedence chain to ensure proper bot assignment permissions.

## Safe Output Tokens

Each safe output type can have its own token configuration, or inherit from the global configuration.

### Per-Output Token

```yaml wrap
safe-outputs:
  create-issue:
    github-token: ${{ secrets.ISSUE_PAT }}
  create-pull-request:
    github-token: ${{ secrets.PR_PAT }}
```

### Global Safe Output Token

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.SAFE_OUTPUT_PAT }}  # Applies to all outputs
  create-issue:
  add-comment:
```

### Cross-Repository Operations

Cross-repository operations require a PAT with access to the target repository:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/other-repo"
```

## GitHub App Token Generation

For enhanced security and fine-grained permissions, use GitHub App installation tokens instead of PATs. These tokens are minted on-demand and automatically revoked after use.

### Configuration

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    owner: "my-org"                    # Optional: defaults to current repo owner
    repositories: ["repo1", "repo2"]   # Optional: scope to specific repos
  create-issue:
```

### How It Works

1. **Token Minting**: At job start, a short-lived installation access token is generated using the `actions/create-github-app-token` action
2. **Automatic Permissions**: Required permissions are automatically computed based on the safe output type
3. **Token Revocation**: At job end, the token is invalidated—even if the job fails

### Benefits

- **On-demand minting**: Tokens are only created when needed
- **Short-lived**: Tokens expire quickly, reducing exposure risk
- **Fine-grained**: Permissions are scoped to exactly what's needed
- **Audit trail**: Clear attribution in GitHub logs
- **No PAT management**: No need to rotate or manage PATs

### Repository Scoping

| Configuration | Scope |
|---------------|-------|
| Not specified | Current repository only |
| `repositories: []` with `owner:` | All repos in installation |
| `repositories: ["repo1"]` | Listed repos only |

### Automatic Permission Mapping

Safe output types automatically map to required permissions:

| Safe Output | Permissions |
|-------------|-------------|
| `create-issue:` | `permission-issues: write` |
| `create-pull-request:` | `permission-contents: write`, `permission-pull-requests: write` |
| `add-comment:` | `permission-issues: write` |
| `create-discussion:` | (Uses GitHub API token directly) |

### Import Support

App configuration can be imported from shared workflows:

```yaml wrap
imports:
  - shared/github-app-config.md

safe-outputs:
  create-issue:
```

Local `app:` configuration takes precedence over imported configuration.

:::tip
Use GitHub App tokens for organization-wide automation. They provide better security, auditability, and don't consume PAT slots.
:::

## Token Security Best Practices

### Use Least Privilege

Configure the minimum permissions needed for each operation:

```yaml wrap
# Good: Minimal permissions with safe outputs
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:

# Avoid: Overly broad permissions
permissions:
  contents: write
  issues: write
```

### Prefer App Tokens

For production workflows, prefer GitHub App tokens over PATs:

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
```

### Scope Tokens Appropriately

Use per-output tokens for operations requiring different permission levels:

```yaml wrap
safe-outputs:
  create-issue:
    github-token: ${{ secrets.ISSUE_PAT }}
  create-pull-request:
    github-token: ${{ secrets.PR_PAT }}
```

### Rotate Tokens Regularly

For PATs, implement regular rotation and use short expiration periods where possible.

## Common Token Configurations

### Basic Workflow (Default Token)

```yaml wrap
permissions:
  contents: read
safe-outputs:
  add-comment:
```

### Cross-Repository Operations

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
  create-issue:
    target-repo: "org/tracking-repo"
```

### Copilot Integration

```yaml wrap
engine: copilot
safe-outputs:
  create-agent-task:
  assign-to-agent:
```

### GitHub App Integration

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  create-issue:
  create-pull-request:
```

## Troubleshooting

### "Resource not accessible by integration"

This error indicates the token lacks required permissions:

- For safe outputs: Configure `github-token:` with appropriate PAT
- For cross-repo: Ensure PAT has access to target repository
- For bot operations: Use `COPILOT_GITHUB_TOKEN`

### Copilot Engine Authentication Failures

Ensure `COPILOT_GITHUB_TOKEN` is configured:

```bash wrap
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "YOUR_COPILOT_PAT"
```

### Remote GitHub Tools Mode Failures

Remote mode requires explicit token configuration:

```yaml wrap
tools:
  github:
    mode: remote
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
```

### Agent Assignment Failures

Bot assignments require special permissions:

```bash wrap
gh secret set GH_AW_AGENT_TOKEN -a actions --body "YOUR_AGENT_PAT"
```

## Related Documentation

- [Engines](/gh-aw/reference/engines/) - Engine-specific authentication
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output token configuration
- [Tools](/gh-aw/reference/tools/) - Tool authentication and modes
- [Permissions](/gh-aw/reference/permissions/) - Permission model overview
