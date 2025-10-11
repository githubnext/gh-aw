---
title: Markdown
description: Learn agentic workflow markdown content
sidebar:
  order: 300
---

The markdown content is where you write natural language instructions for the AI agent. 

For example:

```aw wrap
---
...frontmatter...
---

# Issue Triage

Read the issue #${{ github.event.issue.number }}. Add a comment to the issue listing useful resources and links.
```

The markdown is the most important part of your agentic workflow, and should describe its intended operation.

## Writing Good Agentic Markdown

Effective agentic markdown combines clear instructions, contextual information, and structured guidance to help AI agents perform tasks accurately and consistently.

#### Be Clear and Specific
Write instructions as if you're explaining the task to a new team member. Avoid ambiguity and provide concrete examples.

```aw wrap
# Good: Specific and actionable
Analyze issue #${{ github.event.issue.number }} and add appropriate labels from the repository's label list. Focus on categorizing the issue type (bug, feature, documentation) and priority level (high, medium, low).

# Avoid: Vague and unclear
Look at the issue and do something useful with labels.
```

#### Provide Context
Give the AI agent background information about your project, team preferences, and relevant constraints.

```aw wrap
# Project Context
This repository follows semantic versioning and uses GitHub Flow for branching. 
When reviewing pull requests, ensure:
- All tests pass
- Documentation is updated for API changes
- Breaking changes are clearly marked
```

#### Structure with Headings
Use markdown headings to organize instructions into logical sections that guide the agent's workflow.

```aw wrap
# Weekly Research Report

## Research Areas
Focus on these key areas for ${{ github.repository }}:
- Competitor analysis in the developer tools space
- Emerging trends in AI-powered development
- Community feedback and feature requests

## Output Format
Create a structured report with:
1. Executive summary (2-3 sentences)
2. Key findings by area
3. Recommended actions for next week
```

### Best Practices

#### Use Action-Oriented Language
Start instructions with clear action verbs and specify expected outcomes.

```aw wrap
# Effective action verbs
- "Analyze the pull request and identify potential issues"
- "Create a summary of recent issues tagged as 'bug'"
- "Update the documentation to reflect API changes"
- "Triage incoming issues by applying appropriate labels"
```

#### Include Decision Criteria
Help the agent make consistent decisions by providing clear criteria and examples.

```aw wrap
# Issue Labeling Criteria
Apply labels based on these guidelines:
- `bug`: Reports of incorrect behavior with steps to reproduce
- `enhancement`: Requests for new features or improvements
- `question`: Requests for help or clarification
- `documentation`: Issues related to docs, examples, or guides

Priority levels:
- `high-priority`: Security issues, critical bugs affecting many users
- `medium-priority`: Important features, non-critical bugs
- `low-priority`: Nice-to-have features, minor improvements
```

#### Reference Context Appropriately
Use sanitized context text and GitHub Actions expressions to provide secure, relevant context from the triggering event.

```aw wrap
# RECOMMENDED: Use sanitized context text for security
Analyze issue #${{ github.event.issue.number }} in repository ${{ github.repository }}.

The content to analyze: "${{ needs.activation.outputs.text }}"

# DISCOURAGED: Raw context fields (security risks from untrusted content)
The issue title is: "${{ github.event.issue.title }}"
The issue body is: "${{ github.event.issue.body }}"
```

**Why prefer `needs.activation.outputs.text`?**
- Automatically sanitizes @mentions, bot triggers, XML tags, and malicious URIs
- Prevents prompt injection attacks through user-controlled content
- Limits content size to prevent DoS through excessive text
- Removes control characters that could manipulate terminal output

#### Handle Edge Cases
Anticipate and provide guidance for unusual situations or error conditions.

```aw wrap
# Error Handling
If the workflow fails to complete any step:
1. Create an issue documenting the failure
2. Include relevant error messages and context
3. Tag the issue with 'workflow-failure' label
4. Exit gracefully without making partial changes
```

### Content Organization Patterns

#### Sequential Workflows
For multi-step processes, use numbered lists or clear sequential structure.

```aw wrap
# Code Review Process

1. **Initial Analysis**
   - Check if all required CI checks are passing
   - Verify the PR has an appropriate title and description

2. **Code Quality Review**
   - Scan for common code quality issues
   - Check for proper error handling and logging

3. **Generate Feedback**
   - Create constructive comments on specific lines
   - Summarize overall assessment in PR comment
```

#### Conditional Logic
Use clear conditional statements to guide agent decision-making.

```aw wrap
# Issue Triage Logic

If the issue contains error messages or stack traces:
  - Label as 'bug'
  - Check for similar existing issues
  - Request additional information if needed

If the issue is a feature request:
  - Label as 'enhancement' 
  - Assess scope and complexity
  - Consider impact on existing functionality

Otherwise:
  - Label as 'question' or 'discussion'
  - Provide helpful resources and documentation links
```

#### Template Patterns
Provide templates for consistent output formatting.

```aw wrap
# Weekly Status Report Template

Use this format for the status report:

## Summary
[Brief overview of the week's activities]

## Key Metrics
- Pull requests merged: [number]
- Issues resolved: [number]  
- New contributors: [number]

## Highlights
- [Notable achievements or milestones]
- [Important decisions or changes]

## Next Week
- [Planned activities and priorities]
```

### Common Pitfalls to Avoid

- **Over-complexity**: Keep instructions focused and avoid overwhelming the agent with too many simultaneous tasks
- **Assumption of knowledge**: Don't assume the agent knows your project's specific conventions or history
- **Inconsistent formatting**: Use consistent markdown formatting and structure across workflows
- **Missing error handling**: Always include guidance for what to do when things go wrong
- **Vague success criteria**: Clearly define what constitutes successful completion of the task

Before deploying workflows:
1. **Read aloud**: If instructions sound unclear when spoken, they'll be unclear to the agent
2. **Review examples**: Ensure all examples are accurate and reflect current repository state
3. **Consider edge cases**: Think through unusual scenarios the agent might encounter

## Templating

Agentic markdown supports GitHub Actions expression substitutions and conditional templating for content. See [Templating and Substitutions](/gh-aw/reference/templating/) for details.

## Related Documentation

- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow file organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - YAML configuration options
- [Security Notes](/gh-aw/guides/security/) - Comprehensive security guidance
