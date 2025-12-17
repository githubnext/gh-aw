---
on:
  issues:
    types: [opened, edited]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 1
    post-as-comment: "https://github.com/githubnext/gh-aw/discussions/1234"
---

# Example: Noop with Post-as-Comment

This workflow demonstrates the new `post-as-comment` feature for the noop tool.

## How it works

When the workflow completes without taking any significant actions, instead of just logging a message, 
the noop message will be posted as a comment to the specified issue or discussion URL.

## Configuration

The `post-as-comment` field accepts a GitHub URL, short path, or just a number:
- An issue (full URL): `https://github.com/owner/repo/issues/123`
- An issue (short path): `owner/repo/issues/123`
- An issue (number only): `123` or `#123` (uses current repository)
- A discussion (full URL): `https://github.com/owner/repo/discussions/456`
- A discussion (short path): `owner/repo/discussions/456`
- A discussion (number only): `456` or `#456` (uses current repository, only in discussion context)

When only a number is provided, it uses the current repository. The type (issue or discussion) is inferred from the event context (discussion events default to discussion type, all others default to issue type).

If the URL/path/number is invalid or not provided, the noop tool falls back to its default behavior of just logging the message.

## Messages Template

When posting as a comment, noop includes a footer with workflow metadata (similar to other safe outputs). You can customize this footer using the `messages` configuration:

```yaml
safe-outputs:
  messages:
    footer: "> ✨ Created by [{workflow_name}]({run_url})"
    footer-install: "> Install with `gh aw add {workflow_source}`"
  noop:
    max: 1
    post-as-comment: "123"
```

Available placeholders: `{workflow_name}`, `{run_url}`, `{workflow_source}`, `{workflow_source_url}`, `{triggering_number}`.

## Task

Analyze the issue and determine if any action is needed. If no action is required, output a noop message 
explaining why (e.g., "Issue is already resolved", "No action needed", etc.).

The noop message will be automatically posted as a comment to the discussion specified in the configuration.
