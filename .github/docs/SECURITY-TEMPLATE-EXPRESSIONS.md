# Security Assessment: Template Expression Usage

This document provides a security assessment of GitHub Actions template expressions used in gh-aw workflows, addressing findings from static security analysis tools.

## Overview

GitHub Actions template expressions (e.g., `${{ ... }}`) can pose security risks when they incorporate untrusted data that could be controlled by external attackers. This document reviews identified template injection instances and documents why they are safe or require mitigation.

## Security Classification Framework

We classify template expressions based on data source and control:

### ✅ Safe - Internal References
- Step outputs from the same workflow (`steps.*.outputs.*`)
- Environment variables set within the workflow (`env.*`)
- GitHub context variables (`github.*`)
- Workflow inputs with proper validation

### ⚠️ Caution - Validated External Data
- User inputs with sanitization
- Pull request data with validation
- Issue content with sanitization

### ❌ Dangerous - Untrusted External Data
- Raw user input without validation
- Unvalidated PR titles/bodies
- Issue comments directly in expressions
- External API responses without verification

## Reviewed Template Injection Instances

### Instance 1: copilot-session-insights.md - Step Output Reference

**Location**: `.github/workflows/copilot-session-insights.md` (line 114)

**Template Expression**:
```yaml
echo "::warning::Extension installation status from previous step: ${{ steps.install-extension.outputs.EXTENSION_INSTALLED }}"
```

**Context**:
- Used in a bash script within a workflow step
- References output from a previous step in the same job
- The output value is set internally by the workflow itself (not user-provided)

**Data Source Analysis**:
- `EXTENSION_INSTALLED` is set by the workflow in lines 62, 70, 79, 84
- Possible values: `"true"` or `"false"` (boolean strings)
- No external user input involved
- Complete control by workflow logic

**Security Assessment**:
- ✅ **SAFE** - Internal workflow state reference
- **Risk Level**: Low/Informational
- **Attacker Control**: None - value is set by workflow logic
- **Blast Radius**: Limited to warning message display in workflow logs
- **Exploitation Scenario**: None - attacker cannot influence this value

**Justification**:
This template expression references an internal step output that is entirely controlled by the workflow's own logic. The value can only be `"true"` or `"false"` and is set by conditional branches within the workflow's bash script. There is no path for external input to influence this value.

**Mitigation**: None required - this is safe by design.

---

### Instance 2: shared/mcp/sentry.md - Environment Variable

**Location**: `.github/workflows/shared/mcp/sentry.md` (line 23)

**Template Expression**:
```yaml
env:
  SENTRY_HOST: ${{ env.SENTRY_HOST }} # Optional
```

**Context**:
- Used in MCP server environment variable configuration
- References a workflow-level environment variable
- Variable is not set from external sources in the workflow

**Data Source Analysis**:
- `SENTRY_HOST` is a workflow environment variable
- Not populated from user input, PR data, or issue content
- Typically set in workflow file or repository/organization variables
- Default behavior if unset: empty string or undefined

**Security Assessment**:
- ✅ **SAFE** - Workflow environment variable reference
- **Risk Level**: Low/Informational  
- **Attacker Control**: None - requires workflow file modification or repository admin access
- **Blast Radius**: Limited to MCP server configuration
- **Exploitation Scenario**: None - attacker would need write access to workflow files or repository settings

**Justification**:
This template expression references a workflow environment variable that is set through trusted channels:
1. Workflow file definition (requires PR approval and merge)
2. Repository variables (requires repository admin access)
3. Organization variables (requires organization admin access)

An attacker would need elevated permissions to modify these values, at which point they could directly modify the workflow file itself, making this particular template expression not a unique security concern.

**Mitigation**: None required - this is safe when environment variables are managed through GitHub's standard security model.

---

## Best Practices for Template Expressions

### 1. Prefer Internal References

**Safe Pattern**:
```yaml
- name: Use step output
  run: echo "Status: ${{ steps.previous-step.outputs.result }}"
```

**Reasoning**: Step outputs within the same job are controlled by workflow logic.

### 2. Avoid Direct User Input

**Unsafe Pattern** ❌:
```yaml
- name: Dangerous
  run: echo "Title: ${{ github.event.issue.title }}"
```

**Safe Alternative** ✅:
```yaml
- name: Safe with sanitization
  env:
    ISSUE_TITLE: ${{ github.event.issue.title }}
  run: |
    # Sanitize and validate
    SAFE_TITLE=$(echo "$ISSUE_TITLE" | tr -cd '[:alnum:][:space:]-_.')
    echo "Title: $SAFE_TITLE"
```

**Reasoning**: Always sanitize user-controlled data before use.

### 3. Use Environment Variables for Isolation

**Pattern**:
```yaml
env:
  USER_INPUT: ${{ github.event.comment.body }}
steps:
  - run: |
      # Validate and sanitize USER_INPUT before use
      echo "Processing input..."
```

**Reasoning**: Environment variables provide a layer of isolation and make sanitization easier.

### 4. Validate External Data

When using external data sources, always:
1. Validate format and content
2. Sanitize special characters
3. Limit length and character set
4. Use allowlists instead of denylists

### 5. Document Template Usage

For every template expression involving external data:
1. Document the data source
2. Explain why it's safe or how it's validated
3. Note any sanitization applied
4. Identify potential security implications

## Security Testing Recommendations

### Static Analysis
- Run `zizmor` regularly to identify template injection risks
- Review all findings, even Low/Informational severity
- Document safety justification for each instance

### Code Review Checklist
When reviewing workflows with template expressions:

- [ ] Identify data source (internal vs. external)
- [ ] Check for user-controlled input
- [ ] Verify sanitization if external data is used
- [ ] Confirm safe context (env var vs. direct script injection)
- [ ] Document security assessment

### Testing
- Test workflows with malicious input where applicable
- Verify that template expressions cannot execute arbitrary code
- Confirm that external data is properly escaped

## References

- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [Preventing Script Injection Attacks](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#understanding-the-risk-of-script-injections)
- [Zizmor Security Scanner](https://github.com/woodruffw/zizmor)
- Static Analysis Report: #3280

## Change Log

- **2025-11-05**: Initial security assessment of 2 template injection findings
  - Reviewed `copilot-session-insights.md` step output reference
  - Reviewed `sentry.md` environment variable reference
  - Both instances classified as safe (internal references)
