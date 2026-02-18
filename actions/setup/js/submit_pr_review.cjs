// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { resolveTarget } = require("./safe_output_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "submit_pull_request_review";

/** @type {Set<string>} Valid review event types */
const VALID_EVENTS = new Set(["APPROVE", "REQUEST_CHANGES", "COMMENT"]);

/**
 * Main handler factory for submit_pull_request_review
 * Returns a message handler that stores review metadata (body and event)
 * in the shared PR review buffer. The actual review submission happens
 * during the handler manager's finalization step.
 *
 * The PR review buffer instance is passed via config._prReviewBuffer.
 *
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const maxCount = config.max || 1;
  const targetConfig = config.target || "triggering";
  const buffer = config._prReviewBuffer;

  if (!buffer) {
    core.warning("submit_pull_request_review: No PR review buffer provided in config");
    return async function handleSubmitPRReview() {
      return { success: false, error: "No PR review buffer available" };
    };
  }

  core.info(`Submit PR review handler initialized: max=${maxCount}, target=${targetConfig}`);

  let processedCount = 0;

  /**
   * Message handler that stores review metadata
   * @param {Object} message - The submit_pull_request_review message
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs
   * @returns {Promise<Object>} Result with success status
   */
  return async function handleSubmitPRReview(message, resolvedTemporaryIds) {
    if (processedCount >= maxCount) {
      core.warning(`Skipping submit_pull_request_review: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    // Validate event field — default to COMMENT when not provided
    const event = message.event ? message.event.toUpperCase() : "COMMENT";
    if (!VALID_EVENTS.has(event)) {
      core.warning(`Invalid review event: ${message.event}. Must be one of: APPROVE, REQUEST_CHANGES, COMMENT`);
      return {
        success: false,
        error: `Invalid review event: ${message.event}. Must be one of: APPROVE, REQUEST_CHANGES, COMMENT`,
      };
    }

    // Body is required for REQUEST_CHANGES per GitHub API docs;
    // optional for APPROVE and COMMENT
    const body = message.body || "";
    if (event === "REQUEST_CHANGES" && !body) {
      core.warning("Review body is required for REQUEST_CHANGES");
      return {
        success: false,
        error: "Review body is required for REQUEST_CHANGES",
      };
    }

    // Only increment after validation passes
    processedCount++;

    core.info(`Setting review metadata: event=${event}, bodyLength=${body.length}`);

    // Store the review metadata in the shared buffer
    buffer.setReviewMetadata(body, event);

    // Ensure review context is set for body-only reviews (no inline comments).
    // If create_pull_request_review_comment already set context, this is a no-op.
    // Use target config as single source of truth (same as add_comment): resolveTarget first, then use payload PR only when it matches.
    if (!buffer.getReviewContext()) {
      const targetResult = resolveTarget({
        targetConfig,
        item: message,
        context,
        itemType: "PR review",
        supportsPR: false,
        supportsIssue: false,
      });

      if (!targetResult.success) {
        if (targetResult.shouldFail) {
          core.warning(`Could not resolve PR for review context: ${targetResult.error}`);
        }
      } else if (targetResult.number) {
        const prNum = targetResult.number;
        const repo = `${context.repo.owner}/${context.repo.repo}`;
        const repoParts = { owner: context.repo.owner, repo: context.repo.repo };
        const payloadPR = context.payload?.pull_request;
        const usePayloadPR = payloadPR && payloadPR.number === prNum && payloadPR.head?.sha;

        if (usePayloadPR) {
          buffer.setReviewContext({
            repo,
            repoParts,
            pullRequestNumber: payloadPR.number,
            pullRequest: payloadPR,
          });
          core.info(`Set review context from triggering PR: ${repo}#${payloadPR.number}`);
        } else {
          try {
            const { data: fetchedPR } = await github.rest.pulls.get({
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: prNum,
            });
            if (fetchedPR?.head?.sha) {
              buffer.setReviewContext({
                repo,
                repoParts,
                pullRequestNumber: fetchedPR.number,
                pullRequest: fetchedPR,
              });
              core.info(`Set review context from target: ${repo}#${fetchedPR.number}`);
            } else {
              core.warning("Fetched PR missing head.sha - cannot set review context");
            }
          } catch (fetchErr) {
            core.warning(`Could not fetch PR #${prNum} for review context: ${getErrorMessage(fetchErr)}`);
          }
        }
      }
    }

    return {
      success: true,
      event: event,
      body_length: body.length,
    };
  };
}

module.exports = { main };
