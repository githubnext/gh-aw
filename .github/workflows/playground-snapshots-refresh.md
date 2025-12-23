---
name: Refresh playground snapshots
description: Regenerates docs playground snapshots and adds AI-written job summaries
on:
  workflow_dispatch:
  schedule:
    - cron: '0 8 * * 1' # Weekly on Mondays at 08:00 UTC

permissions:
  contents: read
  pull-requests: read
  issues: read

tools:
  edit:

safe-outputs:
  create-pull-request:
    title-prefix: "[docs] "
    labels: [documentation]

timeout-minutes: 15

steps:
  - name: Checkout
    uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd # v5
    with:
      persist-credentials: false

  - name: Setup Node.js
    uses: actions/setup-node@395ad3262231945c25e8478fd5baf05154b1d79f # v6
    with:
      node-version: '20'

  - name: Regenerate playground snapshots
    env:
      PLAYGROUND_SNAPSHOTS_TOKEN: ${{ secrets.PLAYGROUND_SNAPSHOTS_TOKEN }}
      PLAYGROUND_SNAPSHOTS_MODE: actions
      PLAYGROUND_SNAPSHOTS_REPO: ${{ secrets.PLAYGROUND_SNAPSHOTS_REPO }}
      PLAYGROUND_SNAPSHOTS_WORKFLOW_IDS: ${{ secrets.PLAYGROUND_SNAPSHOTS_WORKFLOW_IDS || 'project-board-draft-updater,project-board-issue-updater' }}
      PLAYGROUND_SNAPSHOTS_INCLUDE_LOGS: '1'
    run: |
      set -euo pipefail
      cd docs
      node scripts/fetch-playground-snapshots.mjs
---

# Playground snapshots refresh

You are updating the documentation playground snapshots in this repository.

## Task

1. Ensure the snapshots are regenerated.
2. For each JSON file in `docs/src/assets/playground-snapshots/*.json`, add or update a `summary` field on every job entry.
   - `jobs[].summary` must be a short, plain-text description (1–2 sentences) of what the job did.
   - Base your summary on the job name, step names, and the most informative log group titles and/or log lines.
   - Keep it factual and specific; avoid fluff.
   - Do not add markdown, headings, or bullet lists.
3. Do not change anything else besides adding/updating `jobs[].summary` values.

## Notes

- These snapshots are intentionally size-limited; keep summaries compact.
- If a job is just scaffolding (e.g. `activation`), say so succinctly.
