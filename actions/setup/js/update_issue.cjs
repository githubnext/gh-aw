// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "update_issue";

const { isIssueContext, getIssueNumber } = require("./update_context_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTarget } = require("./safe_output_helpers.cjs");

/**
 * Execute the issue update API call
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} issueNumber - Issue number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated issue
 */
async function executeIssueUpdate(github, context, issueNumber, updateData) {
  // Remove internal fields used for operation handling
  const { _operation, _rawBody, ...apiData } = updateData;

  const { data: issue } = await github.rest.issues.update({
    owner: context.repo.owner,
    repo: context.repo.repo,
    issue_number: issueNumber,
    ...apiData,
  });

  return issue;
}

/**
 * Main handler factory for update_issue
 * Returns a message handler function that processes individual update_issue messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const updateTarget = config.target || "triggering";
  const maxCount = config.max || 10;

  core.info(`Update issue configuration: max=${maxCount}, target=${updateTarget}`);

  // Track state
  let processedCount = 0;

  /**
   * Message handler function
   * @param {Object} message - The update_issue message
   * @param {Object} resolvedTemporaryIds - Resolved temporary IDs
   * @returns {Promise<Object>} Result
   */
  return async function handleUpdateIssue(message, resolvedTemporaryIds) {
    // Check max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping update_issue: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    // Determine target issue number using resolveTarget helper
    const targetResult = resolveTarget({
      targetConfig: updateTarget,
      item: { ...item, item_number: item.issue_number },
      context: context,
      itemType: "update_issue",
      supportsPR: false, // update_issue only supports issues, not PRs
    });

    if (!targetResult.success) {
      core.warning(targetResult.error);
      return {
        success: false,
        error: targetResult.error,
      };
    }

    const issueNumber = targetResult.number;
    core.info(`Resolved target issue #${issueNumber} (target config: ${updateTarget})`);

    // Build update data
    const updateData = {};
    if (item.title !== undefined) {
      updateData.title = item.title;
    }
    if (item.body !== undefined) {
      updateData.body = item.body;
    }
    if (item.state !== undefined) {
      updateData.state = item.state;
    }
    if (item.labels !== undefined) {
      updateData.labels = item.labels;
    }
    if (item.assignees !== undefined) {
      updateData.assignees = item.assignees;
    }
    if (item.milestone !== undefined) {
      updateData.milestone = item.milestone;
    }

    if (Object.keys(updateData).length === 0) {
      core.warning("No update fields provided");
      return {
        success: false,
        error: "No update fields provided",
      };
    }

    core.info(`Updating issue #${issueNumber} with: ${JSON.stringify(Object.keys(updateData))}`);

    try {
      const updatedIssue = await executeIssueUpdate(github, context, issueNumber, updateData);
      core.info(`Successfully updated issue #${issueNumber}: ${updatedIssue.html_url}`);

      return {
        success: true,
        number: issueNumber,
        url: updatedIssue.html_url,
        title: updatedIssue.title,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to update issue #${issueNumber}: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
