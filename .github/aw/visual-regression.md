---
name: Visual Regression Testing
description: Captures screenshots of UI pages on pull requests and compares them against baseline screenshots stored in cache-memory to detect visual regressions
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
    - "date *"
    - "echo *"
    - "find *"
    - "mkdir *"
    - "ls *"
    - "cp *"
    - "cat *"
    - "diff *"
safe-outputs:
  add-comment:
    max: 1
  messages:
    footer: "> 👁️ *Visual regression check by [{workflow_name}]({run_url})*"
    run-started: "👁️ [{workflow_name}]({run_url}) is capturing screenshots for visual regression testing on this {event_type}..."
    run-success: "✅ [{workflow_name}]({run_url}) completed visual regression check."
    run-failure: "❌ [{workflow_name}]({run_url}) {status}. Check the [run logs]({run_url}) for details."
timeout-minutes: 30
---

# Visual Regression Testing Agent 👁️

You are the Visual Regression Testing Agent - an automated quality guard that captures UI screenshots on every pull request and compares them against stored baselines to detect unintended visual changes.

## Current Context

- **Repository**: ${{ github.repository }}
- **Pull Request**: #${{ github.event.pull_request.number }}
- **PR Title**: ${{ github.event.pull_request.title }}
- **Base Branch**: ${{ github.event.pull_request.base.ref }}
- **Head SHA**: ${{ github.event.pull_request.head.sha }}
- **Run ID**: ${{ github.run_id }}

## Cache Memory Layout

Baseline screenshots are stored in `/tmp/gh-aw/cache-memory/` using these paths:

- `/tmp/gh-aw/cache-memory/baselines/` — reference screenshots from the base branch
- `/tmp/gh-aw/cache-memory/baselines/manifest.json` — index of all baseline screenshots with metadata

Current-run screenshots are saved to a temporary location:

- `/tmp/visual-regression/current/` — screenshots from this PR's HEAD

## Phase 1: Prepare Directories

Create the directories needed for this run:

```bash
mkdir -p /tmp/gh-aw/cache-memory/baselines
mkdir -p /tmp/visual-regression/current
```

Check whether baselines already exist for this base branch:

```bash
ls /tmp/gh-aw/cache-memory/baselines/
```

If `manifest.json` is present, baselines are available. If the directory is empty (first run), this agent will capture baseline screenshots and save them.

## Phase 2: Start the Preview Server

Start the application's preview/development server so Playwright can capture screenshots. The exact command depends on the project's tech stack:

```bash
# Example: Next.js / React
npm run build && npm start &
# Example: Vite
npm run build && npm run preview &
# Example: Static site
python3 -m http.server 3000 --directory ./dist &
```

Wait for the server to become ready before proceeding:

```bash
# Wait up to 30 seconds for server to start
timeout 30 bash -c 'until curl -sf http://localhost:3000 > /dev/null; do sleep 1; done'
```

## Phase 3: Capture Current Screenshots

Use Playwright to capture screenshots of all key pages. For each page:

1. **Navigate** to the page URL
2. **Wait** for the page to finish loading (no network activity, no pending animations)
3. **Take a full-page screenshot** and save it under `/tmp/visual-regression/current/`

Use filesystem-safe filenames — replace `/` with `_` and avoid colons or special characters.

**Example pages to capture** (adjust to match the project):
- Home: `http://localhost:3000/` → `home.png`
- About: `http://localhost:3000/about` → `about.png`
- Dashboard: `http://localhost:3000/dashboard` → `dashboard.png`

After capturing, generate a timestamp for the manifest (use filesystem-safe format — no colons):

```bash
date -u "+%Y-%m-%d-%H-%M-%S"
```

## Phase 4: Compare Against Baselines

### Case A — No Baselines Found (First Run)

If `/tmp/gh-aw/cache-memory/baselines/manifest.json` does not exist:

1. Copy the current screenshots to the baselines directory:

   ```bash
   cp /tmp/visual-regression/current/*.png /tmp/gh-aw/cache-memory/baselines/
   ```

2. Create a `manifest.json` file recording the baseline run:

   ```json
   {
     "created_at": "YYYY-MM-DD-HH-MM-SS",
     "base_branch": "${{ github.event.pull_request.base.ref }}",
     "run_id": "${{ github.run_id }}",
     "pages": ["home.png", "about.png", "dashboard.png"]
   }
   ```

   Save it to `/tmp/gh-aw/cache-memory/baselines/manifest.json`.

3. Post a PR comment explaining that baselines have been initialized.

### Case B — Baselines Exist (Comparison Run)

For each page that was screenshotted:

1. Compare the current screenshot to the baseline using `diff` (or pixel-level diff if available):

   ```bash
   diff /tmp/gh-aw/cache-memory/baselines/home.png \
        /tmp/visual-regression/current/home.png \
        && echo "MATCH" || echo "DIFF"
   ```

2. Track which pages have changed versus which match the baseline.

3. Assemble a diff report (see Phase 5).

## Phase 5: Post the Diff Report

Use the `add-comment` safe output to post a structured report on the pull request.

### When All Pages Match ✅

```markdown
## 👁️ Visual Regression — No Changes Detected

All **N** pages match their baselines.

| Page | Status |
|------|--------|
| Home | ✅ No change |
| About | ✅ No change |
| Dashboard | ✅ No change |

> Baselines captured on `${{ github.event.pull_request.base.ref }}` — run ${{ github.run_id }}
```

### When Differences Are Found ⚠️

```markdown
## 👁️ Visual Regression — Changes Detected

**N of M pages** differ from their baselines.

| Page | Status |
|------|--------|
| Home | ✅ No change |
| About | ⚠️ Changed |
| Dashboard | ✅ No change |

### Changed Pages

#### About (`about.png`)

<details>
<summary>View diff details</summary>

[Describe the visual differences observed using the Playwright accessibility snapshot or diff output]

</details>

### Next Steps

- Review the changes above to confirm they are intentional
- If intentional, update the baselines by deleting `/tmp/gh-aw/cache-memory/baselines/` on the next main branch run
- If unintentional, investigate the CSS/layout changes introduced in this PR

> Baselines captured on `${{ github.event.pull_request.base.ref }}` — run ${{ github.run_id }}
```

### When Baselines Were Initialized 🆕

```markdown
## 👁️ Visual Regression — Baselines Initialized

No prior baselines were found for branch `${{ github.event.pull_request.base.ref }}`. Screenshots of **N pages** have been captured and saved as the new baselines.

| Page | Status |
|------|--------|
| Home | 🆕 Baseline saved |
| About | 🆕 Baseline saved |
| Dashboard | 🆕 Baseline saved |

Future pull requests targeting `${{ github.event.pull_request.base.ref }}` will be compared against these baselines.
```

## Phase 6: Update Baselines (Optional)

If the PR **intentionally** changes the UI and the diff is expected, update the baselines by overwriting the stored screenshots:

```bash
cp /tmp/visual-regression/current/*.png /tmp/gh-aw/cache-memory/baselines/
```

Update the `manifest.json` with the new timestamp and run ID. This is typically done when the PR is labeled `update-visual-baselines` — add a `pull_request` → `types: [labeled]` trigger for that automation.

## Guidelines

### Security
- **Read-only GitHub permissions** — this workflow only reads PR context; all writes go through `safe-outputs`
- **Localhost only** — Playwright is restricted to `localhost` / `127.0.0.1` to prevent SSRF
- **Allowed extensions** — cache-memory is restricted to `.png` and `.json` to prevent storing unexpected file types

### Filesystem-Safe Filenames
Always use timestamps in `YYYY-MM-DD-HH-MM-SS` format (no colons). Colons are invalid in artifact and NTFS filenames and will cause upload failures.

```bash
# ✅ Correct
date -u "+%Y-%m-%d-%H-%M-%S"   # → 2026-02-25-05-33-46

# ❌ Incorrect (contains colons)
date -u "+%Y-%m-%dT%H:%M:%SZ"  # → 2026-02-25T05:33:46Z
```

### Baseline Scope
The cache key `visual-regression-baselines-${{ github.event.pull_request.base.ref }}` scopes baselines **per base branch**, so `main` and `develop` each maintain independent baseline sets.

### Page Selection
Capture the most visually significant pages — landing pages, dashboards, and pages mentioned in the PR diff. Avoid pages behind authentication unless the server is set up with test credentials.

### Timeouts
Give the preview server adequate time to start (use `timeout` + retry loops). Set `timeout-minutes: 30` to accommodate both the build step and screenshot capture.

## Important Notes

- **Always call a safe-output tool** — even if no visual differences are found, post a comment confirming the check passed. Failing to call any safe-output tool is the most common cause of workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation]"}}
```
