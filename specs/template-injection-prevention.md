# Template Injection Prevention in Workflows

## Overview

This document explains the template injection security fix applied to workflows in this repository to prevent potential code injection attacks via GitHub Actions template expansion.

## What is Template Injection?

Template injection occurs when untrusted data flows into GitHub Actions template expressions (`${{ }}`) that are evaluated during workflow execution. This can lead to:

- Code execution in workflow steps
- Information disclosure
- Privilege escalation

## The Vulnerability Pattern

**Unsafe Pattern:**
```yaml
steps:
  - name: My Step
    run: |
      echo "Value: ${{ steps.previous.outputs.value }}"
```

If the output value contains malicious content, it could be executed when the template is expanded.

## The Fix

**Safe Pattern:**
```yaml
steps:
  - name: My Step
    env:
      MY_VALUE: ${{ steps.previous.outputs.value }}
    run: |
      echo "Value: $MY_VALUE"
```

By passing the value through an environment variable, the content is treated as data, not executable code.

## Changes Made

### copilot-session-insights.md

**Issue:** Template expression used directly in bash echo statement
- **Line:** 115
- **Risk:** While using step output (controlled), the pattern could lead to injection if changed to use untrusted data

**Fix Applied:**
```diff
  - name: List and download Copilot agent sessions
    id: download-sessions
    continue-on-error: true
    env:
      GH_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}
+     # Security: Pass step output through environment variable to prevent template injection
+     EXTENSION_INSTALLED: ${{ steps.install-extension.outputs.EXTENSION_INSTALLED }}
    run: |
      # ...
      if ! gh agent-task --help &> /dev/null; then
        echo "::warning::gh agent-task extension is not installed"
-       echo "::warning::Extension installation status from previous step: ${{ steps.install-extension.outputs.EXTENSION_INSTALLED }}"
+       # Security: Use environment variable instead of template expression in bash script
+       echo "::warning::Extension installation status from previous step: $EXTENSION_INSTALLED"
        echo "::warning::This workflow requires GitHub Enterprise Copilot access"
        # ...
```

### mcp-inspector.md

**Status:** No template injection vulnerabilities found
- The "Setup MCPs" step name is static text
- Environment variables use secrets, which are safe
- No untrusted data flows into template expressions

## Security Best Practices

When writing GitHub Actions workflows:

1. **Never use untrusted inputs directly in template expressions:**
   - ❌ `${{ github.event.issue.title }}`
   - ❌ `${{ github.event.issue.body }}`
   - ❌ `${{ github.event.comment.body }}`
   - ❌ `${{ github.head_ref }}` (can be controlled by PR authors)

2. **Use sanitized context instead:**
   - ✅ `${{ needs.activation.outputs.text }}` (sanitized by gh-aw)

3. **Pass data through environment variables:**
   ```yaml
   env:
     UNTRUSTED_VALUE: ${{ github.event.issue.title }}
   run: |
     echo "Title: $UNTRUSTED_VALUE"
   ```

4. **Safe context variables (always safe to use):**
   - `${{ github.actor }}`
   - `${{ github.repository }}`
   - `${{ github.run_id }}`
   - `${{ github.run_number }}`
   - `${{ github.sha }}`

## References

- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [Understanding the risk of script injections](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#understanding-the-risk-of-script-injections)
- Issue #3945 - Static Analysis Report (November 14, 2025)

## Validation

Both workflows compile successfully after the fix:
- ✅ `copilot-session-insights.md` - Template injection fixed
- ✅ `mcp-inspector.md` - No vulnerabilities found

```bash
./gh-aw compile copilot-session-insights --validate
./gh-aw compile mcp-inspector --validate
```
