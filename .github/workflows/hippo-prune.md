---
private: true
emoji: "🦛"
name: Hippo Prune
description: One-time maintenance workflow to inspect and prune the four low-quality Hippo memories flagged by the daily audit
on:
  workflow_dispatch:

permissions:
  contents: read

  copilot-requests: write
tracker-id: hippo-prune
engine:
  id: pi
  model: copilot/gpt-5.4
  bare: true

timeout-minutes: 30

runtimes:
  node:
    version: "22"

network:
  allowed:
    - defaults
    - node

sandbox:
  agent:
    id: awf
    sudo: false

tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [default]
  bash:
    - "*"

safe-outputs:
  close-issue:
    max: 1

imports:
  - shared/hippo-memory.md
  - shared/otlp.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Hippo Memory — Prune Low-Quality Entries

You are an AI agent performing a focused one-time maintenance pass to remove four
low-quality memories flagged by `hippo audit`. These entries were identified in
[issue #46164](https://github.com/${{ github.repository }}/issues/46164) as
too vague to provide reliable signal during semantic recall.

## Flagged memory IDs

- `mem_650c0682ae4c`
- `mem_b78c884146c7`
- `mem_b168e03a0eca`
- `mem_cd88fb9179d1`

## Step 1 — Inspect each flagged memory

Run `inspect` for each ID so you can see the raw text and decide whether to rewrite
or remove it:

```
mcpscripts hippo --args "inspect mem_650c0682ae4c"
mcpscripts hippo --args "inspect mem_b78c884146c7"
mcpscripts hippo --args "inspect mem_b168e03a0eca"
mcpscripts hippo --args "inspect mem_cd88fb9179d1"
```

## Step 2 — Rewrite or remove each memory

For each memory:

- If the text is a short fragment or commit noise with no standalone meaning,
  remove it permanently:
  ```
  mcpscripts hippo --args "forget <id>"
  ```
- If the text contains a real lesson that is merely phrased too vaguely, replace it
  with a fuller, self-contained statement:
  ```
  mcpscripts hippo --args "remember \"<fuller statement>\" --tag <original-tag>"
  mcpscripts hippo --args "forget <id>"
  ```

## Step 3 — Verify audit is clean

Re-run the quality audit and confirm that none of the four IDs appear in the output:

```
mcpscripts hippo --args "audit --fix"
```

The audit should either produce no warnings for these entries, or confirm they have
been removed.

## Step 4 — Close the tracking issue

Once all four memories have been handled and the audit is clean, close issue #46164
using the `close_issue` safe-output tool:

```json
{
  "close_issue": {
    "issue_number": 46164,
    "comment": "Pruned all four flagged low-quality memories (`mem_650c0682ae4c`, `mem_b78c884146c7`, `mem_b168e03a0eca`, `mem_cd88fb9179d1`). Re-ran `hippo audit --fix` and confirmed no warnings for these entries."
  }
}
```

If any memory could not be found (e.g. already pruned by a concurrent run), note
that in the closing comment and still close the issue.

**Important**: You MUST call a safe-output tool. If the audit already shows all four
memories are gone and no action was needed, call `close_issue` with a note explaining
that.
