---
title: Local Workflow Testing
description: Test workflows in seconds, not minutes, without GitHub Actions. Learn validation-only mode, watch mode, and local testing strategies.
sidebar:
  order: 10
---

Test workflows locally in seconds instead of waiting minutes for GitHub Actions runs. This guide shows you how to validate syntax, test configuration changes, and iterate rapidly without consuming Actions minutes or requiring network access.

## Quick Testing Methods

### Validation-Only Mode

The `--no-emit` flag validates your workflow without generating lock files, giving instant feedback on syntax and configuration errors:

```bash
# Validate a single workflow
gh aw compile my-workflow.md --no-emit

# Validate all workflows in the directory
gh aw compile --no-emit

# Validate with strict mode enforcement
gh aw compile my-workflow.md --no-emit --strict

# Validate with additional security scanning
gh aw compile my-workflow.md --no-emit --validate
```

**What gets validated:**
- ✅ Markdown frontmatter syntax and required fields
- ✅ YAML structure and GitHub Actions compatibility
- ✅ Expression size limits (max 21,000 characters)
- ✅ Engine configuration (Copilot, Claude, Codex, Custom)
- ✅ MCP server configuration and toolsets
- ✅ Safe-output configuration
- ✅ Network domain allowlists
- ✅ Permission specifications
- ✅ Tool allowlisting and bash commands
- ✅ Import/fragment references

**What doesn't get validated:**
- ❌ Container image availability (requires `--validate`)
- ❌ Action SHA validity (requires `--validate`)
- ❌ Runtime package availability (requires `--validate`)
- ❌ Actual workflow execution logic
- ❌ Agent behavior with specific inputs

### Watch Mode for Rapid Iteration

Watch mode automatically recompiles workflows when you save changes, providing a tight feedback loop for development:

```bash
# Watch a specific workflow
gh aw compile my-workflow.md --watch

# Watch all workflows in directory
gh aw compile --watch

# Combine watch with validation
gh aw compile my-workflow.md --watch --validate

# Watch with verbose output
gh aw compile my-workflow.md --watch --verbose
```

**Watch mode features:**
- 🔄 Auto-recompiles on file save (300ms debounce)
- 📊 Shows compilation statistics (jobs, steps, file size)
- ⚠️  Displays warnings and errors immediately
- 🔗 Tracks dependencies and recompiles dependent workflows
- 🗑️  Handles file deletion and cleanup
- 📁 Watches subdirectories for imported fragments

**Typical workflow:**
1. Open your workflow file in an editor
2. Start watch mode: `gh aw compile my-workflow.md --watch`
3. Make changes and save
4. See instant feedback in the terminal
5. Fix any errors and save again
6. Stop with Ctrl+C when done

### Quick Syntax Check

For the fastest validation (no schema checking):

```bash
# Just parse frontmatter and check basic syntax
gh aw compile my-workflow.md --no-emit --no-check-update
```

This skips update checks and focuses purely on parsing, giving sub-second feedback.

## Testing Workflow Fragments

Workflow fragments (imports) can be tested independently before using them in main workflows.

### Testing a Fragment File

Create a minimal test wrapper for your fragment:

**Fragment file:** `.github/workflows/shared/mcp/my-tools.md`
```markdown
---
mcp-servers:
  custom-server:
    container: "mcp/my-server"
    allowed: ["tool1", "tool2"]
---
```

**Test wrapper:** `.github/workflows/test-my-tools.md`
```aw wrap
---
on: workflow_dispatch
engine: copilot
imports:
  - shared/mcp/my-tools.md
---

# Test My Tools Fragment

This is a minimal workflow to test the my-tools fragment configuration.
```

Validate the fragment:
```bash
gh aw compile test-my-tools.md --no-emit
```

### Fragment Testing Patterns

**Test network configuration fragments:**
```bash
# Create test workflow with network fragment
gh aw compile test-network-config.md --no-emit --validate
```

**Test MCP server fragments:**
```bash
# Validate MCP server configuration
gh aw compile test-mcp-setup.md --no-emit
gh aw mcp inspect test-mcp-setup
```

**Test safe-output fragments:**
```bash
# Validate safe-output permissions
gh aw compile test-safe-outputs.md --no-emit --strict
```

## Testing MCP Server Configurations

Test MCP server configurations locally without running workflows in GitHub Actions.

### Inspect MCP Configuration

```bash
# View MCP servers for a workflow
gh aw mcp inspect my-workflow

# List all MCP servers in a table format
gh aw mcp list

# View specific server configuration
gh aw mcp inspect my-workflow --server github

# Launch web-based MCP inspector (interactive)
gh aw mcp inspect my-workflow --inspector
```

### Validate MCP Toolsets

```bash
# Compile and check MCP toolset expansion
gh aw compile my-workflow.md --no-emit --verbose

# View which tools are included in toolsets
gh aw mcp inspect my-workflow
```

Example output shows toolset expansion:
```text
GitHub MCP Server
  Mode: remote
  Toolsets: [default]
  Expanded Tools:
    - get_repository_info
    - list_issues
    - get_issue
    - list_pull_requests
    - get_pull_request
    ...
```

### Test Custom MCP Servers

For custom MCP servers, validate the configuration before deployment:

```aw wrap
---
on: workflow_dispatch
engine: copilot
mcp-servers:
  my-server:
    container: "ghcr.io/myorg/my-server:latest"
    allowed: ["custom_tool"]
    network:
      allowed:
        - "api.example.com"
---

# Test Custom MCP Server

Test workflow for validating custom MCP server configuration.
```

Validate:
```bash
# Check configuration syntax
gh aw compile test-custom-mcp.md --no-emit

# Verify container and network settings
gh aw mcp inspect test-custom-mcp

# Full validation with container image check
gh aw compile test-custom-mcp.md --no-emit --validate
```

## Testing Without GitHub Context

Most validation and testing can be done without being in a GitHub repository or having network access. The `gh aw compile` command works on markdown files anywhere:

```bash
# Test workflow files outside a repository
gh aw compile /path/to/workflow.md --no-emit

# Test with custom directory
gh aw compile --dir /path/to/workflows --no-emit

# Validate configuration in any location
cat workflow.md | gh aw compile --no-emit
```

**What works offline:**
- ✅ Syntax validation
- ✅ Frontmatter parsing
- ✅ YAML structure checks
- ✅ Configuration validation
- ✅ Watch mode
- ✅ MCP configuration inspection

**What requires network/GitHub:**
- ❌ Container image validation (`--validate`)
- ❌ Action SHA verification (`--validate`)
- ❌ Update checks (skip with `--no-check-update`)
- ❌ GitHub API operations in actual workflow execution

## Validation Modes Comparison

Different validation flags provide different levels of checking:

| Flag | Speed | Network Required | What It Validates |
|------|-------|------------------|-------------------|
| `--no-emit` | ⚡ Instant (~100ms) | No | Syntax, structure, frontmatter |
| `--no-emit --validate` | 🐢 Slower (~2-5s) | Yes | + Container images, Action SHAs |
| `--no-emit --strict` | ⚡ Instant | No | + Strict mode rules enforced |
| `--no-emit --actionlint` | 🐢 Slower (~1-2s) | No | + GitHub Actions linting |
| `--no-emit --zizmor` | 🐢 Slower (~2-3s) | No | + Security scanning |
| `--no-emit --poutine` | 🐢 Slower (~3-5s) | No | + Supply chain security |

**Recommended combinations:**

```bash
# Fast iteration during development
gh aw compile my-workflow.md --no-emit

# Pre-commit validation
gh aw compile my-workflow.md --no-emit --strict --actionlint

# Pre-merge validation (comprehensive)
gh aw compile my-workflow.md --no-emit --validate --strict --actionlint --zizmor
```

## Comparison: Local vs GitHub Actions Testing

| Aspect | Local Testing | GitHub Actions Testing |
|--------|---------------|------------------------|
| **Feedback Time** | <1 second | 2-5 minutes |
| **Cost** | Free | Consumes Actions minutes |
| **Internet Required** | No (most cases) | Yes |
| **GitHub Authentication** | No (validation only) | Yes |
| **Full Integration Test** | No | Yes |
| **Agent Execution** | No | Yes |
| **Safe-Output Testing** | Configuration only | Full execution |
| **MCP Server Testing** | Configuration only | Full execution |
| **Network Access** | N/A | Real network calls |
| **Container Validation** | Optional (`--validate`) | Automatic |
| **Syntax Errors** | ✅ Detected | ✅ Detected |
| **Configuration Errors** | ✅ Detected | ✅ Detected |
| **Runtime Errors** | ❌ Not detected | ✅ Detected |
| **Agent Behavior** | ❌ Not tested | ✅ Tested |

## When to Use Each Method

### Use Local Testing For:

✅ **Syntax validation** - Catch typos and formatting errors instantly
✅ **Configuration changes** - Test frontmatter modifications without pushing
✅ **Fragment development** - Validate imports and shared configuration
✅ **MCP setup** - Verify toolsets and server configuration
✅ **Rapid iteration** - Make multiple changes quickly with watch mode
✅ **Pre-commit checks** - Validate before committing changes
✅ **Offline development** - Work without network access
✅ **Learning** - Experiment with workflow features safely

### Use GitHub Actions Testing For:

✅ **Agent behavior** - Test actual AI agent responses and decision-making
✅ **Safe-output execution** - Verify GitHub API operations work correctly
✅ **Integration testing** - Test complete workflow with real data
✅ **MCP runtime** - Verify MCP servers connect and function properly
✅ **Network operations** - Test external API calls and web access
✅ **Production validation** - Final verification before deployment
✅ **Scheduled workflows** - Test cron triggers and timing
✅ **Event-driven workflows** - Test issue/PR triggers with real events

### Recommended Workflow

1. **Local first:** Start with `--no-emit` for instant syntax validation
2. **Watch mode:** Use `--watch` during development for rapid iteration
3. **Comprehensive local:** Run `--no-emit --validate --strict` before committing
4. **Push for integration:** Test in GitHub Actions for full integration testing
5. **Iterate:** Fix issues locally, re-test in Actions

## Troubleshooting Local Testing

### Common Issues

**Lock file generated when using --no-emit**

Problem: Lock file appears despite `--no-emit` flag.

Solution: The flag is working correctly - use `--no-emit` only validates, but the lock file might be from a previous compilation. Delete it manually:
```bash
rm my-workflow.lock.yml
gh aw compile my-workflow.md --no-emit
```

**Validation passes locally but fails in GitHub Actions**

Problem: Workflow validates locally but fails during execution.

Causes:
- Container images are invalid (use `--validate` to catch this locally)
- Runtime dependencies missing
- Network access blocked
- Permissions insufficient
- Environment-specific configuration

Solution:
```bash
# Add --validate flag to catch container/action issues
gh aw compile my-workflow.md --no-emit --validate

# Check MCP configuration
gh aw mcp inspect my-workflow

# Review logs from GitHub Actions run
gh aw logs <run-id>
```

**Watch mode not detecting changes**

Problem: File changes don't trigger recompilation.

Solutions:
- Ensure file is in the watched directory (`.github/workflows`)
- Check file extension is `.md` (not `.markdown` or `.txt`)
- Wait for debounce delay (300ms)
- Try running with `--verbose` to see events
- Restart watch mode if file was renamed

**Import validation fails**

Problem: Fragment imports show errors during validation.

Solution: Ensure fragment files exist at the expected paths relative to `.github/workflows`:
```bash
# Check fragment exists
ls -la .github/workflows/shared/my-fragment.md

# Validate main workflow with imports
gh aw compile main-workflow.md --no-emit

# Use absolute paths for debugging
gh aw compile $(pwd)/.github/workflows/main-workflow.md --no-emit
```

**Validation slow with --validate flag**

Problem: `--validate` makes validation very slow.

Explanation: The `--validate` flag checks:
- Container image existence (network calls to registries)
- Action SHA validity (network calls to GitHub)
- Package availability (network calls to PyPI, npm, etc.)

Solution: Use `--validate` only when needed:
```bash
# Fast validation during development
gh aw compile my-workflow.md --no-emit

# Full validation before merge
gh aw compile my-workflow.md --no-emit --validate
```

**Strict mode errors not appearing**

Problem: Expected strict mode violations not showing.

Solution: Explicitly enable strict mode:
```bash
# Force strict mode validation
gh aw compile my-workflow.md --no-emit --strict
```

Or add to frontmatter:
```yaml
---
strict: true
---
```

## Testing Best Practices

### Development Workflow

```bash
# 1. Start with watch mode for rapid iteration
gh aw compile my-workflow.md --watch

# 2. Make changes in your editor, save, see feedback

# 3. Once stable, run comprehensive validation
gh aw compile my-workflow.md --no-emit --validate --strict --actionlint

# 4. Test fragments independently
gh aw compile shared/my-fragment.md --no-emit

# 5. Inspect MCP configuration
gh aw mcp inspect my-workflow

# 6. Commit and push for GitHub Actions integration test
git add .github/workflows/my-workflow.md
git commit -m "Add new workflow"
git push

# 7. Monitor execution
gh aw logs --latest
```

### Pre-Commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Validate all workflows before commit

echo "Validating workflows..."
if ! gh aw compile --no-emit --strict; then
    echo "❌ Workflow validation failed"
    exit 1
fi

echo "✅ All workflows valid"
exit 0
```

Make executable:
```bash
chmod +x .git/hooks/pre-commit
```

### CI/CD Integration

Add to `.github/workflows/validate-workflows.yml`:

```yaml
name: Validate Workflows
on: [pull_request]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install gh-aw
        run: gh extension install githubnext/gh-aw
      - name: Validate workflows
        run: gh aw compile --no-emit --validate --strict --actionlint
```

### Testing Checklist

Before pushing workflows to GitHub:

- [ ] Syntax validates: `gh aw compile --no-emit`
- [ ] Strict mode passes: `gh aw compile --no-emit --strict`
- [ ] Actions lint passes: `gh aw compile --no-emit --actionlint`
- [ ] MCP config inspected: `gh aw mcp inspect my-workflow`
- [ ] Fragments validated independently
- [ ] Container images valid: `gh aw compile --no-emit --validate`
- [ ] Security scans pass: `gh aw compile --no-emit --zizmor`
- [ ] Lock file generated successfully: `gh aw compile`
- [ ] Lock file reviewed for correctness

## Advanced Testing Techniques

### Testing with Trial Mode

Trial mode allows testing workflows against different repositories without modifying the source:

```bash
# Test workflow against another repository
gh aw compile my-workflow.md --trial --logical-repo owner/repo

# This generates a modified lock file for testing
# Original workflow file remains unchanged
```

### Testing Campaigns

Campaign orchestrators can be validated before execution:

```bash
# Validate campaign configuration
gh aw compile my-campaign.yml --no-emit

# Generate orchestrator without execution
gh aw compile my-campaign.yml
```

### Testing with Different Engines

Override the engine for testing compatibility:

```bash
# Test with different engine
gh aw compile my-workflow.md --no-emit --engine claude

# Test with custom engine
gh aw compile my-workflow.md --no-emit --engine custom
```

### Dependency Manifest Generation

Test Dependabot manifest generation:

```bash
# Generate dependency manifests without running workflows
gh aw compile --dependabot --no-emit
```

## Summary

Local testing dramatically improves workflow development:

- **⚡ Speed:** Instant feedback vs 2-5 minute GitHub Actions runs
- **💰 Cost:** Free vs consuming Actions minutes
- **🔒 Offline:** Work without network access
- **🔄 Iteration:** Rapid development with watch mode
- **✅ Validation:** Catch errors before pushing

Remember: Local testing validates configuration and syntax, but GitHub Actions testing validates execution and integration. Use both for robust workflow development.

**Next Steps:**
- [Getting Started with MCP](/gh-aw/guides/getting-started-mcp/) - Configure MCP servers
- [Security Best Practices](/gh-aw/guides/security/) - Secure your workflows
- [Packaging and Imports](/gh-aw/guides/packaging-imports/) - Create reusable fragments
