# Workflow Execution

This campaign is configured to **actively execute workflows**. Your role is to run the workflows listed in sequence, collect their outputs, and use those outputs to drive the campaign forward.

---

## Workflows to Execute

{{ if .Workflows }}
The following workflows should be executed in order:
{{ range $idx, $workflow := .Workflows }}
{{ add1 $idx }}. `{{ $workflow }}`
{{ end }}
{{ end }}

---

## Execution Process

For each workflow:

1. **Check if workflow exists** - Look for `.github/workflows/<workflow-id>.md` or `.github/workflows/<workflow-id>.lock.yml`

2. **Create workflow if needed** - If the workflow doesn't exist:
   - Use your understanding of the campaign objective to design an appropriate workflow
   - Create the workflow file at `.github/workflows/<workflow-id>.md` with:
     - Appropriate trigger: `workflow_dispatch` (required for execution)
     - Required tools and permissions
     - Safe outputs for any GitHub operations (issues, PRs, comments)
     - Clear prompt describing what the workflow should do
   - Compile it with `gh aw compile <workflow-id>.md`

3. **Execute the workflow** - Use GitHub MCP tools:
   - Trigger: `mcp__github__run_workflow(workflow_id: "<workflow-id>", ref: "main")`
   - Wait for completion: Poll `mcp__github__get_workflow_run(run_id)` until status is "completed"
   - Collect outputs: Check `mcp__github__download_workflow_run_artifact()` for any artifacts

4. **Use outputs for next steps** - Use information from workflow runs to:
   - Inform subsequent workflow executions
   - Update project board items
   - Make decisions about campaign progress

---

## Guidelines

- Execute workflows **sequentially** (one at a time)
- Wait for each workflow to complete before starting the next
- If a workflow fails, note the failure and continue with campaign coordination
- Keep workflow designs simple and focused on the campaign objective
- Workflows you create should be reusable for future campaign runs

---

## After Workflow Execution

Once all workflows have been executed (or created and executed), proceed with the normal orchestrator phases:
- Phase 1: Discovery
- Phase 2: Planning
- Phase 3: Project Updates
- Phase 4: Status Reporting
