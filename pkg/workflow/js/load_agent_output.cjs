// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

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
    core.setFailed(errorMessage);
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
    core.setFailed(errorMessage);
    return { success: false, error: errorMessage };
  }

  // Validate items array exists
  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return { success: false };
  }

  return { success: true, items: validatedOutput.items };
}

/**
 * Process agent output with common boilerplate handling.
 *
 * This utility encapsulates the common pattern used across all safe-output handlers:
 * 1. Load and validate agent output
 * 2. Filter items by type
 * 3. Handle empty results
 * 4. Handle staged mode with preview generation
 * 5. Return filtered items for processing
 *
 * @param {Object} options - Configuration options
 * @param {string} options.itemType - The type of items to filter (e.g., "create_issue", "add_labels")
 * @param {Object} [options.stagedPreview] - Configuration for staged mode preview (if omitted, uses inline preview)
 * @param {string} options.stagedPreview.title - Title for the staged preview
 * @param {string} options.stagedPreview.description - Description for the staged preview
 * @param {(item: any, index: number) => string} options.stagedPreview.renderItem - Function to render each item
 * @param {boolean} [options.useWarningForEmpty=false] - Use core.warning instead of core.info for empty results
 * @param {boolean} [options.findOne=false] - Filter to find a single item instead of all matching items
 *
 * @returns {Promise<{
 *   success: true,
 *   items: any[],
 *   isStaged: false
 * } | {
 *   success: true,
 *   items: any[],
 *   isStaged: true
 * } | {
 *   success: false,
 *   items?: undefined
 * }>} Result object with success flag, filtered items array, and staged mode indicator
 */
async function processAgentOutput(options) {
  const { itemType, stagedPreview, useWarningForEmpty = false, findOne = false } = options;

  // Load agent output
  const result = loadAgentOutput();
  if (!result.success) {
    return { success: false };
  }

  // Filter items by type
  let filteredItems;
  if (findOne) {
    const item = result.items.find(item => item.type === itemType);
    filteredItems = item ? [item] : [];
  } else {
    filteredItems = result.items.filter(item => item.type === itemType);
  }

  // Handle empty results
  if (filteredItems.length === 0) {
    // Convert underscores to hyphens for better readability in messages
    const readableType = itemType.replace(/_/g, "-");
    const message = `No ${readableType} items found in agent output`;
    if (useWarningForEmpty) {
      core.warning(message);
    } else {
      core.info(message);
    }
    return { success: false };
  }

  // Log found items (keep underscore format for consistency)
  core.info(`Found ${filteredItems.length} ${itemType} item(s)`);

  // Check for staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Handle staged mode with preview
  if (isStaged && stagedPreview) {
    const { generateStagedPreview } = require("./staged_preview.cjs");
    await generateStagedPreview({
      title: stagedPreview.title,
      description: stagedPreview.description,
      items: filteredItems,
      renderItem: stagedPreview.renderItem,
    });
    return { success: true, items: filteredItems, isStaged: true };
  }

  // Return filtered items for processing
  return { success: true, items: filteredItems, isStaged: false };
}

module.exports = { loadAgentOutput, processAgentOutput };
