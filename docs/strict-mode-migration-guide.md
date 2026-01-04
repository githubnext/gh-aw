# Strict Mode Migration Guide

## Overview

This guide helps you migrate agentic workflows to use `strict: true` for enhanced security validation.

## What is Strict Mode?

When `strict: true` is enabled in a workflow's frontmatter, it enforces security constraints:

1. **Network Access**: Must explicitly declare allowed domains (no wildcards)
2. **Tool Permissions**: Cannot use wildcard tool permissions like `bash: ["*"]`
3. **Write Permissions**: Refuses write permissions on sensitive scopes (contents, issues, pull-requests)
4. **MCP Servers**: Custom MCP servers with containers require network configuration
5. **Input Validation**: Enhanced validation of user inputs and external data

## When to Use Strict Mode

### Always Use (HIGH Priority)

- Workflows with external network access
- Workflows processing user input (issues, PRs, comments)
- Workflows with code execution capabilities
- Workflows creating or modifying code
- Workflows handling secrets or sensitive data

### Should Use (MEDIUM Priority)

- GitHub API-only operations
- Read-only workflows
- Scheduled monitoring workflows
- Analysis and reporting workflows

### Optional (LOW Priority)

- Internal development tooling
- Example workflows
- Testing workflows with minimal external interaction

## Migration Steps

### Step 1: Add `strict: true`

Add the strict mode flag to your workflow frontmatter:

```yaml
---
engine: copilot
strict: true  # Add this line
tools:
  github:
    toolsets: [default]
---
```

### Step 2: Configure Network Access

If your workflow needs network access, explicitly declare allowed domains:

```yaml
---
strict: true
network:
  allowed:
    - defaults          # GitHub API, npm registry, PyPI
    - node              # Node.js ecosystem (npmjs.com, nodejs.org)
    - python            # Python ecosystem (pypi.org, python.org)
    - containers        # Docker Hub, ghcr.io
    - "example.com"     # Custom domains
---
```

Available ecosystem identifiers:
- `defaults` - GitHub API, common package registries
- `node` - Node.js ecosystem
- `python` - Python ecosystem  
- `containers` - Container registries
- `go` - Go ecosystem
- Custom domains in quotes

### Step 3: Remove Wildcard Tool Permissions

Replace wildcard bash permissions with explicit tool lists:

**Before:**
```yaml
tools:
  bash: ["*"]
```

**After:**
```yaml
tools:
  bash: ["gh", "git", "jq", "curl", "make"]
```

### Step 4: Use Safe Outputs Instead of Write Permissions

Replace write permissions with safe-outputs:

**Before:**
```yaml
permissions:
  contents: write
  issues: write
  pull-requests: write
```

**After:**
```yaml
permissions:
  contents: read
  issues: read
  pull-requests: read

safe-outputs:
  create-issue:
    max: 5
  create-pull-request:
    title-prefix: "[automated] "
  add-comment:
    max: 10
```

## Common Migration Patterns

### Pattern 1: Basic GitHub API Workflow

```yaml
---
engine: copilot
strict: true
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [issues, pull_requests]
safe-outputs:
  add-comment:
    max: 5
---
```

### Pattern 2: Workflow with Network Access

```yaml
---
engine: copilot
strict: true
network:
  allowed:
    - defaults
    - node
permissions:
  contents: read
tools:
  bash: ["gh", "git", "npm", "node"]
  github:
    toolsets: [default]
---
```

### Pattern 3: Code Modification Workflow

```yaml
---
engine: copilot
strict: true
network:
  allowed:
    - defaults
permissions:
  contents: read
tools:
  edit:
  bash: ["gh", "git", "make", "go"]
  github:
    toolsets: [repos, pull_requests]
safe-outputs:
  create-pull-request:
    title-prefix: "[automated-fix] "
    labels: [automated]
---
```

## Troubleshooting

### Error: "strict mode: 'network' configuration is required"

**Solution**: Add network configuration to your frontmatter:

```yaml
network:
  allowed:
    - defaults
```

### Error: "strict mode: wildcard '*' is not allowed in network.allowed"

**Solution**: Replace wildcard with specific domains or ecosystem identifiers:

```yaml
network:
  allowed:
    - defaults
    - node
    - "specific-domain.com"
```

### Error: "strict mode: write permission 'contents: write' is not allowed"

**Solution**: Use safe-outputs instead:

```yaml
permissions:
  contents: read

safe-outputs:
  create-pull-request:
```

### Workflow Still Needs Wildcard Bash

Some workflows legitimately need broad bash access (e.g., release workflows). Options:

1. **Preferred**: List all required tools explicitly
2. **Alternative**: Don't use strict mode for that specific workflow (document why)
3. **Future**: Request feature for trusted wildcard bash in strict mode

## Testing Your Migration

After migrating a workflow:

1. Compile the workflow: `gh aw compile .github/workflows/your-workflow.md`
2. Check for validation errors
3. Test the workflow with a manual trigger if possible
4. Monitor the first few runs for issues

## Getting Help

- Review the [strict mode validation code](https://github.com/githubnext/gh-aw/blob/main/pkg/workflow/strict_mode_validation.go)
- Check the [network configuration reference](https://githubnext.github.io/gh-aw/reference/network/)
- Ask in GitHub Discussions

## Current Adoption

Track our progress toward 100% strict mode adoption:

- **Current**: 60% (77/128 workflows) ✅ Exceeded 50% target!
- **Target**: 70%+ (90+ workflows)
- **Remaining**: 51 workflows without strict mode

### By Risk Level

- **HIGH**: 24 workflows (urgent migration needed)
- **MEDIUM**: 23 workflows (should migrate)
- **LOW**: 4 workflows (optional)

## Contributing

Help improve this guide:
- Add migration examples from your workflows
- Document edge cases and solutions
- Share lessons learned

---

**Last Updated**: 2026-01-04
