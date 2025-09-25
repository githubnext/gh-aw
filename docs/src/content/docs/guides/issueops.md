---
title: Issue Ops
description: Learn how to implement Issue Ops workflows using GitHub Agentic Workflows with issue created triggers and automated comment responses for streamlined issue management.
---

Issue Ops is a practice that transforms GitHub issues into powerful automation triggers. Instead of manual triage and responses, you can create intelligent workflows that automatically analyze, categorize, and respond to issues as they're created.

GitHub Agentic Workflows makes Issue Ops natural through issue creation triggers and safe comment outputs that handle automated responses securely without requiring write permissions for the main AI job.

## How Issue Ops Works

Issue Ops workflows activate automatically when new issues are created in your repository. The AI agent analyzes the issue content, applies your defined logic, and provides intelligent responses through automated comments.

```yaml
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

## Key Benefits of Issue Ops

**Immediate Response**: Issues get instant acknowledgment and initial triage, improving user experience even outside business hours.

**Consistent Quality**: Every issue receives the same level of analysis and appropriate response, regardless of maintainer availability.

**Resource Efficiency**: Maintainers can focus on complex issues while routine triage and initial responses are handled automatically.

**Pattern Recognition**: AI agents can identify common issue patterns and provide standardized responses or routing.

## Safe Output Architecture

Issue Ops workflows use the `add-comment` safe output to ensure secure comment creation:

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

Issue Ops workflows have access to sanitized issue content through the `needs.task.outputs.text` variable:

```yaml
# In your workflow instructions:
Analyze this issue: "${{ needs.task.outputs.text }}"
```

The sanitized context provides:
- Issue title and description combined
- Filtered content that removes security risks
- @mention neutralization to prevent unintended notifications
- URI filtering for trusted domains only

**Security Note**: While sanitization reduces risks, always treat user content as potentially untrusted and design workflows to be resilient against prompt injection attempts.

## Common Issue Ops Patterns

### Automated Bug Report Triage

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
---

# Bug Report Triage

Analyze new issues to identify bug reports and guide users through the reporting process.

Look for:
- Steps to reproduce
- Expected vs actual behavior  
- Environment information (OS, browser, version)
- Error messages or stack traces

If the issue appears to be a bug report but is missing key information, respond with a helpful comment requesting the missing details. Use a friendly tone and provide specific guidance on what information would be helpful.

If the issue has sufficient detail, acknowledge it as a properly formatted bug report and indicate next steps.
```

### Feature Request Classification

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
---

# Feature Request Handler

Identify and respond to feature requests with appropriate guidance.

Analyze the issue content for:
- New functionality requests
- Enhancement suggestions  
- API or integration requests
- User experience improvements

For feature requests, provide a welcoming response that:
- Acknowledges the suggestion
- Explains the review process
- Links to contribution guidelines if appropriate
- Suggests community discussion if relevant

Maintain an encouraging tone while setting appropriate expectations about review timelines.
```

### Question and Support Routing

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
---

# Support Question Router

Identify support questions and provide immediate assistance or appropriate routing.

Look for:
- "How do I..." questions
- Configuration help requests
- Usage examples requests
- Troubleshooting questions

For questions that can be answered immediately, provide a helpful response with:
- Direct answers when possible
- Links to relevant documentation
- Code examples if appropriate
- Suggestions for further learning

For complex questions that need maintainer attention, acknowledge the question and explain the support process.
```

## Advanced Issue Ops Techniques

### Multi-Stage Response

```yaml
safe-outputs:
  add-comment:
    max: 3    # Allow multiple comments for complex workflows
```

Use multiple comments for workflows that need to:
- Provide immediate acknowledgment  
- Follow up with detailed analysis
- Add additional context after processing

### Cross-Repository Integration

```yaml
safe-outputs:
  add-comment:
    target: "*"    # Allow comments on any issue (requires issue_number in agent output)
```

For workflows that need to comment on issues in other repositories or reference external issues, use the flexible target configuration.

### Scheduled Issue Analysis

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"    # Weekly analysis
  issues:
    types: [opened]        # Also respond to new issues
```

Combine issue triggers with scheduled analysis for comprehensive issue management that handles both real-time responses and periodic review.

## Security and Access Control

Issue Ops workflows automatically have access to issue content, but follow these security practices:

**Content Validation**: Always validate and sanitize user input before processing
**Permission Boundaries**: Use minimal permissions and rely on safe outputs for write operations  
**Rate Limiting**: Configure appropriate `max` values to prevent abuse
**Context Awareness**: Remember that issue content is public and potentially untrusted

## Getting Started with Issue Ops

1. **Start Simple**: Begin with basic acknowledgment workflows before adding complex logic
2. **Test Thoroughly**: Use private repositories to test workflows before deploying to public projects
3. **Monitor Performance**: Review workflow logs to ensure responses are helpful and appropriate
4. **Iterate Based on Feedback**: Adjust workflows based on user and maintainer feedback
5. **Document Expectations**: Let users know what automated responses they can expect

Issue Ops transforms issue management from reactive to proactive, providing immediate value to users while reducing maintenance burden. Start with one pattern and expand based on your project's specific needs.