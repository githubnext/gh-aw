// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

/**
 * Process agent output items with common safe output handling logic.
 * This function centralizes the duplicate pattern of:
 * 1. Loading agent output from file
 * 2. Filtering items by type
 * 3. Handling staged mode with preview generation
 * 4. Early return on empty item lists
 *
 * @param {Object} options - Configuration options
 * @param {string} options.itemType - The type of items to filter (e.g., "create_issue", "add_comment")
 * @param {Record<string, string>} [options.outputs] - Output keys to initialize (e.g., {issue_number: "", issue_url: ""})
 * @param {Object} [options.stagedPreview] - Staged mode preview configuration
 * @param {string} options.stagedPreview.title - Title for the staged preview
 * @param {string} options.stagedPreview.description - Description for the staged preview
 * @param {(item: any, index: number) => string} options.stagedPreview.renderItem - Function to render each item
 * @param {(items: any[]) => Promise<void>} [options.processItems] - Function to process items in live mode
 * @returns {Promise<{processed: boolean, items?: any[], staged: boolean}>} Result object
 */
async function processAgentOutputItems(options) {
  const { itemType, outputs, stagedPreview, processItems } = options;

  // Initialize outputs to empty strings
  if (outputs) {
    for (const [key, value] of Object.entries(outputs)) {
      core.setOutput(key, value);
    }
  }

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Load agent output
  const result = loadAgentOutput();
  if (!result.success) {
    return { processed: false, staged: isStaged };
  }

  // Filter items by type
  const items = result.items.filter(item => item.type === itemType);
  
  // Check if any items found
  if (items.length === 0) {
    core.info(`No ${itemType} items found in agent output`);
    return { processed: false, items: [], staged: isStaged };
  }

  core.info(`Found ${items.length} ${itemType} item(s)`);

  // Handle staged mode
  if (isStaged) {
    if (stagedPreview) {
      await generateStagedPreview({
        title: stagedPreview.title,
        description: stagedPreview.description,
        items: items,
        renderItem: stagedPreview.renderItem,
      });
    }
    return { processed: true, items, staged: true };
  }

  // Process items in live mode if handler provided
  if (processItems) {
    await processItems(items);
  }

  return { processed: true, items, staged: false };
}

module.exports = { processAgentOutputItems };
