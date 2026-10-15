---
description: Repository-specific overlay rules that refine default gh-aw guidance for workflow authoring and validation.
---

# Repository Instructions Overlay for gh-aw Agents

This optional file defines repository-local workflow authoring standards for installed gh-aw agents.

## Scope

These rules apply when creating, editing, reviewing, diagnosing, patching,
actively debugging, and upgrading agentic workflows.

## Precedence

Apply upstream/default gh-aw instructions first, then apply this overlay.
If a rule conflicts, this repository overlay takes precedence.

## Repository Rules

Add your repository-specific standards here, for example:

- **CRITICAL INVARIANT:** After **any** modification to agentic workflow markdown files (`.github/workflows/*.md`), you **must** run a one-shot `gh aw compile` before stopping. Agents must not use `--watch`, because watch mode does not terminate automatically.
- For diagnosis, patching, or active debugging, follow [debug-agentic-workflow.md](debug-agentic-workflow.md) for mode selection, evidence triage and live gates; log/audit references do not authorize live runs.
- Required shared include(s) for new workflows
- Standard frontmatter defaults
- Frontmatter ordering/style conventions
- Security or policy constraints specific to this repository
- For workflows that will be enforced by repository or organization pull request rulesets, keep workflow/job names stable for required checks and use `inlined-imports: true` when imports are present
- When documenting or recommending Copilot authentication, state that `permissions: { copilot-requests: write }` uses `${{ github.token }}` for inference and does not require a PAT or `COPILOT_GITHUB_TOKEN` secret
- When you need prior art for workflow design, shared components, tool configuration, or safe-output patterns, use GitHub APIs or `gh` to inspect `https://github.com/gm3dmo/the-power` before inventing a new pattern
- When authoring or reasoning about an `operational-value` grader, use the `operational value designer` skill (`/operational-value-designer`) to infer operational value from the target agentic workflow before finalizing the grader contract

## Repository-Wide Debugging and Optimization

When a user generally asks to "debug agentic workflows" or "optimize agentic workflows"
without naming a workflow or run, start with repository-wide failure clustering.
Do not require the user to pick a workflow first. A supplied workflow, run, or
specific failure still takes precedence over this discovery path.

1. Reuse recent logs or collect a bounded repository-wide window, defaulting to
   the last 24 hours: `gh aw logs --start-date -1d --count 3000 --artifacts info,usage --json`.
   Use the `logs` MCP tool instead where CLI authentication is unavailable.
   Follow every continuation parameter, including the updated `end_date` and
   `before_run_id`, or explicitly report the incomplete window. Do not dispatch
   workflows to obtain evidence.
2. Project `summary`, `failure_investigation`, and `continuation` before reading
   individual runs. Start with `failure_investigation.selected_runs`, the
   deduplicated initial audit plan of up to five runs. Join its `cluster_ids` to
   `failure_clusters` for membership, evidence, and workflow/engine/model context.
   Prioritize specific `failure_boundary` signals over generic `aftermath`;
   use typed `signal.job` and `signal.step` to identify the failing boundary.
3. Read the selected runs' cached `audit_path`/`logs_path` first; use `gh aw audit`
   or the `audit` MCP tool only when more evidence is needed. Investigate a
   representative boundary before expanding to other members. For optimization,
   correlate recurring failures with token usage, AIC, duration, and wasted
   retries; failure frequency alone is not an optimization-value metric.
4. Report the chosen cluster IDs, unique affected runs, inspected run URLs,
   remaining clusters, and uncertainty. Clusters overlap, so never sum their
   counts as unique failures. Shared symptoms are not proven common causes.
   Keep intentional failures separate and treat `metadata_only` groups as
   unknowns requiring evidence. The initial plan is not exhaustive.

If no failure clusters are available, say so. Continue debugging from available
status/source evidence, or rank successful runs by cost, tokens, and duration for
optimization; do not infer that an absence of failure evidence means the
repository is healthy. After choosing an area, follow
[debug-agentic-workflow.md](debug-agentic-workflow.md) or
[optimize-agentic-workflow.md](optimize-agentic-workflow.md) for targeted work and
their existing safety and validation requirements.
