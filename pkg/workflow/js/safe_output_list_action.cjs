// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared helper for safe-output list actions (e.g., add_labels, add_reviewer).
 * Provides common functionality for staged preview, validation, and summary rendering.
 */

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { parseAllowedItems, resolveTarget } = require("./safe_output_helpers.cjs");
const { getSafeOutputConfig, validateMaxCount } = require("./safe_output_validator.cjs");

/**
 * @typedef {Object} ListActionConfig
 * @property {string} itemType - Type of item to look for in agent output (e.g., "add_labels", "add_reviewer")
 * @property {string} singularNoun - Singular form of the noun (e.g., "label", "reviewer")
 * @property {string} pluralNoun - Plural form of the noun (e.g., "labels", "reviewers")
 * @property {string} itemsField - Field name in the item containing the list (e.g., "labels", "reviewers")
 * @property {string} configKey - Key in safe output config (e.g., "add_labels", "add_reviewer")
 * @property {string} configAllowedField - Field name for allowed items in config (e.g., "allowed", "reviewers")
 * @property {string} envAllowedVar - Environment variable for allowed items (e.g., "GH_AW_LABELS_ALLOWED")
 * @property {string} envMaxCountVar - Environment variable for max count (e.g., "GH_AW_LABELS_MAX_COUNT")
 * @property {string} envTargetVar - Environment variable for target (e.g., "GH_AW_LABELS_TARGET")
 * @property {string} targetNumberField - Field name for target number in item (e.g., "item_number", "pull_request_number")
 * @property {boolean} supportsPR - Whether this action supports both issues and PRs
 * @property {string} stagedPreviewTitle - Title for staged preview (e.g., "Add Labels")
 * @property {string} stagedPreviewDescription - Description for staged preview
 * @property {(item: any) => string} renderStagedItem - Function to render item in staged preview
 * @property {(items: string[], contextType: string, itemNumber: number) => Promise<void>} applyAction - Function to apply the action
 * @property {string} outputField - Output field name (e.g., "labels_added", "reviewers_added")
 * @property {string} summaryTitle - Title for success summary (e.g., "Label Addition", "Reviewer Addition")
 * @property {(items: string[], contextType: string, itemNumber: number) => string} renderSuccessSummary - Function to render success summary
 * @property {(items: string[]) => {valid: boolean, value?: string[], error?: string}} [validateItems] - Optional custom validation function. When provided, it must handle max count limits internally. The shared helper will NOT apply max count limits when this is set.
 */

/**
 * Execute a list action with shared scaffolding
 * @param {ListActionConfig} config - Configuration for the list action
 * @returns {Promise<void>}
 */
async function executeListAction(config) {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const item = result.items.find(i => i.type === config.itemType);
  if (!item) {
    core.warning(`No ${config.itemType.replace("_", "-")} item found in agent output`);
    return;
  }
  core.info(`Found ${config.itemType.replace("_", "-")} item with ${item[config.itemsField]?.length || 0} ${config.pluralNoun}`);

  // Handle staged preview mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: config.stagedPreviewTitle,
      description: config.stagedPreviewDescription,
      items: [item],
      renderItem: config.renderStagedItem,
    });
    return;
  }

  // Get configuration from config.json
  const safeOutputConfig = getSafeOutputConfig(config.configKey);

  // Parse allowed items (from env or config)
  const allowedItems = parseAllowedItems(process.env[config.envAllowedVar]) || safeOutputConfig[config.configAllowedField];
  if (allowedItems) {
    core.info(`Allowed ${config.pluralNoun}: ${JSON.stringify(allowedItems)}`);
  } else {
    core.info(`No ${config.singularNoun} restrictions - any ${config.pluralNoun} are allowed`);
  }

  // Parse max count (env takes priority, then config)
  const maxCountResult = validateMaxCount(process.env[config.envMaxCountVar], safeOutputConfig.max);
  if (!maxCountResult.valid) {
    core.setFailed(maxCountResult.error);
    return;
  }
  const maxCount = maxCountResult.value;
  core.info(`Max count: ${maxCount}`);

  // Resolve target
  const targetConfig = process.env[config.envTargetVar] || "triggering";
  core.info(`${config.pluralNoun.charAt(0).toUpperCase() + config.pluralNoun.slice(1)} target configuration: ${targetConfig}`);

  const targetResult = resolveTarget({
    targetConfig,
    item,
    context,
    itemType: `${config.singularNoun} addition`,
    supportsPR: config.supportsPR,
  });

  if (!targetResult.success) {
    if (targetResult.shouldFail) {
      core.setFailed(targetResult.error);
    } else {
      core.info(targetResult.error);
    }
    return;
  }

  const itemNumber = targetResult.number;
  const contextType = targetResult.contextType;
  const requestedItems = item[config.itemsField] || [];
  core.info(`Requested ${config.pluralNoun}: ${JSON.stringify(requestedItems)}`);

  // Validate items using custom validation function or default
  let validationResult;
  if (config.validateItems) {
    validationResult = config.validateItems(requestedItems);
  } else {
    // Default validation: filter by allowed, sanitize, deduplicate
    validationResult = validateListItems(requestedItems, allowedItems, maxCount);
  }

  if (!validationResult.valid) {
    // If no valid items, log info and return gracefully instead of failing
    if (validationResult.error && validationResult.error.includes("No valid")) {
      core.info(`No ${config.pluralNoun} to add`);
      core.setOutput(config.outputField, "");
      await core.summary
        .addRaw(
          `
## ${config.summaryTitle}

No ${config.pluralNoun} were added (no valid ${config.pluralNoun} found in agent output).
`
        )
        .write();
      return;
    }
    // For other validation errors, fail the workflow
    core.setFailed(validationResult.error || `Invalid ${config.pluralNoun}`);
    return;
  }

  let uniqueItems = validationResult.value || [];

  // Apply max count limit only when using default validation (not custom validateItems).
  // Custom validators are responsible for handling max count limits internally.
  if (!config.validateItems && uniqueItems.length > maxCount) {
    core.info(`Too many ${config.pluralNoun}, keeping ${maxCount}`);
    uniqueItems = uniqueItems.slice(0, maxCount);
  }

  if (uniqueItems.length === 0) {
    core.info(`No ${config.pluralNoun} to add`);
    core.setOutput(config.outputField, "");
    await core.summary
      .addRaw(
        `
## ${config.summaryTitle}

No ${config.pluralNoun} were added (no valid ${config.pluralNoun} found in agent output).
`
      )
      .write();
    return;
  }

  core.info(`Adding ${uniqueItems.length} ${config.pluralNoun} to ${contextType} #${itemNumber}: ${JSON.stringify(uniqueItems)}`);

  try {
    await config.applyAction(uniqueItems, contextType, itemNumber);
    core.info(`Successfully added ${uniqueItems.length} ${config.pluralNoun} to ${contextType} #${itemNumber}`);
    core.setOutput(config.outputField, uniqueItems.join("\n"));

    const summaryContent = config.renderSuccessSummary(uniqueItems, contextType, itemNumber);
    await core.summary.addRaw(summaryContent).write();
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to add ${config.pluralNoun}: ${errorMessage}`);
    core.setFailed(`Failed to add ${config.pluralNoun}: ${errorMessage}`);
  }
}

/**
 * Default validation for list items: filter by allowed, sanitize, deduplicate
 * @param {any[]} items - Items to validate
 * @param {string[]|undefined} allowedItems - Allowed items list
 * @param {number} maxCount - Maximum number of items
 * @returns {{valid: boolean, value?: string[], error?: string}} Validation result
 */
function validateListItems(items, allowedItems, maxCount) {
  if (!items || !Array.isArray(items)) {
    return { valid: false, error: "items must be an array" };
  }

  // Filter by allowed items if configured
  let validItems = items;
  if (allowedItems && allowedItems.length > 0) {
    validItems = items.filter(item => allowedItems.includes(item));
  }

  // Sanitize and deduplicate
  const uniqueItems = validItems
    .filter(item => item != null && item !== false && item !== 0)
    .map(item => String(item).trim())
    .filter(item => item)
    .filter((item, index, arr) => arr.indexOf(item) === index);

  // Apply max count limit
  if (uniqueItems.length > maxCount) {
    return { valid: true, value: uniqueItems.slice(0, maxCount) };
  }

  if (uniqueItems.length === 0) {
    return { valid: false, error: "No valid items found after filtering" };
  }

  return { valid: true, value: uniqueItems };
}

/**
 * Generate a markdown list from items
 * @param {string[]} items - Items to render as markdown list
 * @returns {string} Markdown list
 */
function renderMarkdownList(items) {
  return items.map(item => `- \`${item}\``).join("\n");
}

module.exports = {
  executeListAction,
  validateListItems,
  renderMarkdownList,
};
