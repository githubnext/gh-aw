---
title: Tools
description: Configure GitHub API tools, browser automation, and AI capabilities available to your agentic workflows, including GitHub tools, Playwright, and custom MCP servers.
sidebar:
  order: 700
---

This guide covers the available tools that can be configured in agentic workflows, including GitHub tools, Playwright browser automation, custom MCP servers, and neutral tools.

Tools are defined in the frontmatter to specify which GitHub API calls, browser automation, and AI capabilities are available to your workflow:

```yaml
tools:
  edit:
  bash: true
```

Some tools are available by default. All tools declared in imported components are merged into the final workflow.

## Edit Tool (`edit:`)

This is an engine-neutral tool that allows file editing by the agentic workflow on the local workspace in GitHub Actions. This means local file edits can be made in the GitHub Actions VM on the checked out copy of the code.

```yaml
tools:
  edit:         # File editing capabilities
```

## Bash Tool (`bash:`)

This is an engine-neutral tool that allows shell command execution by the agentic workflow in the local workspace in GitHub Actions.

```yaml
tools:
  bash: true    # Default safe commands (echo, ls, pwd, cat, head, tail, grep, wc, sort, uniq, date)
  # bash: ["echo", "ls", "git status"]  # Or specify custom commands
```

The bash tool provides access to shell commands with different levels of control and security.

```yaml
tools:
  # Default commands: Provides common safe commands (echo, ls, pwd, cat, etc.)
  bash: true

  # Specific commands: Only allow specified commands
  bash: ["echo", "ls", "git status", "npm test"]

  # No commands: Bash tool enabled but no commands allowed
  bash: []

  # All commands: Unrestricted bash access (use with caution)
  bash: [":*"]
```

**Configuration Options:**

- **`bash: true`** → Adds default safe commands (`echo`, `ls`, `pwd`, `cat`, `head`, `tail`, `grep`, `wc`, `sort`, `uniq`, `date`)
- **`bash: null`** → Adds default commands (only if no git commands needed for safe outputs)  
- **`bash: []`** → No bash commands allowed (empty array preserved)
- **`bash: ["cmd1", "cmd2"]`** → Only specified commands allowed
- **`bash: [":*"]`** → All bash commands allowed (unrestricted access)

**Default Bash Commands:**

When `bash: true` or `bash: null` is specified, the system automatically provides these safe, read-only commands:

**File Operations**: `ls`, `pwd`, `cat`, `head`, `tail`  
**Text Processing**: `grep`, `sort`, `uniq`, `wc`  
**Basic Utilities**: `echo`, `date`

These defaults ensure consistent behavior across Claude and Copilot engines while maintaining security.

**Bash Wildcards:**

```yaml
tools:
  bash: [":*"]                    # Allow ALL bash commands - use with caution
  bash: ["git:*"]                 # Allow all git commands with any arguments
  bash: ["npm:*", "echo", "ls"]   # Mix of wildcards and specific commands
```

**Wildcard Options:**
- **`:*`**: Allows **all bash commands** without restriction
- **`command:*`**: Allows **all invocations of a specific command** with any arguments

## Web Fetch Tool (`web-fetch:`)

This tool allows use of a web fetch capability.

```yaml
tools:
  web-fetch:    # Web content fetching
```

## Web Search Tool (`web-search:`)

This tool allows use of a web search capability if the AI engine supports it.

```yaml
tools:
  web-search:   # Web search capabilities
```

:::note
Some engines (like Copilot) don't have built-in `web-search` support. You can add web search using third-party MCP servers instead. See the [Web Search with MCP guide](/gh-aw/guides/web-search/) for options.
:::

## GitHub Tools (`github:`)

Configure which GitHub API operations are allowed for your workflow.

```yaml
tools:
  github:
    # Uses default GitHub API access with workflow permissions
```

or the extended form:

```yaml
tools:
  github:
    allowed: [create_issue, update_issue, add_issue_comment]  # Optional: specific permissions
    mode: remote                               # Optional: "local" (Docker, default) or "remote" (hosted)
    version: "latest"                          # Optional: MCP server version (local mode only)
    args: ["--verbose", "--debug"]            # Optional: additional arguments (local mode only)
    read-only: true                           # Optional: restrict to read-only operations
    github-token: "${{ secrets.CUSTOM_PAT }}" # Optional: custom GitHub token
    toolset: [repos, issues, pull_requests]   # Optional: array of toolset groups to enable
```

### GitHub Toolsets

The `toolset` field allows you to enable or disable specific groups of GitHub API functionalities. This helps the AI model with tool selection and reduces context size by only exposing relevant tools.

**Configuration:**

```yaml
tools:
  github:
    toolset: [repos, issues, pull_requests, actions]
```

**Available Toolsets:**

| Toolset                 | Description                                                   |
| ----------------------- | ------------------------------------------------------------- |
| `context`               | **Strongly recommended**: Tools that provide context about the current user and GitHub context |
| `actions`               | GitHub Actions workflows and CI/CD operations |
| `code_security`         | Code security related tools, such as GitHub Code Scanning |
| `dependabot`            | Dependabot tools |
| `discussions`           | GitHub Discussions related tools |
| `experiments`           | Experimental features that are not considered stable yet |
| `gists`                 | GitHub Gist related tools |
| `issues`                | GitHub Issues related tools |
| `labels`                | GitHub Labels related tools |
| `notifications`         | GitHub Notifications related tools |
| `orgs`                  | GitHub Organization related tools |
| `projects`              | GitHub Projects related tools |
| `pull_requests`         | GitHub Pull Request related tools |
| `repos`                 | GitHub Repository related tools |
| `secret_protection`     | Secret protection related tools, such as GitHub Secret Scanning |
| `security_advisories`   | Security advisories related tools |
| `stargazers`            | GitHub Stargazers related tools |
| `users`                 | GitHub User related tools |

**Default Toolsets:**

If no `toolset` is specified, the GitHub MCP server uses `toolset: [all]` to enable all available toolsets.

**Default Read-Only Tools**:

**Actions**: `download_workflow_run_artifact`, `get_job_logs`, `get_workflow_run`, `list_workflows`

**Issues & PRs**: `get_issue`, `get_pull_request`, `list_issues`, `list_pull_requests`, `search_issues`

**Repository**: `get_commit`, `get_file_contents`, `list_branches`, `list_commits`, `search_code`

**Security**: `get_code_scanning_alert`, `list_secret_scanning_alerts`, `get_dependabot_alert`

**Users & Organizations**: `search_users`, `search_orgs`, `get_me`

**Note**: The `toolset` field is currently supported in local (Docker) mode only. Remote mode support may be added in future versions.

### GitHub Remote Mode

The GitHub MCP can use the hosted GitHub MCP server at `https://api.githubcopilot.com/mcp/`

**Remote Mode Configuration:**

```yaml
tools:
  github:
    mode: remote
    allowed: [list_issues, create_issue]
```

**Key Differences:**

- **Authentication**: Remote mode uses the `GITHUB_MCP_TOKEN` secret by default (the standard `GITHUB_TOKEN` is not supported by the hosted server)
- **Performance**: Remote mode eliminates the need for Docker, providing faster startup times
- **Read-only**: Controlled via the `X-MCP-Readonly: true` header instead of environment variables

**Setting up GITHUB_MCP_TOKEN:**

To use remote mode, you need to set the `GITHUB_MCP_TOKEN` secret. Create a GitHub Personal Access Token and add it to your repository:

```bash
gh secret set GITHUB_MCP_TOKEN -a actions --body "<your-github-pat>"
```

### GitHub Read-Only Mode

The `read-only` flag restricts the GitHub MCP server to read-only operations, preventing any modifications to repositories, issues, pull requests, etc.

```yaml
tools:
  github:
    read-only: true
```

**Default behavior**: When the GitHub tool is specified without any configuration (just `github:` with no properties), the default behavior provides read-only access with all read-only tools available.

## Playwright Tool (`playwright:`)

Enable browser automation and web testing capabilities using containerized Playwright:

```yaml
tools:
  playwright:
    allowed_domains: ["github.com", "*.example.com"]
```

### Playwright Configuration Options

```yaml
tools:
  playwright:
    version: "latest"                    # Optional: Playwright version
    allowed_domains: ["defaults", "github", "*.custom.com"]  # Domain access control
```

The `args` field allows you to pass additional command-line arguments to the Playwright MCP server:

```yaml
tools:
  playwright:
    args: ["--browser", "chromium"]
```

The `allowed_domains` field supports the same ecosystem bundle resolution as the top-level `network:` configuration, with **localhost-only** as the default for enhanced security:

```yaml
tools:
  playwright:
    allowed_domains: 
      - "defaults"              # Basic infrastructure domains
      - "github"               # GitHub domains (github.com, api.github.com, etc.)
      - "node"                 # Node.js ecosystem
      - "python"               # Python ecosystem
      - "*.example.com"        # Custom domain with wildcard
```

**Security Model:**
- **Default**: `["localhost", "127.0.0.1"]` - localhost access only
- **Ecosystem bundles**: Use same identifiers as `network:` configuration
- **Custom domains**: Support exact matches and wildcard patterns
- **Containerized execution**: Isolated Docker environment for security

**Available Ecosystem Identifiers:**
Same as `network:` configuration: `defaults`, `github`, `node`, `python`, `containers`, `java`, `rust`, `playwright`, etc.

## Other MCP Servers

Use the dedicated `mcp-servers:` section for other MCP server configuration:

```yaml
# In your workflow frontmatter
mcp-servers:
  custom-api:
    command: "node"
    args: ["custom-mcp-server.js"]
    env:
      API_KEY: "${{ secrets.CUSTOM_API_KEY }}"

# Separate tools section for built-in capabilities
tools:
  github:
    allowed: [create_issue, update_issue]
  playwright:
    allowed_domains: ["github.com"]
```

**MCP Server Execution:**
- MCP servers run alongside the AI engine to provide specialized capabilities
- Each server provides specific tools (APIs, database access, etc.)
- Servers run in isolated environments with controlled access
- Domain restrictions apply to network-enabled servers

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - All frontmatter configuration options
- [Network Permissions](/gh-aw/reference/network/) - Network access control for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Complete Model Context Protocol setup and usage
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
