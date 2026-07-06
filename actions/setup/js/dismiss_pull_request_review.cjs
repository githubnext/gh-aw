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
 * @param {any[]} reviews
 * @param {string} expectedAuthor
 * @returns {any[]}
 */
function findAllDismissibleReviewsForActor(reviews, expectedAuthor) {
  if (!Array.isArray(reviews) || !expectedAuthor) return [];
  const expectedAuthorLower = expectedAuthor.toLowerCase();
  const dismissibleStates = new Set(["CHANGES_REQUESTED", "APPROVED"]);
  return reviews.filter(review => {
    const login = typeof review?.user?.login === "string" ? review.user.login.trim().toLowerCase() : "";
    const state = typeof review?.state === "string" ? review.state.trim().toUpperCase() : "";
    return login === expectedAuthorLower && dismissibleStates.has(state);
  });
}

/**
 * @param {any} pullRequest
 * @returns {boolean}
 */
function hasNoRequestedReviewersOrTeams(pullRequest) {
  const hasNoRequestedReviewers = Array.isArray(pullRequest?.requested_reviewers) && pullRequest.requested_reviewers.length === 0;
  const hasNoRequestedTeams = Array.isArray(pullRequest?.requested_teams) && pullRequest.requested_teams.length === 0;
  return hasNoRequestedReviewers && hasNoRequestedTeams;
}

/** Maximum number of pages (100 reviews/page) fetched when enumerating PR reviews. */
const MAX_REVIEW_PAGES = 10;

/**
 * @param {any} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {number} pullRequestNumber
 * @returns {Promise<{reviews: any[], truncated: boolean}>}
 */
async function listAllPullRequestReviews(githubClient, owner, repo, pullRequestNumber) {
  const all = [];
  let page = 1;
  while (page <= MAX_REVIEW_PAGES) {
    const { data } = await githubClient.rest.pulls.listReviews({
      owner,
      repo,
      pull_number: pullRequestNumber,
      per_page: 100,
      page,
    });
    if (!Array.isArray(data) || data.length === 0) break;
    all.push(...data);
    if (data.length < 100) break;
    page++;
  }
  return { reviews: all, truncated: page > MAX_REVIEW_PAGES };
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

    const rawReviewId = String(message.review_id ?? "").trim();
    const useAutoReviewId = rawReviewId === "" || rawReviewId.toLowerCase() === "auto";
    const parsedReviewId = Number.parseInt(rawReviewId, 10);
    if (!useAutoReviewId && (!Number.isInteger(parsedReviewId) || parsedReviewId <= 0)) {
      return {
        success: false,
        error: "review_id must be a positive integer or 'auto'",
      };
    }
    let reviewId = useAutoReviewId ? null : parsedReviewId;

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
      const previewReviewId = useAutoReviewId ? "auto" : reviewId;
      logStagedPreviewInfo(`Would dismiss ${useAutoReviewId ? "all actor-authored" : `review #${previewReviewId}`} on PR #${pullRequestNumber} (${owner}/${repo}) as ${dismisser}`);
      processedCount++;
      return {
        success: true,
        staged: true,
        review_id: previewReviewId,
        pull_request_number: pullRequestNumber,
        repo: `${owner}/${repo}`,
        author: expectedAuthor,
      };
    }

    try {
      if (useAutoReviewId) {
        const [{ data: pullRequest }, { reviews, truncated }] = await Promise.all([
          githubClient.rest.pulls.get({
            owner,
            repo,
            pull_number: pullRequestNumber,
          }),
          listAllPullRequestReviews(githubClient, owner, repo, pullRequestNumber),
        ]);

        const actorReviews = findAllDismissibleReviewsForActor(reviews, expectedAuthor);
        if (actorReviews.length === 0) {
          if (truncated) {
            return {
              success: false,
              error: `review_id=auto could not resolve: review history exceeds ${MAX_REVIEW_PAGES * 100} entries and was truncated; specify an explicit review_id instead`,
            };
          }
          const isBlockedWithoutReviewers = hasNoRequestedReviewersOrTeams(pullRequest) && pullRequest?.mergeable_state === "blocked";
          if (isBlockedWithoutReviewers) {
            return {
              success: false,
              error: "detected a degenerate review-required state (PR is blocked but has no requested reviewers/teams); no dismissible actor-authored review was found",
            };
          }
          return {
            success: false,
            error: `review_id=auto did not find a dismissible review authored by ${expectedAuthor}`,
          };
        }

        const dismissedReviews = [];
        for (const review of actorReviews) {
          const { data: dismissed } = await githubClient.rest.pulls.dismissReview({
            owner,
            repo,
            pull_number: pullRequestNumber,
            review_id: review.id,
            message: justification,
          });
          dismissedReviews.push({ review_id: review.id, review_url: dismissed?.html_url || review?.html_url });
        }

        processedCount++;
        return {
          success: true,
          review_ids: dismissedReviews.map(d => d.review_id),
          dismissed_count: dismissedReviews.length,
          pull_request_number: pullRequestNumber,
          repo: `${owner}/${repo}`,
          author: expectedAuthor,
        };
      }

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
