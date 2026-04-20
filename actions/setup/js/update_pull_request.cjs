// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "update_pull_request";

const { updateBody } = require("./update_pr_description_helpers.cjs");
const { resolveTarget } = require("./safe_output_helpers.cjs");
const { createUpdateHandlerFactory, createStandardResolveNumber, createStandardFormatResult } = require("./update_handler_factory.cjs");
const { sanitizeTitle } = require("./sanitize_title.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { generateHistoryUrl } = require("./generate_history_link.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { withRetry, isTransientError } = require("./error_recovery.cjs");

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
  const includeFooter = updateData._includeFooter !== false; // Default to true

  // Remove internal fields
  const { _operation, _rawBody, _includeFooter, _workflowRepo, ...apiData } = updateData;
  const updateBranch = apiData.update_branch === true;
  delete apiData.update_branch;

  if (updateBranch) {
    core.info(`Updating pull request #${prNumber} branch with base branch changes`);
    try {
      await withRetry(
        () =>
          github.rest.pulls.updateBranch({
            owner: context.repo.owner,
            repo: context.repo.repo,
            pull_number: prNumber,
          }),
        {
          maxRetries: 1,
          initialDelayMs: 0,
          jitterMs: 0,
          shouldRetry: isTransientError,
        },
        `update pull request #${prNumber} branch from base`
      );
    } catch (error) {
      core.warning(`Failed to update pull request #${prNumber} branch from base: ${getErrorMessage(error)}`);
      throw error;
    }
  }

  // If we have a body, process it with the appropriate operation
  if (rawBody !== undefined) {
    // Fetch current PR body for all operations (needed for append/prepend/replace-island/replace)
    const { data: currentPR } = await github.rest.pulls.get({
      owner: context.repo.owner,
      repo: context.repo.repo,
      pull_number: prNumber,
    });
    const currentBody = currentPR.body || "";

    // Get workflow run URL for AI attribution.
    // Use the original workflow repo (_workflowRepo) rather than context.repo, because
    // context may be effectiveContext with repo overridden to a cross-repo target.
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";
    const workflowId = process.env.GH_AW_WORKFLOW_ID || "";
    const callerWorkflowId = process.env.GH_AW_CALLER_WORKFLOW_ID || "";
    const workflowRepo = _workflowRepo || context.repo;
    const runUrl = buildWorkflowRunUrl(context, workflowRepo);

    const historyUrl =
      generateHistoryUrl({
        owner: context.repo.owner,
        repo: context.repo.repo,
        itemType: "pull_request",
        workflowCallId: callerWorkflowId,
        workflowId,
        serverUrl: context.serverUrl,
      }) || undefined;

    // Use helper to update body (handles all operations including replace)
    apiData.body = updateBody({
      currentBody,
      newContent: rawBody,
      operation,
      workflowName,
      runUrl,
      workflowId,
      includeFooter, // Pass footer flag to helper
      historyUrl,
    });

    core.info(`Will update body (length: ${apiData.body.length})`);
  }

  if (Object.keys(apiData).length === 0) {
    return {
      number: prNumber,
      html_url: `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/pull/${prNumber}`,
    };
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
 * Resolve PR number from message and configuration
 * Uses the standard resolve helper for consistency with update_issue
 */
const resolvePRNumber = createStandardResolveNumber({
  itemType: "update_pull_request",
  itemNumberField: "pull_request_number",
  supportsPR: false, // update_pull_request only supports PRs, not issues
  supportsIssue: false,
});

/**
 * Build update data from message
 * @param {Object} item - The message item
 * @param {Object} config - Configuration object
 * @returns {{success: true, data: Object} | {success: true, skipped: true, reason: string} | {success: false, error: string}} Update data result
 */
function buildPRUpdateData(item, config) {
  const canUpdateTitle = config.allow_title !== false; // Default true
  const canUpdateBody = config.allow_body !== false; // Default true

  const updateData = {};
  let hasUpdates = false;

  if (canUpdateTitle && item.title !== undefined) {
    // Sanitize title for Unicode security (no prefix handling needed for updates)
    updateData.title = sanitizeTitle(item.title);
    hasUpdates = true;
  }

  if (canUpdateBody && item.body !== undefined) {
    // Store operation information
    // Use operation from item, or fall back to config default, or use "replace" as final default
    const operation = item.operation || config.default_operation || "replace";
    updateData._operation = operation;
    updateData._rawBody = item.body;
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
  if (item.draft !== undefined) {
    updateData.draft = item.draft;
    hasUpdates = true;
  }

  const updateBranch = item.update_branch !== undefined ? item.update_branch === true : config.update_branch === true;
  if (updateBranch) {
    updateData.update_branch = true;
    hasUpdates = true;
  }

  if (!hasUpdates) {
    return {
      success: true,
      skipped: true,
      reason: "No update fields provided or all fields are disabled",
    };
  }

  // Pass footer config to executeUpdate (default to true)
  updateData._includeFooter = parseBoolTemplatable(config.footer, true);

  return { success: true, data: updateData };
}

/**
 * Format success result for PR update
 * Uses the standard format helper for consistency across update handlers
 */
const formatPRSuccessResult = createStandardFormatResult({
  numberField: "pull_request_number",
  urlField: "pull_request_url",
  urlSource: "html_url",
});

/**
 * Main handler factory for update_pull_request
 * Returns a message handler function that processes individual update_pull_request messages
 * @type {HandlerFactoryFunction}
 */
const main = createUpdateHandlerFactory({
  itemType: "update_pull_request",
  itemTypeName: "pull request",
  supportsPR: false,
  resolveItemNumber: resolvePRNumber,
  buildUpdateData: buildPRUpdateData,
  executeUpdate: executePRUpdate,
  formatSuccessResult: formatPRSuccessResult,
  additionalConfig: {
    allow_title: true,
    allow_body: true,
    update_branch: false,
  },
});

module.exports = { main, buildPRUpdateData };
