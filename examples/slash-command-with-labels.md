---
on:
  slash_command: triage
  issues:
    types: [labeled]
tools:
  github:
    toolsets: [issues]
---

# Issue Triage Workflow

This workflow demonstrates the **label-only exception** for slash command triggers.

## How it works

This workflow is triggered in two ways:

1. **Manual trigger**: When someone comments `/triage` on an issue
2. **Automatic trigger**: When a label is added to an issue

The label-only exception allows combining `slash_command` with `issues` or `pull_request` 
events as long as those events only specify `labeled` or `unlabeled` types. This pattern 
is useful for workflows that need both manual and automatic triggering based on labels.

## What happens when triggered

When triggered by `/triage` command:
- Responds to the slash command in the issue comment
- Can perform triage actions based on the command context

When triggered by label addition:
- Automatically reacts to label changes
- Can perform automatic triage based on the new label

The workflow receives context through `needs.activation.outputs.text` which contains 
the sanitized issue title and body.

## Example use cases

- Automatic routing based on labels, with manual override via command
- Workflow that processes issues when labeled, but can also be triggered on demand
- Combined automation and manual intervention workflows
