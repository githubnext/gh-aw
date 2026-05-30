---
name: visual-regression
description: Reference prompt for visual regression testing using playwright + cache-memory for baseline screenshot storage across pull requests
---

# Visual Regression Testing

Use `playwright` for screenshots and `cache-memory` to persist baselines between PR runs.

## Example Workflow

```markdown
---
description: Capture screenshots on every PR and compare against cached baselines to detect visual regressions
on:
  pull_request:
    types: [opened, synchronize, reopened]
permissions:
  contents: read
  pull-requests: read
engine: copilot
tools:
  playwright:
    allowed_domains:
      - localhost
      - 127.0.0.1
  cache-memory:
    key: visual-regression-baselines-${{ github.event.pull_request.base.ref }}
    retention-days: 30
    allowed-extensions: [".png", ".json"]
  bash:
    - "mkdir *"
    - "cp *"
    - "diff *"
    - "date *"
safe-outputs:
  add-comment:
    max: 1
timeout-minutes: 30
---

Build and serve the app locally, then use Playwright to capture full-page screenshots of key pages into `/tmp/visual-regression/current/`.

Use filesystem-safe timestamps (no colons — colons break artifact uploads):
`date -u "+%Y-%m-%d-%H-%M-%S"`

If `/tmp/gh-aw/cache-memory/baselines/manifest.json` does not exist, copy screenshots there as new baselines and post: "Baselines initialized — N pages captured."

Otherwise compare each screenshot to its baseline. Post a comment summarizing: pages unchanged / pages with diffs. If nothing changed, use the `noop` safe-output.
```

## Key Design Decisions

- **`cache-memory` key per base branch** — scopes baselines to `main`, `develop`, etc. independently
- **`allowed_domains: [localhost, 127.0.0.1]`** — prevents SSRF; app must be served locally
- **`retention-days: 30`** — keeps baselines beyond the default 7-day cache expiry
- **Filesystem-safe timestamps** — `YYYY-MM-DD-HH-MM-SS` format; colons are invalid in artifact filenames
- **Minimal permissions** — all PR writes go through `safe-outputs`, not GitHub tools
