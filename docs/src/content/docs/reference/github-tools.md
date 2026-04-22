---
title: GitHub Tools (for reading from GitHub)
description: Configure reading information from GitHub, including integrity filtering, repository access restrictions, cross-repository access, remote mode, and additional authentication.
sidebar:
  order: 710
---

The GitHub Tools (`tools.github`) allow the agentic step of your workflow to read information such as issues and pull requests from GitHub.

In most workflows, no configuration of the GitHub Tools is necessary since they are included by default with the default toolsets. By default, this provides access to the current repository and all public repositories (if permitted by the network firewall).

## GitHub Toolsets

You can enable specific API groups to increase the available tools or narrow the default selection:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests, actions]
```

**Available**: `context`, `repos`, `issues`, `pull_requests`, `users`, `actions`, `code_security`, `discussions`, `labels`, `notifications`, `orgs`, `projects`, `gists`, `search`, `dependabot`, `experiments`, `secret_protection`, `security_advisories`, `stargazers`

**Shorthand values**:

- `default` — expands to `context`, `repos`, `issues`, `pull_requests`, `users`
- `all` — expands to all available toolsets **except** `dependabot` (see note below)

**Default**: `context`, `repos`, `issues`, `pull_requests`, `users`

Some key toolsets are:

- `context` (user/team info)
- `repos` (repository operations, code search, commits, releases)
- `issues` (issue management, comments, reactions)
- `pull_requests` (PR operations)
- `actions` (workflows, runs, artifacts)
- `code_security` (scanning alerts)
- `discussions` (discussions and comments)
- `labels` (labels management)

:::note
`toolsets: [all]` does **not** include the `dependabot` toolset. The `dependabot` toolset must be opted into explicitly:

```yaml wrap
tools:
  github:
    toolsets: [all, dependabot]
```

See [Using the `dependabot` toolset](#using-the-dependabot-toolset) for authentication requirements.
:::

Some toolsets require [additional authentication](#additional-authentication-for-github-tools).

## GitHub Integrity Filtering (`tools.github.min-integrity`)

Sets the minimum integrity level required for content the agent can access. For public repositories, `min-integrity: approved` is applied automatically. See [Integrity Filtering](/gh-aw/reference/integrity/) for levels, examples, user blocking, and approval labels.

## GitHub Repository Access Restrictions (`tools.github.allowed-repos`)

You can configure the GitHub Tools to be restricted in which repositories can be accessed via the GitHub tools during AI engine execution.

The setting `tools.github.allowed-repos` specifies which repositories the agent can access through GitHub tools:

- `"all"` — All repositories accessible by the configured token
- `"public"` — Public repositories only
- Array of patterns — Specific repositories and wildcards:
  - `"owner/repo"` — Exact repository match
  - `"owner/*"` — All repositories under an owner
  - `"owner/prefix*"` — Repositories with a name prefix under an owner

This defaults to `"all"` when omitted. Patterns must be lowercase. Wildcards are only permitted at the end of the repository name component.

For example:

```yaml wrap
tools:
  github:
    mode: remote
    toolsets: [default]
    allowed-repos:
      - "myorg/*"
      - "partner/shared-repo"
      - "myorg/api-*"
    min-integrity: approved
```

:::note
The `repos` field was renamed to `allowed-repos` to better reflect its purpose. If you have existing workflows using `repos`, run `gh aw fix` to automatically migrate them to `allowed-repos`.
:::

### GitHub Cross-Repository Reading

By default, the GitHub Tools can read from the current repository and all public repositories (if permitted by the network firewall). To read from other private repositories, you must configure additional authentication. See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for details and examples.

## GitHub Tools Access Modes

The `tools.github.mode` field controls how the agent accesses GitHub. Three values are supported:

| Mode | Transport | Notes |
|------|-----------|-------|
| `local` (default) | Docker-based GitHub MCP Server inside the Actions VM | No extra authentication required |
| `remote` | Hosted GitHub MCP Server managed by GitHub | Requires [additional authentication](#additional-authentication-for-github-tools) |
| `gh-proxy` | Pre-authenticated `gh` CLI directly (no MCP server) | Preferred for performance; required for [integrity reactions](/gh-aw/reference/integrity/) |

**`remote` mode** — uses a hosted MCP server managed by GitHub. Requires a GitHub token with appropriate permissions:

```yaml wrap
tools:
  github:
    mode: remote
    github-token: ${{ secrets.CUSTOM_PAT }}  # Required for remote mode
```

**`gh-proxy` mode** — uses the pre-authenticated `gh` CLI directly instead of an MCP server. This offers lower latency because there is no MCP server startup overhead, and it is required for workflows that use [integrity reactions](/gh-aw/reference/integrity/). The legacy `features: {cli-proxy: true}` feature flag is equivalent and is still accepted for backward compatibility.

```yaml wrap
tools:
  github:
    mode: gh-proxy
```

## Additional Authentication for GitHub Tools

In some circumstances you must use a GitHub PAT or GitHub app to give the GitHub tools used by your workflow additional capabilities.

This authentication relates to **reading** information from GitHub. Additional authentication to write to GitHub is handled separately through various [Safe Outputs](/gh-aw/reference/safe-outputs/).

This is required when your workflow requires any of the following:

- Read access to GitHub org or user information
- Read access to other private repos
- Read access to projects
- GitHub tools [Remote Mode](#github-tools-remote-mode)

### Using a Personal Access Token (PAT)

If additional authentication is required, one way is to create a fine-grained PAT with appropriate permissions, add it as a repository secret, and reference it in your workflow:

1. Create a [fine-grained PAT](https://github.com/settings/personal-access-tokens/new?description=GitHub+Agentic+Workflows+-+GitHub+tools+access&contents=read&issues=read&pull_requests=read) (this link pre-fills the description and common read permissions) with:

   - **Repository access**:
     - Select specific repos or "All repositories"
   - **Repository permissions** (based on your GitHub tools usage):
     - Contents: Read (minimum for toolset: repos)
     - Issues: Read (for toolset: issues)
     - Pull requests: Read (for toolset: pull_requests)
     - Projects: Read (for toolset: projects)
     - Security Events: Read (for toolset: dependabot, code_security, secret_protection, security_advisories)
     - Remote mode: no additional permissions required
     - Adjust based on the toolsets you configure in your workflow
   - **Organization permissions** (if accessing org-level info):
     - Members: Read (for org member info in context)
     - Teams: Read (for team info in context)
     - Adjust based on the toolsets you configure in your workflow

2. Add it to your repository secrets, either by CLI or GitHub UI:

   ```bash wrap
   gh aw secrets set MY_PAT_FOR_GITHUB_TOOLS --value "<your-pat-token>"
   ```

3. Configure in your workflow frontmatter:

   ```yaml wrap
   tools:
     github:
       github-token: ${{ secrets.MY_PAT_FOR_GITHUB_TOOLS }}
   ```

### Using a GitHub App

Alternatively, you can use a GitHub App for enhanced security. See [Using a GitHub App for Authentication](/gh-aw/reference/auth/#using-a-github-app-for-authentication) for complete setup instructions.

### Using a magic secret

Alternatively, you can set the magic secret `GH_AW_GITHUB_MCP_SERVER_TOKEN` to a suitable PAT (see the above guide for creating one). This secret name is known to GitHub Agentic Workflows and does not need to be explicitly referenced in your workflow.

```bash wrap
gh aw secrets set GH_AW_GITHUB_MCP_SERVER_TOKEN --value "<your-pat-token>"
```

### Using the `dependabot` toolset

The `dependabot` toolset requires the `vulnerability-alerts: read` and `security-events: read` permissions. These are now supported natively by `GITHUB_TOKEN`. Add them to your workflow's `permissions:` field:

```yaml
permissions:
  vulnerability-alerts: read
  security-events: read
```

Alternatively, you can authenticate with a PAT or GitHub App. If using a GitHub App, add `vulnerability-alerts: read` to your workflow's `permissions:` field and ensure the GitHub App is configured with this permission.

## Related Documentation

- [Tools Reference](/gh-aw/reference/tools/) - All tool configurations
- [Authentication Reference](/gh-aw/reference/auth/) - Token setup and permissions
- [Integrity Filtering](/gh-aw/reference/integrity/) - Public repository content filtering
- [MCPs Guide](/gh-aw/guides/mcps/) - Model Context Protocol setup
