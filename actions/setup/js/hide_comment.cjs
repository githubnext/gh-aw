// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Hide a comment using the GraphQL API.
 * @param {any} github - GitHub GraphQL instance
 * @param {string} nodeId - Comment node ID (e.g., 'IC_kwDOABCD123456')
 * @param {string} reason - Reason for hiding (default: spam)
 * @returns {Promise<{id: string, isMinimized: boolean}>} Hidden comment details
 */
async function hideComment(github, nodeId, reason = "spam") {
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

async function main(config = {}) {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Parse allowed reasons from config object
  const allowedReasons = config.allowed_reasons || null;
  if (allowedReasons && allowedReasons.length > 0) {
    core.info(`Allowed reasons for hiding: [${allowedReasons.join(", ")}]`);
  }

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all hide-comment items
  const hideCommentItems = result.items.filter(/** @param {any} item */ item => item.type === "hide_comment");
  if (hideCommentItems.length === 0) {
    core.info("No hide-comment items found in agent output");
    return;
  }

  core.info(`Found ${hideCommentItems.length} hide-comment item(s)`);

  // If in staged mode, emit step summary instead of hiding comments
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Hide Comments Preview\n\n";
    summaryContent += "The following comments would be hidden if staged mode was disabled:\n\n";

    for (let i = 0; i < hideCommentItems.length; i++) {
      const item = hideCommentItems[i];
      const reason = item.reason || "spam";
      summaryContent += `### Comment ${i + 1}\n`;
      summaryContent += `**Node ID**: ${item.comment_id}\n`;
      summaryContent += `**Action**: Would be hidden as ${reason}\n`;
      summaryContent += "\n";
    }

    core.summary.addRaw(summaryContent).write();
    return;
  }

  // Process each hide-comment item
  for (const item of hideCommentItems) {
    try {
      const commentId = item.comment_id;
      if (!commentId || typeof commentId !== "string") {
        throw new Error("comment_id is required and must be a string (GraphQL node ID)");
      }

      const reason = item.reason || "spam";

      // Normalize reason to uppercase for GitHub API
      const normalizedReason = reason.toUpperCase();

      // Validate reason against allowed reasons if specified (case-insensitive)
      if (allowedReasons && allowedReasons.length > 0) {
        const normalizedAllowedReasons = allowedReasons.map(r => r.toUpperCase());
        if (!normalizedAllowedReasons.includes(normalizedReason)) {
          core.warning(`Reason "${reason}" is not in allowed-reasons list [${allowedReasons.join(", ")}]. Skipping comment ${commentId}.`);
          continue;
        }
      }

      core.info(`Hiding comment: ${commentId} (reason: ${normalizedReason})`);

      const hideResult = await hideComment(github, commentId, normalizedReason);

      if (hideResult.isMinimized) {
        core.info(`Successfully hidden comment: ${commentId}`);
        core.setOutput("comment_id", commentId);
        core.setOutput("is_hidden", "true");
      } else {
        throw new Error(`Failed to hide comment: ${commentId}`);
      }
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to hide comment: ${errorMessage}`);
      core.setFailed(`Failed to hide comment: ${errorMessage}`);
      return;
    }
  }
}

module.exports = { main };
