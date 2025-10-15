---
safe-outputs:
  jobs:
    trigger-workflow:
      description: "Trigger a workflow dispatch for allowed workflows. Requires actions: write permission or a custom GitHub token."
      runs-on: ubuntu-latest
      output: "Workflow triggered successfully!"
      inputs:
        workflow:
          description: "The workflow filename to trigger (e.g., 'build.yml')"
          required: true
          type: string
        payload:
          description: "Optional JSON payload for workflow inputs (e.g., '{\"environment\":\"production\"}')"
          required: false
          type: string
      permissions:
        actions: write
        contents: read
      steps:
        - name: Trigger workflow dispatch
          uses: actions/github-script@v8
          with:
            script: |
              const fs = require('fs');
              const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === 'true';
              const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
              
              // Read allowed workflows from environment variable or config
              let allowedWorkflows = [];
              
              // First, try to read from GH_AW_TRIGGER_WORKFLOW_ALLOWED environment variable
              const allowedEnv = process.env.GH_AW_TRIGGER_WORKFLOW_ALLOWED;
              if (allowedEnv) {
                // Split by comma and trim whitespace
                allowedWorkflows = allowedEnv.split(',').map(w => w.trim()).filter(Boolean);
              } else {
                // Fall back to reading from config (for backwards compatibility)
                const configEnv = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
                if (configEnv) {
                  try {
                    const config = JSON.parse(configEnv);
                    if (config['trigger_workflow'] && Array.isArray(config['trigger_workflow'].allowed)) {
                      allowedWorkflows = config['trigger_workflow'].allowed;
                    }
                  } catch (error) {
                    core.setFailed(`Error parsing safe outputs config: ${error instanceof Error ? error.message : String(error)}`);
                    return;
                  }
                }
              }
              
              if (allowedWorkflows.length === 0) {
                core.setFailed('No allowed workflows configured for trigger-workflow. Please set GH_AW_TRIGGER_WORKFLOW_ALLOWED environment variable with comma-separated workflow filenames.');
                return;
              }
              
              core.info(`Allowed workflows: ${allowedWorkflows.join(', ')}`);
              
              // Read and parse agent output
              if (!outputContent) {
                core.info('No GITHUB_AW_AGENT_OUTPUT environment variable found');
                return;
              }
              
              let agentOutputData;
              try {
                const fileContent = fs.readFileSync(outputContent, 'utf8');
                agentOutputData = JSON.parse(fileContent);
              } catch (error) {
                core.setFailed(`Error reading or parsing agent output: ${error instanceof Error ? error.message : String(error)}`);
                return;
              }
              
              if (!agentOutputData.items || !Array.isArray(agentOutputData.items)) {
                core.info('No valid items found in agent output');
                return;
              }
              
              // Filter for trigger_workflow items
              const triggerItems = agentOutputData.items.filter(item => item.type === 'trigger_workflow');
              
              if (triggerItems.length === 0) {
                core.info('No trigger_workflow items found in agent output');
                return;
              }
              
              core.info(`Found ${triggerItems.length} trigger_workflow item(s)`);
              
              // Process each trigger item
              for (let i = 0; i < triggerItems.length; i++) {
                const item = triggerItems[i];
                const workflow = item.workflow;
                const payload = item.payload;
                
                if (!workflow) {
                  core.warning(`Item ${i + 1}: Missing workflow field, skipping`);
                  continue;
                }
                
                // Validate workflow is in allowed list
                if (!allowedWorkflows.includes(workflow)) {
                  core.warning(`Item ${i + 1}: Workflow '${workflow}' is not in allowed list. Allowed workflows: ${allowedWorkflows.join(', ')}`);
                  continue;
                }
                
                // Parse payload if provided
                let inputs = {};
                if (payload) {
                  try {
                    inputs = JSON.parse(payload);
                    if (typeof inputs !== 'object' || Array.isArray(inputs)) {
                      core.warning(`Item ${i + 1}: Payload must be a JSON object, got ${typeof inputs}`);
                      continue;
                    }
                  } catch (error) {
                    core.warning(`Item ${i + 1}: Invalid JSON payload: ${error instanceof Error ? error.message : String(error)}`);
                    continue;
                  }
                }
                
                if (isStaged) {
                  let summaryContent = "## 🎭 Staged Mode: Workflow Trigger Preview\n\n";
                  summaryContent += "The following workflow dispatch would be triggered if staged mode was disabled:\n\n";
                  summaryContent += `**Workflow:** ${workflow}\n\n`;
                  if (payload) {
                    summaryContent += `**Inputs:**\n\`\`\`json\n${JSON.stringify(inputs, null, 2)}\n\`\`\`\n\n`;
                  } else {
                    summaryContent += `**Inputs:** None\n\n`;
                  }
                  await core.summary.addRaw(summaryContent).write();
                  core.info("📝 Workflow trigger preview written to step summary");
                  continue;
                }
                
                core.info(`Triggering workflow ${i + 1}/${triggerItems.length}: ${workflow}`);
                if (payload) {
                  core.info(`With inputs: ${JSON.stringify(inputs)}`);
                }
                
                // Resolve the ref to use (current ref or repository default branch)
                let ref = context.ref;
                if (!ref) {
                  try {
                    const repoData = await github.rest.repos.get({
                      owner: context.repo.owner,
                      repo: context.repo.repo
                    });
                    ref = repoData.data.default_branch;
                    core.info(`Using repository default branch: ${ref}`);
                  } catch (error) {
                    core.setFailed(`Failed to resolve default branch: ${error instanceof Error ? error.message : String(error)}`);
                    return;
                  }
                }
                
                try {
                  const response = await github.rest.actions.createWorkflowDispatch({
                    owner: context.repo.owner,
                    repo: context.repo.repo,
                    workflow_id: workflow,
                    ref: ref,
                    inputs: inputs
                  });
                  
                  core.info(`✅ Workflow ${i + 1} triggered successfully`);
                  core.info(`Status: ${response.status}`);
                } catch (error) {
                  core.setFailed(`Failed to trigger workflow ${i + 1}: ${error instanceof Error ? error.message : String(error)}`);
                  return;
                }
              }
---
<!--
## Trigger Workflow Integration

This shared configuration provides a custom safe-job for triggering workflow dispatches on allowed workflows.

### Safe Job: trigger-workflow

The `trigger-workflow` safe-job allows agentic workflows to trigger workflow dispatches on specified workflows using the GitHub REST API.

**Agent Output Format:**

The agent should output JSON with items of type `trigger_workflow`:

```json
{
  "items": [
    {
      "type": "trigger_workflow",
      "workflow": "build.yml",
      "payload": "{\"environment\":\"production\",\"version\":\"1.0.0\"}"
    }
  ]
}
```

**Required Configuration:**

You must configure allowed workflows in your workflow's `safe-outputs` configuration:

```yaml
safe-outputs:
  trigger-workflow:
    allowed:
      - "build.yml"
      - "deploy.yml"
      - "test.yml"
```

**Fields:**

- `workflow`: The workflow filename to trigger (must be in the allowed list)
- `payload`: Optional JSON string containing workflow inputs

**Permissions:**

This safe-job requires `actions: write` permission to trigger workflow dispatches. The default `GITHUB_TOKEN` should have sufficient permissions in most cases.

**Example Usage in Workflow:**

```markdown
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
engine: claude
imports:
  - shared/trigger-workflow.md
safe-outputs:
  trigger-workflow:
    allowed:
      - "build.yml"
      - "deploy.yml"
---

# Issue-Triggered Build

When a new issue is created, analyze it and trigger the appropriate workflow.

If the issue mentions "deploy", trigger the deploy.yml workflow.
If the issue mentions "build", trigger the build.yml workflow.

Use the trigger_workflow safe output to trigger the appropriate workflow.
```

**Validation:**

The safe-job validates:
- Workflow filename is in the allowed list
- Payload is valid JSON (if provided)
- Payload is an object, not an array or primitive

**Staged Mode Support:**

This safe-job fully supports staged mode. When `staged: true` is set in the workflow's safe-outputs configuration, workflow triggers will be previewed in the step summary instead of being dispatched.

**Security Considerations:**

- Only workflows explicitly listed in the `allowed` configuration can be triggered
- The `ref` parameter defaults to the current workflow's ref or `main`
- Payload validation ensures only valid JSON objects are passed
- Failed triggers will fail the entire safe-job with an error message
-->
