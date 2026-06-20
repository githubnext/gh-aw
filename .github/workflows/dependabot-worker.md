---
private: true
emoji: "🔧"
name: Dependabot Worker
description: Reusable worker that bundles open Dependabot PRs for generated workflow manifests by editing source workflow markdown and recompiling once
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
      objective:
        description: Shared campaign objective
        type: string
        required: true
      pr-numbers:
        description: Comma-separated selected Dependabot pull request numbers for the bundled batch
        type: string
        required: true
      dependency-batch-json:
        description: JSON array describing the selected Dependabot PR batch
        type: string
        required: false
        default: "[]"
  workflow_dispatch:
    inputs:
      objective:
        description: Shared campaign objective
        type: string
        required: false
        default: Close open Dependabot PRs for generated workflow manifests by updating source workflow markdown and recompiling.
      pr-numbers:
        description: Comma-separated selected Dependabot pull request numbers for the bundled batch
        type: string
        required: false
        default: "0"
      dependency-batch-json:
        description: JSON array describing the selected Dependabot PR batch
        type: string
        required: false
        default: "[]"
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: pi
  model: copilot/gpt-5.4
strict: true
network:
  allowed:
    - defaults
    - node
    - python
    - go
imports:
  - uses: shared/daily-pr-base.md
    with:
      title-prefix: "[dependabot-campaign] "
      expires: "3d"
      labels: [automation, dependencies, dependabot]
      reviewers: [copilot]
  - shared/otlp.md
tools:
  edit:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [default]
  bash:
    - "make dependabot && make build"
    - "make build"
    - "make dependabot"
    - "./gh-aw compile --dependabot"
    - "cd .github/workflows && npm install --package-lock-only"
    - "git status"
    - "git diff *"
    - "cat *"
    - "rg *"
timeout-minutes: 30
---

# Dependabot Worker

You are the executor for one bundled Dependabot campaign wave.

## Goal

Take the selected batch of open Dependabot PRs that touch generated workflow manifests and resolve them the repo-native way: update the source workflow markdown or shared workflow config, regenerate the manifests once, and prepare a single replacement PR if the fix is safe and bounded.

## Context

- Objective: `${{ inputs.objective }}`
- Dependabot PR numbers: `${{ inputs.pr-numbers }}`
- Dependency batch payload: `${{ inputs.dependency-batch-json }}`

## Deterministic worker result

You must always write one result JSON file for this wave, even if the work is blocked or no change is applied.

Write the result file to:

`/tmp/gh-aw/cache-memory/dependabot-worker/results/`

Use a filesystem-safe filename such as:

`${{ github.run_id }}-result.json`

The JSON must include:

- `pr_numbers`
- `dependencies_processed`: array of dependency summaries from the selected batch
- `source_files_updated`: array of workflow markdown or shared files you changed
- `fix_applied`: boolean
- `replacement_pr_created`: boolean
- `status`: `improved`, `unchanged`, or `blocked`
- `validation_commands`: array of commands you ran
- `notes`: concise explanation of what happened

Mark `status` as:

- `improved` when you safely updated source files and regenerated the manifests
- `unchanged` when no matching source change was needed or possible but nothing was wrong locally
- `blocked` when the PR requires risky changes, cannot be traced back to source workflow markdown, or validation fails

## Required approach

1. Inspect the selected Dependabot PR using GitHub tools and confirm it is authored by `dependabot[bot]` or `app/dependabot`.
2. Confirm every selected PR touches only compiler-generated workflow manifests such as `.github/workflows/package.json`, `.github/workflows/package-lock.json`, `.github/workflows/requirements.txt`, or `.github/workflows/go.mod`.
3. Treat `dependency-batch-json` as the JSON payload describing the dependency batch and use it to enumerate the selected dependencies.
4. For each selected dependency, find the source workflow markdown or shared config files that reference the outdated dependency.
5. Apply all safe version updates to source `.md` files in one pass. Do not edit the generated manifest files directly.
6. Regenerate the manifests once with `make dependabot` or `./gh-aw compile --dependabot`.
7. If `.github/workflows/package-lock.json` needs refresh after compilation, run `npm install --package-lock-only` from `.github/workflows`.
8. Keep the change bounded to the selected dependency updates plus the smallest number of related source files needed.

## Required validation

After your first substantial edit, immediately run:

```bash
make dependabot && make build
```

If the generated npm manifest changed, also run:

```bash
cd .github/workflows && npm install --package-lock-only
```

If validation fails, fix only the touched slice and rerun the same focused validation.

## Pull request rule

Create a PR only if:

- the fix is real and bounded
- validation passed
- `git diff --stat` shows an actual code change
- the result JSON would report `status: improved`

The PR body must include:

- original Dependabot PR numbers
- dependency names and version changes
- objective
- which source workflow files were updated
- which manifest files were regenerated
- validation commands you ran

Do not directly merge or modify the generated manifest PR itself.

If no safe bounded remediation is possible, do not create a PR. End with a concise blocker report and still write the worker result JSON.

## Output

End with a concise summary including the selected PR numbers, dependency batch handled, source files updated, validation commands run, result file path, and whether a replacement PR was created.

{{#runtime-import shared/noop-reminder.md}}