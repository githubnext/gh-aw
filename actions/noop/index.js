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
 * Main function to handle noop safe output
 * No-op is a fallback output type that logs messages for transparency
 * without taking any GitHub API actions
 */
async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all noop items
  const noopItems = result.items.filter(/** @param {any} item */ item => item.type === "noop");
  if (noopItems.length === 0) {
    core.info("No noop items found in agent output");
    return;
  }

  core.info(`Found ${noopItems.length} noop item(s)`);

  // If in staged mode, emit step summary instead of logging
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: No-Op Messages Preview\n\n";
    summaryContent += "The following messages would be logged if staged mode was disabled:\n\n";

    for (let i = 0; i < noopItems.length; i++) {
      const item = noopItems[i];
      summaryContent += `### Message ${i + 1}\n`;
      summaryContent += `${item.message}\n\n`;
      summaryContent += "---\n\n";
    }

    await core.summary.addRaw(summaryContent).write();
    core.info("📝 No-op message preview written to step summary");
    return;
  }

  // Process each noop item - just log the messages for transparency
  let summaryContent = "\n\n## No-Op Messages\n\n";
  summaryContent += "The following messages were logged for transparency:\n\n";

  for (let i = 0; i < noopItems.length; i++) {
    const item = noopItems[i];
    core.info(`No-op message ${i + 1}: ${item.message}`);
    summaryContent += `- ${item.message}\n`;
  }

  // Write summary for all noop messages
  await core.summary.addRaw(summaryContent).write();

  // Export the first noop message for use in add-comment default reporting
  if (noopItems.length > 0) {
    core.setOutput("noop_message", noopItems[0].message);
    core.exportVariable("GH_AW_NOOP_MESSAGE", noopItems[0].message);
  }

  core.info(`Successfully processed ${noopItems.length} noop message(s)`);
}

await main();
