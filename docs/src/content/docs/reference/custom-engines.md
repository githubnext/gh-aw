---
title: Custom Engines
description: Guide to custom engines for deterministic workflows and custom error pattern recognition in GitHub Agentic Workflows.
sidebar:
  order: 610
---

## Custom Engines

Define custom GitHub Actions steps without AI interpretation for deterministic workflows.

```yaml wrap
engine: custom
```

Extended configuration:

```yaml wrap
engine:
  id: custom
  steps:
    - name: Install dependencies
      run: npm ci
```

## Custom Engine Error Patterns

All engines (Copilot, Claude, Codex, and Custom) support custom error pattern recognition for enhanced log validation. This allows you to define project-specific error formats that should be detected in agent logs.

### Basic Usage

```yaml wrap
engine:
  id: copilot
  error_patterns:
    - pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)"
      level_group: 2
      message_group: 3
      description: "Custom error format with timestamp"
```

### Multiple Patterns

Define multiple error patterns to catch different error formats:

```yaml wrap
engine:
  id: claude
  error_patterns:
    - pattern: "PROJECT_ERROR:\\s+(.+)"
      message_group: 1
      description: "Project-specific error"
    - pattern: "VALIDATION_FAILED:\\s+(.+)"
      message_group: 1
      description: "Validation error"
```

### Pattern Fields

- **`pattern`** (required): ECMAScript regular expression to match log lines
- **`level_group`** (optional): Capture group index (1-based) containing error level (ERROR, WARNING, etc.). Use 0 to infer from pattern content.
- **`message_group`** (optional): Capture group index (1-based) containing the error message. Use 0 to use the entire match.
- **`description`** (optional): Human-readable description of what this pattern matches

### Shared Error Patterns

Error patterns can be defined in shared workflows and imported:

**`shared/error-patterns.md`:**

```yaml wrap
---
engine:
  error_patterns:
    - pattern: "SHARED_ERROR:\\s+(.+)"
      message_group: 1
      description: "Shared error pattern"
---
```

**Main workflow:**

```yaml wrap
---
imports:
  - ./shared/error-patterns.md
engine: copilot
---
```

Custom error patterns are merged with engine-specific built-in patterns during workflow compilation.

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Complete guide to AI engines
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools](/gh-aw/reference/tools/) - Available tools and MCP servers
