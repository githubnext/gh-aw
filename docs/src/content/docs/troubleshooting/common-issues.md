---
title: Common Issues
description: Frequently encountered issues when working with GitHub Agentic Workflows and their solutions.
sidebar:
  order: 200
---

This reference documents frequently encountered issues when working with GitHub Agentic Workflows, organized by workflow stage and component.

## Workflow Compilation Issues

### Workflow Won't Compile

**Symptoms:** Running `gh aw compile` fails with errors.

**Common Causes:**

1. **YAML syntax errors in frontmatter**
   - Check indentation (use spaces, not tabs)
   - Verify colons have spaces after them
   - Ensure arrays use proper `[item1, item2]` or list syntax

2. **Missing required fields**
   - The `on:` trigger is required
   - Verify all required fields for your chosen trigger type

3. **Invalid field values**
   - Check field types match schema (strings, integers, booleans)
   - Verify enum values are correct (e.g., `engine: copilot`)

**Solution:**
```bash
# Compile with verbose output to see detailed errors
gh aw compile --verbose

# Validate YAML syntax separately
cat .github/workflows/my-workflow.md | head -20 | grep -A 20 "^---"
```

### Lock File Not Generated

**Symptoms:** The `.lock.yml` file is not created after compilation.

**Common Causes:**

1. **Compilation errors prevent generation**
   - Review compilation output for errors
   - Fix all schema validation errors first

2. **File permissions**
   - Ensure write permissions on `.github/workflows/` directory

**Solution:**
```bash
# Check for compilation errors
gh aw compile 2>&1 | grep -i error

# Verify directory permissions
ls -la .github/workflows/
```

### Orphaned Lock Files

**Symptoms:** Old `.lock.yml` files remain after deleting `.md` files.

**Solution:**
```bash
# Remove orphaned lock files
gh aw compile --purge
```

## Import and Include Issues

### Import File Not Found

**Symptoms:** Error message about failed import resolution.

**Common Causes:**

1. **Incorrect path**
   - Import paths are relative to repository root
   - Verify the file exists at the specified location

2. **File not committed**
   - Imported files must be committed to the repository
   - Check `git status` for untracked files

**Solution:**
```yaml
# Use correct import paths
imports:
  - .github/workflows/shared/tools.md  # From repo root
  - shared/security-notice.md          # Relative to .github/workflows/
```

### Multiple Agent Files Error

**Symptoms:** Error about multiple agent files in imports.

**Cause:** More than one file under `.github/agents/` is imported.

**Solution:** Import only one agent file per workflow:

```yaml
# Incorrect
imports:
  - .github/agents/agent1.md
  - .github/agents/agent2.md
  - shared/tools.md

# Correct
imports:
  - .github/agents/agent1.md
  - shared/tools.md
```

### Circular Import Dependencies

**Symptoms:** Compilation hangs or fails with stack overflow.

**Cause:** Import files import each other, creating a circular dependency.

**Solution:** Review import chains and remove circular references:

```yaml
# File A imports File B
# File B imports File A  ← Remove this circular dependency
```

## Tool Configuration Issues

### GitHub Tools Not Available

**Symptoms:** Workflow cannot use GitHub API tools.

**Common Causes:**

1. **Tools not configured**
   - GitHub tools require explicit configuration
   - Check the `tools:` section in frontmatter

2. **Incorrect tool names**
   - Verify tool names match the allowed list
   - See [tools reference](/gh-aw/reference/tools/)

**Solution:**
```yaml
tools:
  github:
    allowed:
      - get_repository
      - list_issues
      - create_issue_comment
```

### MCP Server Connection Failures

**Symptoms:** Workflow fails to connect to MCP servers.

**Common Causes:**

1. **Server not installed**
   - Verify MCP server package is available
   - Check Docker container is accessible

2. **Configuration errors**
   - Validate MCP server configuration syntax
   - Ensure required environment variables are set

**Solution:**
```yaml
mcp-servers:
  my-server:
    command: "npx"
    args: ["@myorg/mcp-server"]
    env:
      API_KEY: "${{ secrets.MCP_API_KEY }}"
```

### Playwright Network Access Denied

**Symptoms:** Playwright tools fail with network errors.

**Cause:** Domain is not in the allowed list.

**Solution:** Add domains to `allowed_domains`:

```yaml
tools:
  playwright:
    allowed_domains:
      - "github.com"
      - "*.github.io"
```

## Permission Issues

### Write Operations Fail

**Symptoms:** Cannot create issues, comments, or pull requests.

**Common Causes:**

1. **Missing permissions**
   - Check the `permissions:` section
   - Verify required permissions are granted

2. **Read-only token**
   - The workflow might be using a read-only token
   - Check token configuration

**Solution:**
```yaml
# For direct write operations
permissions:
  contents: read
  issues: write
  pull-requests: write

# Or use safe-outputs (recommended)
permissions:
  contents: read
safe-outputs:
  create-issue:
  add-comment:
```

### Safe Outputs Not Creating Issues

**Symptoms:** Workflow completes but no issues are created.

**Common Causes:**

1. **Safe outputs not configured correctly**
   - Verify `safe-outputs:` syntax
   - Check the workflow output format

2. **Staged mode enabled**
   - Safe outputs in staged mode only preview
   - Set `staged: false` for actual creation

**Solution:**
```yaml
safe-outputs:
  staged: false  # Ensure not in preview mode
  create-issue:
    title-prefix: "[bot] "
    labels: [automation]
```

### Token Permission Errors

**Symptoms:** "Resource not accessible by integration" errors.

**Cause:** The `GITHUB_TOKEN` lacks required permissions.

**Solution:** Add permissions to the workflow or use a Personal Access Token:

```yaml
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

**Symptoms:** "copilot: command not found" in workflow logs.

**Cause:** The Copilot CLI is not installed or not in PATH.

**Solution:** Ensure the Copilot CLI installation step is included in the workflow. This is typically handled automatically by the compiled workflow.

### Claude Engine Timeout

**Symptoms:** Workflow times out before completing.

**Cause:** Task is too complex or `max-turns` is set too high.

**Solution:** Reduce scope or adjust timeout:

```yaml
timeout_minutes: 30
engine:
  id: claude
  max-turns: 5  # Reduce if timing out
```

### Model Not Available

**Symptoms:** "Model not found" or similar errors.

**Cause:** Specified model is not available for the engine.

**Solution:** Use default model or verify model availability:

```yaml
# Let engine use default model
engine: copilot

# Or specify available model
engine:
  id: copilot
  model: gpt-4
```

## Context Expression Issues

### Unauthorized Expression

**Symptoms:** Compilation fails with "unauthorized expression" error.

**Cause:** Using a GitHub Actions expression that is not in the allowed list.

**Solution:** Use only [allowed expressions](/gh-aw/reference/templating/):

```yaml
# Allowed
${{ github.event.issue.number }}
${{ github.repository }}
${{ needs.activation.outputs.text }}

# Not allowed
${{ secrets.GITHUB_TOKEN }}
${{ env.MY_VAR }}
```

### Sanitized Context Empty

**Symptoms:** `needs.activation.outputs.text` is empty.

**Cause:** Workflow was not triggered by an issue, PR, or comment event.

**Solution:** The sanitized context is only populated for specific events:

```yaml
on:
  issues:
    types: [opened]  # Populates sanitized context
  push:              # Does not populate sanitized context
```

## Build and Test Issues

### Documentation Build Fails

**Symptoms:** `npm run build` fails in the `docs/` directory.

**Common Causes:**

1. **Dependencies not installed**
   ```bash
   cd docs && npm install
   ```

2. **Syntax errors in markdown**
   - Check for malformed frontmatter
   - Verify MDX syntax is valid

3. **Broken links**
   - Fix internal link paths
   - Ensure referenced files exist

**Solution:**
```bash
# Clean install
cd docs
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Tests Failing After Changes

**Symptoms:** `make test` or `make test-unit` fails.

**Common Causes:**

1. **Go code syntax errors**
   - Run `make fmt` to format code
   - Run `make lint` to check for issues

2. **Test expectations outdated**
   - Review test failures and update expectations
   - Ensure changes maintain backward compatibility

**Solution:**
```bash
# Format and lint
make fmt
make lint

# Run tests
make test-unit
```

## Network and Connectivity Issues

### Cannot Download Remote Imports

**Symptoms:** Error downloading workflow from remote repository.

**Cause:** Network connectivity or authentication issues.

**Solution:**
```bash
# Verify network access
curl -I https://raw.githubusercontent.com/githubnext/gh-aw/main/README.md

# Check GitHub authentication
gh auth status
```

### MCP Server Connection Timeout

**Symptoms:** Timeout when connecting to HTTP MCP servers.

**Cause:** Server is not responding or network is blocked.

**Solution:**
```yaml
# Use local MCP server instead of HTTP
mcp-servers:
  my-server:
    type: local  # Change from http
    command: "node"
    args: ["./server.js"]
```

## Cache Issues

### Cache Not Restoring

**Symptoms:** Cache is not restored between workflow runs.

**Common Causes:**

1. **Cache key changed**
   - Verify cache key pattern matches
   - Check if dependencies changed

2. **Cache expired**
   - GitHub Actions caches expire after 7 days
   - Rebuild cache if expired

**Solution:**
```yaml
cache:
  key: deps-${{ hashFiles('package-lock.json') }}
  restore-keys: |
    deps-  # Fallback pattern
```

### Cache Memory Not Persisting

**Symptoms:** Memory MCP server loses data between runs.

**Cause:** Cache configuration is incorrect or cache is not being saved.

**Solution:**
```yaml
tools:
  cache-memory:
    key: memory-${{ github.workflow }}-${{ github.run_id }}
```

## Debugging Strategies

### Enable Verbose Logging

```bash
# Compile with verbose output
gh aw compile --verbose

# Run workflow with debug logging
# (Set repository secret ACTIONS_STEP_DEBUG = true)
```

### Inspect Generated Workflow

```bash
# View the generated lock file
cat .github/workflows/my-workflow.lock.yml

# Compare with source
diff <(cat .github/workflows/my-workflow.md) \
     <(cat .github/workflows/my-workflow.lock.yml)
```

### Check MCP Configuration

```bash
# Inspect MCP servers in workflow
gh aw mcp inspect my-workflow

# List tools available
gh aw mcp list-tools github my-workflow
```

### Review Workflow Logs

```bash
# Download logs for analysis
gh aw logs my-workflow

# Audit specific run
gh aw audit RUN_ID
```

## Getting Help

If the issue persists after trying these solutions:

1. **Check documentation:** Review [reference docs](/gh-aw/reference/workflow-structure/) for detailed configuration options
2. **Search issues:** Look for similar issues in the [GitHub repository](https://github.com/githubnext/gh-aw/issues)
3. **Enable debugging:** Use verbose flags and review logs carefully
4. **Report bugs:** Create an issue with reproduction steps and error messages

For more information, see:
- [Error Reference](/gh-aw/troubleshooting/errors/)
- [Validation Timing](/gh-aw/troubleshooting/validation-timing/)
- [Frontmatter Reference](/gh-aw/reference/frontmatter/)
