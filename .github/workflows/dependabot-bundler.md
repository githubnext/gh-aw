---
name: Dependabot Bundler
description: Bundles Dependabot security alert updates per package.json into a single PR
on:
  schedule: every 6h
  workflow_dispatch:
  skip-if-match: 'is:pr is:open in:title "[dependabot-bundle]"'
permissions:
  contents: read
  pull-requests: read
  security-events: read
engine: copilot
tools:
  github:
    toolsets: [context, repos, dependabot, pull_requests]
  repo-memory:
    - id: campaigns
      branch-name: memory/campaigns
      file-glob: [security-alert-burndown/**]
      campaign-id: security-alert-burndown
  cache-memory:
  edit:
  bash:
safe-outputs:
  add-labels:
    allowed:
      - agentic-campaign
      - z_campaign_security-alert-burndown
  create-pull-request:
    title-prefix: "[dependabot-bundle] "
    labels: [security, dependencies, dependabot, automated-fix, agentic-campaign, z_campaign_security-alert-burndown]
    reviewers: [copilot]
timeout-minutes: 25
---

# Dependabot Bundler Agent

You bundle *multiple* Dependabot security updates that belong to the **same manifest** (same `package.json`) into **one pull request**.

## Campaign Context

This workflow is part of the **Security Alert Burndown Campaign**, which expects to find and address **21 total security findings** across the repository:
- **1 Dependabot alert** (this workflow addresses this)
- **3 Code scanning alerts** (handled by code-scanning-fixer and security-fix-pr workflows)
- **17 Secret scanning alerts** (handled by secret-scanning-triage workflow)

Your focus is on the **1 Dependabot alert**. Bundle all vulnerable packages for a single manifest into one PR per run.

## Ground rules

- Always operate on `owner="githubnext"` and `repo="gh-aw"`.
- Only target **npm** ecosystem manifests (`package.json`).
- Only one PR per run.
- If you cannot produce a clean update safely, exit with a clear explanation (do not guess).

## Goal

1. List open Dependabot alerts.
2. Group by manifest (`dependency.manifest_path` or similar manifest path field).
3. Pick exactly one manifest path per run (round-robin using cache-memory).
4. Update all vulnerable packages for that manifest in one branch.
5. Create a PR with a concise, high-signal summary and links to the relevant alerts.

## Step-by-step

### 0) Load state (cache-memory)

Use `/tmp/gh-aw/cache-memory/dependabot-bundler.json` to persist a cursor.

- If the file exists, parse JSON: `{ "last_manifest": "path/to/package.json" }`.
- If it does not exist, treat it as empty.

### 1) List open Dependabot alerts

Use the GitHub MCP Dependabot toolset.

- Call `github-list_dependabot_alerts` (or the closest available list tool in the `dependabot` toolset) for `owner="githubnext"` and `repo="gh-aw"`.
- Filter to `state="open"`.

From results, collect only alerts where:
- ecosystem is npm, and
- manifest path ends with `package.json`, and
- a patched version exists (e.g. `security_vulnerability.first_patched_version.identifier` or equivalent).

If there are no qualifying alerts, log and exit.

### 2) Group alerts by manifest

Group alerts by the manifest path field.

- Build a stable sorted list of unique manifest paths.
- Select the next manifest path after `last_manifest` (wrap around).

Persist the chosen manifest path back to cache-memory after successful PR creation.

### 3) Apply updates for the selected manifest

Let `manifestPath` be the selected `package.json` path.

- Determine `dir = dirname(manifestPath)`.
- Detect package manager in `dir`:
  - If `pnpm-lock.yaml` exists: use `corepack enable` then `pnpm`.
  - Else if `yarn.lock` exists: use `corepack enable` then `yarn`.
  - Else: use `npm`.

For each alert in this manifest:
- Extract the vulnerable package name and the preferred patched version.
- Apply the minimal update to reach a patched version.
  - npm: `npm install <name>@<patchedVersion>`
  - pnpm: `pnpm add <name>@<patchedVersion>`
  - yarn: `yarn add <name>@<patchedVersion>`

Then run install to ensure lockfile is consistent:
- npm: `npm install`
- pnpm: `pnpm install`
- yarn: `yarn install`

If any command fails, do not create a PR.

### 4) Create the PR

Create a PR (safe output `create_pull_request`) that includes:
- The manifest path you updated
- A bullet list of packages bumped (old → new)
- Links to the Dependabot alerts handled (URLs)
- Notes about any alerts that could not be fixed (and why)

Only emit one `create_pull_request`.

### 5) Record cursor

After the PR is successfully created, write `/tmp/gh-aw/cache-memory/dependabot-bundler.json` with:

```json
{ "last_manifest": "<manifestPath>" }
```
