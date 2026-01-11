// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "update_issue";

const { resolveTarget } = require("./safe_output_helpers.cjs");
const { createUpdateHandlerFactory } = require("./update_handler_factory.cjs");

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
 * Resolve issue number from message and configuration
 * @param {Object} item - The message item
 * @param {string} updateTarget - Target configuration
 * @param {Object} context - GitHub Actions context
 * @returns {{success: true, number: number} | {success: false, error: string}} Resolution result
 */
function resolveIssueNumber(item, updateTarget, context) {
  const targetResult = resolveTarget({
    targetConfig: updateTarget,
    item: { ...item, item_number: item.issue_number },
    context: context,
    itemType: "update_issue",
    supportsPR: false, // Not used when supportsIssue is true
    supportsIssue: true, // update_issue only supports issues, not PRs
  });

  if (!targetResult.success) {
    return { success: false, error: targetResult.error };
  }

  return { success: true, number: targetResult.number };
}

/**
 * Build update data from message
 * @param {Object} item - The message item
 * @param {Object} config - Configuration object
 * @returns {{success: true, data: Object} | {success: false, error: string}} Update data result
 */
function buildIssueUpdateData(item, config) {
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

  return { success: true, data: updateData };
}

/**
 * Format success result for issue update
 * @param {number} issueNumber - Issue number
 * @param {Object} updatedIssue - Updated issue object
 * @returns {Object} Formatted success result
 */
function formatIssueSuccessResult(issueNumber, updatedIssue) {
  return {
    success: true,
    number: issueNumber,
    url: updatedIssue.html_url,
    title: updatedIssue.title,
  };
}

/**
 * Main handler factory for update_issue
 * Returns a message handler function that processes individual update_issue messages
 * @type {HandlerFactoryFunction}
 */
const main = createUpdateHandlerFactory({
  itemType: "update_issue",
  itemTypeName: "issue",
  supportsPR: false, // Not used by factory, but kept for documentation
  resolveItemNumber: resolveIssueNumber,
  buildUpdateData: buildIssueUpdateData,
  executeUpdate: executeIssueUpdate,
  formatSuccessResult: formatIssueSuccessResult,
});

module.exports = { main };
