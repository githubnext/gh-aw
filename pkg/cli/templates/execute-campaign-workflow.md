# Workflow Execution

This campaign is configured to **actively execute workflows**. Your role is to dispatch the configured workflows (via `workflow_dispatch`) so they can create/advance work items that the orchestrator will discover and track.

**IMPORTANT:** In the current implementation, dispatching a workflow does **not** provide a workflow run ID and the orchestrator should **not** wait/poll for completion. Treat dispatch as "kick off work" and rely on the next orchestrator run's discovery to observe outcomes.

---

## Workflows to Execute

{{ if .Workflows }}
The following workflows should be executed in order:
{{ range $idx, $workflow := .Workflows }}
{{ add1 $idx }}. `{{ $workflow }}`
{{ end }}
{{ end }}

---

## Requirements

Before dispatching any workflow:

1. The workflow must already exist in `.github/workflows/` as either:

   - a compiled agentic workflow (`<id>.lock.yml`), or
   - a standard GitHub Actions workflow (`<id>.yml`).

1. The workflow must support `workflow_dispatch`.

If a workflow is missing or not dispatchable, report it in the campaign status update and continue with the remaining workflows.

---

## Execution Process

For each workflow:

1. **Check availability**

   - The workflow must be present as `.github/workflows/<workflow-id>.lock.yml` or `.github/workflows/<workflow-id>.yml`.
   - If only `.github/workflows/<workflow-id>.md` exists, it must be compiled first (outside this run).

1. **Dispatch the workflow**

   - Use the per-workflow safe-output tool generated for this campaign.
   - The tool name is the workflow ID with dashes normalized to underscores.
     - Example: workflow `dependency-updater` → tool `dependency_updater`
   - Provide inputs only if needed; the tool schema will expose any `workflow_dispatch.inputs`.

1. **Do not wait for completion**

   - Dispatching is fire-and-forget. The orchestrator should continue to discovery + project updates.
   - Outcomes are observed on later runs via the discovery manifest (`./.gh-aw/campaign.discovery.json`).

---

## Guidelines

**Execution order:**

- Dispatch workflows **sequentially** (one dispatch at a time).
- Do **not** wait/poll for completion.
- Keep dispatch volume low (the campaign safe-output max is capped).

**Failure handling:**

- If dispatch fails for a workflow, record it in the status update and continue.
- If a workflow is missing/not compiled/not dispatchable, record it in the status update and continue.

**Permissions and safety:**

- Keep workflow permissions minimal (only what's needed)
- **Why minimal?** - Reduces risk and follows principle of least privilege
- Prefer draft PRs over direct merges for code changes
- **Why drafts?** - Requires human review before merging changes
- Escalate to humans when uncertain about decisions
- **Why escalate?** - Human oversight prevents risky autonomous actions

---

## After Workflow Execution

Once all workflows have been dispatched (or skipped with rationale), proceed with the normal orchestrator steps:

- Step 1: Discovery (read state from manifest and project board)
- Step 2: Planning (determine what needs updating)
- Step 3: Project Updates (write state to project board)
- Step 4: Status Reporting (report progress, failures, and next steps)
