# Workflow Execution

This campaign references the following campaign workers. These workers follow the first-class worker pattern: they are dispatch-only workflows with standardized input contracts.

**IMPORTANT: Workers are orchestrated, not autonomous. They accept `campaign_id` and `payload` inputs via workflow_dispatch.**

---

## Campaign Workers

{{ if .Workflows }}
The following campaign workers are referenced by this campaign:
{{ range $idx, $workflow := .Workflows }}
{{ add1 $idx }}. `{{ $workflow }}`
{{ end }}
{{ end }}

**Worker Pattern**: All workers MUST:
- Use `workflow_dispatch` as the ONLY trigger (no schedule/push/pull_request)
- Accept `campaign_id` (string) and `payload` (string; JSON) inputs
- Implement idempotency via deterministic work item keys
- Label all created items with `z_campaign_{{ .CampaignID }}`

---

## Workflow Creation Guardrails

### Before Creating Any Worker Workflow, Ask:

1. **Does this workflow already exist?** - Check `.github/workflows/` thoroughly
2. **Can an existing workflow be adapted?** - Even if not perfect, existing is safer
3. **Is the requirement clear?** - Can you articulate exactly what it should do?
4. **Is it testable?** - Can you verify it works with test inputs?
5. **Is it reusable?** - Could other campaigns benefit from this worker?

### Only Create New Workers When:

✅ **All these conditions are met:**
- No existing workflow does the required task
- The campaign objective explicitly requires this capability
- You have a clear, specific design for the worker
- The worker has a focused, single-purpose scope
- You can test it independently before campaign use

❌ **Never create workers when:**
- You're unsure about requirements
- An existing workflow "mostly" works
- The worker would be complex or multi-purpose
- You haven't verified it doesn't already exist
- You can't clearly explain what it does in one sentence

---

## Worker Creation Template

If you must create a new worker (only after checking ALL guardrails above), use this template:

**Create the workflow file at `.github/workflows/<workflow-id>.md`:**

```yaml
---
name: <Worker Name>
description: <One sentence describing what it does>

on:
  workflow_dispatch:
    inputs:
      campaign_id:
        description: 'Campaign identifier'
        required: true
        type: string
      payload:
        description: 'JSON payload with work item details'
        required: true
        type: string

tracker-id: <workflow-id>

tools:
  github:
    toolsets: [default]
  # Add minimal additional tools as needed

safe-outputs:
  create-pull-request:
    max: 1  # Start conservative
  add-comment:
    max: 2
---

# <Worker Name>

You are a campaign worker that processes work items.

## Input Contract

Parse inputs:
```javascript
const campaignId = context.payload.inputs.campaign_id;
const payload = JSON.parse(context.payload.inputs.payload);
```

Expected payload structure:
```json
{
  "repository": "owner/repo",
  "work_item_id": "unique-id",
  "target_ref": "main",
  // Additional context...
}
```

## Idempotency Requirements

1. **Generate deterministic key**:
   ```
   const workKey = `campaign-${campaignId}-${payload.repository}-${payload.work_item_id}`;
   ```

2. **Check for existing work**:
   - Search for PRs/issues with `workKey` in title
  - Filter by label: `z_campaign_${campaignId}`
   - If found: Skip or update
   - If not: Create new

3. **Label all created items**:
  - Apply `z_campaign_${campaignId}` label
   - This enables discovery by orchestrator

## Task

<Specific task description>

## Output

Report:
- Link to created/updated PR or issue
- Whether work was skipped (exists) or completed
- Any errors or blockers
```

**After creating:**
- Compile: `gh aw compile <workflow-id>.md`
- **CRITICAL: Test with sample inputs** (see testing requirements below)

---

## Worker Testing (MANDATORY)

**Why test?** - Untested workers may fail during campaign execution. Test with sample inputs first to catch issues early.

**Testing steps:**

1. **Prepare test payload**:
   ```json
   {
     "repository": "test-org/test-repo",
     "work_item_id": "test-1",
     "target_ref": "main"
   }
   ```

2. **Trigger test run**:
   ```bash
   gh workflow run <workflow-id>.yml \
     -f campaign_id={{ .CampaignID }} \
     -f payload='{"repository":"test-org/test-repo","work_item_id":"test-1"}'
   ```
   
   Or via GitHub MCP:
   ```javascript
   mcp__github__run_workflow(
     workflow_id: "<workflow-id>", 
     ref: "main",
     inputs: {
       campaign_id: "{{ .CampaignID }}",
       payload: JSON.stringify({repository: "test-org/test-repo", work_item_id: "test-1"})
     }
   )
   ```

3. **Wait for completion**: Poll until status is "completed"

4. **Verify success**:
   - Check that workflow succeeded
   - Verify idempotency: Run again with same inputs, should skip/update
   - Review created items have correct labels
   - Confirm deterministic keys are used

5. **Test failure actions**:
   - DO NOT use the worker if testing fails
   - Analyze failure logs
   - Make corrections
   - Recompile and retest
   - If unfixable after 2 attempts, report in status and skip

**Note**: Workflows that accept `workflow_dispatch` inputs can receive parameters from the orchestrator. This enables the orchestrator to provide context, priorities, or targets based on its decisions. See [DispatchOps documentation](https://githubnext.github.io/gh-aw/guides/dispatchops/#with-input-parameters) for input parameter examples.

---

## Orchestration Guidelines

**Execution pattern:**
- Workers are **orchestrated, not autonomous**
- Orchestrator discovers work items via discovery manifest
- Orchestrator decides which workers to run and with what inputs
- Workers receive `campaign_id` and `payload` via workflow_dispatch
- Sequential vs parallel execution is orchestrator's decision

**Worker dispatch:**
- Parse discovery manifest (`./.gh-aw/campaign.discovery.json`)
- For each work item needing processing:
  1. Determine appropriate worker for this item type
  2. Construct payload with work item details
  3. Dispatch worker via workflow_dispatch with campaign_id and payload
  4. Track dispatch status

**Input construction:**
```javascript
// Example: Dispatching security-fix worker
const workItem = discoveryManifest.items[0];
const payload = {
  repository: workItem.repo,
  work_item_id: `alert-${workItem.number}`,
  target_ref: "main",
  alert_type: "sql-injection",
  file_path: "src/db.go",
  line_number: 42
};

await github.actions.createWorkflowDispatch({
  owner: context.repo.owner,
  repo: context.repo.repo,
  workflow_id: "security-fix-worker.yml",
  ref: "main",
  inputs: {
    campaign_id: "{{ .CampaignID }}",
    payload: JSON.stringify(payload)
  }
});
```

**Idempotency by design:**
- Workers implement their own idempotency checks
- Orchestrator doesn't need to track what's been processed
- Can safely re-dispatch work items across runs
- Workers will skip or update existing items

**Failure handling:**
- If a worker dispatch fails, note it but continue
- Worker failures don't block entire campaign
- Report all failures in status update with context
- Humans can intervene if needed

---

## After Worker Orchestration

Once workers have been dispatched (or new workers created and tested), proceed with normal orchestrator steps:

1. **Discovery** - Read state from discovery manifest and project board
2. **Planning** - Determine what needs updating on project board
3. **Project Updates** - Write state changes to project board  
4. **Status Reporting** - Report progress, worker dispatches, failures, next steps

---

## Key Differences from Fusion Approach

**Old fusion approach (REMOVED)**:
- Workers had mixed triggers (schedule + workflow_dispatch)
- Fusion dynamically added workflow_dispatch to existing workflows
- Workers stored in campaign-specific folders
- Ambiguous ownership and trigger precedence

**New first-class worker approach**:
- Workers are dispatch-only (on: workflow_dispatch)
- Standardized input contract (campaign_id, payload)
- Explicit idempotency via deterministic keys
- Clear ownership: workers are orchestrated, not autonomous
- Workers stored with regular workflows (not campaign-specific folders)
- Orchestration policy kept explicit in orchestrator

This eliminates duplicate execution problems and makes orchestration concerns explicit.
