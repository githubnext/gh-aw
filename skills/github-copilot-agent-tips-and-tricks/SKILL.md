---
name: github-copilot-agent-tips-and-tricks
description: Tips and Tricks for Working with GitHub Copilot Agent PRs
---


# GitHub Copilot Agent Tips and Tricks

This document provides guidance for discovering, reviewing, and working with pull requests created by the GitHub Copilot agent in the gh-aw repository.

## Identifying Copilot Agent PRs

### Branch Naming Convention

The GitHub Copilot agent creates branches with the `copilot/` prefix. This makes them easy to identify and filter.

**Examples from this repository:**
- `copilot/add-cache-for-imported-workflows`
- `copilot/fix-istruthy-bundling-issue`
- `copilot/update-audit-command-copilot`
- `copilot/refactor-mcp-tool-rendering`

### Author Attribution

Copilot agent PRs are typically authored by:
- `app/github-copilot` - The GitHub Copilot bot account
- Individual developers using Copilot as an assistant

## Searching for Copilot Agent PRs

### Using GitHub CLI (`gh`)

**Prerequisites:**
```bash
# Authenticate with GitHub CLI
gh auth login
```

**Search by author (GitHub Copilot bot):**
```bash
# List all PRs created by the Copilot bot
gh pr list --author "app/github-copilot" --limit 100

# Include closed PRs
gh pr list --author "app/github-copilot" --state all --limit 100

# Get detailed JSON output
gh pr list --author "app/github-copilot" --json number,title,author,headRefName,createdAt,state
```

**Search by branch prefix:**
```bash
# Find all PRs from copilot/* branches
gh pr list --search "head:copilot/" --state all

# Combine with other filters
gh pr list --search "head:copilot/ is:open"
gh pr list --search "head:copilot/ is:merged"
```

**Filter with jq:**
```bash
# Extract specific fields
gh pr list --limit 100 --json author,number,title,headRefName \
  --jq '.[] | select(.headRefName | startswith("copilot/")) | {number, title, branch: .headRefName}'

# Filter by author containing "copilot"
gh pr list --limit 100 --json author,number,title \
  --jq '.[] | select(.author.login | contains("copilot"))'
```

### Using Git Commands

**List copilot branches:**
```bash
# Local and remote copilot branches
git branch -a | grep copilot

# Remote copilot branches only
git branch -r | grep copilot
```

**Search commit history:**
```bash
# Find commits with "copilot" in message
git log --all --grep="copilot" --oneline

# Find commits by copilot author
git log --all --author="copilot" --oneline

# Show graph with copilot-related commits
git log --all --grep="copilot" --oneline --graph
```

**Find merged copilot PRs:**
```bash
# Search for merge commits
git log --all --merges --grep="copilot" --oneline

# With PR numbers
git log --all --merges --oneline | grep -i copilot
```

## Common Copilot Agent PR Patterns

### Recent Examples from gh-aw Repository

Based on analysis of this repository, Copilot agent PRs typically address:

1. **Refactoring and Code Organization**
   - Example: "Refactor ALL_TOOLS to separate JSON file with runtime filtering"
   - Example: "Eliminate duplicate MCP tool table rendering logic"

2. **Documentation Improvements**
   - Example: "Document strict mode enforcement areas and CLI flag in schema"
   - Example: "Add comprehensive strict mode reference documentation"

3. **Bug Fixes**
   - Example: "Fix JavaScript test assertions for loadAgentOutput error handling"
   - Example: "Remove duplicate formatFileSize() function"

4. **Testing Enhancements**
   - Example: "Add integration tests for playwright MCP configuration across all engines"

5. **Security Fixes**
   - Example: "Fix template injection risk in copilot-session-insights workflow"

### PR Metadata to Check

When reviewing Copilot agent PRs, pay attention to:
- **Branch name**: Should follow `copilot/descriptive-name` pattern
- **Commit messages**: Often include "Initial plan" commits
- **PR description**: Should explain the problem and solution
- **Linked issues**: May reference issues being addressed

## Workflow Tips

### Finding Related PRs

```bash
# Find PRs related to a specific feature
gh pr list --search "head:copilot/ refactor" --state all

# Find PRs in a date range
gh pr list --search "head:copilot/ created:>=2024-01-01" --state all

# Find PRs with specific labels
gh pr list --search "head:copilot/ label:enhancement"
```

### Reviewing Copilot PRs

```bash
# Check out a copilot PR locally
gh pr checkout <PR-number>

# View PR diff
gh pr diff <PR-number>

# View PR details
gh pr view <PR-number>

# View PR in browser
gh pr view <PR-number> --web
```

### Tracking Copilot Contributions

```bash
# Count merged copilot PRs
gh pr list --author "app/github-copilot" --state merged --json number --jq 'length'

# List recent copilot PRs with dates
gh pr list --author "app/github-copilot" --state all --limit 20 \
  --json number,title,createdAt,state \
  --jq '.[] | "\(.number): \(.title) (\(.state)) - \(.createdAt)"'

# Export to CSV for analysis
gh pr list --author "app/github-copilot" --state all --limit 100 \
  --json number,title,createdAt,state,author \
  --jq -r '.[] | [.number, .title, .state, .createdAt] | @csv' > copilot-prs.csv
```

## Troubleshooting

### Authentication Issues

If you see "gh auth login" prompts:
```bash
# Authenticate with GitHub CLI
gh auth login

# Or set token environment variable
export GH_TOKEN="your-github-token"
```

### No Results Found

If searches return no results:
1. Verify you're in the correct repository
2. Check if the author name is correct (try `app/github-copilot` or `github-copilot`)
3. Try searching by branch prefix instead: `gh pr list --search "head:copilot/"`
4. Check if PRs exist: `git branch -r | grep copilot`

### Rate Limiting

If you hit GitHub API rate limits:
```bash
# Check rate limit status
gh api rate_limit

# Use authenticated requests (higher limits)
gh auth login
```

## Best Practices

1. **Use branch prefix search** when author search is unavailable
2. **Export PR lists** regularly for tracking and analysis
3. **Review commit history** to understand Copilot's implementation approach
4. **Check for "Initial plan" commits** to see Copilot's planning process
5. **Verify tests pass** before merging Copilot PRs
6. **Review security implications** especially for workflow changes

## Additional Resources

- GitHub CLI documentation: https://cli.github.com/manual/
- GitHub Copilot documentation: https://docs.github.com/en/copilot
- Git branch filtering: https://git-scm.com/docs/git-branch
- jq JSON processing: https://stedolan.github.io/jq/manual/
