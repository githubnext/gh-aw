# Enterprise Configuration for Copilot Agents

This guide explains how to configure GitHub Agentic Workflows for GitHub Enterprise Cloud (GHEC) and GitHub Enterprise Server (GHES) customers.

## Overview

GitHub Agentic Workflows automatically detects your GitHub environment and configures the appropriate Copilot API endpoints through AWF (Agentic Workflow Firewall). The system intelligently routes GitHub Copilot API traffic based on your enterprise configuration.

## Automatic Detection (Recommended)

AWF automatically detects GitHub Enterprise environments based on the `GITHUB_SERVER_URL` environment variable, which is set by GitHub Actions in enterprise environments, unless you explicitly override detection with `engine.enterprise.server-url` in your workflow frontmatter.

### GitHub Enterprise Cloud (GHEC)

For GHEC tenants (domains ending with `.ghe.com`), AWF automatically extracts the subdomain and routes to the tenant-specific API endpoint.

**Workflow Configuration (automatic detection):**

```yaml
---
engine:
  id: copilot
network:
  allowed:
    - defaults
    - acme.ghe.com
    - api.acme.ghe.com
---
```

**How it works:**
1. AWF reads `GITHUB_SERVER_URL` from the environment
2. Detects that the hostname ends with `.ghe.com`
3. Extracts the subdomain (e.g., `acme` from `acme.ghe.com`)
4. Routes Copilot API traffic to `api.acme.ghe.com`

If `GITHUB_SERVER_URL` is not set (for example, when running outside of GitHub Actions) or you need to force a specific tenant, you can override automatic detection by adding an explicit server URL:

```yaml
engine:
  id: copilot
  enterprise:
    server-url: "https://acme.ghe.com"
```

**Required domains in network allowlist:**
- `acme.ghe.com` - Your GHEC tenant domain (git operations, web UI)
- `api.acme.ghe.com` - Your tenant-specific Copilot API endpoint
- `raw.githubusercontent.com` - Raw content access (if using GitHub MCP server)

### GitHub Enterprise Server (GHES)

For GHES instances (custom domains), AWF automatically routes to the enterprise Copilot endpoint based on `GITHUB_SERVER_URL`.

**Workflow Configuration (automatic detection):**

```yaml
---
engine:
  id: copilot
network:
  allowed:
    - defaults
    - github.company.com
    - api.enterprise.githubcopilot.com
---
```

If `GITHUB_SERVER_URL` is not available or you need to force a specific GHES URL, you can override automatic detection with:

```yaml
engine:
  id: copilot
  enterprise:
    server-url: "https://github.company.com"
```

**How it works:**
1. AWF reads `GITHUB_SERVER_URL` from the environment
2. Detects that the hostname is not `github.com` or `*.ghe.com`
3. Routes Copilot API traffic to `api.enterprise.githubcopilot.com`

**Required domains in network allowlist:**
- `github.company.com` - Your GHES instance (git operations, web UI)
- `api.enterprise.githubcopilot.com` - Enterprise Copilot API endpoint (used for all GHES instances)

## Manual Override

If automatic detection doesn't work for your setup, you can manually specify the Copilot API endpoint.

**Workflow Configuration:**

```yaml
---
engine:
  id: copilot
  enterprise:
    copilot-api-target: "api.custom.endpoint.com"
network:
  allowed:
    - defaults
    - custom.endpoint.com
    - api.custom.endpoint.com
---
```

The `copilot-api-target` field takes precedence over automatic detection.

## Priority Order

AWF determines the Copilot API endpoint in this order:

1. **`engine.enterprise.copilot-api-target`** (highest priority) - Manual override
2. **`engine.enterprise.server-url`** with `*.ghe.com` - Automatic GHEC detection → `api.<subdomain>.ghe.com`
3. **`engine.enterprise.server-url`** with custom domain - Automatic GHES detection → `api.enterprise.githubcopilot.com`
4. **GitHub Actions `GITHUB_SERVER_URL`** - Uses the environment variable set by GitHub Actions
5. **Default** - Public GitHub → `api.githubcopilot.com`

## Complete Examples

### GHEC with GitHub MCP Server

```yaml
---
description: Workflow for GHEC environment with GitHub API access
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
  pull-requests: write
engine:
  id: copilot
  enterprise:
    server-url: "https://acme.ghe.com"
tools:
  github:
    mode: remote
    toolsets: [default]
network:
  allowed:
    - defaults
    - acme.ghe.com
    - api.acme.ghe.com
    - raw.githubusercontent.com
---

# Your workflow prompt here
```

### GHES with Custom Endpoint

```yaml
---
description: Workflow for GHES environment
on:
  issue_comment:
    types: [created]
permissions:
  contents: read
  issues: write
engine:
  id: copilot
  enterprise:
    server-url: "https://github.company.com"
network:
  allowed:
    - defaults
    - github.company.com
    - api.enterprise.githubcopilot.com
---

# Your workflow prompt here
```

### Manual Override Example

```yaml
---
description: Workflow with manual API endpoint override
on:
  workflow_dispatch:
permissions:
  contents: read
engine:
  id: copilot
  enterprise:
    copilot-api-target: "api.custom.endpoint.com"
network:
  allowed:
    - defaults
    - custom.endpoint.com
    - api.custom.endpoint.com
---

# Your workflow prompt here
```

## Verification

To verify your configuration is working correctly:

### 1. Check Compiled Workflow

After compiling your workflow, check the generated `.lock.yml` file:

```bash
gh aw compile your-workflow.md
```

Look for:
- `GITHUB_SERVER_URL` environment variable in the agent job
- `--copilot-api-target` flag in AWF command (if using manual override)

### 2. Check Workflow Runs

In GitHub Actions workflow runs:
1. Go to the agent job
2. Check the "Run Copilot Agent" step
3. Verify the AWF command includes the correct API target
4. Check AWF logs for "Copilot proxy listening" messages

## Troubleshooting

### Wrong API Endpoint

**Problem:** Traffic is going to the wrong Copilot API endpoint

**Solutions:**
1. Verify `engine.enterprise.server-url` is set correctly in your workflow frontmatter
2. Check that the domain is in your `network.allowed` list
3. Use `copilot-api-target` to manually override if automatic detection fails
4. Review AWF logs in the workflow run for endpoint detection messages

### Domain Not Whitelisted

**Problem:** Requests are blocked with network errors

**Solution:** Add the missing domain to your `network.allowed` list:
- For GHEC: `[acme.ghe.com, api.acme.ghe.com]`
- For GHES: `[github.company.com, api.enterprise.githubcopilot.com]`

### GitHub MCP Server Issues

**Problem:** GitHub MCP server fails to connect to your enterprise instance

**Solutions:**
1. Ensure your GHEC/GHES domain is in `network.allowed`
2. Verify the GitHub token has appropriate scopes for your enterprise tenant
3. Use `mode: remote` for the GitHub MCP server when on GHEC/GHES

## Related Documentation

- [AWF Enterprise Configuration](https://github.com/github/gh-aw-firewall/blob/main/docs/enterprise-configuration.md) - Detailed AWF documentation
- [GitHub Actions Environment Variables](https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables) - Default GitHub Actions variables
- [Network Permissions](network.md) - Network access configuration
- [Tools Configuration](tools.md) - MCP server and tool setup

## See Also

For more information about the underlying AWF firewall configuration that enables enterprise support, see the [gh-aw-firewall PR #1264](https://github.com/github/gh-aw-firewall/pull/1264) which adds automatic endpoint detection.
