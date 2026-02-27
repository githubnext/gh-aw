---
title: GitHub Tools
description: Configure GitHub API operations, toolsets, modes, and authentication for your agentic workflows
sidebar:
  order: 710
---

Configure GitHub API operations available to your workflow through the Model Context Protocol (MCP).

```yaml wrap
tools:
  github:                                      # Default read-only access
  github:
    toolsets: [repos, issues, pull_requests]   # Recommended: toolset groups
    mode: remote                               # "local" (Docker) or "remote" (hosted)
    read-only: true                            # Read-only operations
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
     - Lockdown mode: no additional permissions required
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

## Related Documentation

- [Tools Reference](/gh-aw/reference/tools/) - All tool configurations
- [Authentication Reference](/gh-aw/reference/auth/) - Token setup and permissions
- [Lockdown Mode](/gh-aw/reference/lockdown-mode/) - Public repository security
- [MCPs Guide](/gh-aw/guides/mcps/) - Model Context Protocol setup
