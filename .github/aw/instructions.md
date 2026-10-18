---
description: Repository overrides for workflow authoring, validation, and investigation.
---

# Repository Instructions

Apply to workflow creation, editing, review, diagnosis, debugging, and upgrades.
These rules override conflicting upstream guidance.

## Repository Rules

- After any `.github/workflows/*.md` edit, run one-shot `gh aw compile`; never `--watch`.
- Follow [debug-agentic-workflow.md](debug-agentic-workflow.md) for diagnosis,
  patching, mode selection, and live gates. Logs/audits do not authorize live runs.
- For ruleset-required workflows, keep workflow/job names stable; set
  `inlined-imports: true` when imports exist.
- Explain Copilot authentication accurately: `permissions: { copilot-requests: write }`
  uses `${{ github.token }}` for inference, without a PAT or `COPILOT_GITHUB_TOKEN`.
- Before inventing workflow/shared-component/tool/safe-output patterns, inspect
  `https://github.com/gm3dmo/the-power` through GitHub APIs or `gh` for prior art.
- Graders are reviewed built-ins that compute only over the in-memory unified
  session in conclusion. Never add custom `script`, `run`, or `operational-value`
  graders. Use `/operational-value-designer` to define outcome evidence before
  proposing a new pure built-in.

## Repository-Wide Debugging and Optimization

General "debug/optimize agentic workflows" requests start with failure clustering,
not a request to choose a workflow. Supplied workflow/run/failure scope takes
precedence. Follow [analyzing-runs.md](analyzing-runs.md) for collection, selection,
cached evidence, audits, and uncertainty reporting. Do not dispatch workflows for evidence.
