// @ts-check
/// <reference types="@actions/github-script" />

// === Inlined from ./update_runner.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared update runner for safe-output scripts (update_issue, update_pull_request, etc.)
 *
 * This module depends on GitHub Actions environment globals provided by actions/github-script:
 * - core: @actions/core module for logging and outputs
 * - github: @octokit/rest instance for GitHub API calls
 * - context: GitHub Actions context with event payload and repository info
 *
 * @module update_runner
 */

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

// === Inlined from ./staged_preview.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generate a staged mode preview summary and write it to the step summary.
 *
 * @param {Object} options - Configuration options for the preview
 * @param {string} options.title - The main title for the preview (e.g., "Create Issues")
 * @param {string} options.description - Description of what would happen if staged mode was disabled
 * @param {Array<any>} options.items - Array of items to preview
 * @param {(item: any, index: number) => string} options.renderItem - Function to render each item as markdown
 * @returns {Promise<void>}
 */
async function generateStagedPreview(options) {
  const { title, description, items, renderItem } = options;

  let summaryContent = `## 🎭 Staged Mode: ${title} Preview\n\n`;
  summaryContent += `${description}\n\n`;

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    summaryContent += renderItem(item, i);
    summaryContent += "---\n\n";
  }

  try {
    await core.summary.addRaw(summaryContent).write();
    core.info(summaryContent);
    core.info(`📝 ${title} preview written to step summary`);
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

// === End of ./staged_preview.cjs ===


/**
 * @typedef {Object} UpdateRunnerConfig
 * @property {string} itemType - Type of item in agent output (e.g., "update_issue", "update_pull_request")
 * @property {string} displayName - Human-readable name (e.g., "issue", "pull request")
 * @property {string} displayNamePlural - Human-readable plural name (e.g., "issues", "pull requests")
 * @property {string} numberField - Field name for explicit number (e.g., "issue_number", "pull_request_number")
 * @property {string} outputNumberKey - Output key for number (e.g., "issue_number", "pull_request_number")
 * @property {string} outputUrlKey - Output key for URL (e.g., "issue_url", "pull_request_url")
 * @property {(eventName: string, payload: any) => boolean} isValidContext - Function to check if context is valid
 * @property {(payload: any) => number|undefined} getContextNumber - Function to get number from context payload
 * @property {boolean} supportsStatus - Whether this type supports status updates
 * @property {boolean} supportsOperation - Whether this type supports operation (append/prepend/replace)
 * @property {(item: any, index: number) => string} renderStagedItem - Function to render item for staged preview
 * @property {(github: any, context: any, targetNumber: number, updateData: any) => Promise<any>} executeUpdate - Function to execute the update API call
 * @property {(result: any) => string} getSummaryLine - Function to generate summary line for an updated item
 */

/**
 * Resolve the target number for an update operation
 * @param {Object} params - Resolution parameters
 * @param {string} params.updateTarget - Target configuration ("triggering", "*", or explicit number)
 * @param {any} params.item - Update item with optional explicit number field
 * @param {string} params.numberField - Field name for explicit number
 * @param {boolean} params.isValidContext - Whether current context is valid
 * @param {number|undefined} params.contextNumber - Number from triggering context
 * @param {string} params.displayName - Display name for error messages
 * @returns {{success: true, number: number} | {success: false, error: string}}
 */
function resolveTargetNumber(params) {
  const { updateTarget, item, numberField, isValidContext, contextNumber, displayName } = params;

  if (updateTarget === "*") {
    // For target "*", we need an explicit number from the update item
    const explicitNumber = item[numberField];
    if (explicitNumber) {
      const parsed = parseInt(explicitNumber, 10);
      if (isNaN(parsed) || parsed <= 0) {
        return { success: false, error: `Invalid ${numberField} specified: ${explicitNumber}` };
      }
      return { success: true, number: parsed };
    } else {
      return { success: false, error: `Target is "*" but no ${numberField} specified in update item` };
    }
  } else if (updateTarget && updateTarget !== "triggering") {
    // Explicit number specified in target
    const parsed = parseInt(updateTarget, 10);
    if (isNaN(parsed) || parsed <= 0) {
      return { success: false, error: `Invalid ${displayName} number in target configuration: ${updateTarget}` };
    }
    return { success: true, number: parsed };
  } else {
    // Default behavior: use triggering context
    if (isValidContext && contextNumber) {
      return { success: true, number: contextNumber };
    }
    return { success: false, error: `Could not determine ${displayName} number` };
  }
}

/**
 * Build update data based on allowed fields and provided values
 * @param {Object} params - Build parameters
 * @param {any} params.item - Update item with field values
 * @param {boolean} params.canUpdateStatus - Whether status updates are allowed
 * @param {boolean} params.canUpdateTitle - Whether title updates are allowed
 * @param {boolean} params.canUpdateBody - Whether body updates are allowed
 * @param {boolean} params.supportsStatus - Whether this type supports status
 * @returns {{hasUpdates: boolean, updateData: any, logMessages: string[]}}
 */
function buildUpdateData(params) {
  const { item, canUpdateStatus, canUpdateTitle, canUpdateBody, supportsStatus } = params;

  /** @type {any} */
  const updateData = {};
  let hasUpdates = false;
  const logMessages = [];

  // Handle status update (only for types that support it, like issues)
  if (supportsStatus && canUpdateStatus && item.status !== undefined) {
    if (item.status === "open" || item.status === "closed") {
      updateData.state = item.status;
      hasUpdates = true;
      logMessages.push(`Will update status to: ${item.status}`);
    } else {
      logMessages.push(`Invalid status value: ${item.status}. Must be 'open' or 'closed'`);
    }
  }

  // Handle title update
  if (canUpdateTitle && item.title !== undefined) {
    const trimmedTitle = typeof item.title === "string" ? item.title.trim() : "";
    if (trimmedTitle.length > 0) {
      updateData.title = trimmedTitle;
      hasUpdates = true;
      logMessages.push(`Will update title to: ${trimmedTitle}`);
    } else {
      logMessages.push("Invalid title value: must be a non-empty string");
    }
  }

  // Handle body update (basic - without operation logic)
  if (canUpdateBody && item.body !== undefined) {
    if (typeof item.body === "string") {
      updateData.body = item.body;
      hasUpdates = true;
      logMessages.push(`Will update body (length: ${item.body.length})`);
    } else {
      logMessages.push("Invalid body value: must be a string");
    }
  }

  return { hasUpdates, updateData, logMessages };
}

/**
 * Run the update workflow with the provided configuration
 * @param {UpdateRunnerConfig} config - Configuration for the update runner
 * @returns {Promise<any[]|undefined>} Array of updated items or undefined
 */
async function runUpdateWorkflow(config) {
  const {
    itemType,
    displayName,
    displayNamePlural,
    numberField,
    outputNumberKey,
    outputUrlKey,
    isValidContext,
    getContextNumber,
    supportsStatus,
    supportsOperation,
    renderStagedItem,
    executeUpdate,
    getSummaryLine,
  } = config;

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all update items
  const updateItems = result.items.filter(/** @param {any} item */ item => item.type === itemType);
  if (updateItems.length === 0) {
    core.info(`No ${itemType} items found in agent output`);
    return;
  }

  core.info(`Found ${updateItems.length} ${itemType} item(s)`);

  // If in staged mode, emit step summary instead of updating
  if (isStaged) {
    await generateStagedPreview({
      title: `Update ${displayNamePlural.charAt(0).toUpperCase() + displayNamePlural.slice(1)}`,
      description: `The following ${displayName} updates would be applied if staged mode was disabled:`,
      items: updateItems,
      renderItem: renderStagedItem,
    });
    return;
  }

  // Get the configuration from environment variables
  const updateTarget = process.env.GH_AW_UPDATE_TARGET || "triggering";
  const canUpdateStatus = process.env.GH_AW_UPDATE_STATUS === "true";
  const canUpdateTitle = process.env.GH_AW_UPDATE_TITLE === "true";
  const canUpdateBody = process.env.GH_AW_UPDATE_BODY === "true";

  core.info(`Update target configuration: ${updateTarget}`);
  if (supportsStatus) {
    core.info(`Can update status: ${canUpdateStatus}, title: ${canUpdateTitle}, body: ${canUpdateBody}`);
  } else {
    core.info(`Can update title: ${canUpdateTitle}, body: ${canUpdateBody}`);
  }

  // Check context validity
  const contextIsValid = isValidContext(context.eventName, context.payload);
  const contextNumber = getContextNumber(context.payload);

  // Validate context based on target configuration
  if (updateTarget === "triggering" && !contextIsValid) {
    core.info(`Target is "triggering" but not running in ${displayName} context, skipping ${displayName} update`);
    return;
  }

  const updatedItems = [];

  // Process each update item
  for (let i = 0; i < updateItems.length; i++) {
    const updateItem = updateItems[i];
    core.info(`Processing ${itemType} item ${i + 1}/${updateItems.length}`);

    // Resolve target number
    const targetResult = resolveTargetNumber({
      updateTarget,
      item: updateItem,
      numberField,
      isValidContext: contextIsValid,
      contextNumber,
      displayName,
    });

    if (!targetResult.success) {
      core.info(targetResult.error);
      continue;
    }

    const targetNumber = targetResult.number;
    core.info(`Updating ${displayName} #${targetNumber}`);

    // Build update data
    const { hasUpdates, updateData, logMessages } = buildUpdateData({
      item: updateItem,
      canUpdateStatus,
      canUpdateTitle,
      canUpdateBody,
      supportsStatus,
    });

    // Log all messages
    for (const msg of logMessages) {
      core.info(msg);
    }

    // Handle body operation for types that support it (like PRs with append/prepend)
    if (supportsOperation && canUpdateBody && updateItem.body !== undefined && typeof updateItem.body === "string") {
      // The body was already added by buildUpdateData, but we need to handle operations
      // This will be handled by the executeUpdate function for PR-specific logic
      updateData._operation = updateItem.operation || "append";
      updateData._rawBody = updateItem.body;
    }

    if (!hasUpdates) {
      core.info("No valid updates to apply for this item");
      continue;
    }

    try {
      // Execute the update using the provided function
      const updatedItem = await executeUpdate(github, context, targetNumber, updateData);
      core.info(`Updated ${displayName} #${updatedItem.number}: ${updatedItem.html_url}`);
      updatedItems.push(updatedItem);

      // Set output for the last updated item (for backward compatibility)
      if (i === updateItems.length - 1) {
        core.setOutput(outputNumberKey, updatedItem.number);
        core.setOutput(outputUrlKey, updatedItem.html_url);
      }
    } catch (error) {
      core.error(`✗ Failed to update ${displayName} #${targetNumber}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all updated items
  if (updatedItems.length > 0) {
    let summaryContent = `\n\n## Updated ${displayNamePlural.charAt(0).toUpperCase() + displayNamePlural.slice(1)}\n`;
    for (const item of updatedItems) {
      summaryContent += getSummaryLine(item);
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully updated ${updatedItems.length} ${displayName}(s)`);
  return updatedItems;
}

/**
 * @typedef {Object} RenderStagedItemConfig
 * @property {string} entityName - Display name for the entity (e.g., "Issue", "Pull Request")
 * @property {string} numberField - Field name for the target number (e.g., "issue_number", "pull_request_number")
 * @property {string} targetLabel - Label for the target (e.g., "Target Issue:", "Target PR:")
 * @property {string} currentTargetText - Text when targeting current entity (e.g., "Current issue", "Current pull request")
 * @property {boolean} [includeOperation=false] - Whether to include operation field for body updates
 */

/**
 * Create a render function for staged preview items
 * @param {RenderStagedItemConfig} config - Configuration for the renderer
 * @returns {(item: any, index: number) => string} Render function
 */
function createRenderStagedItem(config) {
  const { entityName, numberField, targetLabel, currentTargetText, includeOperation = false } = config;

  return function renderStagedItem(item, index) {
    let content = `### ${entityName} Update ${index + 1}\n`;
    if (item[numberField]) {
      content += `**${targetLabel}** #${item[numberField]}\n\n`;
    } else {
      content += `**Target:** ${currentTargetText}\n\n`;
    }

    if (item.title !== undefined) {
      content += `**New Title:** ${item.title}\n\n`;
    }
    if (item.body !== undefined) {
      if (includeOperation) {
        const operation = item.operation || "append";
        content += `**Operation:** ${operation}\n`;
        content += `**Body Content:**\n${item.body}\n\n`;
      } else {
        content += `**New Body:**\n${item.body}\n\n`;
      }
    }
    if (item.status !== undefined) {
      content += `**New Status:** ${item.status}\n\n`;
    }
    return content;
  };
}

/**
 * @typedef {Object} SummaryLineConfig
 * @property {string} entityPrefix - Prefix for the summary line (e.g., "Issue", "PR")
 */

/**
 * Create a summary line generator function
 * @param {SummaryLineConfig} config - Configuration for the summary generator
 * @returns {(item: any) => string} Summary line generator function
 */
function createGetSummaryLine(config) {
  const { entityPrefix } = config;

  return function getSummaryLine(item) {
    return `- ${entityPrefix} #${item.number}: [${item.title}](${item.html_url})\n`;
  };
}

// === End of ./update_runner.cjs ===


/**
 * Check if the current context is a valid issue context
 * @param {string} eventName - GitHub event name
 * @param {any} _payload - GitHub event payload (unused but kept for interface consistency)
 * @returns {boolean} Whether context is valid for issue updates
 */
function isIssueContext(eventName, _payload) {
  return eventName === "issues" || eventName === "issue_comment";
}

/**
 * Get issue number from the context payload
 * @param {any} payload - GitHub event payload
 * @returns {number|undefined} Issue number or undefined
 */
function getIssueNumber(payload) {
  return payload.issue?.number;
}

// Use shared helper for staged preview rendering
const renderStagedItem = createRenderStagedItem({
  entityName: "Issue",
  numberField: "issue_number",
  targetLabel: "Target Issue:",
  currentTargetText: "Current issue",
  includeOperation: false,
});

/**
 * Execute the issue update API call
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} issueNumber - Issue number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated issue
 */
async function executeIssueUpdate(github, context, issueNumber, updateData) {
  // Remove internal fields used for operation handling
  const { _operation, _rawBody, ...apiData } = updateData;

  const { data: issue } = await github.rest.issues.update({
    owner: context.repo.owner,
    repo: context.repo.repo,
    issue_number: issueNumber,
    ...apiData,
  });

  return issue;
}

// Use shared helper for summary line generation
const getSummaryLine = createGetSummaryLine({
  entityPrefix: "Issue",
});

async function main() {
  return await runUpdateWorkflow({
    itemType: "update_issue",
    displayName: "issue",
    displayNamePlural: "issues",
    numberField: "issue_number",
    outputNumberKey: "issue_number",
    outputUrlKey: "issue_url",
    isValidContext: isIssueContext,
    getContextNumber: getIssueNumber,
    supportsStatus: true,
    supportsOperation: false,
    renderStagedItem,
    executeUpdate: executeIssueUpdate,
    getSummaryLine,
  });
}

await main();
