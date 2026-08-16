---
private: true
emoji: "🧹"
description: Daily cleanup of TODO/FIXME comments and simple Go code debt using Aider
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
sandbox:
  agent:
    id: awf
    sudo: false
tracker-id: daily-code-debt-aider
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
    labels: [automation, cleanup]
    draft: false
  missing-tool:
timeout-minutes: 30
imports:
  - shared/aider.md
  - shared/otlp.md
  - shared/reporting.md
---

# Daily Code Debt Cleanup — Aider

You are an automated coding agent that reduces Go code debt by resolving actionable TODO/FIXME comments
and removing trivially dead code. Aider has no MCP client; all safe-output events must be written as
JSONL lines to `$GH_AW_SAFE_OUTPUTS`.

## Step 1 — Find actionable TODO/FIXME comments

```bash
grep -rn "TODO\|FIXME" \
  --include="*.go" \
  --exclude-dir=vendor \
  --exclude-dir=.git \
  . | grep -v "_test.go" | head -20
```

From the output, identify at most **3 comments** that are:
- Self-contained (do not require external API changes or large refactors)
- Resolvable with ≤ 20 lines of code
- In a single file

Skip any comment that references an issue number (`#\d+`) or requires discussion.

## Step 2 — Resolve selected comments

For each selected comment:
1. Read the surrounding function to understand context.
2. Apply a minimal, correct fix using the `edit` tool.
3. Remove or update the TODO/FIXME marker after the fix.

## Step 3 — Verify the build still compiles

```bash
GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build ./...
```

If the build fails, revert the last change:

```bash
git diff --name-only | xargs git checkout --
```

## Step 4 — Format and create PR

If any code was changed:
1. Run `make fmt || true`
2. Write the following JSONL to `$GH_AW_SAFE_OUTPUTS`:

```
{"type":"create_pull_request","title":"Resolve actionable TODO/FIXME comments","body":"Automated cleanup of self-contained TODO and FIXME comments.\n\nChanges applied:\n<list each file and comment resolved>"}
```

If nothing was changed (no actionable items found), write:

```
{"type":"noop","reason":"No actionable TODO/FIXME comments found — skipping cleanup."}
```

## Exit rule

**Always** write at least one JSONL line to `$GH_AW_SAFE_OUTPUTS` before finishing.
