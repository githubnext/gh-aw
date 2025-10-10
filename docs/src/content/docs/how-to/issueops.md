---
title: IssueOps
description: Learn how to implement IssueOps workflows using GitHub Agentic Workflows with issue created triggers and automated comment responses for streamlined issue management.
---

IssueOps is a practice that transforms GitHub issues into powerful automation triggers. Instead of manual triage and responses, you can create intelligent workflows that automatically analyze, categorize, and respond to issues as they're created.

GitHub Agentic Workflows makes IssueOps natural through issue creation triggers and safe comment outputs that handle automated responses securely without requiring write permissions for the main AI job.

## How IssueOps Works

IssueOps workflows activate automatically when new issues are created in your repository. The AI agent analyzes the issue content, applies your defined logic, and provides intelligent responses through automated comments.

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 2
---

# Issue Triage Assistant

When a new issue is created, analyze the issue content and provide helpful guidance.

Examine the issue title and description for:
- Bug reports that need additional information
- Feature requests that should be categorized
- Questions that can be answered immediately
- Issues that might be duplicates

Respond with a helpful comment that guides the user on next steps or provides immediate assistance.
```

This workflow creates an intelligent issue triage system that automatically responds to new issues with contextual guidance and assistance.

## Safe Output Architecture

IssueOps workflows use the `add-comment` safe output to ensure secure comment creation:

```yaml
safe-outputs:
  add-comment:
    max: 3                    # Optional: allow multiple comments (default: 1)
    target: "triggering"      # Default: comment on the triggering issue/PR
```

**Security Benefits**:
- Main job runs with minimal `contents: read` permissions
- Comment creation happens in a separate job with appropriate `issues: write` permissions  
- Automatic sanitization of AI-generated content
- Built-in limits prevent comment spam

## Accessing Issue Context

IssueOps workflows have access to sanitized issue content through the `needs.activation.outputs.text` variable:

```yaml
# In your workflow instructions:
Analyze this issue: "${{ needs.activation.outputs.text }}"
```

The sanitized context provides:
- Issue title and description combined
- Filtered content that removes security risks
- @mention neutralization to prevent unintended notifications
- URI filtering for trusted domains only

**Security Note**: While sanitization reduces risks, always treat user content as potentially untrusted and design workflows to be resilient against prompt injection attempts.

## Common IssueOps Patterns

### Automated Bug Report Triage

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-labels:
    allowed: [bug, needs-info, enhancement, question, documentation]  # Restrict to specific labels
    max: 2                                                            # Maximum 2 labels per issue
---

# Bug Report Triage

Analyze new issues to identify bug reports and automatically add appropriate labels.

Look for:
- Steps to reproduce
- Expected vs actual behavior  
- Environment information (OS, browser, version)
- Error messages or stack traces

Based on your analysis:
- If the issue appears to be a bug report, add the "bug" label
- If it's missing key information, also add the "needs-info" label
- For feature requests, add the "enhancement" label
- For questions or documentation issues, use the "question" or "documentation" labels

You can only add labels from the allowed list and a maximum of 2 labels per issue.
```

