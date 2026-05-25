---
emoji: "🛫"
name: Gemini to Antigravity Migration
description: Perform a breaking migration from Gemini CLI to Antigravity CLI with codemods, strict legacy rejection, validation, and PR creation
on:
  roles:
    - admin
    - maintainer
  workflow_dispatch:
    inputs:
      target_ref:
        description: Branch or ref to migrate
        required: false
        default: main
        type: string
      dry_run:
        description: Research and report only without editing files or opening a PR
        required: false
        default: false
        type: boolean
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: gemini-to-antigravity
engine:
  id: copilot
strict: true
timeout-minutes: 45
network:
  allowed:
    - defaults
    - github
    - go
    - node
    - python
    - ai.google.dev
    - developers.google.com
    - developers.googleblog.com
tools:
  cli-proxy: true
  cache-memory: true
  github:
    mode: gh-proxy
    toolsets: [default]
  bash:
    - "*"
  edit:
  web-fetch:
steps:
  - name: Setup Go
    uses: actions/setup-go@v6.4.0
    with:
      go-version-file: go.mod
      cache: true
  - name: Setup Node.js
    uses: actions/setup-node@v6.4.0
    with:
      node-version: "24"
      cache: npm
      cache-dependency-path: actions/setup/js/package-lock.json
  - name: Install JavaScript dependencies
    run: npm ci
    working-directory: ./actions/setup/js
  - name: Install development dependencies
    run: make deps-dev
safe-outputs:
  create-pull-request:
    expires: 2d
    title-prefix: "[migration] "
    labels: [automation, breaking-change, migration]
    draft: false
    protected-files: allowed
  noop:
---

# Gemini to Antigravity Migration

You are the migration agent for a **breaking** engine replacement in `${{ github.repository }}`.

## Objective

Replace the existing Gemini CLI engine with a new Antigravity CLI engine and prepare a single reviewable pull request that completes the repo migration.

This is intentionally a breaking migration:

- remove legacy Gemini CLI execution
- rename the public engine from `gemini` to `antigravity`
- require a new Antigravity credential
- provide codemods for repo users and this source tree
- fail fast on unmigrated Gemini usage

If `${{ github.event.inputs.dry_run }}` is `true`, do not edit files and do not create a pull request. Produce a concise migration report and call `noop`.

## Non-Negotiable Rules

- Do **not** preserve Gemini compatibility.
- Do **not** keep Gemini fallback branches.
- Do **not** silently map `GEMINI_API_KEY` to `ANTIGRAVITY_API_KEY`.
- Do **not** allow user-global Antigravity config, plugins, hooks, skills, MCP servers, or subagents to bypass gh-aw policy.
- Do **not** emit credentials into logs, comments, artifacts, or generated files.
- Keep changes bounded to this migration. Do not fix unrelated failures.

## Phase 0 — Research and Inventory

1. Read the current Antigravity CLI documentation using `web-fetch`.
2. Confirm and record:
   - the non-interactive CI invocation for `agy`
   - authentication requirements
   - supported model names or aliases
   - configuration paths for plugins, hooks, skills, MCP, and subagents
3. Inventory every active Gemini reference in the repository.
4. Classify each reference as one of:
   - engine API
   - CLI invocation
   - credential
   - docs
   - tests
   - examples
   - generated output
   - historical changelog

Treat historical references as allowed only when they are clearly historical and do not instruct users to keep using Gemini.

## Phase 1 — Define the Breaking Target

Implement the new target model in source code first:

- add `antigravity` as the canonical engine name
- remove `gemini` from supported runtime engine validation
- remove Gemini CLI installer and invocation logic
- use `agy` as the CLI executable
- rename Gemini-specific engine files, symbols, tests, fixtures, and docs where appropriate
- fail compilation when workflow frontmatter still contains `engine: gemini`

Add this migration diagnostic and use it consistently in strict validation:

`engine: gemini is no longer supported. Run the Antigravity migration codemod and configure ANTIGRAVITY_API_KEY.`

## Phase 2 — Credentials and Runtime Policy

Require the new Antigravity credential:

- introduce `ANTIGRAVITY_API_KEY` unless implementation research proves a better official secret name
- validate the new credential before invoking `agy`
- reject `GEMINI_API_KEY` as insufficient
- emit a clear diagnostic when only `GEMINI_API_KEY` is present

Ensure runtime configuration is deterministic in CI:

- do not load user-global config by default
- keep gh-aw as the source of truth for tool access and policy
- preserve safe-output, allowed-files, protected-files, and network-policy behavior

## Phase 3 — Codemods

Provide codemods for repository users and for this repository's own source tree.

Codemods must support:

- dry-run mode
- write mode
- file list output
- diff output
- failure summary
- manual migration annotations for ambiguous model mappings

Update all relevant tracked text files:

- workflow frontmatter
- example workflows
- tests and fixtures
- documentation
- shell scripts
- generated references
- environment variable names
- engine aliases
- error messages
- installation references

Required rewrite rules:

- `engine: gemini` → `engine: antigravity`
- `GEMINI_API_KEY` → `ANTIGRAVITY_API_KEY`
- Gemini CLI command references → `agy`
- Gemini install/version checks → Antigravity install/version checks
- Gemini docs/setup text → Antigravity docs/setup text

Mark any model alias that cannot be mapped with confidence as a manual migration item instead of guessing.

## Phase 4 — Tests, Validation, and Generated Files

After the first substantial edit, run:

```bash
make build && make fmt
```

Before finishing, run the existing repo validation commands needed for the touched areas:

```bash
make build
make lint
make test-unit
make recompile
```

Also run any focused existing tests for touched JavaScript or integration surfaces when the migration changes them. Add and update tests so the suite proves:

- `engine: antigravity` is supported
- `engine: gemini` fails with the migration diagnostic
- `GEMINI_API_KEY` alone is rejected
- `ANTIGRAVITY_API_KEY` is accepted
- `agy` discovery and invocation work

Do not hand-edit generated references when an existing repo command can regenerate them.

## Phase 5 — Pull Request Requirements

Create exactly one PR when the migration is implemented and validation passes.

The change set must include:

- the source implementation changes
- codemods
- updated docs/examples/tests/fixtures/scripts/generated references
- a breaking changeset in `.changeset/` with migration guidance

The PR summary must clearly call out:

- the engine rename from `gemini` to `antigravity`
- the new `ANTIGRAVITY_API_KEY` requirement
- the removal of Gemini compatibility
- the codemods added
- any manual migration follow-ups for ambiguous model mappings
- the validation commands you ran

## Exit Rules

- If the work is blocked by missing Antigravity CLI facts or unsafe ambiguous mappings, stop with a concise blocker summary and call `noop`.
- If `${{ github.event.inputs.dry_run }}` is `true`, stop after the research and migration report and call `noop`.
- Otherwise, do not end without either creating the PR or calling `noop` with a clear reason.
