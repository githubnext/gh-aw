---
title: Rate Limiting
description: Configure rate limits in GitHub Agentic Workflows to prevent denial-of-service scenarios and protect external services from excessive API calls.
sidebar:
  order: 15
---

Rate limiting protects your workflows and external services from excessive resource consumption. GitHub Agentic Workflows uses a token bucket algorithm to enforce configurable limits on API calls, MCP requests, network requests, and file operations.

## How It Works

Rate limiting provides protection against runaway agents, excessive API consumption, and accidental denial-of-service patterns. The system tracks request rates across different operation types and automatically throttles when limits are exceeded.

**Rate Limiting Architecture:**

```
┌─────────────────┐
│ Agentic Job     │
│ Makes Requests  │
└────────┬────────┘
         │ requests
         ▼
┌─────────────────┐
│ Token Bucket    │ (Per operation type)
│ Rate Limiter    │
└────────┬────────┘
         │ allowed/throttled
         ▼
┌─────────────────┐
│ Exponential     │ (Automatic retry on throttle)
│ Backoff Handler │
└────────┬────────┘
         │ retry or fail
         ▼
┌─────────────────┐
│ External Service│
│ GitHub API, MCP │
└─────────────────┘
```

## Token Bucket Algorithm

Rate limiting uses a token bucket algorithm that provides burst handling with sustained rate enforcement:

1. **Bucket capacity** — Maximum tokens available for bursts
2. **Refill rate** — Tokens added per time unit
3. **Request cost** — Each request consumes one token
4. **Throttling** — Requests wait when bucket is empty

This approach allows short bursts of activity while preventing sustained overuse. When limits are exceeded, requests are automatically retried with exponential backoff.

## Default Configuration

Rate limits are pre-configured with sensible defaults for each operation type:

| Operation | Default Limit | Retries | Initial Backoff | Max Backoff |
|-----------|---------------|---------|-----------------|-------------|
| `github-api` | 100/hour | 3 | 1s | 5min |
| `mcp-requests` | 50/minute | 3 | 500ms | 1min |
| `network-requests` | 60/minute | 3 | 500ms | 1min |
| `file-read` | 1000/minute | 1 | 10ms | 1s |

These defaults balance protection with practical workflow needs. Most workflows operate within these limits without modification.

## Configuration Options

Configure rate limits in your workflow frontmatter using the `rate-limits` field:

```yaml wrap
rate-limits:
  github-api: "200/hour"
  mcp-requests: "100/minute"
  network-requests: "120/minute"
  file-read: "2000/minute"
```

### Rate Limit Format

Specify limits using the format `N/unit` where `N` is the number of requests and `unit` is the time period:

**Supported time units:**

| Unit | Aliases |
|------|---------|
| second | `sec`, `s` |
| minute | `min`, `m` |
| hour | `hr`, `h` |
| day | `d` |

**Examples:**

```yaml wrap
rate-limits:
  github-api: "100/hour"      # 100 requests per hour
  mcp-requests: "50/min"      # 50 requests per minute
  network-requests: "1/s"     # 1 request per second
  file-read: "5000/day"       # 5000 reads per day
```

### Operation Types

| Operation | Description |
|-----------|-------------|
| `github-api` | GitHub REST and GraphQL API calls via the GitHub MCP server |
| `mcp-requests` | Requests to any MCP server (GitHub, custom, registry-based) |
| `network-requests` | HTTP requests to allowed external domains |
| `file-read` | File system read operations within the workflow |

## Exponential Backoff

When rate limits are exceeded, requests automatically retry with exponential backoff:

1. **Initial delay** — First retry waits for the configured initial backoff
2. **Exponential increase** — Each subsequent retry doubles the delay
3. **Maximum cap** — Delays are capped at the maximum backoff value
4. **Retry limit** — Requests fail after exhausting configured retries

**Backoff calculation:**

```
delay = min(initial_backoff * 2^(attempt - 1), max_backoff)
```

For example, with `github-api` defaults (1s initial, 5min max):
- Retry 1: 1s delay
- Retry 2: 2s delay
- Retry 3: 4s delay
- Retry 4: fails (exceeds 3 retries)

## Practical Examples

### High-Volume Repository Analysis

For workflows analyzing large repositories with many files:

```aw wrap
---
on:
  workflow_dispatch:
permissions:
  contents: read
tools:
  github:
    toolsets: [default]
rate-limits:
  github-api: "500/hour"
  file-read: "5000/minute"
---

# Repository Analysis Agent

Analyze repository structure, dependencies, and code patterns.
```

### Frequent MCP Interactions

For workflows with heavy MCP tool usage:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [default, actions]
mcp-servers:
  notion:
    container: "mcp/notion"
    allowed: ["*"]
rate-limits:
  mcp-requests: "200/minute"
  github-api: "150/hour"
---

# Issue Triage with External Integration

Analyze issues and sync with external project management tools.
```

### Conservative Rate Limiting

For workflows that should minimize external service impact:

```aw wrap
---
on:
  schedule:
    - cron: "0 0 * * 0"
permissions:
  contents: read
tools:
  github:
    toolsets: [default]
rate-limits:
  github-api: "50/hour"
  mcp-requests: "20/minute"
  network-requests: "30/minute"
---

# Weekly Audit Agent

Perform weekly repository audits with minimal API impact.
```

### Strict Production Workflow

For production workflows requiring explicit rate control:

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
strict: true
permissions:
  contents: read
  pull-requests: read
tools:
  github:
    toolsets: [default]
rate-limits:
  github-api: "100/hour"
  mcp-requests: "50/minute"
  network-requests: "60/minute"
  file-read: "1000/minute"
safe-outputs:
  add-comment:
---

# PR Review Agent

Review pull requests with controlled resource usage.
```

## Best Practices

**Start with defaults.** The default rate limits are designed for typical workflows. Only adjust when you have specific requirements or encounter throttling.

**Monitor workflow logs.** Review logs for rate limit warnings to understand actual usage patterns before adjusting limits.

**Use appropriate time units.** Choose time units that match your workflow's execution pattern:
- Use `/minute` for short-running workflows
- Use `/hour` for long-running analysis tasks
- Use `/day` for scheduled batch operations

**Consider external service limits.** Your rate limits should stay below external API limits (e.g., GitHub API has its own rate limits).

**Test with realistic data.** Validate rate limit configuration with representative workloads before production deployment.

**Combine with strict mode.** For production workflows, use `strict: true` alongside rate limits for comprehensive resource control.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| **Workflow times out** | Increase rate limits or reduce workload scope; check if backoff delays consume timeout budget |
| **API errors after retries** | Verify limits are within external service quotas; consider longer time units for sustained operations |
| **Unexpected throttling** | Review default limits; some operations may count against multiple rate limiters |
| **File operations slow** | `file-read` limits may be too restrictive for large repositories; increase limit or reduce file access patterns |

## Related Documentation

- [Security Guide](/gh-aw/guides/security/) — Overall security best practices including resource limits
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) — All configuration options
- [Using MCPs](/gh-aw/guides/mcps/) — MCP server configuration and usage
- [Network Configuration](/gh-aw/reference/network/) — Network access controls
