---
emoji: "🔍"
description: Scans documentation samples for CLI command examples that vary by agentic engine and verifies they use tabs to present copilot, claude, codex, and pi options
on:
  schedule:
    - cron: "daily around 08:00"
  workflow_dispatch:
max-daily-ai-credits: 5000
permissions:
  contents: read
  issues: read
strict: true
engine:
  id: claude
  model: claude-haiku-4.5
network:
  allowed: [defaults]
imports:
  - shared/otlp.md
tools:
  bash:
    - find docs -name "*.md" -o -name "*.mdx"
    - grep:*
    - cat:*
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  create-issue:
    expires: 3d
    title-prefix: "[doc-cli-tab-scanner] "
    labels: [documentation, automation]
    close-older-issues: true
    max: 1
  noop: null
timeout-minutes: 20
tracker-id: daily-doc-cli-tab-scanner
features:
  gh-aw-detection: true
sandbox:
  agent:
    sudo: false
---

# Daily Documentation CLI Tab Scanner

Scan documentation samples for CLI command examples that vary by agentic engine (copilot, claude, codex, pi) and verify they use `<Tabs>` with `<TabItem>` components to present all relevant engine options.

**Repository**: ${{ github.repository }} | **Run**: ${{ github.run_id }}

## Context

In this repository's documentation, CLI commands often include engine-specific paths or flags — for example, commands with `--engine copilot`, `--engine claude`, `--engine codex`, or `--engine pi`. These engine-specific variants must be presented using the Astro Starlight `<Tabs>` / `<TabItem>` components so that readers can see the command for their chosen engine.

The engines to check for are: **copilot**, **claude**, **codex**, and **pi**.

Documentation files are located under `docs/src/content/docs/`.

## Step 1: Discover Documentation Files

List all markdown and MDX documentation sample files:

```bash
find docs/src/content/docs -name "*.md" -o -name "*.mdx"
```

Focus primarily on files that already contain:
- CLI command code blocks (` ```bash `, ` ```text `, ` ```sh `)
- Mentions of engines: `copilot`, `claude`, `codex`, `pi`
- The `--engine` flag

To narrow down candidates efficiently:

```bash
grep -rl "\-\-engine\|engine copilot\|engine claude\|engine codex\|engine pi" docs/src/content/docs/
```

## Step 2: Identify CLI Command Examples Varying by Engine

For each candidate file, read its contents and identify:

1. **CLI code blocks** — fenced code blocks containing `gh aw` commands or other CLI commands that include engine-specific flags or paths.
2. **Engine-varying paths** — any path, URL, command flag, or configuration value that differs depending on which engine the user has chosen (copilot, claude, codex, pi).

Signs that a CLI example is engine-specific:
- The `--engine <name>` flag appears in a command
- Different secrets are referenced per engine (e.g. `COPILOT_GITHUB_TOKEN`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`)
- Different `engine:` frontmatter values are shown
- The instruction says "if using Claude, run…" or similar inline engine branching

## Step 3: Verify Tab Usage

For each identified engine-varying CLI example, check whether:

### ✅ Correct — tabs are present and complete

The file imports `{ Tabs, TabItem }` from `@astrojs/starlight/components` and wraps the engine-specific commands in a `<Tabs>` block with a `<TabItem>` for each relevant engine. All four engines (copilot, claude, codex, pi) should be represented unless a specific engine genuinely does not support the command.

Example of a correct pattern:
```mdx
import { Tabs, TabItem } from '@astrojs/starlight/components';

<Tabs>
  <TabItem label="Copilot">
    ```bash
    gh aw add-wizard githubnext/agentics/daily-repo-status --engine copilot
    ```
  </TabItem>
  <TabItem label="Claude">
    ```bash
    gh aw add-wizard githubnext/agentics/daily-repo-status --engine claude
    ```
  </TabItem>
  <TabItem label="Codex">
    ```bash
    gh aw add-wizard githubnext/agentics/daily-repo-status --engine codex
    ```
  </TabItem>
  <TabItem label="Pi">
    ```bash
    gh aw add-wizard githubnext/agentics/daily-repo-status --engine pi
    ```
  </TabItem>
</Tabs>
```

### ❌ Incorrect — tabs are missing or incomplete

- An engine-specific CLI command appears in a plain code block without tabs
- Tabs exist but are missing one or more engine options (e.g. only copilot and claude, missing codex and pi)
- Engine-specific prose describes commands inline (e.g. "For Claude, run `gh aw ... --engine claude`") without using the tab component

## Step 4: Report Findings

Compile a list of all issues found. For each issue:

- **File**: the relative path of the documentation file
- **Location**: the section heading or approximate line context
- **Issue type**: missing tabs entirely / incomplete tabs (list missing engines) / inline engine branching in prose without tabs
- **Suggested fix**: brief description of how to apply the correct tab pattern

## Step 5: Create Issue or Noop

**If issues were found**, create one consolidated issue using `create_issue`. The issue must include:

- **Title**: `Documentation CLI Tab Gaps — [Date]`
- **Body**:
  - A brief summary: how many files were scanned, how many gaps were found
  - A table or grouped list of each gap with file, location, issue type, and suggested fix
  - Use `<details><summary>Full findings</summary>...</details>` if there are more than 5 gaps

Use `###` or lower heading levels throughout the issue body.

**If no issues were found**, call `noop` with a message summarizing what was scanned and confirming all CLI tab patterns are compliant.

{{#runtime-import shared/noop-reminder.md}}
