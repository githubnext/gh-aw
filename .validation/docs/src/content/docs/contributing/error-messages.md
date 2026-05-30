---
title: Error Messages
description: Write actionable, constructive error messages with examples.
---

# Error message style guide

Use actionable messages that explain what went wrong, what is expected, and how to fix it.

## Prefer constructive language

- Avoid: `invalid`, `cannot`, `must`, `failed` without guidance.
- Prefer adding: `expected`, `requires`, `should`, `example`.

✅ `invalid repo format 'gh-aw' — expected 'owner/repo' format (for example: 'github/gh-aw')`

❌ `invalid repo format`

## When to use `NewValidationError` vs `fmt.Errorf`

Use `NewValidationError(field, value, reason, suggestion)` in validation code (`*_validation.go`) so users get a structured reason and suggestion.

Use `fmt.Errorf` for operational wrapping (`%w`) outside validation logic when you include specific context and recovery guidance.

## Error type selection

- `NewValidationError(...)`: bad input/config shape, missing fields, unsupported values.
- `NewOperationError(...)`: runtime actions fail (fetching, file IO, network, command execution).
- `NewConfigurationError(...)`: safe-outputs/config wiring errors.
- `fmt.Errorf(...%w...)`: wrap lower-level errors with actionable context.

## Suggestion text requirements

Good suggestions:

1. Say what to change
2. Include a concrete YAML/code example
3. Prefer ✓/✗ examples when ambiguity is likely

Example:

```text
Use a supported engine.
✓ Example:
engine: copilot

✗ Avoid:
engine: unknown
```

## YAML example guidance

- Keep examples minimal and valid YAML
- Use real field names from frontmatter
- Quote only when required by YAML syntax
