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

The `post-as-comment` field accepts a GitHub URL to:
- An issue: `https://github.com/owner/repo/issues/123`
- A discussion: `https://github.com/owner/repo/discussions/456`

If the URL is invalid or not provided, the noop tool falls back to its default behavior of just logging the message.

## Task

Analyze the issue and determine if any action is needed. If no action is required, output a noop message 
explaining why (e.g., "Issue is already resolved", "No action needed", etc.).

The noop message will be automatically posted as a comment to the discussion specified in the configuration.
