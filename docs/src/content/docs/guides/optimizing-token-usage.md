---
title: Optimizing Token Usage
description: Best practices for reducing token consumption in agentic workflows
---

This guide provides strategies for optimizing token usage in agentic workflows, particularly when using MCP tools like the GitHub MCP server.

## Overview

Token consumption directly impacts:
- **Cost**: Higher token usage increases operational costs
- **Performance**: Larger payloads slow down agent processing
- **Rate limits**: Excessive token usage can trigger rate limiting

## GitHub MCP Tool Optimization

Certain GitHub MCP tools return large payloads that can consume 5-20x more tokens than necessary:

### High Token Usage Tools

**`list_code_scanning_alerts`**
- **Typical payload**: ~24,000 tokens (97KB)
- **Issue**: Returns complete alert objects with extensive metadata
- **Impact**: Most expensive GitHub MCP tool by token count

**`list_pull_requests`**
- **Issue**: Duplicates repository object in every PR result
- **Impact**: Scales poorly with number of PRs

### Low Token Usage Tools (Efficient)

These tools demonstrate efficient payload design:
- `list_labels`
- `list_branches`
- `list_workflows`
- `list_discussions`

## Best Practices

### 1. Use Targeted Queries

Instead of listing all alerts, use specific queries:

```yaml
# ❌ Inefficient - retrieves all alerts
tools:
  github:
    toolsets: [code_security]

# Workflow asks agent to: "List all code scanning alerts"
```

```yaml
# ✅ Efficient - use targeted parameters
tools:
  github:
    toolsets: [code_security]

# Workflow asks agent to: "Get code scanning alert #42"
# Or: "List code scanning alerts with state=open and severity=critical"
```

### 2. Limit Result Counts

When listing resources, use pagination and limit results:

```markdown
## Prompt

List the 10 most recent pull requests, not all PRs.
Focus only on open PRs from the last 7 days.
```

### 3. Request Specific Fields

Guide agents to extract only necessary information:

```markdown
## Prompt

For each code scanning alert, extract only:
- Alert number
- Severity level
- Rule description
- File location

Do not include full rule objects or detailed metadata.
```

### 4. Avoid Listing When Possible

Direct access is more efficient than listing:

```yaml
# ❌ Inefficient
# "List all PRs and find PR #123"

# ✅ Efficient
# "Get details for PR #123"
```

### 5. Use Alternative Toolsets

Consider if you need code security tools at all:

```yaml
# If you only need repository and issue operations:
tools:
  github:
    toolsets: [repos, issues]  # Omit code_security
```

### 6. Post-Filter Results

When you must use high-token tools, filter results in the workflow:

```markdown
## Prompt

1. List code scanning alerts with severity=critical
2. From the results, extract only alert numbers and descriptions
3. Discard all other metadata
```

## Workflow Examples

### Efficient Code Security Review

```yaml
---
name: Efficient Security Scan
engine: copilot
tools:
  github:
    toolsets: [code_security, repos]
---

Review critical code scanning alerts:

1. List alerts with severity=critical and state=open
2. For each alert (limit to top 5):
   - Alert number
   - Rule ID
   - File path
   - Line number
3. Create a summary report with only these fields
4. Do NOT include full alert objects or rule details
```

### Efficient PR Review

```yaml
---
name: Efficient PR Review
engine: copilot
tools:
  github:
    toolsets: [pull_requests, repos]
---

Review recent pull requests:

1. List open PRs from the last 3 days (maximum 10)
2. For each PR, extract:
   - PR number and title
   - Author
   - Status
3. Skip repository object (we already know the repo)
4. Generate a concise summary
```

## Monitoring Token Usage

Track token consumption in your workflows:

1. **Enable token reporting**: Use safe-outputs to track token usage
2. **Review workflow runs**: Check GitHub Actions logs for token counts
3. **Compare alternatives**: Test different approaches and measure impact

```yaml
---
name: Token Tracking Workflow
engine: copilot
safe-outputs:
  enabled: true
tools:
  github:
    toolsets: [repos, issues]
---

# Your workflow here
# Safe-outputs will track token usage automatically
```

## Future Improvements

### Response Mode Configuration (Future)

When the upstream GitHub MCP server supports response optimization, you'll be able to configure it:

```yaml
# This is a future API - not currently supported
tools:
  github:
    toolsets: [code_security]
    options:
      response-mode: summary  # Request lightweight responses
```

This configuration is accepted by gh-aw but has no effect until the upstream GitHub MCP server implements it.

### Field Selection (Future)

Future versions may support field-level filtering:

```yaml
# Future API
tools:
  github:
    toolsets: [code_security]
    options:
      response-mode: summary
      fields: [number, state, severity, description]
```

## Measuring Impact

To measure token savings:

1. **Baseline**: Run workflow with `list_code_scanning_alerts`
2. **Optimized**: Use targeted queries and filtering
3. **Compare**: Check safe-outputs reports or workflow logs

Expected savings:
- **Targeted queries**: 40-60% reduction
- **Field filtering**: 30-50% reduction
- **Pagination limits**: 50-80% reduction

## Additional Resources

- [GitHub MCP Server Documentation](./getting-started-mcp)
- [Safe Outputs Guide](../reference/safe-outputs-specification)
- [GitHub API Best Practices](https://docs.github.com/en/rest/guides/best-practices-for-using-the-rest-api)

## Summary

**Key takeaways:**
1. Avoid `list_code_scanning_alerts` unless essential
2. Use targeted queries with specific parameters
3. Limit result counts through prompts
4. Request only necessary fields
5. Consider if you need code_security toolset at all
6. Monitor token usage with safe-outputs

By following these practices, you can reduce token usage by 30-50% for workflows that use GitHub MCP tools.
