---
title: Common Issues
description: Frequently encountered issues when working with GitHub Agentic Workflows and their solutions.
sidebar:
  order: 200
---

This reference documents frequently encountered issues when working with GitHub Agentic Workflows, organized by workflow stage and component.

## Workflow Compilation Issues

### Workflow Won't Compile

If `gh aw compile` fails, check YAML frontmatter syntax (proper indentation with spaces, colons with spaces after them), verify required fields like `on:` are present, and ensure field types match the schema. Use `gh aw compile --verbose` for detailed error messages.

### Lock File Not Generated

If `.lock.yml` isn't created, fix compilation errors first (`gh aw compile 2>&1 | grep -i error`) and verify write permissions on `.github/workflows/`.

### Orphaned Lock Files

Remove old `.lock.yml` files after deleting `.md` files with `gh aw compile --purge`.

## Import and Include Issues

### Import File Not Found

Import paths are relative to repository root. Verify the file exists and is committed (`git status`). Example paths: `.github/workflows/shared/tools.md` (from repo root) or `shared/security-notice.md` (relative to `.github/workflows/`).

### Multiple Agent Files Error

Import only one file from `.github/agents/` per workflow. Use other imports for shared content like tools.

### Circular Import Dependencies

If compilation hangs, check for import files that import each other. Remove circular references by reviewing the import chain.

## Tool Configuration Issues

### GitHub Tools Not Available

Configure GitHub tools explicitly in the `tools:` section with correct tool names from the [tools reference](/gh-aw/reference/tools/):

```yaml wrap
tools:
  github:
    allowed: [get_repository, list_issues, create_issue_comment]
```

### MCP Server Connection Failures

Verify the MCP server package is installed and configuration syntax is valid. Ensure required environment variables are set:

```yaml wrap
mcp-servers:
  my-server:
    command: "npx"
    args: ["@myorg/mcp-server"]
    env:
      API_KEY: "${{ secrets.MCP_API_KEY }}"
```

### Playwright Network Access Denied

Add blocked domains to the `allowed_domains` list:

```yaml wrap
tools:
  playwright:
    allowed_domains: ["github.com", "*.github.io"]
```

## Permission Issues

### Write Operations Fail

Grant required permissions in the `permissions:` section or use safe-outputs (recommended):

```yaml wrap
# Direct write
permissions:
  contents: read
  issues: write
  pull-requests: write

# Safe-outputs (recommended)
permissions:
  contents: read
safe-outputs:
  create-issue:
  add-comment:
```

### Safe Outputs Not Creating Issues

Disable staged mode to create issues (not just preview):

```yaml wrap
safe-outputs:
  staged: false
  create-issue:
    title-prefix: "[bot] "
    labels: [automation]
```

### Token Permission Errors

Add permissions to `GITHUB_TOKEN` or use a custom token:

```yaml wrap
# Increase GITHUB_TOKEN permissions
permissions:
  contents: write
  issues: write

# Or use custom token
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}
  create-issue:
```

## Engine-Specific Issues

### Copilot CLI Not Found

The compiled workflow should automatically include CLI installation steps. If missing, verify compilation succeeded.

### Model Not Available

Use the default model or specify an available one:

```yaml wrap
engine: copilot  # Default model

# Or specify model
engine:
  id: copilot
  model: gpt-4
```

## Context Expression Issues

### Unauthorized Expression

Use only [allowed expressions](/gh-aw/reference/templating/) like `github.event.issue.number`, `github.repository`, or `needs.activation.outputs.text`. Expressions like `secrets.GITHUB_TOKEN` or `env.MY_VAR` are not allowed.

### Sanitized Context Empty

`needs.activation.outputs.text` is only populated for issue, PR, or comment events (e.g., `on: issues:`) but not for other triggers like `push:`.

## Build and Test Issues

### Documentation Build Fails

Install dependencies, check for malformed frontmatter or MDX syntax, and fix broken links:

```bash wrap
cd docs
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Tests Failing After Changes

Format code and check for issues before running tests:

```bash wrap
make fmt
make lint
make test-unit
```

## Network and Connectivity Issues

### Cannot Download Remote Imports

Verify network access and GitHub authentication:

```bash wrap
curl -I https://raw.githubusercontent.com/githubnext/gh-aw/main/README.md
gh auth status
```

### MCP Server Connection Timeout

Use a local MCP server if HTTP connections timeout:

```yaml wrap
mcp-servers:
  my-server:
    command: "node"
    args: ["./server.js"]
```

## Cache Issues

### Cache Not Restoring

Verify cache key patterns match and note that caches expire after 7 days:

```yaml wrap
cache:
  key: deps-${{ hashFiles('package-lock.json') }}
  restore-keys: deps-
```

### Cache Memory Not Persisting

Configure cache properly for the memory MCP server:

```yaml wrap
tools:
  cache-memory:
    key: memory-${{ github.workflow }}-${{ github.run_id }}
```

## Debugging Strategies

Enable verbose compilation (`gh aw compile --verbose`), set `ACTIONS_STEP_DEBUG = true` for debug logging, inspect generated lock files (`cat .github/workflows/my-workflow.lock.yml`), check MCP configuration (`gh aw mcp inspect my-workflow`), and review logs (`gh aw logs my-workflow` or `gh aw audit RUN_ID`).

## Getting Help

Review [reference docs](/gh-aw/reference/workflow-structure/), search [existing issues](https://github.com/githubnext/gh-aw/issues), enable debugging with verbose flags, or create a new issue with reproduction steps. See also: [Error Reference](/gh-aw/troubleshooting/errors/) and [Frontmatter Reference](/gh-aw/reference/frontmatter/).
