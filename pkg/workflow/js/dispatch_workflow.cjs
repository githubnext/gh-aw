// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  // Initialize outputs to empty strings
  core.setOutput("workflow_name", "");
  core.setOutput("workflow_ref", "");

  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const dispatchWorkflowItems = result.items.filter(item => item.type === "dispatch_workflow");
  if (dispatchWorkflowItems.length === 0) {
    core.info("No dispatch-workflow items found in agent output");
    return;
  }

  core.info(`Found ${dispatchWorkflowItems.length} dispatch-workflow item(s)`);

  // Get allowed workflows from environment
  const allowedWorkflowsEnv = process.env.GH_AW_ALLOWED_WORKFLOWS;
  let allowedWorkflows = [];
  if (allowedWorkflowsEnv) {
    allowedWorkflows = allowedWorkflowsEnv
      .split(",")
      .map(w => w.trim())
      .filter(w => w);
    core.info(`Allowed workflows: ${allowedWorkflows.join(", ")}`);
  } else {
    core.setFailed("No allowed workflows configured. Set GH_AW_ALLOWED_WORKFLOWS environment variable.");
    return;
  }

  if (isStaged) {
    await generateStagedPreview({
      title: "Dispatch Workflows",
      description: "The following workflows would be dispatched if staged mode was disabled:",
      items: dispatchWorkflowItems,
      renderItem: (item, index) => {
        let content = `### Workflow ${index + 1}\n`;
        content += `**Workflow:** ${item.workflow}\n`;

        // Check if workflow is allowed
        const isAllowed = allowedWorkflows.includes(item.workflow);
        content += `**Status:** ${isAllowed ? "✅ Allowed" : "❌ Not in allowlist"}\n`;

        if (item.ref) {
          content += `**Ref:** ${item.ref}\n`;
        }
        if (item.inputs && Object.keys(item.inputs).length > 0) {
          content += `**Inputs:**\n`;
          for (const [key, value] of Object.entries(item.inputs)) {
            content += `  - ${key}: ${JSON.stringify(value)}\n`;
          }
        }
        content += "\n";
        return content;
      },
    });
    return;
  }

  // Process each dispatch_workflow item
  const dispatchedWorkflows = [];
  for (let i = 0; i < dispatchWorkflowItems.length; i++) {
    const item = dispatchWorkflowItems[i];
    core.info(`Processing dispatch-workflow item ${i + 1}/${dispatchWorkflowItems.length}: workflow=${item.workflow}`);

    // Validate workflow is in allowlist
    if (!allowedWorkflows.includes(item.workflow)) {
      core.error(`Workflow '${item.workflow}' is not in the allowed workflows list`);
      core.setFailed(`Workflow '${item.workflow}' is not allowed. Allowed workflows: ${allowedWorkflows.join(", ")}`);
      return;
    }

    try {
      // Prepare workflow dispatch parameters
      const dispatchParams = {
        owner: context.repo.owner,
        repo: context.repo.repo,
        workflow_id: item.workflow,
        ref: item.ref || context.ref || "main",
      };

      // Add inputs if provided
      if (item.inputs && Object.keys(item.inputs).length > 0) {
        dispatchParams.inputs = item.inputs;
      }

      core.info(`Dispatching workflow '${item.workflow}' on ref '${dispatchParams.ref}'`);
      if (dispatchParams.inputs) {
        core.info(`With inputs: ${JSON.stringify(dispatchParams.inputs)}`);
      }

      // Dispatch the workflow
      await github.rest.actions.createWorkflowDispatch(dispatchParams);

      core.info(`Successfully dispatched workflow '${item.workflow}'`);
      dispatchedWorkflows.push({
        workflow: item.workflow,
        ref: dispatchParams.ref,
      });

      // Set outputs for the first dispatched workflow
      if (i === 0) {
        core.setOutput("workflow_name", item.workflow);
        core.setOutput("workflow_ref", dispatchParams.ref);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to dispatch workflow '${item.workflow}': ${errorMessage}`);
      core.setFailed(`Failed to dispatch workflow '${item.workflow}': ${errorMessage}`);
      return;
    }
  }

  // Generate success summary
  let summaryContent = "## ✅ Workflows Dispatched\n\n";
  summaryContent += `Successfully dispatched ${dispatchedWorkflows.length} workflow(s):\n\n`;

  for (const dispatched of dispatchedWorkflows) {
    summaryContent += `- **${dispatched.workflow}** on ref \`${dispatched.ref}\`\n`;
  }

  summaryContent += "\n";
  summaryContent += `> **Note**: Workflow runs may take a few moments to appear in the Actions tab.\n`;

  core.summary.addRaw(summaryContent).write();
}

// Call the main function
await main();
