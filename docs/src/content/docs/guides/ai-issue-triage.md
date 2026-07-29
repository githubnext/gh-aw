---
title: AI issue triage on GitHub
description: Use gh-aw to triage GitHub issues automatically by labeling issues, detecting duplicates, and asking follow-up questions through safe outputs.
---

AI issue triage with gh-aw means running an agent whenever a new issue arrives so it can classify the report, set priority labels, detect likely duplicates, and ask for missing details. The workflow stays read-only during analysis, and gh-aw performs the resulting GitHub writes through safe outputs.

Install the starter with `gh aw add-wizard githubnext/agentics/issue-triage`. The pattern is adapted from the workflows published in [githubnext/agentics](https://github.com/githubnext/agentics).

```aw wrap title=".github/workflows/issue-triage.md"
---
on:
  issues:
    types: [opened, edited]

permissions:
  contents: read
  issues: read

safe-outputs:
  add-labels:
    allowed:
      - bug
      - feature
      - question
      - needs-info
      - priority/p0
      - priority/p1
      - priority/p2
      - duplicate
    max: 4
  add-comment:
    max: 1
---

# Issue Triage Assistant

Review the triggering issue and do three things:

1. Classify the issue by type and priority, then add the matching labels.
2. Identify likely duplicates from existing open and recent closed issues.
3. If required details are missing, ask concise clarifying questions in one comment.

Use `duplicate` only when the match is strong and include the issue number in the comment. Use `needs-info` when reproduction steps, expected behavior, or environment details are missing.
```

`add-labels` and `add-comment` matter for security because the agent does not receive direct write access to issues. gh-aw validates label names and comment output before posting, which reduces the risk of prompt injection turning repository analysis into unrestricted writes.

## Related pages

- [Run Claude Code in GitHub Actions with gh-aw](/gh-aw/engines/claude/)
- [Run GitHub Copilot agents in GitHub Actions with gh-aw](/gh-aw/engines/copilot/)
- [IssueOps](/gh-aw/patterns/issue-ops/)
- [Quick start](/gh-aw/setup/quick-start/)
