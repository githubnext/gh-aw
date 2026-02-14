// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "update_issue";

const { resolveTarget } = require("./safe_output_helpers.cjs");
const { createUpdateHandlerFactory } = require("./update_handler_factory.cjs");
const { updateBody } = require("./update_pr_description_helpers.cjs");
const { loadTemporaryProjectMap, replaceTemporaryProjectReferences } = require("./temporary_id.cjs");
const { sanitizeTitle } = require("./sanitize_title.cjs");

/**
 * Maximum limits for issue update parameters to prevent resource exhaustion.
 * These limits align with GitHub's API constraints and security best practices.
 */
/** @type {number} Maximum number of labels allowed per issue */
const MAX_LABELS = 10;

/** @type {number} Maximum number of assignees allowed per issue */
const MAX_ASSIGNEES = 5;

/**
 * Enforces maximum limits on issue update parameters to prevent resource exhaustion attacks.
 * Per Safe Outputs specification requirement SEC-003, limits must be enforced before API calls.
 *
 * @param {string[]|undefined} labels - Labels array to validate
 * @param {string[]|undefined} assignees - Assignees array to validate
 * @throws {Error} When any limit is exceeded, with error code E003 and details
 */
function enforceIssueUpdateLimits(labels, assignees) {
  // Check labels count - max limit exceeded check
  if (labels && Array.isArray(labels) && labels.length > MAX_LABELS) {
    throw new Error(`E003: Cannot update issue with more than ${MAX_LABELS} labels (received ${labels.length})`);
  }

  // Check assignees count - max limit exceeded check
  if (assignees && Array.isArray(assignees) && assignees.length > MAX_ASSIGNEES) {
    throw new Error(`E003: Cannot update issue with more than ${MAX_ASSIGNEES} assignees (received ${assignees.length})`);
  }
}

/**
 * Execute the issue update API call
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} issueNumber - Issue number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated issue
 */
async function executeIssueUpdate(github, context, issueNumber, updateData) {
  // Handle body operation (append/prepend/replace/replace-island)
  // Default to "append" to add footer with AI attribution
  const operation = updateData._operation || "append";
  let rawBody = updateData._rawBody;
  const includeFooter = updateData._includeFooter !== false; // Default to true

  // Remove internal fields
  const { _operation, _rawBody, _includeFooter, ...apiData } = updateData;

  // If we have a body, process it with the appropriate operation
  if (rawBody !== undefined) {
    // Load and apply temporary project URL replacements FIRST
    // This resolves any temporary project IDs (e.g., #aw_abc123def456) to actual project URLs
    const temporaryProjectMap = loadTemporaryProjectMap();
    if (temporaryProjectMap.size > 0) {
      rawBody = replaceTemporaryProjectReferences(rawBody, temporaryProjectMap);
      core.debug(`Applied ${temporaryProjectMap.size} temporary project URL replacement(s)`);
    }

    // Fetch current issue body for all operations (needed for append/prepend/replace-island/replace)
    const { data: currentIssue } = await github.rest.issues.get({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
    });
    const currentBody = currentIssue.body || "";

    // Get workflow run URL for AI attribution
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";
    const workflowId = process.env.GH_AW_WORKFLOW_ID || "";
    const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

    // Use helper to update body (handles all operations including replace)
    apiData.body = updateBody({
      currentBody,
      newContent: rawBody,
      operation,
      workflowName,
      runUrl,
      workflowId,
      includeFooter, // Pass footer flag to helper
    });

    core.info(`Will update body (length: ${apiData.body.length})`);
  }

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
    // Sanitize title for Unicode security (no prefix handling needed for updates)
    updateData.title = sanitizeTitle(item.title);
  }
  // Check if body updates are allowed (defaults to true if not specified)
  const canUpdateBody = config.allow_body !== false;
  if (item.body !== undefined && canUpdateBody) {
    // Store operation information for consistent footer/append behavior.
    // Default to "append" so we preserve the original issue text.
    updateData._operation = item.operation || "append";
    updateData._rawBody = item.body;
  } else if (item.body !== undefined && !canUpdateBody) {
    // Body update attempted but not allowed by configuration
    core.warning("Body update not allowed by safe-outputs configuration");
  }
  // The safe-outputs schema uses "status" (open/closed), while the GitHub API uses "state".
  // Accept both for compatibility.
  if (item.state !== undefined) {
    updateData.state = item.state;
  } else if (item.status !== undefined) {
    updateData.state = item.status;
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

  // Enforce max limits on labels and assignees before API calls
  enforceIssueUpdateLimits(updateData.labels, updateData.assignees);

  // Pass footer config to executeUpdate (default to true)
  updateData._includeFooter = config.footer !== false;

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

module.exports = { main, buildIssueUpdateData };
