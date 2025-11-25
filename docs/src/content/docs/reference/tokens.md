---
title: GitHub Tokens
description: Comprehensive reference for all GitHub tokens used in gh-aw, including authentication, token precedence, and security best practices
sidebar:
  order: 650
---

GitHub Agentic Workflows authenticate using multiple tokens depending on the operation. This reference explains which token to use and how precedence works.

## Token Types

| Token | Purpose | When Required |
|-------|---------|--------------|
| `GITHUB_TOKEN` | Default Actions token | Automatically provided, used as fallback |
| `GH_AW_GITHUB_TOKEN` | Enhanced PAT | Cross-repo operations, remote GitHub tools |
| `COPILOT_GITHUB_TOKEN` | Copilot authentication | Copilot engine, bot assignments |
| `GITHUB_MCP_SERVER_TOKEN` | GitHub MCP Server | Auto-set based on GitHub tools config |
| `GH_AW_AGENT_TOKEN` | Agent assignments | Assigning Copilot to issues |

## `GITHUB_TOKEN` (Default)

Automatically provided by GitHub Actions with scoped access to the current repository. Used as a fallback when no custom token is configured.

**Limitations**: Cannot access other repositories, trigger workflows, assign bots, or authenticate with Copilot CLI.

## `GH_AW_GITHUB_TOKEN` (Enhanced PAT)

Personal Access Token providing enhanced capabilities beyond `GITHUB_TOKEN`. Required for cross-repository operations, remote GitHub tools mode, and Codex engine operations.

**Setup**: Create a [fine-grained PAT](https://github.com/settings/personal-access-tokens/new) with appropriate repository access and permissions (issues/PRs: read+write, contents: read+write for PRs).

```bash wrap
gh secret set GH_AW_GITHUB_TOKEN -a actions --body "YOUR_PAT"
```

**Token precedence** (highest to lowest):
1. Per-output `github-token:`
2. Global `safe-outputs.github-token:`
3. Workflow-level `github-token:` frontmatter
4. `${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`

## `COPILOT_GITHUB_TOKEN` (Copilot Auth)

Required for Copilot engine, agent tasks, and bot assignments. Create a [PAT](https://github.com/settings/personal-access-tokens/new) with your user account (not org) as resource owner, public repo access, and "Copilot Requests" permission.

```bash wrap
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "YOUR_COPILOT_PAT"
```

**Copilot token precedence**:
1. Per-output `github-token:`
2. Global `safe-outputs.github-token:`
3. Workflow-level `github-token:`
4. `COPILOT_GITHUB_TOKEN` (or legacy: `COPILOT_CLI_TOKEN`, `GH_AW_COPILOT_TOKEN`, `GH_AW_GITHUB_TOKEN`)

Note: `GITHUB_TOKEN` cannot be used for Copilot operations.

## `GITHUB_MCP_SERVER_TOKEN` (Auto-configured)

Automatically set by gh-aw based on your GitHub tools configuration. Passed as `GITHUB_PERSONAL_ACCESS_TOKEN` env var (local mode) or `Authorization: Bearer` header (remote mode).

```yaml wrap
tools:
  github:
    mode: remote  # Requires GH_AW_GITHUB_TOKEN or custom PAT
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
```

## `GH_AW_AGENT_TOKEN` (Agent Assignment)

Used for `assign-to-agent:` safe output and assigning `copilot` as assignee or reviewer. Requires `actions`, `contents`, `issues`, and `pull-requests` write permissions.

```bash wrap
gh secret set GH_AW_AGENT_TOKEN -a actions --body "YOUR_AGENT_PAT"
```

**Precedence**: Per-output → global safe-outputs → workflow-level → `GH_AW_AGENT_TOKEN || GH_AW_GITHUB_TOKEN`

## Token Configuration Patterns

### Per-Output vs Global

Configure tokens globally or per-output. When `copilot` is specified as reviewer, the Copilot token chain is automatically used.

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.GLOBAL_PAT }}  # Applied to all outputs
  create-issue:
    github-token: ${{ secrets.ISSUE_PAT }}  # Override for this output
  create-pull-request:
    reviewers: copilot  # Automatically uses Copilot token chain
```

### Cross-Repository Operations

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/other-repo"
```

## GitHub App Tokens

GitHub App installation tokens provide enhanced security with short-lived, auto-revoked credentials. Tokens are minted on-demand at job start and automatically revoked at job end (even on failure). Permissions are computed automatically based on safe output types.

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    owner: "my-org"                    # Optional
    repositories: ["repo1", "repo2"]   # Optional: omit for current repo only
  create-issue:
```

**Benefits**: On-demand minting, short-lived tokens, automatic fine-grained permissions, audit trail, no PAT rotation needed.

**Permission mapping**: `create-issue:` → issues write, `create-pull-request:` → contents + pull-requests write, `add-comment:` → issues write.

App configuration can be imported from shared workflows (local config takes precedence).

## Security Best Practices

- **Use least privilege**: Configure minimal `permissions:` and let safe outputs handle API access
- **Prefer App tokens**: Use GitHub Apps for production workflows (better security, auditability)
- **Scope appropriately**: Use per-output tokens when operations need different permission levels
- **Rotate PATs**: Implement regular rotation with short expiration periods

## Common Configurations

**Basic workflow** (uses default `GITHUB_TOKEN`):
```yaml wrap
safe-outputs:
  add-comment:
```

**Cross-repository**:
```yaml wrap
safe-outputs:
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
  create-issue:
    target-repo: "org/tracking-repo"
```

**Copilot integration** (requires `COPILOT_GITHUB_TOKEN`):
```yaml wrap
engine: copilot
safe-outputs:
  create-agent-task:
```

**GitHub App**:
```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  create-issue:
```

## Troubleshooting

**"Resource not accessible by integration"**: Token lacks required permissions. Configure `github-token:` with appropriate PAT (cross-repo requires target repo access, bot operations require `COPILOT_GITHUB_TOKEN`).

**Copilot authentication failures**: Set `COPILOT_GITHUB_TOKEN` secret with user-scoped PAT.

**Remote GitHub tools failures**: Remote mode requires `GH_AW_GITHUB_TOKEN` or custom PAT (default `GITHUB_TOKEN` not supported).

**Agent assignment failures**: Set `GH_AW_AGENT_TOKEN` with actions/contents/issues/pull-requests write permissions.

## Related Documentation

- [Engines](/gh-aw/reference/engines/) - Engine-specific authentication
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output token configuration
- [Tools](/gh-aw/reference/tools/) - Tool authentication and modes
- [Permissions](/gh-aw/reference/permissions/) - Permission model overview
