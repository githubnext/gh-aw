---
name: visual-regression
description: Reference workflow for visual regression testing using playwright to capture screenshots and cache-memory to store baselines across pull requests
---

# Visual Regression Testing Reference

This is a reference workflow that demonstrates how to combine `playwright` (for capturing screenshots) with `cache-memory` (for storing baselines between runs) to implement visual regression testing on pull requests.

## Example Workflow

```markdown
---
description: Captures screenshots on every PR and compares them against baselines stored in cache-memory to detect visual regressions
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
    - "echo *"
safe-outputs:
  add-comment:
    max: 1
  messages:
    footer: "> 👁️ *Visual regression check by [{workflow_name}]({run_url})*"
    run-started: "👁️ [{workflow_name}]({run_url}) is running visual regression on this {event_type}..."
    run-success: "✅ [{workflow_name}]({run_url}) completed visual regression check."
    run-failure: "❌ [{workflow_name}]({run_url}) {status}. Check the [run logs]({run_url}) for details."
timeout-minutes: 30
---

# Visual Regression Agent 👁️

You are the Visual Regression Agent. On every pull request, capture screenshots of key UI pages and compare them against baseline screenshots stored in cache-memory.

## Context

- **Repository**: ${{ github.repository }}
- **Pull Request**: #${{ github.event.pull_request.number }}
- **Base Branch**: ${{ github.event.pull_request.base.ref }}
- **Run ID**: ${{ github.run_id }}

## Steps

### 1. Prepare directories

```bash
mkdir -p /tmp/gh-aw/cache-memory/baselines
mkdir -p /tmp/visual-regression/current
```

### 2. Start the preview server

Build and serve the app locally (adjust for the project's stack), then wait for it to be ready:

```bash
npm run build && npm start &
timeout 30 bash -c 'until curl -sf http://localhost:3000 > /dev/null; do sleep 1; done'
```

### 3. Capture screenshots

Use Playwright to navigate to each key page and take a full-page screenshot saved under `/tmp/visual-regression/current/`. Use filesystem-safe filenames (replace `/` with `_`, no special characters).

Generate a filesystem-safe timestamp when needed (no colons — colons break artifact uploads):

```bash
date -u "+%Y-%m-%d-%H-%M-%S"
```

### 4. Compare against baselines

**First run** — no `manifest.json` in `/tmp/gh-aw/cache-memory/baselines/`:

Copy current screenshots to `/tmp/gh-aw/cache-memory/baselines/` and write a `manifest.json`:

```json
{
  "created_at": "YYYY-MM-DD-HH-MM-SS",
  "base_branch": "${{ github.event.pull_request.base.ref }}",
  "run_id": "${{ github.run_id }}",
  "pages": ["home.png", "dashboard.png"]
}
```

**Subsequent runs** — baselines exist:

Compare each screenshot with its baseline and collect the list of changed pages.

### 5. Post diff report

Use `add-comment` to post the results. Examples:

**No changes:**
```
## 👁️ Visual Regression — No Changes Detected
All N pages match their baselines.
```

**Changes found:**
```
## 👁️ Visual Regression — Changes Detected
N of M pages differ from their baselines.
List each changed page with a description of the difference.
```

**Baselines initialized:**
```
## 👁️ Visual Regression — Baselines Initialized
No prior baselines found. N pages captured and saved as new baselines.
```

Always call a safe-output tool. If nothing changed, use noop:

```json
{"noop": {"message": "Visual regression complete: all pages match baselines"}}
```
```

## Key Design Decisions

- **`cache-memory` key per base branch** — `visual-regression-baselines-${{ github.event.pull_request.base.ref }}` scopes baselines to `main`, `develop`, etc. independently
- **`allowed-extensions: [".png", ".json"]`** — restricts cache to screenshots and the manifest only
- **`playwright.allowed_domains: [localhost, 127.0.0.1]`** — prevents SSRF; the app must be served locally
- **`retention-days: 30`** — keeps baselines accessible beyond the default 7-day cache expiry
- **Filesystem-safe timestamps** — `YYYY-MM-DD-HH-MM-SS` (no colons); colons are invalid in NTFS/artifact filenames
- **Minimal permissions** — `contents: read` + `pull-requests: read`; all PR writes go through `safe-outputs`
