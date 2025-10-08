---
"githubnext/gh-aw": patch
---

Add GITHUB_AW_WORKFLOW_NAME environment variable to add_reaction job

Fixed a bug where the `add_reaction` job was missing the `GITHUB_AW_WORKFLOW_NAME` environment variable, causing the workflow name to fall back to the generic "Workflow" instead of displaying the actual workflow name in comments and reactions. The environment variable is now consistently set across all safe-output jobs.
