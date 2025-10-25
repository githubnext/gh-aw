---
title: IssueOps
description: Learn how to implement IssueOps workflows using GitHub Agentic Workflows with issue created triggers and automated comment responses for streamlined issue management.
---

IssueOps transforms GitHub issues into powerful automation triggers that automatically analyze, categorize, and respond to issues as they're created. GitHub Agentic Workflows makes IssueOps natural through issue creation triggers and safe comment outputs that handle automated responses securely without requiring write permissions for the main AI job.

## Overview

Through issue triggers, workflows activate automatically when new issues are created in your repository. The AI agent analyzes the issue content, applies your defined logic, and provides intelligent responses through automated comments.

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

IssueOps workflows use the `add-comment` safe output to ensure secure comment creation with minimal permissions. The main job runs with `contents: read` while comment creation happens in a separate job with `issues: write` permissions, automatically sanitizing AI content and preventing spam:

```yaml
safe-outputs:
  add-comment:
    max: 3                    # Optional: allow multiple comments (default: 1)
    target: "triggering"      # Default: comment on the triggering issue/PR
```

## Accessing Issue Context

IssueOps workflows access sanitized issue content through the `needs.activation.outputs.text` variable, which combines the issue title and description while removing security risks (@mention neutralization, URI filtering, injection protection):

```yaml
# In your workflow instructions:
Analyze this issue: "${{ needs.activation.outputs.text }}"
```

**Security Note**: Always treat user content as potentially untrusted and design workflows to be resilient against prompt injection attempts.

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

Analyze new issues and add appropriate labels based on content:

- Bug reports (with repro steps, environment info, error messages) → "bug" label
- Missing information → also add "needs-info" label
- Feature requests → "enhancement" label
- Questions or docs issues → "question" or "documentation" labels

Maximum of 2 labels per issue from the allowed list.
```

