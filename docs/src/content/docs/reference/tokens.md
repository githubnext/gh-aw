---
title: Token Reference
description: Comprehensive reference for all tokens, secrets, and credentials used by GitHub Agentic Workflows
sidebar:
  order: 660
---

This page is a consolidated reference for every token and secret used by GitHub Agentic Workflows — where each comes from, what permissions it needs, and what it is used for.

## Quick Reference

| Secret Name | Purpose | Required? | Fallback |
|---|---|---|---|
| [`COPILOT_GITHUB_TOKEN`](#copilot_github_token) | Copilot CLI authentication | Yes (Copilot engine) | None |
| [`ANTHROPIC_API_KEY`](#anthropic_api_key) | Claude engine authentication | Yes (Claude engine) | None |
| [`OPENAI_API_KEY`](#openai_api_key) | Codex engine authentication | Yes (Codex engine) | `CODEX_API_KEY` → `OPENAI_API_KEY` |
| [`GEMINI_API_KEY`](#gemini_api_key) | Gemini engine authentication | Yes (Gemini engine) | None |
| [`GH_AW_GITHUB_MCP_SERVER_TOKEN`](#gh_aw_github_mcp_server_token) | GitHub MCP server authentication | Optional | `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN` |
| [`GH_AW_GITHUB_TOKEN`](#gh_aw_github_token) | General-purpose GitHub token | Optional | `GITHUB_TOKEN` |
| [`GH_AW_AGENT_TOKEN`](#gh_aw_agent_token) | Assign AI agents to issues/PRs | Optional | `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN` |
| [`GH_AW_PROJECT_GITHUB_TOKEN`](#gh_aw_project_github_token) | GitHub Projects v2 operations | Yes (Projects) | None |
| [`GH_AW_CI_TRIGGER_TOKEN`](#gh_aw_ci_trigger_token) | Trigger CI on workflow-created PRs | Yes (CI trigger) | None |
| [`GH_AW_PLUGINS_TOKEN`](#gh_aw_plugins_token) | APM plugin/dependency access | Optional | `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN` |
| [`APP_ID` / `APP_PRIVATE_KEY`](#github-app-tokens) | GitHub App authentication | Optional | PAT-based tokens |
| [`GITHUB_TOKEN`](#github_token) | Default GitHub Actions token | Automatic | None (always available) |

## AI Engine Tokens

These tokens authenticate the AI coding agent. Exactly one is required, depending on which engine your workflow uses.

### `COPILOT_GITHUB_TOKEN`

Authenticates the [GitHub Copilot CLI](/gh-aw/reference/engines/#available-coding-agents) as the AI engine.

| Property | Value |
|---|---|
| **Source** | User-created fine-grained PAT stored as repository secret |
| **Required** | Yes, when using `engine: copilot` (the default engine) |
| **Permissions** | Account permission: **Copilot Requests: Read** |
| **Resource owner** | Must be the user's **personal account**, not an organization |
| **Fallback** | None — the `GITHUB_TOKEN` does not have Copilot permissions |
| **Used by** | Copilot CLI inference step, secret validation, Copilot-related safe outputs |

When the `copilot-requests` [feature flag](/gh-aw/reference/frontmatter/#feature-flags-features) is enabled, the compiler uses `${{ github.token }}` instead of this secret, allowing the built-in GitHub Actions token to authenticate Copilot directly. This feature is currently in **private preview** and will not work unless your account has been onboarded.

**Setup:**

```bash wrap
gh aw secrets set COPILOT_GITHUB_TOKEN --value "<your-fine-grained-pat>"
```

See [Authentication](/gh-aw/reference/auth/#copilot_github_token) for detailed setup instructions.

---

### `ANTHROPIC_API_KEY`

Authenticates the [Claude by Anthropic](/gh-aw/reference/engines/#available-coding-agents) engine.

| Property | Value |
|---|---|
| **Source** | Anthropic API key stored as repository secret |
| **Required** | Yes, when using `engine: claude` |
| **Permissions** | Anthropic API access (external service) |
| **Fallback** | None |
| **Used by** | Claude inference step |

```bash wrap
gh aw secrets set ANTHROPIC_API_KEY --value "<your-anthropic-api-key>"
```

---

### `OPENAI_API_KEY`

Authenticates the [Codex by OpenAI](/gh-aw/reference/engines/#available-coding-agents) engine.

| Property | Value |
|---|---|
| **Source** | OpenAI API key stored as repository secret |
| **Required** | Yes, when using `engine: codex` |
| **Permissions** | OpenAI API access (external service) |
| **Fallback** | Both `CODEX_API_KEY` and `OPENAI_API_KEY` are accepted; the runtime tries `CODEX_API_KEY` first (`${{ secrets.CODEX_API_KEY \|\| secrets.OPENAI_API_KEY }}`). No fallback to `GITHUB_TOKEN`. |
| **Used by** | Codex inference step |

```bash wrap
gh aw secrets set OPENAI_API_KEY --value "<your-openai-api-key>"
```

---

### `GEMINI_API_KEY`

Authenticates the [Gemini by Google](/gh-aw/reference/engines/#available-coding-agents) engine.

| Property | Value |
|---|---|
| **Source** | Google AI Studio API key stored as repository secret |
| **Required** | Yes, when using `engine: gemini` |
| **Permissions** | Google AI Studio API access (external service) |
| **Fallback** | None |
| **Used by** | Gemini inference step |

```bash wrap
gh aw secrets set GEMINI_API_KEY --value "<your-gemini-api-key>"
```

---

## GitHub API Tokens

These tokens control access to the GitHub API across different parts of a compiled workflow.

### `GH_AW_GITHUB_MCP_SERVER_TOKEN`

The primary token for authenticating GitHub API operations through the [GitHub MCP server](/gh-aw/reference/github-tools/). This is a **"magic secret"** — if present in repository secrets, it is automatically used without being referenced in your workflow frontmatter.

| Property | Value |
|---|---|
| **Source** | User-created PAT stored as repository secret |
| **Required** | Depends on mode — required for [remote mode](/gh-aw/reference/github-tools/#github-tools-remote-mode) and [lockdown mode](/gh-aw/reference/lockdown-mode/) |
| **Permissions** | Depends on [toolsets](/gh-aw/reference/github-tools/#github-toolsets) used (e.g., repo scope for repos toolset, project scope for projects) |
| **Fallback** | `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN` |
| **Used by** | GitHub MCP server (local and remote modes), guard policy enforcement |

**How the magic secret works:** The compiler automatically injects this token into the workflow-level environment. You do not need to reference it in your frontmatter — if the secret exists in your repository, it is used. The resolved expression is:

```yaml wrap
${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
```

**When additional permissions are needed:**
- **Remote mode**: Requires a PAT (classic or fine-grained) — the default `GITHUB_TOKEN` is not sufficient
- **Lockdown mode**: Requires a PAT for elevated access control
- **Projects toolset**: Requires `project` scope (see [Authentication (Projects)](/gh-aw/reference/auth-projects/))
- **Cross-repository access**: Requires access to the target repositories

```bash wrap
gh aw secrets set GH_AW_GITHUB_MCP_SERVER_TOKEN --value "<your-pat>"
```

---

### `GH_AW_GITHUB_TOKEN`

A general-purpose GitHub token that serves as a fallback across multiple token chains. Use this when you want a single PAT to cover multiple workflow operations without configuring purpose-specific tokens.

| Property | Value |
|---|---|
| **Source** | User-created PAT stored as repository secret |
| **Required** | Optional — provides a middle-tier fallback |
| **Permissions** | Depends on operations (typically `repo` scope) |
| **Fallback** | `GITHUB_TOKEN` |
| **Used by** | MCP server (when `GH_AW_GITHUB_MCP_SERVER_TOKEN` is not set), safe outputs, agent assignment, APM dependencies |

This token appears in the fallback chain of almost every token resolution function. It is a good choice when you need a PAT with broader permissions than `GITHUB_TOKEN` but don't want to configure multiple purpose-specific secrets.

---

### `GITHUB_TOKEN`

The default token automatically provided by GitHub Actions to every workflow run. It is repository-scoped and cannot access cross-repo resources, Projects v2, or trigger other workflows.

| Property | Value |
|---|---|
| **Source** | Automatically provided by GitHub Actions |
| **Required** | No (always available) |
| **Permissions** | Repository-scoped, controlled by workflow `permissions:` block |
| **Fallback** | None (this is the final fallback) |
| **Used by** | Last-resort fallback for MCP server, safe outputs, `gh` CLI, checkout steps |

> [!CAUTION]
> `GITHUB_TOKEN` cannot access GitHub Projects v2, trigger other workflow runs, or access resources outside the current repository. Use purpose-specific tokens for those operations.

---

## Purpose-Specific Tokens

These tokens serve specific workflow features and have tailored fallback chains.

### `GH_AW_AGENT_TOKEN`

Used when a workflow assigns an AI coding agent (e.g., Copilot coding agent) to issues or pull requests via [safe outputs](/gh-aw/reference/assign-to-copilot/).

| Property | Value |
|---|---|
| **Source** | User-created PAT stored as repository secret |
| **Required** | Recommended for agent assignment operations |
| **Permissions** | `issues: write`, `pull_requests: write`, and bot assignment permissions |
| **Fallback** | `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN` (may lack permissions) |
| **Used by** | `assign-to-copilot` safe output, agent assignment steps |

```bash wrap
gh aw secrets set GH_AW_AGENT_TOKEN --value "<your-pat>"
```

---

### `GH_AW_PROJECT_GITHUB_TOKEN`

Used for GitHub Projects v2 operations. This token has **no fallback** because the default `GITHUB_TOKEN` cannot access the Projects GraphQL API.

| Property | Value |
|---|---|
| **Source** | User-created PAT stored as repository secret |
| **Required** | Yes, for any Projects v2 operation |
| **Permissions** | Classic PAT: `project` + `repo` scopes. Fine-grained: Organization permissions → Projects: Read and write |
| **Fallback** | None — `GITHUB_TOKEN` cannot access Projects v2 |
| **Used by** | `update-project`, `create-project`, `create-project-status-update` safe outputs, `projects` toolset |

```bash wrap
gh aw secrets set GH_AW_PROJECT_GITHUB_TOKEN --value "<your-pat>"
```

For separate read/write tokens, see [Authentication (Projects)](/gh-aw/reference/auth-projects/#recommended-secret-layout).

---

### `GH_AW_CI_TRIGGER_TOKEN`

Used when a workflow needs to push commits that trigger CI checks on pull requests it creates. This token has **no fallback** because events created with `GITHUB_TOKEN` [do not trigger other workflow runs](https://docs.github.com/en/actions/using-workflows/triggering-a-workflow#triggering-a-workflow-from-a-workflow) by design.

| Property | Value |
|---|---|
| **Source** | User-created PAT stored as repository secret |
| **Required** | Yes, when using [CI triggering](/gh-aw/reference/triggering-ci/) |
| **Permissions** | `contents: write` on the target repository |
| **Fallback** | None — `GITHUB_TOKEN` events cannot trigger workflows |
| **Used by** | Empty commit push step for CI triggering on workflow-created PRs |

```bash wrap
gh aw secrets set GH_AW_CI_TRIGGER_TOKEN --value "<your-pat>"
```

See [Triggering CI](/gh-aw/reference/triggering-ci/) for usage details.

---

### `GH_AW_PLUGINS_TOKEN`

Used for APM (Agentic Package Manager) plugin and dependency operations, such as fetching packages from private registries or cross-organization repositories.

| Property | Value |
|---|---|
| **Source** | User-created PAT stored as repository secret |
| **Required** | Optional — only needed for private/cross-org dependencies |
| **Permissions** | `repo` scope for accessing private repositories containing plugins |
| **Fallback** | `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN` |
| **Used by** | APM dependency resolution and installation steps |

```bash wrap
gh aw secrets set GH_AW_PLUGINS_TOKEN --value "<your-pat>"
```

---

## GitHub App Tokens

For enhanced security, you can configure a [GitHub App](https://docs.github.com/en/apps/creating-github-apps/about-creating-github-apps/about-creating-github-apps) instead of PATs. App tokens are short-lived, automatically scoped, and revoked at workflow end.

### Configuration

Store the App ID as a repository **variable** and the private key as a repository **secret**:

```bash wrap
gh variable set APP_ID --body "123456"
gh aw secrets set APP_PRIVATE_KEY --value "$(cat path/to/private-key.pem)"
```

Reference them in your workflow frontmatter:

```yaml wrap
tools:
  github:
    github-app:
      client-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
      owner: "my-org"                    # Optional: defaults to current repo owner
      repositories: ["repo1", "repo2"]   # Optional: defaults to current repo only
```

### How App tokens work

When a GitHub App is configured, the compiler generates [`actions/create-github-app-token`](https://github.com/actions/create-github-app-token) steps that mint short-lived installation access tokens at workflow runtime. These tokens:

- **Override all PAT-based tokens** — when configured, App tokens take highest precedence
- **Are scoped to the job's `permissions:` block** — the token receives only the permissions declared in your workflow
- **Are automatically revoked** at the end of the workflow run (even on failure)
- **Support repository scoping** — limit access to specific repositories for least privilege

### Where App tokens are minted

The compiler generates App token mint steps in multiple locations depending on what features your workflow uses:

| Location | Step ID | Used For |
|---|---|---|
| Pre-activation job | `pre-activation-app-token` | Membership checks, skip-if evaluations |
| Activation job | `activation-app-token` | Reactions, status comments, label removal |
| Agent job (GitHub MCP) | `github-mcp-app-token` | GitHub MCP server authentication |
| Safe outputs job | `safe-outputs-app-token` | Write operations (issues, PRs, comments) |
| APM dependencies | `apm-app-token` | Plugin and dependency access |

Each mint step includes `github-api-url: ${{ github.api_url }}` for [GitHub Enterprise compatibility](/gh-aw/troubleshooting/debug-ghe/).

### App token vs PAT precedence

When a GitHub App is configured, it always takes precedence:

```
GitHub App token (highest priority)
  └── Custom github-token field
        └── Purpose-specific secret (e.g., GH_AW_GITHUB_MCP_SERVER_TOKEN)
              └── GH_AW_GITHUB_TOKEN
                    └── GITHUB_TOKEN (lowest priority)
```

> [!NOTE]
> `COPILOT_GITHUB_TOKEN` is the one token that **cannot** use a GitHub App. It must be a fine-grained PAT with Copilot Requests permission on a personal account.

See [Authentication](/gh-aw/reference/auth/#using-a-github-app-for-authentication) for full GitHub App setup instructions.

---

## Internal Runtime Tokens

These tokens are set by the compiler at workflow runtime. You do not configure them as repository secrets — they are derived from the tokens above.

### `GITHUB_MCP_SERVER_TOKEN`

An environment variable set on the agent job step that carries the resolved GitHub token into the MCP gateway container. Its value is determined by the [token precedence chain](#token-precedence-chains) and may be a GitHub App token, a custom PAT, or the default `GITHUB_TOKEN`.

| Property | Value |
|---|---|
| **Source** | Compiler-generated from configured tokens |
| **Set on** | Agent job steps, MCP gateway Docker container |
| **Resolves to** | App token → custom `github-token` → `GH_AW_GITHUB_MCP_SERVER_TOKEN` → `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN` |

### `GITHUB_PERSONAL_ACCESS_TOKEN`

An environment variable passed to the GitHub MCP server Docker container. It carries the same resolved token as `GITHUB_MCP_SERVER_TOKEN` and is used internally by the MCP server process.

### `GH_TOKEN`

Set on various workflow steps (checkout, `gh` CLI commands, pre-agentic steps) to authenticate GitHub CLI operations. Typically resolves to `${{ github.token }}` or a minted App token.

---

## Token Precedence Chains

The compiler uses different precedence chains depending on the operation context. Each chain is a cascade expression — the first non-empty secret wins.

### GitHub MCP Server

```
getEffectiveGitHubToken():
  1. Custom github-token (from tools.github.github-token)
  2. GH_AW_GITHUB_MCP_SERVER_TOKEN
  3. GH_AW_GITHUB_TOKEN
  4. GITHUB_TOKEN
```

If a GitHub App is configured, the App token overrides this entire chain.

### Safe Outputs

```
getEffectiveSafeOutputGitHubToken():
  1. Custom github-token (from safe-outputs.<type>.github-token)
  2. GH_AW_GITHUB_TOKEN
  3. GITHUB_TOKEN
```

> [!NOTE]
> `GH_AW_GITHUB_MCP_SERVER_TOKEN` is intentionally **not** in the safe outputs chain. Safe outputs use a simpler chain because MCP server-specific tokens should not leak into write operations.

### Copilot Operations

```
getEffectiveCopilotRequestsToken():
  1. Custom github-token
  2. COPILOT_GITHUB_TOKEN
  (no further fallback)
```

### Agent Assignment

```
getEffectiveCopilotCodingAgentGitHubToken():
  1. Custom github-token
  2. GH_AW_AGENT_TOKEN
  3. GH_AW_GITHUB_TOKEN
  4. GITHUB_TOKEN
```

### Projects v2

```
getEffectiveProjectGitHubToken():
  1. Custom github-token
  2. GH_AW_PROJECT_GITHUB_TOKEN
  (no further fallback)
```

### CI Triggering

```
getEffectiveCITriggerGitHubToken():
  1. Custom github-token
  2. GH_AW_CI_TRIGGER_TOKEN
  (no further fallback)
```

### APM Dependencies

```
getEffectiveAPMGitHubToken():
  1. Custom github-token (from dependencies.github-token)
  2. GH_AW_PLUGINS_TOKEN
  3. GH_AW_GITHUB_TOKEN
  4. GITHUB_TOKEN
```

---

## Overriding Tokens with `github-token`

Many workflow features accept an explicit `github-token` field that takes highest priority in the precedence chain (above any repository secret):

```yaml wrap
tools:
  github:
    github-token: ${{ secrets.MY_CUSTOM_PAT }}

safe-outputs:
  create-issue:
    github-token: ${{ secrets.MY_ISSUES_PAT }}
  update-project:
    github-token: ${{ secrets.MY_PROJECT_PAT }}
```

This allows fine-grained control over which token is used for each operation, independent of the default fallback chains.

---

## Tokens That Do Not Fall Back to `GITHUB_TOKEN`

Three tokens intentionally have **no fallback** to `GITHUB_TOKEN` because the default token lacks the necessary capabilities:

| Token | Reason `GITHUB_TOKEN` cannot be used |
|---|---|
| `COPILOT_GITHUB_TOKEN` | `GITHUB_TOKEN` does not have Copilot Requests permission |
| `GH_AW_PROJECT_GITHUB_TOKEN` | `GITHUB_TOKEN` is repository-scoped and cannot access the Projects v2 GraphQL API |
| `GH_AW_CI_TRIGGER_TOKEN` | Events created by `GITHUB_TOKEN` [do not trigger other workflows](https://docs.github.com/en/actions/using-workflows/triggering-a-workflow#triggering-a-workflow-from-a-workflow) |

If these secrets are not configured and no custom `github-token` is provided, the relevant workflow steps will fail at runtime.

---

## Related Documentation

- [Authentication](/gh-aw/reference/auth/) — AI engine secrets and GitHub App setup
- [Authentication (Projects)](/gh-aw/reference/auth-projects/) — Projects-specific token configuration
- [GitHub Tools](/gh-aw/reference/github-tools/) — MCP server modes and toolset authentication
- [Permissions](/gh-aw/reference/permissions/) — Permission model and scopes
- [Environment Variables](/gh-aw/reference/environment-variables/) — CLI configuration, model overrides, guard policy fallbacks, and runtime variable scopes
- [Triggering CI](/gh-aw/reference/triggering-ci/) — CI trigger token usage
- [Lockdown Mode](/gh-aw/reference/lockdown-mode/) — Elevated authentication requirements
