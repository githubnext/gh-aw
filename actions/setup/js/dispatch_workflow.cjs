// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "dispatch_workflow";

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Main handler factory for dispatch_workflow
 * Returns a message handler function that processes individual dispatch_workflow messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedWorkflows = config.workflows || [];
  const maxCount = config.max || 1;

  core.info(`Dispatch workflow configuration: max=${maxCount}`);
  if (allowedWorkflows.length > 0) {
    core.info(`Allowed workflows: ${allowedWorkflows.join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  // Get the current repository context and ref
  const repo = context.repo;
  const ref = process.env.GITHUB_REF || context.ref || "refs/heads/main";

  /**
   * Message handler function that processes a single dispatch_workflow message
   * @param {Object} message - The dispatch_workflow message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleDispatchWorkflow(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping dispatch_workflow: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;
    const workflowName = item.workflow_name;

    if (!workflowName || workflowName.trim() === "") {
      core.warning("Workflow name is empty, skipping");
      return {
        success: false,
        error: "Workflow name is empty",
      };
    }

    // Validate workflow is in allowed list
    if (allowedWorkflows.length > 0 && !allowedWorkflows.includes(workflowName)) {
      const error = `Workflow "${workflowName}" is not in the allowed workflows list: ${allowedWorkflows.join(", ")}`;
      core.warning(error);
      return {
        success: false,
        error: error,
      };
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

      core.info(`✓ Successfully dispatched workflow: ${workflowName}`);

      return {
        success: true,
        workflow_name: workflowName,
        inputs: inputs,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to dispatch workflow "${workflowName}": ${errorMessage}`);

      // Check if it's a 404 error (workflow not found)
      if (errorMessage.includes("404")) {
        return {
          success: false,
          error: `Workflow "${workflowName}.lock.yml" not found. Make sure the workflow file exists and supports workflow_dispatch trigger.`,
        };
      } else {
        return {
          success: false,
          error: `Failed to dispatch workflow "${workflowName}": ${errorMessage}`,
        };
      }
    }
  };
}

module.exports = { main };
