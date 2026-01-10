# Workflow Execution Instructions

This campaign is configured for **active workflow execution**. As the orchestrator, you actively run workflows,
wait for their completion, and use their outputs to drive work forward and make decisions about next steps.

---

## Execution Architecture

You have access to GitHub MCP tools for workflow execution:
- `mcp__github__run_workflow` - Trigger a workflow via workflow_dispatch
- `mcp__github__get_workflow_run` - Check workflow run status and conclusion
- `mcp__github__list_workflow_runs` - List recent workflow runs
- `mcp__github__download_workflow_run_artifact` - Download artifacts containing outputs
- `mcp__github__get_job_logs` - Get job logs to extract outputs

Use these tools to execute the configured workflow sequence, collect outputs, and make decisions.

---

## Execution Sequence

{{ if .ExecutionSequence }}
Execute the following workflows in order:

{{ range $idx, $step := .ExecutionSequence }}
### Step {{ add1 $idx }}: {{ $step.Workflow }}

**Workflow:** `{{ $step.Workflow }}`

{{ if $step.Condition }}
**Condition:** Only run if: `{{ $step.Condition }}`
{{ end }}

{{ if $step.Inputs }}
**Inputs:**
{{ range $key, $value := $step.Inputs }}
- `{{ $key }}`: {{ $value }}
{{ end }}
{{ end }}

{{ if $step.Outputs }}
**Outputs to Collect:**
{{ range $output := $step.Outputs }}
- `{{ $output.Name }}` from `{{ $output.From }}`{{ if $output.Required }} (REQUIRED){{ end }}
{{ end }}
{{ end }}

**On Failure:** {{ if $step.ContinueOnFailure }}Continue to next step{{ else }}Stop execution{{ end }}

{{ end }}
{{ end }}

---

## Execution Process

Follow this process for each workflow execution step:

### 1. Check Condition (if specified)
- Evaluate the condition expression using available outputs from previous steps
- If condition is false, skip this step and move to the next
- If no condition is specified, always execute

### 2. Prepare Inputs
- Resolve any expressions in inputs (e.g., `${{ outputs.previous_result }}`)
- Convert all input values to strings as required by workflow_dispatch
- Validate that all required inputs are provided

### 3. Trigger Workflow
Use `mcp__github__run_workflow` to trigger the workflow:
```
mcp__github__run_workflow(
  workflow_id: "{{ .WorkflowID }}",
  ref: "main",  # or appropriate branch
  inputs: { /* resolved inputs */ }
)
```

**Note:** The tool may return immediately without a run ID. If so, you'll need to list recent runs to find it.

### 4. Find Workflow Run ID
After triggering, identify the workflow run:
```
mcp__github__list_workflow_runs(
  workflow_id: "{{ .WorkflowID }}",
  per_page: 5
)
```
- Look for the most recent run with status "queued" or "in_progress"
- Match by triggered time (should be within last few seconds)
- Store the run ID for subsequent operations

### 5. Wait for Completion
Poll the workflow run status until it completes:
```
mcp__github__get_workflow_run(
  run_id: <RUN_ID>
)
```

**Polling Strategy:**
- Initial wait: 30 seconds
- Poll interval: 30-60 seconds
- Maximum wait time: {{ if .TimeoutMinutes }}{{ .TimeoutMinutes }}{{ else }}60{{ end }} minutes
- Acceptable status values: "queued", "in_progress", "completed"
- When status is "completed", check the conclusion: "success", "failure", "cancelled", "timed_out"

**Important:** Be patient! Workflows can take several minutes to complete. Don't give up too early.

### 6. Collect Outputs
Based on the output configuration, collect data from the completed workflow run:

**For `artifact:filename.json` format:**
```
mcp__github__download_workflow_run_artifact(
  run_id: <RUN_ID>,
  artifact_name: "filename"  # without .json extension
)
```
- The tool returns the artifact content
- Parse as JSON if needed
- Extract specific fields using JSONPath if specified (e.g., `artifact:results.json:$.services`)

**For `artifact:filename.json:$.path` format:**
- Download the artifact
- Parse the JSON content
- Navigate to the specified JSONPath (e.g., `$.path.to.field`)
- Extract the value at that path

**For `logs:pattern` format:**
```
mcp__github__get_job_logs(
  run_id: <RUN_ID>,
  failed_only: false
)
```
- Get logs from all jobs
- Search for the specified pattern using regex
- Extract matching values

**For `conclusion` format:**
- Simply store the workflow run conclusion value
- Values: "success", "failure", "cancelled", "timed_out", "action_required", "neutral", "skipped"

### 7. Store Outputs
- Save collected outputs in memory for use in subsequent steps
- Outputs can be referenced in later steps as `${{ outputs.name }}`
- If output is marked as `required` and not found, fail the step

### 8. Handle Failures
If the workflow fails (conclusion != "success"):
- Log the failure details (conclusion, run URL, error messages from logs)
- If `continue-on-failure` is true, proceed to next step
- If `continue-on-failure` is false, stop execution and report

---

## Parallel Execution

{{ if and .MaxConcurrentWorkflows (gt .MaxConcurrentWorkflows 1) }}
This campaign allows up to {{ .MaxConcurrentWorkflows }} workflows to run concurrently.

**Parallel Execution Strategy:**
- Group steps that don't depend on each other's outputs
- Trigger multiple workflows simultaneously
- Track all run IDs
- Wait for all to complete before moving to dependent steps
- Collect outputs from all completed runs
{{ else }}
This campaign uses **sequential execution** (one workflow at a time).
{{ end }}

---

## Error Handling

### Workflow Trigger Failures
- If `mcp__github__run_workflow` fails, retry once after 10 seconds
- If still failing, log the error and skip to next step (if continue-on-failure is true)

### Timeout Handling  
- If workflow doesn't complete within timeout, mark as timed out
- Log the run URL for manual inspection
- If continue-on-failure is false, stop execution

### Output Collection Failures
- If required output is missing, fail the step
- If optional output is missing, log warning and continue
- If artifact download fails, retry once

---

## Reporting

After workflow execution completes (success or failure), include in your status update:

**Workflow Execution Summary:**
- Which workflows were executed
- Input parameters provided
- Outputs collected
- Success/failure status
- Execution time for each workflow
- Any errors or warnings

**Example:**
```markdown
## Workflow Execution Results

**Step 1: framework-scanner**
- Status: ✓ Success
- Execution time: 3m 45s
- Outputs collected: services_to_upgrade (12 services), scan_report_url
- Next action: Proceeding to framework-upgrader with 12 services

**Step 2: framework-upgrader**
- Status: ✓ Success  
- Execution time: 8m 12s
- Inputs: 12 services (batch_size: 5)
- Outputs collected: upgrade_results (10 success, 2 failed)
- Issues: 2 services failed upgrade (logged to project board)
```

---

## Best Practices

1. **Always check workflow status** - Don't assume success, verify with get_workflow_run
2. **Be patient** - Workflows can take minutes; use appropriate wait times
3. **Handle errors gracefully** - Log details for human review
4. **Validate outputs** - Ensure collected data is in expected format
5. **Report transparently** - Document all execution steps in status update
6. **Respect timeouts** - Don't wait indefinitely; respect configured timeout
7. **Clean up on failure** - Document failed runs for human investigation

---

## Authority

Workflow execution happens **before** Phase 1 (Discovery). The execution sequence is:

1. **Phase 0**: Workflow Execution (this section) - Active execution of configured workflows
2. **Phase 1**: Read State (Discovery) - Discover worker outputs
3. **Phase 2**: Make Decisions (Planning) - Plan project updates
4. **Phase 3**: Write State (Execution) - Update project board
5. **Phase 4**: Report & Status Update - Summarize campaign progress

Workflow outputs collected in Phase 0 can influence decisions in later phases.
