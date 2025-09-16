---
title: Markdown Content
description: Learn agentic workflow markdown content
sidebar:
  order: 3
---

The markdown content is where you write natural language instructions for the AI agent. 

## Writing Good Agentic Markdown

Effective agentic markdown combines clear instructions, contextual information, and structured guidance to help AI agents perform tasks accurately and consistently.

### Core Principles

#### Be Clear and Specific
Write instructions as if you're explaining the task to a new team member. Avoid ambiguity and provide concrete examples.

```markdown
# Good: Specific and actionable
Analyze issue #${{ github.event.issue.number }} and add appropriate labels from the repository's label list. Focus on categorizing the issue type (bug, feature, documentation) and priority level (high, medium, low).

# Avoid: Vague and unclear
Look at the issue and do something useful with labels.
```

#### Provide Context
Give the AI agent background information about your project, team preferences, and relevant constraints.

```markdown
# Project Context
This repository follows semantic versioning and uses GitHub Flow for branching. 
When reviewing pull requests, ensure:
- All tests pass
- Documentation is updated for API changes
- Breaking changes are clearly marked
```

#### Structure with Headings
Use markdown headings to organize instructions into logical sections that guide the agent's workflow.

```markdown
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

```markdown
# Effective action verbs
- "Analyze the pull request and identify potential issues"
- "Create a summary of recent issues tagged as 'bug'"
- "Update the documentation to reflect API changes"
- "Triage incoming issues by applying appropriate labels"
```

#### Include Decision Criteria
Help the agent make consistent decisions by providing clear criteria and examples.

```markdown
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
Use GitHub Actions expressions to provide relevant context from the triggering event.

```markdown
# Contextual Information
You are reviewing pull request #${{ github.event.pull_request.number }} 
in repository ${{ github.repository }}, submitted by ${{ github.actor }}.

This is workflow run #${{ github.run_number }}.
```

#### Handle Edge Cases
Anticipate and provide guidance for unusual situations or error conditions.

```markdown
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

```markdown
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

```markdown
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

```markdown
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

## Expression Security in Markdown Content

Agentic workflows restrict which GitHub Actions expressions can be used in **markdown content**. This prevents potential security vulnerabilities where access to secrets or environment variables is passed to workflows.

> **Note**: These restrictions apply only to expressions in the markdown content portion of workflows. The YAML frontmatter can still use secrets and environment variables as needed for workflow configuration (e.g., `env:` and authentication).

The following GitHub Actions context expressions are permitted in the markdown content:

### GitHub Context Expressions

- `${{ github.event.after }}` - The SHA of the most recent commit on the ref after the push
- `${{ github.event.before }}` - The SHA of the most recent commit on the ref before the push
- `${{ github.event.check_run.id }}` - The ID of the check run that triggered the workflow
- `${{ github.event.check_suite.id }}` - The ID of the check suite that triggered the workflow
- `${{ github.event.comment.id }}` - The ID of the comment that triggered the workflow
- `${{ github.event.deployment.id }}` - The ID of the deployment that triggered the workflow
- `${{ github.event.deployment_status.id }}` - The ID of the deployment status that triggered the workflow
- `${{ github.event.head_commit.id }}` - The ID of the head commit for the push event
- `${{ github.event.installation.id }}` - The ID of the GitHub App installation
- `${{ github.event.issue.number }}` - The number of the issue that triggered the workflow
- `${{ github.event.label.id }}` - The ID of the label that triggered the workflow
- `${{ github.event.milestone.id }}` - The ID of the milestone that triggered the workflow
- `${{ github.event.organization.id }}` - The ID of the organization that triggered the workflow
- `${{ github.event.page.id }}` - The ID of the page build that triggered the workflow
- `${{ github.event.project.id }}` - The ID of the project that triggered the workflow
- `${{ github.event.project_card.id }}` - The ID of the project card that triggered the workflow
- `${{ github.event.project_column.id }}` - The ID of the project column that triggered the workflow
- `${{ github.event.pull_request.number }}` - The number of the pull request that triggered the workflow
- `${{ github.event.release.assets[0].id }}` - The ID of the first asset in a release
- `${{ github.event.release.id }}` - The ID of the release that triggered the workflow
- `${{ github.event.release.tag_name }}` - The tag name of the release that triggered the workflow
- `${{ github.event.repository.id }}` - The ID of the repository that triggered the workflow
- `${{ github.event.review.id }}` - The ID of the pull request review that triggered the workflow
- `${{ github.event.review_comment.id }}` - The ID of the review comment that triggered the workflow
- `${{ github.event.sender.id }}` - The ID of the user who triggered the workflow
- `${{ github.event.workflow_run.id }}` - The ID of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.conclusion }}` - The conclusion of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.html_url }}` - The URL of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.head_sha }}` - The head SHA of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.run_number }}` - The run number of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.event }}` - The event that triggered the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.status }}` - The status of the workflow run that triggered the current workflow
- `${{ github.actor }}` - The username of the user who triggered the workflow
- `${{ github.job }}` - Job ID of the current workflow run
- `${{ github.owner }}` - The owner of the repository (user or organization name)
- `${{ github.repository }}` - The owner and repository name (e.g., `octocat/Hello-World`)
- `${{ github.run_id }}` - A unique number for each workflow run within a repository
- `${{ github.run_number }}` - A unique number for each run of a particular workflow in a repository
- `${{ github.server_url }}` - Base URL of the server, e.g. https://github.com
- `${{ github.workflow }}` - The name of the workflow
- `${{ github.workspace }}` - The default working directory on the runner for steps

### Special Pattern Expressions
- `${{ needs.* }}` - Any outputs from previous jobs (e.g., `${{ needs.task.outputs.text }}`)
- `${{ steps.* }}` - Any outputs from previous steps in the same job
- `${{ github.event.inputs.* }}` - Any workflow inputs when triggered by workflow_dispatch (e.g., `${{ github.event.inputs.name }}`)

## Prohibited Expressions

All other expressions are disallowed, including:
- `${{ secrets.* }}` - All secrets
- `${{ env.* }}` - All environment variables
- `${{ vars.* }}` - All repository variables
- Complex functions like `${{ toJson(...) }}`, `${{ fromJson(...) }}`, etc.

## Security Rationale

This restriction prevents:
- **Secret leakage**: Prevents accidentally exposing secrets in AI prompts or logs
- **Environment variable exposure**: Protects sensitive configuration from being accessed
- **Code injection**: Prevents complex expressions that could execute unintended code
- **Expression injection**: Prevents malicious expressions from being injected into AI prompts
- **Prompt hijacking**: Stops unauthorized modification of workflow instructions through expression values
- **Cross-prompt information attacks (XPIA)**: Blocks attempts to leak information between different workflow executions

## Validation

Expression safety is validated during compilation with `gh aw compile`. If unauthorized expressions are found, you'll see an error like:

```
error: unauthorized expressions: [secrets.TOKEN, env.MY_VAR]. 
allowed: [github.repository, github.actor, github.workflow, ...]
```

## Example Valid Usage

```markdown
# Valid expressions
Repository: ${{ github.repository }}
Triggered by: ${{ github.actor }}  
Issue number: ${{ github.event.issue.number }}
Previous output: ${{ needs.task.outputs.text }}
User input: ${{ github.event.inputs.environment }}
Workflow run conclusion: ${{ github.event.workflow_run.conclusion }}

# Invalid expressions (will cause compilation error)
Token: ${{ secrets.GITHUB_TOKEN }}
Environment: ${{ env.MY_VAR }}
Complex: ${{ toJson(github.workflow) }}
```

## Best Practices

- **Use allowed expressions**: Stick to the permitted GitHub context expressions
- **Reference event data**: Use `${{ github.event.* }}` to access trigger-specific information  
- **Leverage workflow context**: Use `${{ github.repository }}`, `${{ github.actor }}` for basic context
- **Pass data via frontmatter**: Use YAML frontmatter for secrets and sensitive configuration
- **Test compilation**: Always run `gh aw compile` to validate expression usage

## Related Documentation

- [Workflow Structure](./workflow-structure/) - Overall workflow file organization
- [Frontmatter Options](./frontmatter/) - YAML configuration options
- [Security Notes](../guides/security-notes/) - Comprehensive security guidance
