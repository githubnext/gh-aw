// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");

/**
 * Type constant for handler identification
 */
const HANDLER_TYPE = "hide_comment";

/**
 * Hide a comment using the GraphQL API.
 * @param {any} github - GitHub GraphQL instance
 * @param {string} nodeId - Comment node ID (e.g., 'IC_kwDOABCD123456')
 * @param {string} reason - Reason for hiding (default: spam)
 * @returns {Promise<{id: string, isMinimized: boolean}>} Hidden comment details
 */
async function hideCommentAPI(github, nodeId, reason = "spam") {
  const query = /* GraphQL */ `
    mutation ($nodeId: ID!, $classifier: ReportedContentClassifiers!) {
      minimizeComment(input: { subjectId: $nodeId, classifier: $classifier }) {
        minimizedComment {
          isMinimized
        }
      }
    }
  `;

  const result = await github.graphql(query, { nodeId, classifier: reason });

  return {
    id: nodeId,
    isMinimized: result.minimizeComment.minimizedComment.isMinimized,
  };
}

/**
 * Resolve a safe-output comment_id into a GraphQL node ID.
 * Supports both GraphQL node IDs and numeric REST comment IDs.
 * @param {any} github - GitHub client
 * @param {{owner?: string, repo?: string}|null|undefined} repoContext - Repository context
 * @param {string|number} commentId - GraphQL node ID or numeric REST comment ID
 * @returns {Promise<string>} GraphQL node ID
 */
async function resolveCommentNodeId(github, repoContext, commentId) {
  if (typeof commentId === "string") {
    const trimmed = commentId.trim();
    if (!trimmed) {
      throw new Error("comment_id is required");
    }

    // GraphQL node IDs (e.g., IC_kwDOABCD123456) can be used directly.
    if (!/^\d+$/.test(trimmed)) {
      return trimmed;
    }

    commentId = Number.parseInt(trimmed, 10);
  }

  if (!Number.isInteger(commentId) || commentId <= 0) {
    throw new Error("comment_id must be a GraphQL node ID string or a positive numeric REST comment ID");
  }

  if (!repoContext || !repoContext.owner || !repoContext.repo) {
    throw new Error("Unable to resolve numeric comment_id: repository context (owner/repo) is not available");
  }

  const comment = await github.rest.issues.getComment({
    owner: repoContext.owner,
    repo: repoContext.repo,
    comment_id: commentId,
  });

  const nodeId = comment && comment.data ? comment.data.node_id : null;
  if (!nodeId || typeof nodeId !== "string") {
    throw new Error(`Failed to resolve GraphQL node ID for comment_id ${commentId}: comment not found or node_id unavailable`);
  }

  return nodeId;
}

/**
 * Main handler factory for hide_comment
 * Returns a message handler function that processes individual hide_comment messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedReasons = config.allowed_reasons || [];
  const maxCount = config.max || 5;
  const githubClient = await createAuthenticatedGitHubClient(config);

  // Check if we're in staged mode
  const isStaged = isStagedMode(config);

  core.info(`Hide comment configuration: max=${maxCount}`);
  if (allowedReasons.length > 0) {
    core.info(`Allowed reasons: ${allowedReasons.join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single hide_comment message
   * @param {Object} message - The hide_comment message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleHideComment(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping hide_comment: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    try {
      const commentId = message.comment_id;
      if (commentId === undefined || commentId === null || (typeof commentId === "string" && !commentId.trim())) {
        core.warning("comment_id is required");
        return {
          success: false,
          error: "comment_id is required",
        };
      }

      // Normalize reason to uppercase for GitHub API
      const normalizedReason = (message.reason || "SPAM").toUpperCase();

      // Validate reason against allowed reasons if specified (case-insensitive)
      if (allowedReasons.length > 0) {
        const normalizedAllowedReasons = allowedReasons.map(r => r.toUpperCase());
        if (!normalizedAllowedReasons.includes(normalizedReason)) {
          core.warning(`Reason "${message.reason}" is not in allowed-reasons list [${allowedReasons.join(", ")}]. Skipping comment ${commentId}.`);
          return {
            success: false,
            error: `Reason "${message.reason}" is not in allowed-reasons list`,
          };
        }
      }

      core.info(`Hiding comment: ${commentId} (reason: ${normalizedReason})`);

      // If in staged mode, preview without executing
      if (isStaged) {
        logStagedPreviewInfo(`Would hide comment ${commentId}`);
        return {
          success: true,
          staged: true,
          previewInfo: {
            commentId,
            reason: normalizedReason,
          },
        };
      }

      const resolvedNodeId = await resolveCommentNodeId(githubClient, context && context.repo ? context.repo : null, commentId);
      const hideResult = await hideCommentAPI(githubClient, resolvedNodeId, normalizedReason);

      if (hideResult.isMinimized) {
        core.info(`Successfully hidden comment: ${resolvedNodeId}`);
        return {
          success: true,
          comment_id: resolvedNodeId,
          is_hidden: true,
        };
      } else {
        core.error(`Failed to hide comment: ${commentId}`);
        return {
          success: false,
          error: `Failed to hide comment: ${commentId}`,
        };
      }
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to hide comment: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main, HANDLER_TYPE };
