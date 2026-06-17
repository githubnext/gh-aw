---
name: Example Failure Category Filter
on:
  workflow_dispatch:
safe-outputs:
  report-failure-as-issue:
    - agent_failure           # Only report genuine agent-side failures
    - missing_safe_outputs    # Only report when outputs are missing
    - missing_tool            # Only report when functionality is missing
    - missing_data            # Only report when required data is unavailable
    # Excluded categories (won't create issues):
    # - report_incomplete: Infrastructure/tool failures
    # - inference_access_error: AI server transient errors
    # - ai_credits_rate_limit_error: AI rate limits
    # - mcp_policy_error: MCP policy violations
  create-issue:
---

# Example: Failure Category Filtering

This workflow demonstrates the new `report-failure-as-issue` category filtering feature.

## Context

For scheduled workflows that frequently encounter transient infrastructure failures:
- Docker registry timeouts
- AI server 5xx errors
- Firewall startup failures
- MCP image pull intermittent failures

Traditional `report-failure-as-issue: false` suppresses ALL failure reports, including genuine agent bugs.

## Solution

Use category filtering to only report actionable failures:

```yaml
safe-outputs:
  report-failure-as-issue:
    - agent_failure
    - missing_safe_outputs
    - missing_tool
    - missing_data
```

This prevents noise while preserving actionable signals.

## Task

Create an issue summarizing this feature.
