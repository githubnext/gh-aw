// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { getPRNumber } = require("./update_context_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { resolveTargetRepoConfig, validateTargetRepo } = require("./repo_helpers.cjs");

/**
 * Type constant for handler identification
 */
const HANDLER_TYPE = "resolve_pull_request_review_thread";

/**
 * Look up a review thread's parent PR number and repository via the GraphQL API.
 * Used to validate the thread before resolving.
 * @param {any} github - GitHub GraphQL instance
 * @param {string} threadId - Review thread node ID (e.g., 'PRRT_kwDOABCD...')
 * @returns {Promise<{prNumber: number, repoNameWithOwner: string|null}|null>} The PR number and repo, or null if not found
 */
async function getThreadPullRequestInfo(github, threadId) {
  const query = /* GraphQL */ `
    query ($threadId: ID!) {
      node(id: $threadId) {
        ... on PullRequestReviewThread {
          pullRequest {
            number
            repository {
              nameWithOwner
            }
          }
        }
      }
    }
  `;

  const result = await github.graphql(query, { threadId });

  const pullRequest = result?.node?.pullRequest;
  if (!pullRequest) {
    return null;
  }

  return {
    prNumber: pullRequest.number,
    repoNameWithOwner: pullRequest.repository?.nameWithOwner ?? null,
  };
}

/**
 * Resolve a pull request review thread using the GraphQL API.
 * @param {any} github - GitHub GraphQL instance
 * @param {string} threadId - Review thread node ID (e.g., 'PRRT_kwDOABCD...')
 * @returns {Promise<{threadId: string, isResolved: boolean}>} Resolved thread details
 */
async function resolveReviewThreadAPI(github, threadId) {
  const query = /* GraphQL */ `
    mutation ($threadId: ID!) {
      resolveReviewThread(input: { threadId: $threadId }) {
        thread {
          id
          isResolved
        }
      }
    }
  `;

  const result = await github.graphql(query, { threadId });

  return {
    threadId: result.resolveReviewThread.thread.id,
    isResolved: result.resolveReviewThread.thread.isResolved,
  };
}

/**
 * Main handler factory for resolve_pull_request_review_thread
 * Returns a message handler function that processes individual resolve messages.
 *
 * By default, resolution is scoped to the triggering PR only. When target-repo or
 * allowed-repos are specified, cross-repository thread resolution is supported.
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const maxCount = config.max || 10;
  const resolveTarget = config.target || "triggering";
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);

  // Whether the user explicitly configured cross-repo targeting.
  // defaultTargetRepo always has a value (falls back to context.repo), so we check
  // the raw config keys to distinguish user-configured from default.
  const hasExplicitTargetConfig = !!(config["target-repo"] || config.allowed_repos?.length > 0);

  const authClient = await createAuthenticatedGitHubClient(config);

  // Determine the triggering PR number from context
  const triggeringPRNumber = getPRNumber(context.payload);

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Resolve PR review thread configuration: max=${maxCount}, target=${resolveTarget}, triggeringPR=${triggeringPRNumber || "none"}`);
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single resolve_pull_request_review_thread message
   * @param {Object} message - The resolve message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleResolvePRReviewThread(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping resolve_pull_request_review_thread: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    try {
      // Validate required fields
      const threadId = item.thread_id;
      if (!threadId || typeof threadId !== "string" || threadId.trim().length === 0) {
        core.warning('Missing or invalid required field "thread_id" in resolve message');
        return {
          success: false,
          error: 'Missing or invalid required field "thread_id" - must be a non-empty string (GraphQL node ID)',
        };
      }

      // Look up the thread's PR number and repository
      const threadInfo = await getThreadPullRequestInfo(authClient, threadId);
      if (threadInfo === null) {
        core.warning(`Review thread not found or not a PullRequestReviewThread: ${threadId}`);
        return {
          success: false,
          error: `Review thread not found: ${threadId}`,
        };
      }

      const { prNumber: threadPRNumber, repoNameWithOwner: threadRepo } = threadInfo;

      // When the user explicitly configured target-repo or allowed-repos, validate the thread's
      // repository using validateTargetRepo (supports wildcards like "*", "org/*").
      // Otherwise, fall back to the legacy behavior of scoping to the triggering PR only.
      if (hasExplicitTargetConfig) {
        // Cross-repo mode: validate thread repo against configured repos (fail closed if missing)
        if (!threadRepo) {
          core.warning(`Could not determine repository for thread ${threadId}`);
          return {
            success: false,
            error: `Could not determine the repository for thread ${threadId}`,
          };
        }
        const repoValidation = validateTargetRepo(threadRepo, defaultTargetRepo, allowedRepos);
        if (!repoValidation.valid) {
          core.warning(`Thread ${threadId} belongs to repo ${threadRepo}, which is not in the allowed repos`);
          return {
            success: false,
            error: repoValidation.error,
          };
        }

        // Determine target PR number based on target config
        if (resolveTarget === "triggering") {
          if (!triggeringPRNumber) {
            core.warning("Cannot resolve review thread: not running in a pull request context");
            return {
              success: false,
              error: "Cannot resolve review threads outside of a pull request context",
            };
          }
          if (threadPRNumber !== triggeringPRNumber) {
            core.warning(`Thread ${threadId} belongs to PR #${threadPRNumber}, not triggering PR #${triggeringPRNumber}`);
            return {
              success: false,
              error: `Thread belongs to PR #${threadPRNumber}, but only threads on the triggering PR #${triggeringPRNumber} can be resolved`,
            };
          }
        } else if (resolveTarget !== "*") {
          // Explicit PR number target
          const targetPRNumber = parseInt(resolveTarget, 10);
          if (Number.isNaN(targetPRNumber) || targetPRNumber <= 0) {
            core.warning(`Invalid target PR number: '${resolveTarget}'`);
            return {
              success: false,
              error: `Invalid target: '${resolveTarget}' - must be 'triggering', '*', or a positive integer`,
            };
          }
          if (threadPRNumber !== targetPRNumber) {
            core.warning(`Thread ${threadId} belongs to PR #${threadPRNumber}, not target PR #${targetPRNumber}`);
            return {
              success: false,
              error: `Thread belongs to PR #${threadPRNumber}, but target is PR #${targetPRNumber}`,
            };
          }
        }
        // resolveTarget === "*": any PR in allowed repos — no further PR number check needed
      } else {
        // Default (legacy) mode: scope to triggering PR only
        if (!triggeringPRNumber) {
          core.warning("Cannot resolve review thread: not running in a pull request context");
          return {
            success: false,
            error: "Cannot resolve review threads outside of a pull request context",
          };
        }

        if (threadPRNumber !== triggeringPRNumber) {
          core.warning(`Thread ${threadId} belongs to PR #${threadPRNumber}, not triggering PR #${triggeringPRNumber}`);
          return {
            success: false,
            error: `Thread belongs to PR #${threadPRNumber}, but only threads on the triggering PR #${triggeringPRNumber} can be resolved`,
          };
        }
      }

      core.info(`Resolving review thread: ${threadId} (PR #${threadPRNumber}${threadRepo ? " in " + threadRepo : ""})`);

      // If in staged mode, preview without executing
      if (isStaged) {
        logStagedPreviewInfo(`Would resolve review thread ${threadId}`);
        return {
          success: true,
          staged: true,
          previewInfo: {
            thread_id: threadId,
            pr_number: threadPRNumber,
          },
        };
      }

      const resolveResult = await resolveReviewThreadAPI(authClient, threadId);

      if (resolveResult.isResolved) {
        core.info(`Successfully resolved review thread: ${threadId}`);
        return {
          success: true,
          thread_id: threadId,
          is_resolved: true,
        };
      } else {
        core.error(`Failed to resolve review thread: ${threadId}`);
        return {
          success: false,
          error: `Failed to resolve review thread: ${threadId}`,
        };
      }
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to resolve review thread: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main, HANDLER_TYPE };
