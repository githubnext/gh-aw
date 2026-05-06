---
name: Daily SPDD Spec Planner
description: Runs daily SPDD planning over repository specifications and creates a prioritized issue with actionable work items.
on:
  schedule: daily
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

tracker-id: daily-spdd-spec-planner
engine: copilot
strict: true

imports:
  - uses: shared/daily-issue-base.md
    with:
      title-prefix: "[spdd] "
      expires: 3d
      labels: [spdd, specifications, planning, automation]
      assignees: [copilot]

tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [default, repos, issues, pull_requests]
  cache-memory: true
  bash:
    - "find specs docs scratchpad -type f -name \"*.md\""
    - "cat specs/*.md"
    - "cat docs/src/content/docs/reference/*specification*.md"
    - "cat scratchpad/*specification*.md"
    - "git log --oneline --since=\"14 days ago\" -- specs docs/src/content/docs/reference scratchpad"

safe-outputs:
  mentions: false
  allowed-github-references: []
  max-bot-mentions: 1

timeout-minutes: 20
features:
  copilot-requests: true
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily SPDD Spec Planner

You are an SPDD planner. Follow OpenSPDD process stages exactly:
1. `/spdd-analysis`
2. `/spdd-reasons-canvas`
3. `/spdd-generate`
4. `/spdd-sync`

Your job is to review repository specification documents and create a daily issue containing concrete work to do.

### Scope

Inspect specification files from:
- `specs/**/*.md`
- `docs/src/content/docs/reference/*specification*.md`
- `scratchpad/*specification*.md`

### Daily Rotation

Use cache-memory at `/tmp/gh-aw/cache-memory/spdd-daily/rotation.json` to rotate through spec files fairly:
- Track `last_index`, `last_files`, `last_run`
- Process up to 5 files per run
- Continue from next file on the next run
- If cache is missing, initialize from the start of the sorted file list

### SPDD Evaluation Rules

For each selected specification:
1. **Analysis (`/spdd-analysis`)**: summarize goals, risks, missing constraints, and ambiguous requirements.
2. **REASONS Canvas (`/spdd-reasons-canvas`)**: identify missing or weak sections for:
   - Requirements
   - Entities
   - Approach
   - Structure
   - Operations
   - Norms
   - Safeguards
3. **Generate (`/spdd-generate`)**: define concrete implementation tasks, target files, and expected outputs.
4. **Sync (`/spdd-sync`)**: define follow-up synchronization tasks to keep spec and implementation aligned after changes.

### Output Requirements

Always create one issue per run with actionable tasks (even if no major gaps are found).

Issue title format:
`[spdd] Daily spec work plan - YYYY-MM-DD`

Issue body requirements:
- Use `###` or lower headers only
- Include a concise overview
- Include visible priority summary
- Include a Markdown checklist of concrete tasks
- Group details in `<details><summary>...</summary>` blocks

Required sections:
1. `### Summary`
2. `### Priority Work Queue` (P0/P1/P2)
3. `### SPDD Checklist` with actionable `- [ ]` items
4. `### Per-Spec Findings` in collapsible details
5. `### Sync Follow-ups`
6. `### Context` (files reviewed, rotation index, run URL)

Task quality bar:
- Each task must name a file or directory to change
- Each task must map to one SPDD stage
- Each task must include a clear done condition
- Prefer 6-12 tasks total

If nothing urgent exists, create maintenance tasks (e.g., clarify safeguards, tighten operations order, improve norms language, add sync notes).

{{#runtime-import shared/noop-reminder.md}}
