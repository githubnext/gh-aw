// Embedded files for bundling
const FILES = {
    "load_agent_output.cjs": "// @ts-check\n/// \u003creference types=\"@actions/github-script\" /\u003e\n\nconst fs = require(\"fs\");\n\n/**\n * Maximum content length to log for debugging purposes\n * @type {number}\n */\nconst MAX_LOG_CONTENT_LENGTH = 10000;\n\n/**\n * Truncate content for logging if it exceeds the maximum length\n * @param {string} content - Content to potentially truncate\n * @returns {string} Truncated content with indicator if truncated\n */\nfunction truncateForLogging(content) {\n  if (content.length \u003c= MAX_LOG_CONTENT_LENGTH) {\n    return content;\n  }\n  return content.substring(0, MAX_LOG_CONTENT_LENGTH) + `\\n... (truncated, total length: ${content.length})`;\n}\n\n/**\n * Load and parse agent output from the GH_AW_AGENT_OUTPUT file\n *\n * This utility handles the common pattern of:\n * 1. Reading the GH_AW_AGENT_OUTPUT environment variable\n * 2. Loading the file content\n * 3. Validating the JSON structure\n * 4. Returning parsed items array\n *\n * @returns {{\n *   success: true,\n *   items: any[]\n * } | {\n *   success: false,\n *   items?: undefined,\n *   error?: string\n * }} Result object with success flag and items array (if successful) or error message\n */\nfunction loadAgentOutput() {\n  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;\n\n  // No agent output file specified\n  if (!agentOutputFile) {\n    core.info(\"No GH_AW_AGENT_OUTPUT environment variable found\");\n    return { success: false };\n  }\n\n  // Read agent output from file\n  let outputContent;\n  try {\n    outputContent = fs.readFileSync(agentOutputFile, \"utf8\");\n  } catch (error) {\n    const errorMessage = `Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`;\n    core.error(errorMessage);\n    return { success: false, error: errorMessage };\n  }\n\n  // Check for empty content\n  if (outputContent.trim() === \"\") {\n    core.info(\"Agent output content is empty\");\n    return { success: false };\n  }\n\n  core.info(`Agent output content length: ${outputContent.length}`);\n\n  // Parse the validated output JSON\n  let validatedOutput;\n  try {\n    validatedOutput = JSON.parse(outputContent);\n  } catch (error) {\n    const errorMessage = `Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`;\n    core.error(errorMessage);\n    core.info(`Failed to parse content:\\n${truncateForLogging(outputContent)}`);\n    return { success: false, error: errorMessage };\n  }\n\n  // Validate items array exists\n  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {\n    core.info(\"No valid items found in agent output\");\n    core.info(`Parsed content: ${truncateForLogging(JSON.stringify(validatedOutput))}`);\n    return { success: false };\n  }\n\n  return { success: true, items: validatedOutput.items };\n}\n\nmodule.exports = { loadAgentOutput, truncateForLogging, MAX_LOG_CONTENT_LENGTH };\n"
  };

// Helper to load embedded files
function requireFile(filename) {
  const content = FILES[filename];
  if (!content) {
    throw new Error(`File not found: ${filename}`);
  }
  const exports = {};
  const module = { exports };
  const func = new Function('exports', 'module', 'require', content);
  func(exports, module, requireFile);
  return module.exports;
}

// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = requireFile('load_agent_output.cjs');

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
