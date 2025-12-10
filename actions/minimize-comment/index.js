// @ts-check
/// <reference types="@actions/github-script" />

// === Inlined from ./load_agent_output.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

/**
 * Maximum content length to log for debugging purposes
 * @type {number}
 */
const MAX_LOG_CONTENT_LENGTH = 10000;

/**
 * Truncate content for logging if it exceeds the maximum length
 * @param {string} content - Content to potentially truncate
 * @returns {string} Truncated content with indicator if truncated
 */
function truncateForLogging(content) {
  if (content.length <= MAX_LOG_CONTENT_LENGTH) {
    return content;
  }
  return content.substring(0, MAX_LOG_CONTENT_LENGTH) + `\n... (truncated, total length: ${content.length})`;
}

/**
 * Load and parse agent output from the GH_AW_AGENT_OUTPUT file
 *
 * This utility handles the common pattern of:
 * 1. Reading the GH_AW_AGENT_OUTPUT environment variable
 * 2. Loading the file content
 * 3. Validating the JSON structure
 * 4. Returning parsed items array
 *
 * @returns {{
 *   success: true,
 *   items: any[]
 * } | {
 *   success: false,
 *   items?: undefined,
 *   error?: string
 * }} Result object with success flag and items array (if successful) or error message
 */
function loadAgentOutput() {
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;

  // No agent output file specified
  if (!agentOutputFile) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    return { success: false };
  }

  // Read agent output from file
  let outputContent;
  try {
    outputContent = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    const errorMessage = `Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`;
    core.error(errorMessage);
    return { success: false, error: errorMessage };
  }

  // Check for empty content
  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return { success: false };
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    const errorMessage = `Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`;
    core.error(errorMessage);
    core.info(`Failed to parse content:\n${truncateForLogging(outputContent)}`);
    return { success: false, error: errorMessage };
  }

  // Validate items array exists
  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    core.info(`Parsed content: ${truncateForLogging(JSON.stringify(validatedOutput))}`);
    return { success: false };
  }

  return { success: true, items: validatedOutput.items };
}

// === End of ./load_agent_output.cjs ===


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
