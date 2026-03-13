---
title: Footer Control
description: Learn how to control AI-generated footers in safe output operations and customize footer messages for GitHub issues, pull requests, discussions, and releases.
sidebar:
  order: 805
---

Control whether AI-generated footers are added to created and updated GitHub items (issues, pull requests, discussions, releases). Footers provide attribution and links to workflow runs, but you may want to omit them for cleaner content or when using custom branding.

## Global Footer Control

Set `footer: false` at the safe-outputs level to hide footers for all output types:

```yaml wrap
safe-outputs:
  footer: false                      # hide footers globally
  create-issue:
    title-prefix: "[ai] "
  create-pull-request:
    title-prefix: "[ai] "
```

When `footer: false` is set, visible attribution text is omitted from item bodies but hidden XML markers (workflow-id, tracker-id) remain for searchability. Applies to all output types: create-issue, create-pull-request, create-discussion, update-issue, update-pull-request, update-discussion, and update-release.

## Per-Handler Footer Control

Override the global setting for specific output types by setting `footer` at the handler level:

```yaml wrap
safe-outputs:
  footer: false                      # global default: no footers
  create-issue:
    title-prefix: "[issue] "
    # inherits footer: false
  create-pull-request:
    title-prefix: "[pr] "
    footer: true                     # override: show footer for PRs only
```

Individual handler settings always take precedence over the global setting.

## PR Review Footer Control

For PR reviews (`submit-pull-request-review`), the `footer` field supports conditional control over when the footer is added to the review body:

```yaml wrap
safe-outputs:
  create-pull-request-review-comment:
  submit-pull-request-review:
    footer: "if-body"         # conditional footer based on review body
```

The `footer` field accepts `"always"` (default), `"none"`, or `"if-body"` (footer only when the review has body text). Booleans are accepted: `true` → `"always"`, `false` → `"none"`. Use `"if-body"` for clean approval reviews — approvals without body text appear without a footer, while reviews with comments include it.

## What's Preserved When Footer is Hidden

Even with `footer: false`, hidden HTML markers remain:
- `<!-- gh-aw-workflow-id: WORKFLOW_NAME -->` — for search and tracking
- `<!-- gh-aw-tracker-id: unique-id -->` — for issue/discussion tracking (when applicable)

These markers enable searching for workflow-created items even when footers are hidden.

### Searching for Workflow-Created Items

You can use the workflow-id marker to find all items created by a specific workflow on GitHub.com. The marker is always included in the body of issues, pull requests, discussions, and comments, regardless of the `footer` setting.

**Search Examples:**

Find all open issues created by the `daily-team-status` workflow:
```
repo:owner/repo is:issue is:open "gh-aw-workflow-id: daily-team-status" in:body
```

Find all pull requests created by the `security-audit` workflow:
```
repo:owner/repo is:pr "gh-aw-workflow-id: security-audit" in:body
```

Find all items (issues, PRs, discussions) from any workflow in your organization:
```
org:your-org "gh-aw-workflow-id:" in:body
```

Find comments from a specific workflow:
```
repo:owner/repo "gh-aw-workflow-id: bot-responder" in:comments
```

> [!TIP]
> The workflow name in the marker is the filename without `.md`. Combine with filters like `is:open` or `created:>2024-01-01`, and use `in:body` or `in:comments` as appropriate. See [GitHub advanced search](https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests).

## Use Cases

**Clean content for public repositories:**
```yaml wrap
safe-outputs:
  footer: false
  create-issue:
    title-prefix: "[report] "
    labels: [automated]
```

**Custom branding - footers on PRs only:**
```yaml wrap
safe-outputs:
  footer: false                      # hide for issues
  create-issue:
    title-prefix: "[issue] "
  create-pull-request:
    footer: true                     # show for PRs
    title-prefix: "[pr] "
```

**Minimal documentation updates:**
```yaml wrap
safe-outputs:
  update-release:
    footer: false                    # clean release notes
    max: 1
```

## Backward Compatibility

The default value for `footer` is `true`, maintaining backward compatibility with existing workflows. To hide footers, you must explicitly set `footer: false`.

## Customizing Footer Messages

Instead of hiding footers entirely, you can customize the footer message text using the `messages.footer` template. This allows you to maintain attribution while using custom branding:

```yaml wrap
safe-outputs:
  messages:
    footer: "> 🤖 Powered by [{workflow_name}]({run_url})"
  create-issue:
    title-prefix: "[bot] "
```

The `messages.footer` template supports variables like `{workflow_name}`, `{run_url}`, `{triggering_number}`, and more. See [Custom Messages](/gh-aw/reference/safe-outputs/#custom-messages-messages) for complete documentation on message templates and available variables.

## Related Documentation

- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Complete safe outputs reference
- [Custom Messages](/gh-aw/reference/safe-outputs/#custom-messages-messages) - Message templates and variables
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
