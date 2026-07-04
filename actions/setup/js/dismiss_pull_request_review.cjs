// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTarget, isStagedMode, logStagedPreviewInfo, checkRequiredFilter } = require("./safe_output_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "dismiss_pull_request_review";

/**
 * Resolve the effective actor used as both the dismisser and default expected author.
 * @returns {string}
 */
function getEffectiveActor() {
  const actor = (process.env.GITHUB_ACTOR || context?.actor || "github-actions[bot]").trim();
  return actor || "github-actions[bot]";
}

/**
 * Main handler factory for dismiss_pull_request_review.
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const maxCount = config.max || 10;
  const targetConfig = config.target || "triggering";
  const isStaged = isStagedMode(config);
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const githubClient = await createAuthenticatedGitHubClient(config);
  const requiredLabels = Array.isArray(config.required_labels) ? config.required_labels : [];
  const requiredTitlePrefix = config.required_title_prefix || "";
  const dismisser = getEffectiveActor();

  let processedCount = 0;

  return async function handleDismissPullRequestReview(message) {
    if (processedCount >= maxCount) {
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    const reviewId = Number.parseInt(String(message.review_id || ""), 10);
    if (!Number.isInteger(reviewId) || reviewId <= 0) {
      return {
        success: false,
        error: "review_id must be a positive integer",
      };
    }

    const justification = typeof message.justification === "string" ? message.justification.trim() : "";
    if (justification.length < 20) {
      return {
        success: false,
        error: "justification must be at least 20 characters",
      };
    }

    const expectedAuthor = typeof message.author === "string" && message.author.trim().length > 0 ? message.author.trim() : dismisser;
    if (expectedAuthor !== dismisser) {
      return {
        success: false,
        error: `author must match the current workflow actor (${dismisser})`,
      };
    }

    const targetResult = resolveTarget({
      targetConfig,
      item: message,
      context,
      itemType: "pull request review dismissal",
      // In resolveTarget conventions, supportsPR=false means PR-only handlers.
      supportsPR: false,
    });
    if (!targetResult.success) {
      return {
        success: false,
        error: targetResult.error,
      };
    }
    const pullRequestNumber = targetResult.number;

    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "pull request review dismissal");
    if (!repoResult.success) {
      return {
        success: false,
        error: repoResult.error,
      };
    }
    const { owner, repo } = repoResult.repoParts;

    const filterResult = await checkRequiredFilter(githubClient, repoResult.repoParts, pullRequestNumber, requiredLabels, requiredTitlePrefix, HANDLER_TYPE);
    if (filterResult) return filterResult;

    if (isStaged) {
      logStagedPreviewInfo(`Would dismiss review #${reviewId} on PR #${pullRequestNumber} (${owner}/${repo}) as ${dismisser}`);
      processedCount++;
      return {
        success: true,
        staged: true,
        review_id: reviewId,
        pull_request_number: pullRequestNumber,
        repo: `${owner}/${repo}`,
        author: expectedAuthor,
      };
    }

    try {
      const { data: review } = await githubClient.rest.pulls.getReview({
        owner,
        repo,
        pull_number: pullRequestNumber,
        review_id: reviewId,
      });

      const reviewAuthorLogin = review?.user?.login;
      if (typeof reviewAuthorLogin !== "string" || reviewAuthorLogin.trim() === "") {
        return {
          success: false,
          error: "review author is unavailable for dismissal validation",
        };
      }
      const reviewAuthor = reviewAuthorLogin.trim();
      if (reviewAuthor !== expectedAuthor) {
        return {
          success: false,
          error: `review author (${reviewAuthor || "unknown"}) must match dismisser (${dismisser})`,
        };
      }

      const { data: dismissed } = await githubClient.rest.pulls.dismissReview({
        owner,
        repo,
        pull_number: pullRequestNumber,
        review_id: reviewId,
        message: justification,
      });

      processedCount++;
      return {
        success: true,
        review_id: reviewId,
        pull_request_number: pullRequestNumber,
        repo: `${owner}/${repo}`,
        author: reviewAuthor,
        review_url: dismissed?.html_url || review?.html_url,
      };
    } catch (error) {
      return {
        success: false,
        error: getErrorMessage(error),
      };
    }
  };
}

module.exports = { main, HANDLER_TYPE };
