// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 * @typedef {import('./types/handler-factory').HandlerConfig} HandlerConfig
 * @typedef {import('./types/handler-factory').ResolvedTemporaryIds} ResolvedTemporaryIds
 * @typedef {import('./types/handler-factory').HandlerResult} HandlerResult
 */

/**
 * @typedef {{ reviewers?: Array<string|null|undefined|false>, team_reviewers?: Array<string|null|undefined|false>, pull_request_number?: number|string }} AddReviewerMessage
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "add_reviewer";

const { processItems } = require("./safe_output_processor.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { getPullRequestNumber } = require("./pr_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { COPILOT_REVIEWER_BOT } = require("./constants.cjs");

/**
 * Main handler factory for add_reviewer
 * Returns a message handler function that processes individual add_reviewer messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const allowedReviewers = config.allowed ?? [];
  const allowedTeamReviewers = config.allowed_team_reviewers ?? [];
  const maxCount = config.max ?? 10;
  const githubClient = await createAuthenticatedGitHubClient(config);
  const isStaged = isStagedMode(config);

  core.info(`Add reviewer configuration: max=${maxCount}`);
  if (allowedReviewers.length > 0) {
    core.info(`Allowed reviewers: ${allowedReviewers.join(", ")}`);
  }
  if (allowedTeamReviewers.length > 0) {
    core.info(`Allowed team reviewers: ${allowedTeamReviewers.join(", ")}`);
  }

  let processedCount = 0;

  /**
   * @param {AddReviewerMessage} message - The add_reviewer message to process
   * @param {ResolvedTemporaryIds} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<HandlerResult>} Result with success/error status
   */
  return async function handleAddReviewer(message, resolvedTemporaryIds) {
    if (processedCount >= maxCount) {
      core.warning(`Skipping add_reviewer: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const { prNumber, error } = getPullRequestNumber(message, context);

    if (error) {
      core.warning(error);
      return {
        success: false,
        error,
      };
    }
    if (prNumber === null) {
      return {
        success: false,
        error: "Pull request number is required",
      };
    }

    const requestedReviewers = message.reviewers ?? [];
    const requestedTeamReviewers = message.team_reviewers ?? [];
    core.info(`Requested reviewers: ${JSON.stringify(requestedReviewers)}`);
    core.info(`Requested team reviewers: ${JSON.stringify(requestedTeamReviewers)}`);

    // Use shared helper to filter, sanitize, dedupe, and limit across both reviewer types
    const uniqueReviewers = processItems(requestedReviewers, allowedReviewers, maxCount);
    const remainingReviewerSlots = Math.max(0, maxCount - uniqueReviewers.length);
    const uniqueTeamReviewers = processItems(requestedTeamReviewers, allowedTeamReviewers, remainingReviewerSlots);

    if (uniqueReviewers.length === 0 && uniqueTeamReviewers.length === 0) {
      core.info("No reviewers to add");
      return {
        success: true,
        prNumber,
        reviewersAdded: [],
        teamReviewersAdded: [],
        message: "No valid reviewers found",
      };
    }

    core.info(`Adding reviewers to PR #${prNumber}: reviewers=${JSON.stringify(uniqueReviewers)}, team_reviewers=${JSON.stringify(uniqueTeamReviewers)}`);

    // If in staged mode, preview without executing
    if (isStaged) {
      logStagedPreviewInfo(`Would add reviewers to PR #${prNumber}`);
      return {
        success: true,
        staged: true,
        previewInfo: {
          number: prNumber,
          reviewers: uniqueReviewers,
          team_reviewers: uniqueTeamReviewers,
        },
      };
    }

    try {
      // Special handling for "copilot" reviewer - separate it from other reviewers
      const hasCopilot = uniqueReviewers.includes("copilot");
      const otherReviewers = uniqueReviewers.filter(r => r !== "copilot");

      // Add non-copilot reviewers first
      if (otherReviewers.length > 0 || uniqueTeamReviewers.length > 0) {
        /** @type {{ owner: string, repo: string, pull_number: number, reviewers: string[], team_reviewers?: string[] }} */
        const reviewerRequest = {
          owner: context.repo.owner,
          repo: context.repo.repo,
          pull_number: prNumber,
          reviewers: otherReviewers,
        };
        if (uniqueTeamReviewers.length > 0) {
          reviewerRequest.team_reviewers = uniqueTeamReviewers;
        }
        await githubClient.rest.pulls.requestReviewers(reviewerRequest);
        core.info(`Successfully added reviewers to PR #${prNumber}: reviewers=${JSON.stringify(otherReviewers)}, team_reviewers=${JSON.stringify(uniqueTeamReviewers)}`);
      }

      // Add copilot reviewer separately if requested
      if (hasCopilot) {
        try {
          await githubClient.rest.pulls.requestReviewers({
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
        prNumber,
        reviewersAdded: uniqueReviewers,
        teamReviewersAdded: uniqueTeamReviewers,
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
