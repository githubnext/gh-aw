---
# Recommended engine limits for cost control
# Import this in your workflow with: imports: [shared/engine-limits.md]

# These limits prevent runaway costs from:
# - Infinite loops (max-turns)
# - Long-running workflows (timeout_minutes)

# For Claude/Codex engines: add max-turns to prevent excessive iterations
# For all engines: add timeout_minutes to cap total runtime

# Example usage in workflow frontmatter:
# engine:
#   id: claude
#   max-turns: 25
# timeout_minutes: 15
---

# Engine Limits Best Practices

This shared configuration provides recommended defaults for cost control based on live workflow analysis.

## Recommended Limits

- **max-turns: 25** - Prevents infinite loops and excessive API calls (Claude/Codex only)
- **timeout_minutes: 15** - Caps total workflow runtime to prevent runaway costs

## Why These Limits?

Based on analysis of workflow runs:
- Workflows without max-turns can run 80+ iterations, consuming 1M+ tokens
- Workflows without timeout can run indefinitely if they encounter errors
- Setting conservative limits early prevents costly mistakes

## When to Adjust

- **Increase max-turns** for complex tasks requiring multiple iterations
- **Increase timeout_minutes** for workflows with large data processing
- **Decrease limits** for simple automation tasks

## Usage

Import this in your workflow frontmatter:
```yaml
imports:
  - shared/engine-limits.md
```

Then configure the limits appropriate for your workflow:
```yaml
engine:
  id: claude
  max-turns: 30  # Adjust based on complexity
timeout_minutes: 20  # Adjust based on expected runtime
```
