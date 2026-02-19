// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { processItems } = require("./safe_output_processor.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { getPullRequestNumber } = require("./pr_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");

// GitHub Copilot reviewer bot username
const COPILOT_REVIEWER_BOT = "copilot-pull-request-reviewer[bot]";

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "add_reviewer";

/**
 * Main handler factory for add_reviewer
 * Returns a message handler function that processes individual add_reviewer messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedReviewers = config.allowed || [];
  const maxCount = config.max || 10;

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Add reviewer configuration: max=${maxCount}`);
  if (allowedReviewers.length > 0) {
    core.info(`Allowed reviewers: ${allowedReviewers.join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single add_reviewer message
   * @param {Object} message - The add_reviewer message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleAddReviewer(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping add_reviewer: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const reviewerItem = message;

    // Determine PR number using helper
    const { prNumber, error } = getPullRequestNumber(reviewerItem, context);

    if (error) {
      core.warning(error);
      return {
        success: false,
        error: error,
      };
    }

    const requestedReviewers = reviewerItem.reviewers || [];
    core.info(`Requested reviewers: ${JSON.stringify(requestedReviewers)}`);

    // Use shared helper to filter, sanitize, dedupe, and limit
    const uniqueReviewers = processItems(requestedReviewers, allowedReviewers, maxCount);

    if (uniqueReviewers.length === 0) {
      core.info("No reviewers to add");
      return {
        success: true,
        prNumber: prNumber,
        reviewersAdded: [],
        message: "No valid reviewers found",
      };
    }

    core.info(`Adding ${uniqueReviewers.length} reviewers to PR #${prNumber}: ${JSON.stringify(uniqueReviewers)}`);

    // If in staged mode, preview without executing
    if (isStaged) {
      logStagedPreviewInfo(`Would add reviewers to PR #${prNumber}`);
      return {
        success: true,
        staged: true,
        previewInfo: {
          number: prNumber,
          reviewers: uniqueReviewers,
        },
      };
    }

    try {
      // Special handling for "copilot" reviewer - separate it from other reviewers
      const hasCopilot = uniqueReviewers.includes("copilot");
      const otherReviewers = uniqueReviewers.filter(r => r !== "copilot");

      // Add non-copilot reviewers first
      if (otherReviewers.length > 0) {
        await github.rest.pulls.requestReviewers({
          owner: context.repo.owner,
          repo: context.repo.repo,
          pull_number: prNumber,
          reviewers: otherReviewers,
        });
        core.info(`Successfully added ${otherReviewers.length} reviewer(s) to PR #${prNumber}`);
      }

      // Add copilot reviewer separately if requested
      if (hasCopilot) {
        try {
          await github.rest.pulls.requestReviewers({
            owner: context.repo.owner,
            repo: context.repo.repo,
            pull_number: prNumber,
            reviewers: [COPILOT_REVIEWER_BOT],
          });
          core.info(`Successfully added copilot as reviewer to PR #${prNumber}`);
        } catch (copilotError) {
          core.warning(`Failed to add copilot as reviewer: ${getErrorMessage(copilotError)}`);
          // Don't fail the whole step if copilot reviewer fails
        }
      }

      return {
        success: true,
        prNumber: prNumber,
        reviewersAdded: uniqueReviewers,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to add reviewers: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
