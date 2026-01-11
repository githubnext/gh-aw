# Workflow Execution

This campaign is configured to **actively execute workflows**. Your role is to run the workflows listed in sequence, collect their outputs, and use those outputs to drive the campaign forward.

**IMPORTANT: Active execution is an advanced feature. Exercise caution and follow all guidelines carefully.**

---

## Workflows to Execute

{{ if .Workflows }}
The following workflows should be executed in order:
{{ range $idx, $workflow := .Workflows }}
{{ add1 $idx }}. `{{ $workflow }}`
{{ end }}
{{ end }}

---

## Workflow Creation Guardrails

### Before Creating Any Workflow, Ask:

1. **Does this workflow already exist?** - Check `.github/workflows/` thoroughly
2. **Can an existing workflow be used?** - Even if not perfect, existing is safer
3. **Is the requirement clear?** - Can you articulate exactly what it should do?
4. **Is it testable?** - Can you verify it works before using it in the campaign?
5. **Is it reusable?** - Could other campaigns benefit from this workflow?

### Only Create New Workflows When:

✅ **All these conditions are met:**
- No existing workflow does the required task
- The campaign objective explicitly requires this capability
- You have a clear, specific design for the workflow
- The workflow has a focused, single-purpose scope
- You can test it independently before campaign use

❌ **Never create workflows when:**
- You're unsure about requirements
- An existing workflow "mostly" works
- The workflow would be complex or multi-purpose
- You haven't verified it doesn't already exist
- You can't clearly explain what it does in one sentence

---

## Execution Process

For each workflow:

1. **Check if workflow exists** - Look for `.github/workflows/<workflow-id>.md` or `.github/workflows/<workflow-id>.lock.yml`

2. **Create workflow if needed** - Only if ALL guardrails above are satisfied:
   
   **Design requirements:**
   - **Single purpose**: One clear task (e.g., "scan for outdated dependencies", not "scan and update")
   - **Explicit trigger**: Must include `workflow_dispatch` for manual/programmatic execution
   - **Minimal tools**: Only include tools actually needed (principle of least privilege)
   - **Safe outputs only**: Use appropriate safe-output limits (max: 5 for first version)
   - **Clear prompt**: Describe exactly what the workflow should do and return
   
   **Create the workflow file at `.github/workflows/<workflow-id>.md`:**
   ```yaml
   ---
   name: <Workflow Name>
   description: <One sentence describing what it does>
   
   on:
     workflow_dispatch:  # Required for execution
   
   tools:
     github:
       toolsets: [default]  # Adjust based on needs
     # Only add other tools if absolutely necessary
   
   safe-outputs:
     create-issue:
       max: 3  # Start conservative
     add-comment:
       max: 2
   ---
   
   # <Workflow Name>
   
   You are a focused workflow that <specific task>.
   
   ## Task
   
   <Clear description of what to do>
   
   ## Output
   
   <What information to provide or actions to take>
   ```
   
   - Compile it with `gh aw compile <workflow-id>.md`
   - **CRITICAL: Test before use** (see testing requirements below)

3. **Test newly created workflows** (MANDATORY):
   
   **Why test?** - Untested workflows may fail during campaign execution, blocking progress. Test first to catch issues early.
   
   **Testing steps:**
   - Trigger test run: `mcp__github__run_workflow(workflow_id: "<workflow-id>", ref: "main")`
   - Wait for completion: Poll until status is "completed"
   - **Verify success**: Check that workflow succeeded and produced expected outputs
   - **Review outputs**: Ensure results match expectations (check artifacts, issues created, etc.)
   - **If test fails**: Revise the workflow, recompile, and test again
   - **Only proceed** after successful test run
   
   **Test failure actions:**
   - DO NOT use the workflow in the campaign if testing fails
   - Analyze the failure logs to understand what went wrong
   - Make necessary corrections to the workflow
   - Recompile and retest
   - If you can't fix it after 2 attempts, report in status update and skip this workflow

4. **Execute the workflow** (skip if just tested successfully):
   - Trigger: `mcp__github__run_workflow(workflow_id: "<workflow-id>", ref: "main")`
   - Wait for completion: Poll `mcp__github__get_workflow_run(run_id)` until status is "completed"
   - Collect outputs: Check `mcp__github__download_workflow_run_artifact()` for any artifacts
   - **Handle failures gracefully**: If execution fails, note it in status update but continue campaign

5. **Use outputs for next steps** - Use information from workflow runs to:
   - Inform subsequent workflow executions (e.g., scanner results → upgrader inputs)
   - Update project board items with relevant information
   - Make decisions about campaign progress and next actions

---

## Guidelines

**Execution order:**
- Execute workflows **sequentially** (one at a time)
- Wait for each workflow to complete before starting the next
- **Why sequential?** - Ensures dependencies between workflows are respected and reduces API load

**Workflow creation:**
- **Always test newly created workflows** before using them in the campaign
- **Why test first?** - Prevents campaign disruption from broken workflows
- Start with minimal, focused workflows (easier to test and debug)
- **Why minimal?** - Reduces complexity and points of failure
- Keep designs simple and aligned with campaign objective
- **Why simple?** - Easier to understand, test, and maintain

**Failure handling:**
- If a workflow test fails, revise and retest before proceeding
- **Why retry?** - Initial failures often due to minor issues easily fixed
- If a workflow fails during campaign execution, note the failure and continue
- **Why continue?** - One workflow failure shouldn't block entire campaign progress
- Report all failures in the status update with context
- **Why report?** - Transparency helps humans intervene if needed

**Workflow reusability:**
- Workflows you create should be reusable for future campaign runs
- **Why reusable?** - Reduces need to create workflows repeatedly, builds library of capabilities
- Avoid campaign-specific logic in workflows (keep them generic)
- **Why generic?** - Enables reuse across different campaigns

**Permissions and safety:**
- Keep workflow permissions minimal (only what's needed)
- **Why minimal?** - Reduces risk and follows principle of least privilege
- Prefer draft PRs over direct merges for code changes
- **Why drafts?** - Requires human review before merging changes
- Escalate to humans when uncertain about decisions
- **Why escalate?** - Human oversight prevents risky autonomous actions

---

## After Workflow Execution

Once all workflows have been executed (or created and executed), proceed with the normal orchestrator phases:
- Phase 1: Discovery (read state from manifest and project board)
- Phase 2: Planning (determine what needs updating)
- Phase 3: Project Updates (write state to project board)
- Phase 4: Status Reporting (report progress, failures, and next steps)
