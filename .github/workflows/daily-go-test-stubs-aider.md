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
`$GH_AW_SAFE_OUTPUTS`.

## Step 1 — Find packages with no test files

```bash
find . -name "*.go" -not -name "*_test.go" -not -path "*/vendor/*" -not -path "*/.git/*" \
  | sed 's|/[^/]*\.go$||' | sort -u \
  | while read pkg; do
      test_count=$(find "$pkg" -maxdepth 1 -name "*_test.go" 2>/dev/null | wc -l)
      if [ "$test_count" -eq 0 ]; then echo "$pkg"; fi
    done | head -5
```

Pick at most 3 packages from the output.

## Step 2 — Add minimal test stubs

For each selected package, create a `<package>_test.go` stub that:
- Declares the correct package name (use `package <pkg>_test` if the source uses external test convention, otherwise `package <pkg>`)
- Imports `"testing"`
- Contains one `TestPlaceholder_TODO` function with `t.Skip("not yet implemented")`

Write the file using the `edit` tool.

## Step 3 — Commit and create PR

If any stubs were created:
1. Run `make fmt` (ignore errors if no Makefile target exists — catch with `|| true`)
2. Confirm the build still passes: `GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build ./...`
3. Write the following JSONL line to `$GH_AW_SAFE_OUTPUTS` to create a pull request:

```
{"type":"create_pull_request","title":"Add test stubs for uncovered packages","body":"Automatically generated test stubs for packages with zero test coverage.\n\nPackages updated: <list>\n\nStubs use `t.Skip` and are ready to be filled in."}
```

If no packages needed stubs (all packages already had tests), write:

```
{"type":"noop","reason":"All Go packages already have test files — no stubs needed."}
```

## Step 4 — Exit rule

**Always** write at least one JSONL line to `$GH_AW_SAFE_OUTPUTS` before finishing.
