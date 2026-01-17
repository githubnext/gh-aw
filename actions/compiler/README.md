# Agentic Workflow Compiler

Compile `.github/workflows/*.md` into deterministic `.lock.yml` files and fail CI if lockfiles would change.

## What it does

- Runs `gh-aw compile --dir <workflows_dir> --check`
- Fails the step when it detects lockfile drift

## Quickstart

Use as a PR-required check (recommended):

```yaml
name: Compile agentic workflows

on:
  pull_request:

permissions:
  contents: read

jobs:
  compile:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@<PINNED_SHA>
      - uses: githubnext/gh-aw/actions/compiler@v1
        with:
          # Pin this to a tag like v1.2.3 or a full commit SHA.
          gh_aw_ref: v1.2.3
          workflows_dir: .github/workflows
```

## Inputs

|Input|Required|Default|Description|
|---|---|---|---|
|`gh_aw_ref`|yes|-|Ref used for `go install github.com/githubnext/gh-aw/cmd/gh-aw@<ref>` (pin this).|
|`workflows_dir`|no|`.github/workflows`|Workflows directory to compile.|
|`extra_args`|no|-|Extra `gh-aw compile` args, newline-delimited (one token per line).|

Example `extra_args`:

```yaml
      - uses: githubnext/gh-aw/actions/compiler@v1
        with:
          gh_aw_ref: v1.2.3
          extra_args: |
            --verbose
            --engine=copilot
```

## Enterprise / org install

- **Org-wide rollout:** put a reusable workflow in a central repo (often the org’s `.github` repo), then require it via rulesets/branch protection.
- **Permissions:** workflow/job `permissions:` are set in the workflow YAML (or org defaults). A composite action cannot raise or lower permissions by itself.
- **Restricted runners:** if your runners can’t `go install` from the public internet, vendor this Action into your org and replace the install step to fetch an internal `gh-aw` binary, or use an internal Go module proxy/mirror.

## Security

- Intended for read-only permissions (example above uses `contents: read`).
- No secrets required.
- Does not push commits.

For a ready-to-use workflow file, see [templates/compiler-pr-check.yml](../../templates/compiler-pr-check.yml).
