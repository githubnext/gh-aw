// @ts-check

const core = require("@actions/core");
const github = require("@actions/github");

/**
 * Main function to dispatch workflows
 */
async function main() {
  const context = github.context;

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from file
  const outputFile = process.env.GH_AW_AGENT_OUTPUT;
  if (!outputFile) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  const fs = require("fs");
  let outputContent;
  try {
    outputContent = fs.readFileSync(outputFile, "utf8");
  } catch (error) {
    core.setFailed(`Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all dispatch_workflow items
  const items = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "dispatch_workflow");
  if (items.length === 0) {
    core.info("No dispatch_workflow items found in agent output");
    return;
  }

  core.info(`Found ${items.length} dispatch_workflow item(s)`);

  // Get GitHub token
  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN;
  if (!token) {
    core.setFailed("GITHUB_TOKEN not found in environment");
    return;
  }

  const octokit = github.getOctokit(token);

  // If in staged mode, emit step summary instead of performing actions
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Workflow Dispatch Preview\n\n";
    summaryContent += "The following workflow dispatches would be performed if staged mode was disabled:\n\n";

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      summaryContent += `### Dispatch ${i + 1}\n`;
      summaryContent += `**Workflow**: \`${item.workflow_id}\`\n`;
      if (item.ref) {
        summaryContent += `**Ref**: \`${item.ref}\`\n`;
      }
      if (item.inputs && Object.keys(item.inputs).length > 0) {
        summaryContent += `**Inputs**:\n\`\`\`json\n${JSON.stringify(item.inputs, null, 2)}\n\`\`\`\n`;
      }
      summaryContent += "\n";
    }

    core.summary.addRaw(summaryContent).write();
    return;
  }

  // Process each workflow dispatch item
  for (const item of items) {
    try {
      core.info(`Dispatching workflow: ${item.workflow_id}`);

      // Prepare workflow dispatch parameters
      const params = {
        owner: context.repo.owner,
        repo: context.repo.repo,
        workflow_id: item.workflow_id,
        ref: item.ref || context.ref || "main",
        inputs: item.inputs || {},
      };

      core.info(`  Owner: ${params.owner}`);
      core.info(`  Repo: ${params.repo}`);
      core.info(`  Workflow ID: ${params.workflow_id}`);
      core.info(`  Ref: ${params.ref}`);
      if (Object.keys(params.inputs).length > 0) {
        core.info(`  Inputs: ${JSON.stringify(params.inputs)}`);
      }

      // Dispatch the workflow
      await octokit.rest.actions.createWorkflowDispatch(params);

      core.info(`✓ Successfully dispatched workflow: ${item.workflow_id}`);

      // Set outputs
      core.setOutput("workflow_id", item.workflow_id);
      core.setOutput("ref", params.ref);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to dispatch workflow ${item.workflow_id}: ${errorMessage}`);
      core.setFailed(`Failed to dispatch workflow ${item.workflow_id}: ${errorMessage}`);
      return;
    }
  }

  core.info("All workflow dispatches completed successfully");
}

// Call the main function
main().catch(error => {
  core.setFailed(`Unexpected error: ${error instanceof Error ? error.message : String(error)}`);
});
