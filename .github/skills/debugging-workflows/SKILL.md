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

Use a matching local guidance checkout. For Contents API reads, use the guidance
owner/repository (`owner=github`, `repo=gh-aw` by default),
`path=.github/aw/debug-workflow-evidence.md`, retaining the ref of this skill.
The other API paths are `.github/aw/debug-bootstrap.md` and
`.github/aw/debug-agentic-workflow.md`. Normalize relative links against their
containing file before API reads; `../../aw/...` is not an API path.
For default raw guidance, use `https://raw.githubusercontent.com/github/gh-aw/<guidance-ref>/`
plus the repository-root-relative path, including `.github/`. Preserve supplied
refs containing slashes and keep guidance identity separate from the target
workflow repository/ref. The bootstrap documents size-aware reads and public
raw fallback after API denial.

Read the canonical instructions before running commands. This compatibility
router defines no separate setup, review, or live-execution procedure.
