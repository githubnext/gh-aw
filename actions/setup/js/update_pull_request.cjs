// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "update_pull_request";

const { updateBody } = require("./update_pr_description_helpers.cjs");
const { isPRContext, getPRNumber } = require("./update_context_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Execute the pull request update API call
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} prNumber - PR number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated pull request
 */
async function executePRUpdate(github, context, prNumber, updateData) {
  // Handle body operation (append/prepend/replace/replace-island)
  const operation = updateData._operation || "replace";
  const rawBody = updateData._rawBody;

  // Remove internal fields
  const { _operation, _rawBody, ...apiData } = updateData;

  // If we have a body with operation, handle it
  if (rawBody !== undefined && operation !== "replace") {
    // Fetch current PR body for operations that need it
    const { data: currentPR } = await github.rest.pulls.get({
      owner: context.repo.owner,
      repo: context.repo.repo,
      pull_number: prNumber,
    });
    const currentBody = currentPR.body || "";

    // Get workflow run URL for AI attribution
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";
    const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

    // Use helper to update body
    apiData.body = updateBody({
      currentBody,
      newContent: rawBody,
      operation,
      workflowName,
      runUrl,
      runId: context.runId,
    });

    core.info(`Will update body (length: ${apiData.body.length})`);
  } else if (rawBody !== undefined) {
    // Replace: just use the new content as-is (already in apiData.body)
    core.info("Operation: replace (full body replacement)");
  }

  const { data: pr } = await github.rest.pulls.update({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: prNumber,
    ...apiData,
  });

  return pr;
}

/**
 * Main handler factory for update_pull_request
 * Returns a message handler function that processes individual update_pull_request messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const updateTarget = config.target || "triggering";
  const maxCount = config.max || 10;
  const canUpdateTitle = config.allow_title !== false; // Default true
  const canUpdateBody = config.allow_body !== false; // Default true

  core.info(`Update pull request configuration: max=${maxCount}, target=${updateTarget}, allow_title=${canUpdateTitle}, allow_body=${canUpdateBody}`);

  // Track state
  let processedCount = 0;

  /**
   * Message handler function
   * @param {Object} message - The update_pull_request message
   * @param {Object} resolvedTemporaryIds - Resolved temporary IDs
   * @returns {Promise<Object>} Result
   */
  return async function handleUpdatePullRequest(message, resolvedTemporaryIds) {
    // Check max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping update_pull_request: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    // Determine target PR number
    let prNumber;
    if (item.pull_request_number !== undefined) {
      prNumber = parseInt(String(item.pull_request_number), 10);
      if (isNaN(prNumber)) {
        core.warning(`Invalid pull request number: ${item.pull_request_number}`);
        return {
          success: false,
          error: `Invalid pull request number: ${item.pull_request_number}`,
        };
      }
    } else {
      // Use triggering context
      if (updateTarget === "triggering" && isPRContext(context.eventName, context.payload)) {
        prNumber = getPRNumber(context.payload);
        if (!prNumber) {
          core.warning("No PR number in triggering context");
          return {
            success: false,
            error: "No PR number available",
          };
        }
      } else {
        core.warning("No pull_request_number provided");
        return {
          success: false,
          error: "No pull request number provided",
        };
      }
    }

    // Build update data
    const updateData = {};
    let hasUpdates = false;

    if (canUpdateTitle && item.title !== undefined) {
      updateData.title = item.title;
      hasUpdates = true;
    }

    if (canUpdateBody && item.body !== undefined) {
      // Store operation information
      if (item.operation !== undefined) {
        updateData._operation = item.operation;
        updateData._rawBody = item.body;
      }
      updateData.body = item.body;
      hasUpdates = true;
    }

    // Other fields (always allowed)
    if (item.state !== undefined) {
      updateData.state = item.state;
      hasUpdates = true;
    }
    if (item.base !== undefined) {
      updateData.base = item.base;
      hasUpdates = true;
    }

    if (!hasUpdates) {
      core.warning("No update fields provided or all fields are disabled");
      return {
        success: false,
        error: "No update fields provided",
      };
    }

    core.info(`Updating pull request #${prNumber} with: ${JSON.stringify(Object.keys(updateData).filter(k => !k.startsWith("_")))}`);

    try {
      const updatedPR = await executePRUpdate(github, context, prNumber, updateData);
      core.info(`Successfully updated pull request #${prNumber}: ${updatedPR.html_url}`);

      return {
        success: true,
        pull_request_number: prNumber,
        pull_request_url: updatedPR.html_url,
        title: updatedPR.title,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to update pull request #${prNumber}: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
