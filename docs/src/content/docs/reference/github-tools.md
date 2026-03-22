---
title: GitHub Tools
description: Configure GitHub API operations, toolsets, modes, and authentication for your agentic workflows
sidebar:
  order: 710
---

Configure GitHub API operations available to your workflow through the Model Context Protocol (MCP).

```yaml wrap
tools:
  github:                                      # Default read-only access (always enforced)
  github:
    toolsets: [repos, issues, pull_requests]   # Recommended: toolset groups
    mode: remote                               # "local" (Docker) or "remote" (hosted)
    read-only: true                            # Always true; false is not permitted
    github-token: "${{ secrets.CUSTOM_PAT }}"  # Custom token
```

## GitHub Toolsets

Enable specific API groups to improve tool selection and reduce context size:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests, actions]
```

**Available**: `context`, `repos`, `issues`, `pull_requests`, `users`, `actions`, `code_security`, `discussions`, `labels`, `notifications`, `orgs`, `projects`, `gists`, `search`, `dependabot`, `experiments`, `secret_protection`, `security_advisories`, `stargazers`

**Default**: `context`, `repos`, `issues`, `pull_requests`, `users`

Note `toolsets: [default]` expands to `[context, repos, issues, pull_requests]` (excluding `users`) since `GITHUB_TOKEN` lacks user permissions. Use a PAT for the full default set.

Key toolsets: **context** (user/team info), **repos** (repository operations, code search, commits, releases), **issues** (issue management, comments, reactions), **pull_requests** (PR operations), **actions** (workflows, runs, artifacts), **code_security** (scanning alerts), **discussions**, **labels**.

## Remote vs Local Mode

**Remote Mode**: Use hosted MCP server for faster startup (no Docker). Requires [Additional Authentication for GitHub Tools](/gh-aw/reference/github-tools/#additional-authentication-for-github-tools):

```yaml wrap
tools:
  github:
    mode: remote  # Default: "local" (Docker)
    github-token: ${{ secrets.CUSTOM_PAT }}  # Required for remote mode
```

**Local Mode**: Use Docker container for isolation. Requires `docker` tool and appropriate permissions:

```yaml wrap
tools:
  docker:
  github:
    mode: local
```

## Guard Policies

Restrict which repositories and integrity levels the GitHub MCP server can access during agent execution. Guard policies apply fine-grained access control at the MCP gateway level.

For **public repositories** without explicit guard policy configuration, `min-integrity: approved` is applied automatically at runtime, ensuring content is filtered to owners, members, and collaborators even without additional authentication. See [Automatic Minimum-Integrity Protection](/gh-aw/reference/lockdown-mode/#automatic-minimum-integrity-protection) for details.

```yaml wrap
tools:
  github:
    mode: remote
    toolsets: [default]
    repos: "all"
    min-integrity: unapproved
```

`min-integrity` can be specified alone; `repos` defaults to `"all"` when omitted. If `repos` is also specified, both fields must be valid.

### `repos`

Specifies which repositories the agent can access through GitHub tools:

- `"all"` — All repositories accessible by the configured token
- `"public"` — Public repositories only
- Array of patterns — Specific repositories and wildcards:
  - `"owner/repo"` — Exact repository match
  - `"owner/*"` — All repositories under an owner
  - `"owner/prefix*"` — Repositories with a name prefix under an owner

Patterns must be lowercase. Wildcards are only permitted at the end of the repository name component.

```yaml wrap
tools:
  github:
    mode: remote
    toolsets: [default]
    repos:
      - "myorg/*"
      - "partner/shared-repo"
      - "myorg/api-*"
    min-integrity: approved
```

### `min-integrity`

Sets the minimum integrity level required for repository access.

#### Integrity Level Definitions

Integrity levels are determined based on the combination of the `author_association` field associated with GitHub objects (issues, pull requests, comments, etc.) and whether an object is reachable from the main branch:

| Level | Criteria |
|-------|----------|
| `merged` | Objects reachable from the main branch (regardless of authorship) |
| `approved` | Objects with `author_association` of `OWNER`, `MEMBER`, or `COLLABORATOR` |
| `unapproved` | Objects with `author_association` of `CONTRIBUTOR` or `FIRST_TIME_CONTRIBUTOR` |
| `none` | Objects with `author_association` of `FIRST_TIMER` or `NONE` |

**How it works:**
- **Merged content** has the highest integrity because it has been reviewed and merged into the main branch
- **Approved contributors** (owners, members, collaborators) have established trust relationships with the repository
- **Unapproved contributors** have made contributions but lack formal repository access
- **None level** includes first-time interactions and users with no prior contribution history

### Examples

**Restrict to public repositories only:**

```yaml wrap
tools:
  github:
    repos: "public"
    min-integrity: none
```

**Restrict to repositories in multiple organizations:**

```yaml wrap
tools:
  github:
    mode: remote
    toolsets: [repos, issues]
    repos:
      - "frontend-org/*"
      - "backend-org/*"
    min-integrity: approved
```

### Non-GitHub MCP Server Integration

When you configure `repos` in the GitHub guard policy, the compiler automatically derives a linked write-sink guard-policy for **all non-GitHub MCP servers** — including [safe outputs](/gh-aw/reference/safe-outputs/), playwright, serena, mcp-scripts, agentic-workflows, web-fetch, and any custom tools. This ensures that as guard policies are applied to GitHub inputs, the corresponding write operations to non-GitHub servers are permitted.

The transformation rules are:

- **`repos: "all"` or `repos: "public"`**: Creates a write-sink policy with `accept: ["*"]` to allow all write operations
- **`repos: [patterns]`**: Each entry is transformed and added as an accept entry:
  - `"owner/*"` → `"private:owner"` (owner wildcard → strip wildcard)
  - `"owner/prefix*"` → `"private:owner/prefix*"` (prefix wildcard → keep as-is)
  - `"owner/repo"` → `"private:owner/repo"` (specific repo → keep as-is)

This derivation happens at compile time and requires no additional configuration, allowing the MCP gateway to read repository data through the GitHub tools and write outputs to any configured non-GitHub MCP server.

```yaml wrap
tools:
  github:
    mode: remote
    toolsets: [default]
    repos: "public"          # Creates write-sink with accept: ["*"]
    min-integrity: approved
safe-outputs:
  create-issue:              # safe outputs can write with accept: ["*"]
```

```yaml wrap
tools:
  github:
    mode: remote
    toolsets: [default]
    repos:
      - "myorg/private-repo"   # → accept: ["private:myorg/private-repo"]
      - "myorg/another-repo"   # → accept: ["private:myorg/another-repo"]
    min-integrity: approved
safe-outputs:
  create-issue:                # safe outputs can write to the guard-policy repos
```

## Lockdown Mode for Public Repositories

Lockdown Mode is a security feature that filters public repository content to only show issues, PRs, and comments from users with push access. Automatically enabled for public repositories when using custom tokens. See [Lockdown Mode](/gh-aw/reference/lockdown-mode/) for complete documentation.

```yaml wrap
tools:
  github:
    lockdown: true   # Force enable (automatic for public repos)
    lockdown: false  # Disable (for workflows processing all user input)
```

## Additional Authentication for GitHub Tools

In some circumstances you must use a GitHub PAT or GitHub app to give the GitHub tools used by your workflow additional capabilities.

This authentication relates to **reading** information from GitHub. Additional authentication to write to GitHub is handled separately through various [Safe Outputs](/gh-aw/reference/safe-outputs/).

This is required when your workflow requires any of the following:

- Read access to GitHub org or user information
- Read access to other private repos
- Read access to projects
- GitHub tools [Lockdown Mode](/gh-aw/reference/lockdown-mode/)
- GitHub tools [Remote Mode](#remote-vs-local-mode)

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
     - Lockdown mode: no additional permissions required
     - Remote mode: no additional permissions required
     - Adjust based on the toolsets you configure in your workflow

   > [!NOTE]
   > The `dependabot` toolset also requires the `vulnerability-alerts` GitHub App permission. If you are using a GitHub App (rather than a PAT), add `vulnerability-alerts: read` to your workflow's `permissions:` field and ensure the GitHub App is configured with this permission. See [GitHub App-Only Permissions](/gh-aw/reference/permissions/#github-app-only-permissions).
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

## Cross-Repository Reading

When GitHub Tools need to read information from repositories other than the one where the workflow is running, additional authorization is required. The default `GITHUB_TOKEN` only has access to the current repository.

Configure cross-repository read access using the same authentication methods described above:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests]
    github-token: ${{ secrets.CROSS_REPO_PAT }}
```

This enables operations like:
- Reading files and searching code in external repositories
- Querying issues and pull requests from other repos
- Accessing commits, releases, and workflow runs across repositories
- Reading organization-level information

> [!NOTE]
> This authorization is for **reading** from GitHub. For **writing** to other repositories (creating issues, PRs, comments), configure authentication separately through [Safe Outputs](/gh-aw/reference/safe-outputs/) with cross-repository operations.

For complete cross-repository workflow patterns and examples, see [Cross-Repository Operations](/gh-aw/reference/cross-repository/).

## Related Documentation

- [Tools Reference](/gh-aw/reference/tools/) - All tool configurations
- [Authentication Reference](/gh-aw/reference/auth/) - Token setup and permissions
- [Lockdown Mode](/gh-aw/reference/lockdown-mode/) - Public repository security
- [MCPs Guide](/gh-aw/guides/mcps/) - Model Context Protocol setup
