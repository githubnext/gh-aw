---
description: GitHub context expression variables and Handlebars-style template conditionals ({{#if}}) for agentic workflows.
---

## GitHub Context Expression Interpolation

**For security reasons, only specific expressions are allowed.**

### Allowed Context Variables

- **`${{ github.event.after }}`** - SHA of the most recent commit after the push
- **`${{ github.event.before }}`** - SHA of the most recent commit before the push
- **`${{ github.event.check_run.id }}`** - ID of the check run
- **`${{ github.event.check_suite.id }}`** - ID of the check suite
- **`${{ github.event.comment.id }}`** - ID of the comment
- **`${{ github.event.deployment.id }}`** - ID of the deployment
- **`${{ github.event.deployment_status.id }}`** - ID of the deployment status
- **`${{ github.event.head_commit.id }}`** - ID of the head commit
- **`${{ github.event.installation.id }}`** - ID of the GitHub App installation
- **`${{ github.event.issue.number }}`** - Issue number
- **`${{ github.event.issue.state }}`** - State of the issue (open/closed)
- **`${{ github.event.issue.title }}`** - Title of the issue
- **`${{ github.event.label.id }}`** - ID of the label
- **`${{ github.event.milestone.id }}`** - ID of the milestone
- **`${{ github.event.milestone.number }}`** - Number of the milestone
- **`${{ github.event.organization.id }}`** - ID of the organization
- **`${{ github.event.page.id }}`** - ID of the GitHub Pages page
- **`${{ github.event.project.id }}`** - ID of the project
- **`${{ github.event.project_card.id }}`** - ID of the project card
- **`${{ github.event.project_column.id }}`** - ID of the project column
- **`${{ github.event.pull_request.number }}`** - Pull request number
- **`${{ github.event.pull_request.state }}`** - State of the pull request (open/closed)
- **`${{ github.event.pull_request.title }}`** - Title of the pull request
- **`${{ github.event.pull_request.head.sha }}`** - SHA of the PR head commit
- **`${{ github.event.pull_request.base.sha }}`** - SHA of the PR base commit
- **`${{ github.event.discussion.number }}`** - Discussion number
- **`${{ github.event.discussion.title }}`** - Title of the discussion
- **`${{ github.event.discussion.category.name }}`** - Category name of the discussion
- **`${{ github.event.release.assets[0].id }}`** - ID of the first release asset
- **`${{ github.event.release.id }}`** - ID of the release
- **`${{ github.event.release.name }}`** - Name of the release
- **`${{ github.event.release.tag_name }}`** - Tag name of the release
- **`${{ github.event.repository.id }}`** - ID of the repository
- **`${{ github.event.repository.default_branch }}`** - Default branch of the repository
- **`${{ github.event.review.id }}`** - ID of the review
- **`${{ github.event.review_comment.id }}`** - ID of the review comment
- **`${{ github.event.sender.id }}`** - ID of the user who triggered the event
- **`${{ github.event.deployment.environment }}`** - Deployment environment name
- **`${{ github.event.workflow_job.id }}`** - ID of the workflow job
- **`${{ github.event.workflow_job.run_id }}`** - Run ID of the workflow job
- **`${{ github.event.workflow_run.id }}`** - ID of the workflow run
- **`${{ github.event.workflow_run.number }}`** - Number of the workflow run
- **`${{ github.event.workflow_run.conclusion }}`** - Conclusion of the workflow run
- **`${{ github.event.workflow_run.status }}`** - Status of the workflow run
- **`${{ github.event.workflow_run.event }}`** - Event that triggered the workflow run
- **`${{ github.event.workflow_run.html_url }}`** - HTML URL of the workflow run
- **`${{ github.event.workflow_run.head_sha }}`** - Head SHA of the workflow run
- **`${{ github.event.workflow_run.run_number }}`** - Run number of the workflow run
- **`${{ github.actor }}`** - Username of the person who initiated the workflow
- **`${{ github.event_name }}`** - Name of the event that triggered the workflow
- **`${{ github.job }}`** - Job ID of the current workflow run
- **`${{ github.owner }}`** - Owner of the repository
- **`${{ github.repository }}`** - Repository name in "owner/name" format
- **`${{ github.repository_owner }}`** - Owner of the repository (organization or user)
- **`${{ github.run_id }}`** - Unique ID of the workflow run
- **`${{ github.run_number }}`** - Number of the workflow run
- **`${{ github.server_url }}`** - Base URL of the server, e.g. <https://github.com>
- **`${{ github.workflow }}`** - Name of the workflow
- **`${{ github.workspace }}`** - The default working directory on the runner for steps

#### Special Pattern Expressions

- **`${{ needs.* }}`** - Any outputs from previous jobs (e.g., `${{ needs.pre_activation.outputs.activated }}`)
- **`${{ steps.* }}`** - Any outputs from previous steps (e.g., `${{ steps.my-step.outputs.result }}`)
- **`${{ github.event.inputs.* }}`** - Any workflow inputs when triggered by workflow_dispatch (e.g., `${{ github.event.inputs.environment }}`)

All other expressions are disallowed.

### Sanitized Context Text (`steps.sanitized.outputs.text`)

**RECOMMENDED**: Use `${{ steps.sanitized.outputs.text }}` instead of individual `github.event` fields for accessing issue/PR content.

The `steps.sanitized.outputs.text` value provides automatically sanitized content based on the triggering event:

- **Issues**: `title + "\n\n" + body`
- **Pull Requests**: `title + "\n\n" + body`
- **Issue Comments**: `comment.body`
- **PR Review Comments**: `comment.body`
- **PR Reviews**: `review.body`
- **Other events**: Empty string

**Security Benefits of Sanitized Context:**

- **@mention neutralization**: Prevents unintended user notifications (converts `@user` to `` `@user` ``)
- **Bot trigger protection**: Prevents accidental bot invocations (converts `fixes #123` to `` `fixes #123` ``)
- **XML tag safety**: Converts XML tags to parentheses format to prevent injection
- **URI filtering**: Only allows HTTPS URIs from trusted domains; others become "(redacted)"
- **Content limits**: Automatically truncates excessive content (0.5MB max, 65k lines max)
- **Control character removal**: Strips ANSI escape sequences and non-printable characters

**Example Usage:**

```markdown
# RECOMMENDED: Use sanitized context text
Analyze this content: "${{ steps.sanitized.outputs.text }}"

# Less secure alternative (use only when specific fields are needed)
Issue number: ${{ github.event.issue.number }}
Repository: ${{ github.repository }}
```

### Security Validation

Expression safety is automatically validated during compilation. If unauthorized expressions are found, compilation will fail with an error listing the prohibited expressions.

### Example Usage

```markdown
# Valid expressions - RECOMMENDED: Use sanitized context text for security
Analyze issue #${{ github.event.issue.number }} in repository ${{ github.repository }}.

The issue content is: "${{ steps.sanitized.outputs.text }}"

# Alternative approach using individual fields (less secure)
The issue was created by ${{ github.actor }} with title: "${{ github.event.issue.title }}"

Using output from previous task: "${{ steps.sanitized.outputs.text }}"

Deploy to environment: "${{ github.event.inputs.environment }}"

# Invalid expressions (will cause compilation errors)
# Token: ${{ secrets.GITHUB_TOKEN }}
# Environment: ${{ env.MY_VAR }}
# Complex: ${{ toJson(github.workflow) }}
```

## Prompt Template Conditionals (`{{#if}}`)

The workflow markdown body supports a lightweight template language for conditional blocks. Template tags are resolved **at runtime, before the agent receives the prompt** — the agent always sees the final resolved text.

### Syntax

```
{{#if <condition>}}
...true branch content...
{{#else}}
...false branch content (optional)...
{{#endif}}
```

- **`{{#if <condition>}}`** — opens a conditional block; the content is included only when `<condition>` is truthy
- **`{{#else}}`** — optional separator; splits the block into a true branch and a false branch
- **`{{#endif}}`** — closes the block (**primary closing tag**; preferred)
- **`{{/if}}`** — alternate closing tag (both forms are permanently supported; `{{#endif}}` is preferred for consistency)

Tags may appear on their own line (block form) or inline. Block form (tag on its own line) is recommended for readability.

### Supported Conditions

| Form | Example | Truthy when |
|---|---|---|
| Bare value | `{{#if experiments.flag }}` | value is non-empty and not `"false"` |
| Equality | `{{#if experiments.style == "concise" }}` | value equals the quoted string |
| Inequality | `{{#if experiments.style != "verbose" }}` | value does not equal the quoted string |
| Strict equality | `{{#if experiments.style === "concise" }}` | value strictly equals the quoted string |
| Strict inequality | `{{#if experiments.style !== "verbose" }}` | value strictly differs from the quoted string |

### Example: Conditional Without Else

```markdown
{{#if experiments.skill_hint == "enabled" }}
Check `skills/` for SKILL.md files relevant to this task and apply their guidance.
{{#endif}}
```

### Example: Conditional With Else

```markdown
{{#if experiments.output_style == "concise" }}
Write a maximum of 5 bullet points. Each bullet is one sentence.
{{#else}}
Write a structured report with sections for new features, bug fixes, and refactors.
Include a one-paragraph executive summary at the top.
{{#endif}}
```

### Integration with Experiments

When the `experiments:` frontmatter field is set, the selected variant value is substituted into `{{#if experiments.<name> == "..." }}` conditions before template rendering. See [A/B Testing Experiments](../aw/experiments.md) for full experiment design guidance.

### Notes

- **Fenced code blocks are preserved** — `{{#if}}` tags inside `` ``` `` blocks are never processed; they appear verbatim in the output.
- **Nested conditionals are not supported** — do not place `{{#if}}` inside another `{{#if}}` block; the inner tags will be treated as literal text and appear verbatim in the agent prompt.
- **Template tags are not visible to the agent** — all `{{#if}}` / `{{#else}}` / `{{#endif}}` tags are stripped from the prompt before the agent runs.

