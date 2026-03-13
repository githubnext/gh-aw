# Custom API Endpoint Configuration

This guide explains how to configure GitHub Agentic Workflows to use custom API endpoints for GitHub Enterprise Cloud (GHEC), GitHub Enterprise Server (GHES), or custom AI endpoints.

## Overview

GitHub Agentic Workflows supports custom API endpoints through the `engine.api-target` configuration field. This allows you to specify custom endpoints for:

- **GitHub Enterprise Cloud (GHEC)** - Tenant-specific Copilot API endpoints
- **GitHub Enterprise Server (GHES)** - Enterprise Copilot API endpoints
- **Custom AI Endpoints** - Custom OpenAI-compatible or Anthropic-compatible endpoints

## Configuration

To configure a custom API endpoint, add the `api-target` field to your engine configuration:

**Basic Configuration:**

```yaml
---
engine:
  id: copilot
  api-target: api.acme.ghe.com
network:
  allowed:
    - defaults
    - acme.ghe.com
    - api.acme.ghe.com
---
```

The `api-target` field accepts a hostname (without protocol or path) and works with any agentic engine.

## Examples

### GitHub Enterprise Cloud (GHEC)

For GHEC tenants (domains ending with `.ghe.com`), specify your tenant-specific API endpoint:

**Workflow Configuration:**

```yaml
---
engine:
  id: copilot
  api-target: api.acme.ghe.com
network:
  allowed:
    - defaults
    - acme.ghe.com
    - api.acme.ghe.com
---
```

**Required domains in network allowlist:**
- `acme.ghe.com` - Your GHEC tenant domain (git operations, web UI)
- `api.acme.ghe.com` - Your tenant-specific Copilot API endpoint
- `raw.githubusercontent.com` - Raw content access (if using GitHub MCP server)

### GitHub Enterprise Server (GHES)

For GHES instances (custom domains), specify the enterprise Copilot endpoint:

**Workflow Configuration:**

```yaml
---
engine:
  id: copilot
  api-target: api.enterprise.githubcopilot.com
network:
  allowed:
    - defaults
    - github.company.com
    - api.enterprise.githubcopilot.com
---
```

**Required domains in network allowlist:**
- `github.company.com` - Your GHES instance (git operations, web UI)
- `api.enterprise.githubcopilot.com` - Enterprise Copilot API endpoint (used for all GHES instances)

### Custom AI Endpoints

The `api-target` field works with any agentic engine, allowing you to use custom AI endpoints:

**Workflow Configuration:**

```yaml
---
engine:
  id: codex
  api-target: api.custom.ai-provider.com
network:
  allowed:
    - defaults
    - api.custom.ai-provider.com
---
```

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
  api-target: api.acme.ghe.com
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
  api-target: api.enterprise.githubcopilot.com
network:
  allowed:
    - defaults
    - github.company.com
    - api.enterprise.githubcopilot.com
---

# Your workflow prompt here
```

### Custom AI Provider

```yaml
---
description: Workflow with custom AI endpoint
on:
  workflow_dispatch:
permissions:
  contents: read
engine:
  id: codex
  api-target: api.custom.ai-provider.com
network:
  allowed:
    - defaults
    - api.custom.ai-provider.com
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
- `--copilot-api-target` flag in AWF command (if using Copilot engine)
- Correct API endpoint hostname in the flag value

### 2. Check Workflow Runs

In GitHub Actions workflow runs:
1. Go to the agent job
2. Check the "Run Copilot Agent" (or equivalent) step
3. Verify the AWF command includes the correct API target
4. Check AWF logs for API connection messages

## Troubleshooting

### Wrong API Endpoint

**Problem:** Traffic is going to the wrong API endpoint

**Solutions:**
1. Verify `engine.api-target` is set correctly in your workflow frontmatter
2. Check that the domain is in your `network.allowed` list
3. Review AWF logs in the workflow run for endpoint configuration messages
4. Ensure you're not using a full URL (use hostname only: `api.acme.ghe.com` not `https://api.acme.ghe.com`)

### Domain Not Whitelisted

**Problem:** Requests are blocked with network errors

**Solution:** Add the missing domain to your `network.allowed` list:
- For GHEC: `[acme.ghe.com, api.acme.ghe.com]`
- For GHES: `[github.company.com, api.enterprise.githubcopilot.com]`
- For custom AI: `[api.custom.ai-provider.com]`

### GitHub MCP Server Issues

**Problem:** GitHub MCP server fails to connect to your enterprise instance

**Solutions:**
1. Ensure your GHEC/GHES domain is in `network.allowed`
2. Verify the GitHub token has appropriate scopes for your enterprise tenant
3. Use `mode: remote` for the GitHub MCP server when on GHEC/GHES

## Related Documentation

- [AWF Firewall Configuration](https://github.com/github/gh-aw-firewall) - Detailed AWF documentation
- [GitHub Actions Environment Variables](https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables) - Default GitHub Actions variables
- [Network Permissions](network.md) - Network access configuration
- [Tools Configuration](tools.md) - MCP server and tool setup
