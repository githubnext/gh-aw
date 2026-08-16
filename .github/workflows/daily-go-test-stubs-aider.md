---
private: true
emoji: "🧪"
description: Daily scan of Go packages for missing test coverage and automatic addition of test stubs using Aider
on:
  schedule: daily
  workflow_dispatch:
permissions:
  copilot-requests: write
  contents: read
  issues: read
  pull-requests: read
sandbox:
  agent:
    id: awf
    sudo: false
tracker-id: daily-go-test-stubs-aider
engine:
  id: aider
model: copilot/claude-sonnet-4.5
strict: true
network:
  allowed: []
tools:
  edit:
  bash:
    - "*"
safe-outputs:
  create-pull-request:
    expires: 2d
    title-prefix: "[aider] "
    labels: [automation, testing]
    draft: false
  missing-tool:
timeout-minutes: 30
imports:
  - shared/aider.md
  - shared/otlp.md
  - shared/reporting.md
---

# Daily Go Test Stubs — Aider

You are an automated coding agent that improves Go test coverage by finding untested packages and adding
minimal test stubs. Aider has no MCP client; all safe-output events must be written as JSONL lines to
`$GH_AW_SAFE_OUTPUTS`. Follow the Aider execution constraints above: one shell command per line, and edit
files with *SEARCH/REPLACE* blocks rather than heredocs.

## Step 1 — Find packages with no test files

```bash
for pkg in $(find . -name '*.go' -not -name '*_test.go' -not -path './vendor/*' -not -path './.git/*' | sed 's|/[^/]*\.go$||' | sort -u); do [ -z "$(find "$pkg" -maxdepth 1 -name '*_test.go')" ] && echo "$pkg"; done | head -5
```

Pick at most 3 packages from the output.

## Step 2 — Add minimal test stubs

For each selected package, create a `<package>_test.go` stub with a *SEARCH/REPLACE* block that:
- Declares the correct package name (use `package <pkg>_test` if the source uses external test convention, otherwise `package <pkg>`)
- Imports `"testing"`
- Contains one `TestPlaceholder_TODO` function with `t.Skip("not yet implemented")`

## Step 3 — Commit and create the pull request

If any stubs were created, run these commands (one per line):

```bash
make fmt || true
GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build ./... && git checkout -b add-test-stubs-$GITHUB_RUN_ID && git add -A && git commit -m "Add test stubs for uncovered packages" && printf '%s\n' "{\"type\":\"create_pull_request\",\"title\":\"Add test stubs for uncovered packages\",\"body\":\"Automatically generated test stubs for packages with zero test coverage. Stubs use t.Skip and are ready to be filled in.\",\"branch\":\"add-test-stubs-$GITHUB_RUN_ID\"}" >> "$GH_AW_SAFE_OUTPUTS"
```

If no packages needed stubs (all packages already had tests), run instead:

```bash
printf '%s\n' '{"type":"noop","message":"All Go packages already have test files — no stubs needed."}' >> "$GH_AW_SAFE_OUTPUTS"
```

## Step 4 — Exit rule

**Always** write exactly one JSONL line to `$GH_AW_SAFE_OUTPUTS` before finishing.
