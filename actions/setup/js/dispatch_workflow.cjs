// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const fs = require("fs");

async function main() {
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("count", "");

  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  if (!agentOutputFile) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  // Read agent output from file
  let outputContent;
  try {
    outputContent = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.setFailed(`Error reading agent output file: ${getErrorMessage(error)}`);
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }
  core.info(`Agent output content length: ${outputContent.length}`);

  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${getErrorMessage(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  const dispatchWorkflowItems = validatedOutput.items.filter(item => item.type === "dispatch_workflow");
  if (dispatchWorkflowItems.length === 0) {
    core.info("No dispatch-workflow items found in agent output");
    return;
  }

  core.info(`Found ${dispatchWorkflowItems.length} dispatch-workflow item(s)`);

  // Get allowed workflows and max count from environment
  const allowedWorkflowsJSON = process.env.GH_AW_DISPATCH_WORKFLOW_ALLOWED;
  const maxCount = parseInt(process.env.GH_AW_DISPATCH_WORKFLOW_MAX_COUNT || "1", 10);

  let allowedWorkflows = [];
  if (allowedWorkflowsJSON) {
    try {
      allowedWorkflows = JSON.parse(allowedWorkflowsJSON);
    } catch (error) {
      core.setFailed(`Error parsing GH_AW_DISPATCH_WORKFLOW_ALLOWED: ${getErrorMessage(error)}`);
      return;
    }
  }

  // Check max count
  if (dispatchWorkflowItems.length > maxCount) {
    core.setFailed(`Too many dispatch-workflow items: ${dispatchWorkflowItems.length} (max: ${maxCount})`);
    return;
  }

  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Dispatch Workflow Preview\n\n";
    summaryContent += "The following workflows would be dispatched if staged mode was disabled:\n\n";

    for (const [index, item] of dispatchWorkflowItems.entries()) {
      summaryContent += `### Workflow ${index + 1}\n\n`;
      summaryContent += `**Workflow:** ${item.workflow_name}\n\n`;

      if (item.inputs && Object.keys(item.inputs).length > 0) {
        summaryContent += `**Inputs:**\n`;
        for (const [key, value] of Object.entries(item.inputs)) {
          summaryContent += `- \`${key}\`: ${JSON.stringify(value)}\n`;
        }
        summaryContent += "\n";
      } else {
        summaryContent += `**Inputs:** None\n\n`;
      }

      summaryContent += "---\n\n";
    }

    core.info(summaryContent);
    core.summary.addRaw(summaryContent);
    await core.summary.write();
    return;
  }

  // Get the current repository context
  const repo = context.repo;
  const ref = process.env.GITHUB_REF || context.ref || "refs/heads/main";

  // Process all dispatch workflow items
  let dispatchedCount = 0;
  let summaryContent = "## ✅ Workflows Dispatched\n\n";

  for (const [index, item] of dispatchWorkflowItems.entries()) {
    const workflowName = item.workflow_name;

    if (!workflowName || workflowName.trim() === "") {
      core.warning(`Item ${index + 1}: Workflow name is empty, skipping`);
      continue;
    }

    // Validate workflow is in allowed list
    if (allowedWorkflows.length > 0 && !allowedWorkflows.includes(workflowName)) {
      core.setFailed(`Workflow "${workflowName}" is not in the allowed workflows list: ${allowedWorkflows.join(", ")}`);
      return;
    }

    try {
      core.info(`Dispatching workflow: ${workflowName}`);

      // Prepare inputs - convert all values to strings as required by workflow_dispatch
      /** @type {Record<string, string>} */
      const inputs = {};
      if (item.inputs && typeof item.inputs === "object") {
        for (const [key, value] of Object.entries(item.inputs)) {
          // Convert value to string
          if (value === null || value === undefined) {
            inputs[key] = "";
          } else if (typeof value === "object") {
            inputs[key] = JSON.stringify(value);
          } else {
            inputs[key] = String(value);
          }
        }
      }

      // Dispatch the workflow using the GitHub REST API
      await github.rest.actions.createWorkflowDispatch({
        owner: repo.owner,
        repo: repo.repo,
        workflow_id: `${workflowName}.lock.yml`,
        ref: ref,
        inputs: inputs,
      });

      dispatchedCount++;
      core.info(`✓ Successfully dispatched workflow: ${workflowName}`);

      summaryContent += `### ${workflowName}\n\n`;
      summaryContent += `**Status:** ✅ Dispatched\n\n`;
      if (Object.keys(inputs).length > 0) {
        summaryContent += `**Inputs:**\n`;
        for (const [key, value] of Object.entries(inputs)) {
          summaryContent += `- \`${key}\`: ${value}\n`;
        }
        summaryContent += "\n";
      }
      summaryContent += "---\n\n";
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to dispatch workflow "${workflowName}": ${errorMessage}`);

      // Check if it's a 404 error (workflow not found)
      if (errorMessage.includes("404")) {
        core.setFailed(`Workflow "${workflowName}.lock.yml" not found. Make sure the workflow file exists and supports workflow_dispatch trigger.`);
      } else {
        core.setFailed(`Failed to dispatch workflow "${workflowName}": ${errorMessage}`);
      }
      return;
    }
  }

  // Set output with count of dispatched workflows
  core.setOutput("count", dispatchedCount.toString());
  core.info(`Total workflows dispatched: ${dispatchedCount}`);

  // Write summary
  core.info(summaryContent);
  core.summary.addRaw(summaryContent);
  await core.summary.write();
}

module.exports = { main };
