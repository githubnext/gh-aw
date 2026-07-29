---
title: Automated AI pull request review on GitHub
description: Use gh-aw to review pull requests automatically, post summary feedback, and add inline review comments through safe outputs.
---

Automated AI pull request review with gh-aw means running an agent on each pull request update so it can inspect the diff, identify likely defects, and return review feedback in GitHub-native review surfaces. The workflow keeps the agent read-only and uses safe outputs for the review summary and inline comments.

Install the starter with `gh aw add-wizard githubnext/agentics/pr-review`.

```aw wrap title=".github/workflows/pr-review.md"
---
on:
  pull_request:
    types: [opened, synchronize]

permissions:
  contents: read
  pull-requests: read

safe-outputs:
  add-comment:
    max: 1
  create-pull-request-review-comment:
    max: 10
  submit-pull-request-review:
    allowed-events: [COMMENT]
---

# Pull Request Review Assistant

Review the pull request diff for correctness, security, maintainability, and test coverage.

Create inline review comments only for specific problems or concrete improvements. Add one summary comment that groups findings by severity and notes anything that needs human follow-up. Do not restate unchanged code or provide style-only feedback.
```

This pattern uses `add-comment` for the summary and `create-pull-request-review-comment` for inline findings. Both are safe outputs, so the agent cannot write directly to the pull request; gh-aw validates the review payload before posting it.

## Related pages

- [Run Claude Code in GitHub Actions with gh-aw](/gh-aw/engines/claude/)
- [Run GitHub Copilot agents in GitHub Actions with gh-aw](/gh-aw/engines/copilot/)
- [ChatOps](/gh-aw/patterns/chat-ops/)
- [Quick start](/gh-aw/setup/quick-start/)
