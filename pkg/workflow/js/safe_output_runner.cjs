// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

/**
 * @typedef {Object} SafeOutputRunnerOptions
 * @property {string} itemType - The type of item to filter for (e.g., "add_labels", "create_issue")
 * @property {string} itemTypePlural - Plural form for logging (e.g., "add-labels", "create-issue")
 * @property {boolean} [warnIfNotFound=false] - Whether to log warning (true) or info (false) when no items found
 * @property {string} [stagedTitle] - Title for staged mode preview (e.g., "Add Labels")
 * @property {string} [stagedDescription] - Description for staged mode preview
 * @property {(item: any, index: number) => string} [renderStagedItem] - Function to render each item in staged preview
 * @property {(items: any[]) => Promise<any>} processItems - Function to process the items when not in staged mode
 */

/**
 * @typedef {Object} SafeOutputRunnerResult
 * @property {boolean} handled - Whether the runner handled the request (returned early due to no items, staged mode, etc.)
 * @property {any} [result] - The result from processItems if it was called
 */

/**
 * Run a safe output script with common bootstrap logic.
 *
 * This handles the common pattern of:
 * 1. Loading agent output from file
 * 2. Filtering items by type
 * 3. Handling staged mode with preview
 * 4. Delegating to the actual processing function
 *
 * @param {SafeOutputRunnerOptions} options - Configuration options
 * @returns {Promise<SafeOutputRunnerResult>} Result indicating if handled and any processing result
 */
async function runSafeOutput(options) {
  const { itemType, itemTypePlural, warnIfNotFound = false, stagedTitle, stagedDescription, renderStagedItem, processItems } = options;

  // Load agent output
  const result = loadAgentOutput();
  if (!result.success) {
    return { handled: true };
  }

  // Filter items by type
  const items = result.items.filter(/** @param {any} item */ item => item.type === itemType);

  // Handle no items found
  if (items.length === 0) {
    const message = `No ${itemTypePlural} items found in agent output`;
    if (warnIfNotFound) {
      core.warning(message);
    } else {
      core.info(message);
    }
    return { handled: true };
  }

  core.info(`Found ${items.length} ${itemTypePlural} item(s)`);

  // Handle staged mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    if (stagedTitle && stagedDescription && renderStagedItem) {
      await generateStagedPreview({
        title: stagedTitle,
        description: stagedDescription,
        items: items,
        renderItem: renderStagedItem,
      });
    }
    return { handled: true };
  }

  // Process items
  const processResult = await processItems(items);
  return { handled: false, result: processResult };
}

/**
 * Convenience function for safe outputs that expect a single item.
 * Returns the first item after filtering.
 *
 * @typedef {Object} SingleItemRunnerOptions
 * @property {string} itemType - The type of item to filter for
 * @property {string} itemTypePlural - Plural form for logging
 * @property {boolean} [warnIfNotFound=true] - Whether to log warning when no items found
 * @property {string} [stagedTitle] - Title for staged mode preview
 * @property {string} [stagedDescription] - Description for staged mode preview
 * @property {(item: any, index: number) => string} [renderStagedItem] - Function to render item in staged preview
 * @property {(item: any) => Promise<any>} processSingleItem - Function to process the single item
 */

/**
 * Run a safe output script that expects a single item.
 *
 * @param {SingleItemRunnerOptions} options - Configuration options
 * @returns {Promise<SafeOutputRunnerResult>} Result indicating if handled and any processing result
 */
async function runSingleItemSafeOutput(options) {
  const { processSingleItem, ...restOptions } = options;

  return runSafeOutput({
    ...restOptions,
    warnIfNotFound: options.warnIfNotFound !== undefined ? options.warnIfNotFound : true,
    processItems: async items => {
      // For single item, just pass the first one
      const item = items[0];
      core.info(`Processing ${options.itemTypePlural} item`);
      return processSingleItem(item);
    },
  });
}

module.exports = {
  runSafeOutput,
  runSingleItemSafeOutput,
};
