// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");

/**
 * Minimize (hide) a comment using the GraphQL API.
 * @param {any} github - GitHub GraphQL instance
 * @param {string} nodeId - Comment node ID (e.g., 'IC_kwDOABCD123456')
 * @returns {Promise<{id: string, isMinimized: boolean}>} Minimized comment details
 */
async function minimizeComment(github, nodeId) {
  const query = /* GraphQL */ `
    mutation ($nodeId: ID!) {
      minimizeComment(input: { subjectId: $nodeId, classifier: SPAM }) {
        minimizedComment {
          isMinimized
        }
      }
    }
  `;

  const result = await github.graphql(query, { nodeId });

  return {
    id: nodeId,
    isMinimized: result.minimizeComment.minimizedComment.isMinimized,
  };
}

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all minimize-comment items
  const minimizeCommentItems = result.items.filter(/** @param {any} item */ item => item.type === "minimize_comment");
  if (minimizeCommentItems.length === 0) {
    core.info("No minimize-comment items found in agent output");
    return;
  }

  core.info(`Found ${minimizeCommentItems.length} minimize-comment item(s)`);

  // If in staged mode, emit step summary instead of minimizing comments
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Minimize Comments Preview\n\n";
    summaryContent += "The following comments would be minimized if staged mode was disabled:\n\n";

    for (let i = 0; i < minimizeCommentItems.length; i++) {
      const item = minimizeCommentItems[i];
      summaryContent += `### Comment ${i + 1}\n`;
      summaryContent += `**Node ID**: ${item.comment_id}\n`;
      summaryContent += `**Action**: Would be minimized as SPAM\n`;
      summaryContent += "\n";
    }

    core.summary.addRaw(summaryContent).write();
    return;
  }

  // Process each minimize-comment item
  for (const item of minimizeCommentItems) {
    try {
      const commentId = item.comment_id;
      if (!commentId || typeof commentId !== "string") {
        throw new Error("comment_id is required and must be a string (GraphQL node ID)");
      }

      core.info(`Minimizing comment: ${commentId}`);

      const minimizeResult = await minimizeComment(github, commentId);

      if (minimizeResult.isMinimized) {
        core.info(`Successfully minimized comment: ${commentId}`);
        core.setOutput("comment_id", commentId);
        core.setOutput("is_minimized", "true");
      } else {
        throw new Error(`Failed to minimize comment: ${commentId}`);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to minimize comment: ${errorMessage}`);
      core.setFailed(`Failed to minimize comment: ${errorMessage}`);
      return;
    }
  }
}

// Call the main function
await main();
