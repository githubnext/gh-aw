---
description: Deprecation Policy and Guidelines for GitHub Agentic Workflows
---

# Deprecation Policy

This document defines the formal deprecation lifecycle for fields, features, and APIs in GitHub Agentic Workflows (gh-aw).

## Overview

GitHub Agentic Workflows follows a structured deprecation process to ensure users have adequate time to migrate while maintaining a clean, maintainable codebase. This policy applies to:

- Frontmatter field names and structure
- Configuration options and values
- Command-line flags and arguments
- API interfaces and function signatures

## Deprecation Lifecycle

### Stage 1: Mark as Deprecated

**Duration**: Minimum 3 releases (typically 3-6 weeks)

**Actions**:
- Mark field as deprecated in JSON schema with `"deprecated": true`
- Add clear deprecation message in schema description
- Document the new preferred approach
- Create migration guide if needed

**Example** (from `main_workflow_schema.json`):
```json
{
  "timeout_minutes": {
    "type": "integer",
    "description": "Deprecated: Use 'timeout-minutes' instead. Workflow timeout in minutes.",
    "deprecated": true
  },
  "timeout-minutes": {
    "type": "integer",
    "description": "Workflow timeout in minutes (GitHub Actions standard field)."
  }
}
```

**User Impact**: None - deprecated fields continue to work normally without any warnings.

### Stage 2: Emit Warnings

**Duration**: Minimum 3 releases (typically 3-6 weeks)

**Actions**:
- Add runtime warning when deprecated field is used
- Warning should be clear, actionable, and non-blocking
- Use `console.FormatWarningMessage()` for consistent styling
- Continue supporting both old and new fields

**Example** (from `pkg/workflow/compiler.go`):
```go
// Prefer timeout-minutes (new) over timeout_minutes (deprecated)
workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout-minutes")
if workflowData.TimeoutMinutes == "" {
    workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout_minutes")
    if workflowData.TimeoutMinutes != "" {
        // Emit deprecation warning
        fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Field 'timeout_minutes' is deprecated. Please use 'timeout-minutes' instead to follow GitHub Actions naming convention."))
    }
}
```

**User Impact**: Warning messages displayed during compilation. Workflows continue to work normally.

### Stage 3: Remove (Breaking Change)

**Duration**: Coordinated with major version release

**Actions**:
- Remove deprecated field from schema
- Remove runtime support code
- Update all documentation and examples
- Add breaking change notice to CHANGELOG
- Announce in release notes

**Requirements**:
- Must be part of a major version bump (0.x.0 → 1.0.0, or 1.x.0 → 2.0.0)
- Minimum 6 releases (Stages 1 + 2) must have passed
- Clear migration path must be documented
- Breaking changes should be batched when possible

**User Impact**: Workflows using deprecated fields will fail compilation with clear error messages.

## Semantic Versioning

GitHub Agentic Workflows follows semantic versioning:

- **Major version** (x.0.0): Breaking changes, including removal of deprecated features
- **Minor version** (0.x.0): New features, non-breaking changes, deprecation warnings
- **Patch version** (0.0.x): Bug fixes, documentation updates, dependency updates

**Current Status**: Project is pre-1.0, using 0.x.x versioning. Breaking changes may occur more frequently but still follow the deprecation lifecycle above.

## Timeline Expectations

| Phase | Minimum Duration | Typical Duration | Example Versions |
|-------|-----------------|------------------|------------------|
| Stage 1: Mark | 3 releases | 3-6 weeks | v0.25.0, v0.26.0, v0.27.0 |
| Stage 2: Warn | 3 releases | 3-6 weeks | v0.28.0, v0.29.0, v0.30.0 |
| Stage 3: Remove | Major release | Coordinated | v1.0.0 or v0.x.0 |
| **Total minimum** | **6 releases** | **6-12 weeks** | - |

**Note**: Pre-1.0 releases may have shorter deprecation cycles for critical issues, but the minimum 6-release lifecycle should be maintained when possible.

## Announcing Deprecations

### Documentation Updates

When deprecating a field:

1. **Update schema** with `"deprecated": true` and clear message
2. **Update CHANGELOG** with deprecation notice under "Deprecations" section
3. **Update documentation** to show both old and new approaches with migration guide
4. **Update examples** to use new approach (but don't remove old examples immediately)

### Communication Channels

Deprecation notices should be communicated through:

- **CHANGELOG.md**: Detailed deprecation notices with examples
- **Release notes**: Summary of deprecations in each release
- **Schema validation**: Clear error/warning messages during compilation
- **Documentation**: Migration guides and best practices

### CHANGELOG Format

```markdown
## v0.28.0 - 2025-11-04

### Deprecations

#### Field 'timeout_minutes' deprecated in favor of 'timeout-minutes'

The `timeout_minutes` field is deprecated to align with GitHub Actions naming conventions. Use `timeout-minutes` instead.

**Migration**:
```yaml
# Before (deprecated)
---
timeout_minutes: 30
---

# After (preferred)
---
timeout-minutes: 30
---
```

The deprecated field will continue to work with warnings until v1.0.0.
```

## Exemplary Case: timeout_minutes → timeout-minutes

The `timeout_minutes` → `timeout-minutes` migration demonstrates best practices:

### ✅ What Was Done Well

1. **Schema-first approach**: Marked as deprecated in JSON schema with clear message
2. **Graceful fallback**: Compiler checks new field first, falls back to old field
3. **Clear warning**: User-friendly warning message with actionable guidance
4. **Documentation**: Both fields documented with migration path explained
5. **Non-breaking**: Old field continues to work during deprecation period
6. **Alignment reasoning**: Clear explanation (GitHub Actions naming convention)

### Implementation Details

**JSON Schema** (`pkg/parser/schemas/main_workflow_schema.json`):
```json
{
  "timeout-minutes": {
    "type": "integer",
    "description": "Workflow timeout in minutes (GitHub Actions standard field).",
    "examples": [5, 10, 30]
  },
  "timeout_minutes": {
    "type": "integer",
    "description": "Deprecated: Use 'timeout-minutes' instead. Workflow timeout in minutes.",
    "examples": [5, 10, 30],
    "deprecated": true
  }
}
```

**Compiler Logic** (`pkg/workflow/compiler.go`):
```go
// Prefer timeout-minutes (new) over timeout_minutes (deprecated)
workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout-minutes")
if workflowData.TimeoutMinutes == "" {
    workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout_minutes")
    if workflowData.TimeoutMinutes != "" {
        fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
            "Field 'timeout_minutes' is deprecated. Please use 'timeout-minutes' instead to follow GitHub Actions naming convention.",
        ))
    }
}
```

**Test Coverage** (`pkg/workflow/timeout_minutes_test.go`):
```go
{
    name: "timeout_minutes (deprecated format)",
    frontmatter: map[string]any{
        "timeout_minutes": 15,
    },
    expectTimeout: 15,
    expectWarning: true,
},
```

## Guidelines for Contributors

### When Adding New Fields

- Use GitHub Actions-compatible naming (kebab-case: `field-name`)
- Document clearly in schema with description and examples
- Add validation and tests
- Consider future deprecation needs upfront

### When Deprecating Fields

1. **Create Issue**: Document the deprecation plan
2. **Stage 1 PR**: Mark as deprecated in schema
3. **Wait**: Minimum 3 releases
4. **Stage 2 PR**: Add runtime warnings
5. **Wait**: Minimum 3 releases
6. **Stage 3 PR**: Remove in major version with breaking change notice

### When to Skip Deprecation

Deprecation can be skipped only in these cases:

- **Experimental features** explicitly marked as unstable
- **Security vulnerabilities** requiring immediate removal
- **Pre-release versions** (0.0.x) with no known users
- **Internal APIs** not exposed to users

### Error Message Quality

When adding deprecation warnings, follow the error message template:

**Template**: [what's deprecated]. [what to use instead]. [example]

```go
// ✅ Good - clear, actionable, with example
fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
    "Field 'timeout_minutes' is deprecated. Use 'timeout-minutes' instead. Example: timeout-minutes: 30",
))

// ❌ Bad - vague, no guidance
fmt.Fprintln(os.Stderr, "timeout_minutes is deprecated")
```

## Breaking Change Checklist

Before removing a deprecated field:

- [ ] Deprecated for minimum 6 releases (Stages 1 + 2)
- [ ] Coordinated with major version release
- [ ] Migration guide documented
- [ ] All examples updated
- [ ] CHANGELOG includes breaking change notice
- [ ] Release notes highlight breaking change
- [ ] Clear error messages for removed fields
- [ ] Tests verify error messages for removed fields

## Version Upgrade Guidance

Users should expect:

- **Patch updates** (0.0.x): Safe to update immediately, no breaking changes
- **Minor updates** (0.x.0): Safe to update, may see new deprecation warnings
- **Major updates** (x.0.0): Review CHANGELOG for breaking changes, test before deploying

**Best Practice**: Monitor deprecation warnings during minor updates and plan migrations before the next major release.

## References

- [Semantic Versioning](https://semver.org/)
- [GitHub Actions Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
- [Error Message Style Guide](../.github/instructions/error-messages.instructions.md)
- [Developer Instructions](../.github/instructions/developer.instructions.md)

## Questions?

For questions about deprecation policy:
- Check existing deprecation PRs for patterns
- Review `timeout_minutes` case study above
- Ask in GitHub issues or Discord (#continuous-ai)

**Last Updated**: 2025-11-17
