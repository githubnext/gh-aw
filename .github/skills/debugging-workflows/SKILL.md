---
name: debugging-workflows
description: Diagnose gh-aw failures using the canonical debugging strategy and evidence reference.
---

# Workflow Debugging Router

This skill remains available for existing invocations. Load and follow
[the evidence reference](../../aw/debug-workflow-evidence.md), where the
instructions now live. It delegates setup to
[the bootstrap](../../aw/debug-bootstrap.md) and execution policy to
[the shared strategy](../../aw/debug-agentic-workflow.md).

Use these local paths in a gh-aw checkout. Otherwise fetch the same paths from
`https://raw.githubusercontent.com/github/gh-aw/main/`, retaining the ref when
this skill was loaded from a different gh-aw revision. Resolve relative links
against their containing file, not the target workflow repository.

Read the canonical instructions before running commands. This compatibility
router defines no separate setup, review, or live-execution procedure.
