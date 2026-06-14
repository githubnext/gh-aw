---
on:
  schedule: daily
  workflow_dispatch: null
permissions:
  contents: read
  issues: read
  pull-requests: read
  copilot-requests: write
imports:
- uses: shared/daily-audit-base.md
  with:
    expires: 3d
    title-prefix: "[safe-output-integrator] "
- shared/otlp.md
safe-outputs:
  create-pull-request:
    draft: false
    expires: 3d
    labels:
    - safe-outputs
    - testing
    - automation
    title-prefix: "[safe-output-integrator] "
  noop: null
description: Daily workflow that inspects test workflows in pkg/cli/workflows for safe-output coverage, detects missing safe-output types, creates test workflows and Go compilation tests for any missing types, then creates a PR or reports NOOP
emoji: 🔧
engine:
  id: copilot
  copilot-sdk: true
name: Daily Safe Output Integrator
strict: true
timeout-minutes: 20
tools:
  bash:
  - find pkg/cli/workflows -name "test-*.md" -type f
  - ls pkg/cli/workflows/
  - grep -rn "safe-outputs:" pkg/cli/workflows/*.md
  - grep -n "yaml:.*" pkg/workflow/compiler_types.go
  - cat pkg/workflow/compiler_types.go
  - cat pkg/workflow/safe_outputs_validation_config.go
  - cat pkg/workflow/js/safe_outputs_tools.json
  - cat pkg/parser/schemas/main_workflow_schema.json
  - cat pkg/cli/workflows/*.md
  - git status
  - git diff --name-only
  - python3 *
  cli-proxy: true
  edit: null
  github:
    mode: gh-proxy
    toolsets:
    - default
tracker-id: daily-safe-output-integrator
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Safe Output Integrator

Ensure every supported safe-output type has both:
1) a `pkg/cli/workflows/test-*.md` fixture, and
2) coverage in `pkg/workflow/compiler_safe_outputs_config_test.go`.

## Current Context

- **Repository**: ${{ github.repository }}
- **Run Date**: $(date +%Y-%m-%d)
- **Workspace**: ${{ github.workspace }}

## Execution Plan

### Phase 1: Build authoritative type list

- Read `SafeOutputsConfig` in `pkg/workflow/compiler_types.go`.
- Extract YAML keys from `yaml:"<key>,omitempty"` fields.
- Exclude config-only keys: `jobs`, `github-app`, `env`, `github-token`, `allowed-domains`, `allowed-github-references`, `staged`, `threat-detection`.

### Phase 2: Detect workflow fixture gaps

- Scan frontmatter of `pkg/cli/workflows/*.md` and map each type to files that already cover it.
- Use one compact script (saved under `/tmp/gh-aw/agent/`) to emit:
  - covered types (with file examples),
  - missing types requiring new fixtures.
- Do not rely on repeated hardcoded frontmatter examples; generate fixtures from a single template pattern.

### Phase 3: Detect Go compiler-test gaps

- Inspect `pkg/workflow/compiler_safe_outputs_config_test.go` (especially `TestAddHandlerManagerConfigEnvVar`).
- For each missing type, add a case using existing table-test style:
  - correct `SafeOutputsConfig` field,
  - default `Max` from validation config,
  - expected handler key in underscore form.

### Phase 4: Create missing fixture files only

For each uncovered type, create `pkg/cli/workflows/test-copilot-<type>.md` with:

- minimal valid frontmatter (`on.workflow_dispatch`, least-required permissions, `engine: copilot`, `safe-outputs.<type>`),
- one short task that exercises that type,
- JSONL-output instruction naming the corresponding safe-output tool.

Do not modify unrelated existing fixture files and do not create duplicates when another file already covers the type.

### Phase 5: Verify changes

- Confirm `git status` and `git diff --name-only` include only intended files.
- Ensure every new workflow has valid frontmatter and target safe-output key.
- Ensure each new Go test follows existing naming and assertions pattern.

### Phase 6: Final action (mandatory)

- If changes were made, call `create_pull_request` summarizing:
  - new workflow fixtures,
  - new Go test cases,
  - total newly covered types.
- If no changes were needed, call `noop` with a clear “coverage already complete” message.

## Guardrails

- Keep additions minimal and deterministic.
- Use `###`/`####` headings only in generated report text.
- Use `<details>` blocks for long sections.
- This workflow must end with either `create_pull_request` or `noop`.
