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

// === Inlined from ./update_pr_description_helpers.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Helper functions for updating pull request descriptions
 * Handles append, prepend, replace, and replace-island operations
 * @module update_pr_description_helpers
 */

// === Inlined from ./messages_footer.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Footer Message Module
 *
 * This module provides footer and installation instructions generation
 * for safe-output workflows.
 */

// === Inlined from ./messages_core.cjs ===
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Core Message Utilities Module
 *
 * This module provides shared utilities for message template processing.
 * It includes configuration parsing and template rendering functions.
 *
 * Supported placeholders:
 * - {workflow_name} - Name of the workflow
 * - {run_url} - URL to the workflow run
 * - {workflow_source} - Source specification (owner/repo/path@ref)
 * - {workflow_source_url} - GitHub URL for the workflow source
 * - {triggering_number} - Issue/PR/Discussion number that triggered this workflow
 * - {operation} - Operation name (for staged mode titles/descriptions)
 * - {event_type} - Event type description (for run-started messages)
 * - {status} - Workflow status text (for run-failure messages)
 *
 * Both camelCase and snake_case placeholder formats are supported.
 */

/**
 * @typedef {Object} SafeOutputMessages
 * @property {string} [footer] - Custom footer message template
 * @property {string} [footerInstall] - Custom installation instructions template
 * @property {string} [stagedTitle] - Custom staged mode title template
 * @property {string} [stagedDescription] - Custom staged mode description template
 * @property {string} [runStarted] - Custom workflow activation message template
 * @property {string} [runSuccess] - Custom workflow success message template
 * @property {string} [runFailure] - Custom workflow failure message template
 * @property {string} [detectionFailure] - Custom detection job failure message template
 * @property {string} [closeOlderDiscussion] - Custom message for closing older discussions as outdated
 */

/**
 * Get the safe-output messages configuration from environment variable.
 * @returns {SafeOutputMessages|null} Parsed messages config or null if not set
 */
function getMessages() {
  const messagesEnv = process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
  if (!messagesEnv) {
    return null;
  }

  try {
    // Parse JSON with camelCase keys from Go struct (using json struct tags)
    return JSON.parse(messagesEnv);
  } catch (error) {
    core.warning(`Failed to parse GH_AW_SAFE_OUTPUT_MESSAGES: ${error instanceof Error ? error.message : String(error)}`);
    return null;
  }
}

/**
 * Replace placeholders in a template string with values from context.
 * Supports {key} syntax for placeholder replacement.
 * @param {string} template - Template string with {key} placeholders
 * @param {Record<string, string|number|undefined>} context - Key-value pairs for replacement
 * @returns {string} Template with placeholders replaced
 */
function renderTemplate(template, context) {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = context[key];
    return value !== undefined && value !== null ? String(value) : match;
  });
}

/**
 * Convert context object keys to snake_case for template rendering
 * @param {Record<string, any>} obj - Object with camelCase keys
 * @returns {Record<string, any>} Object with snake_case keys
 */
function toSnakeCase(obj) {
  /** @type {Record<string, any>} */
  const result = {};
  for (const [key, value] of Object.entries(obj)) {
    // Convert camelCase to snake_case
    const snakeKey = key.replace(/([A-Z])/g, "_$1").toLowerCase();
    result[snakeKey] = value;
    // Also keep original key for backwards compatibility
    result[key] = value;
  }
  return result;
}

// === End of ./messages_core.cjs ===


/**
 * @typedef {Object} FooterContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} [workflowSource] - Source of the workflow (owner/repo/path@ref)
 * @property {string} [workflowSourceUrl] - GitHub URL for the workflow source
 * @property {number|string} [triggeringNumber] - Issue, PR, or discussion number that triggered this workflow
 */

/**
 * Get the footer message, using custom template if configured.
 * @param {FooterContext} ctx - Context for footer generation
 * @returns {string} Footer message
 */
function getFooterMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default footer template - pirate themed! 🏴‍☠️
  const defaultFooter = "> Ahoy! This treasure was crafted by [🏴‍☠️ {workflow_name}]({run_url})";

  // Use custom footer if configured
  let footer = messages?.footer ? renderTemplate(messages.footer, templateContext) : renderTemplate(defaultFooter, templateContext);

  // Add triggering reference if available
  if (ctx.triggeringNumber) {
    footer += ` fer issue #{triggering_number} 🗺️`.replace("{triggering_number}", String(ctx.triggeringNumber));
  }

  return footer;
}

/**
 * Get the footer installation instructions, using custom template if configured.
 * @param {FooterContext} ctx - Context for footer generation
 * @returns {string} Footer installation message or empty string if no source
 */
function getFooterInstallMessage(ctx) {
  if (!ctx.workflowSource || !ctx.workflowSourceUrl) {
    return "";
  }

  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default installation template - pirate themed! 🏴‍☠️
  const defaultInstall =
    "> Arr! To plunder this workflow fer yer own ship, run `gh aw add {workflow_source}`. Chart yer course at [🦜 {workflow_source_url}]({workflow_source_url})!";

  // Use custom installation message if configured
  return messages?.footerInstall
    ? renderTemplate(messages.footerInstall, templateContext)
    : renderTemplate(defaultInstall, templateContext);
}

/**
 * Generates an XML comment marker with agentic workflow metadata for traceability.
 * This marker enables searching and tracing back items generated by an agentic workflow.
 *
 * The marker format is:
 * <!-- agentic-workflow: workflow-name, engine: copilot, version: 1.0.0, model: gpt-5, run: https://github.com/... -->
 *
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {string} XML comment marker with workflow metadata
 */
function generateXMLMarker(workflowName, runUrl) {
  // Read engine metadata from environment variables
  const engineId = process.env.GH_AW_ENGINE_ID || "";
  const engineVersion = process.env.GH_AW_ENGINE_VERSION || "";
  const engineModel = process.env.GH_AW_ENGINE_MODEL || "";
  const trackerId = process.env.GH_AW_TRACKER_ID || "";

  // Build the key-value pairs for the marker
  const parts = [];

  // Always include agentic-workflow name
  parts.push(`agentic-workflow: ${workflowName}`);

  // Add tracker-id if available (for searchability and tracing)
  if (trackerId) {
    parts.push(`tracker-id: ${trackerId}`);
  }

  // Add engine ID if available
  if (engineId) {
    parts.push(`engine: ${engineId}`);
  }

  // Add version if available
  if (engineVersion) {
    parts.push(`version: ${engineVersion}`);
  }

  // Add model if available
  if (engineModel) {
    parts.push(`model: ${engineModel}`);
  }

  // Always include run URL
  parts.push(`run: ${runUrl}`);

  // Return the XML comment marker
  return `<!-- ${parts.join(", ")} -->`;
}

/**
 * Generate the complete footer with AI attribution and optional installation instructions.
 * This is a drop-in replacement for the original generateFooter function.
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @param {string} workflowSource - Source of the workflow (owner/repo/path@ref)
 * @param {string} workflowSourceURL - GitHub URL for the workflow source
 * @param {number|undefined} triggeringIssueNumber - Issue number that triggered this workflow
 * @param {number|undefined} triggeringPRNumber - Pull request number that triggered this workflow
 * @param {number|undefined} triggeringDiscussionNumber - Discussion number that triggered this workflow
 * @returns {string} Complete footer text
 */
function generateFooterWithMessages(
  workflowName,
  runUrl,
  workflowSource,
  workflowSourceURL,
  triggeringIssueNumber,
  triggeringPRNumber,
  triggeringDiscussionNumber
) {
  // Determine triggering number (issue takes precedence, then PR, then discussion)
  let triggeringNumber;
  if (triggeringIssueNumber) {
    triggeringNumber = triggeringIssueNumber;
  } else if (triggeringPRNumber) {
    triggeringNumber = triggeringPRNumber;
  } else if (triggeringDiscussionNumber) {
    triggeringNumber = `discussion #${triggeringDiscussionNumber}`;
  }

  const ctx = {
    workflowName,
    runUrl,
    workflowSource,
    workflowSourceUrl: workflowSourceURL,
    triggeringNumber,
  };

  let footer = "\n\n" + getFooterMessage(ctx);

  // Add installation instructions if source is available
  const installMessage = getFooterInstallMessage(ctx);
  if (installMessage) {
    footer += "\n>\n" + installMessage;
  }

  // Add XML comment marker for traceability
  footer += "\n\n" + generateXMLMarker(workflowName, runUrl);

  footer += "\n";
  return footer;
}

// === End of ./messages_footer.cjs ===


/**
 * Build the AI footer with workflow attribution
 * Uses the messages system to support custom templates from frontmatter
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {string} AI attribution footer
 */
function buildAIFooter(workflowName, runUrl) {
  return "\n\n" + getFooterMessage({ workflowName, runUrl });
}

/**
 * Build the island start marker for replace-island mode
 * @param {number} runId - Workflow run ID
 * @returns {string} Island start marker
 */
function buildIslandStartMarker(runId) {
  return `<!-- gh-aw-island-start:${runId} -->`;
}

/**
 * Build the island end marker for replace-island mode
 * @param {number} runId - Workflow run ID
 * @returns {string} Island end marker
 */
function buildIslandEndMarker(runId) {
  return `<!-- gh-aw-island-end:${runId} -->`;
}

/**
 * Find and extract island content from body
 * @param {string} body - The body content to search
 * @param {number} runId - Workflow run ID
 * @returns {{found: boolean, startIndex: number, endIndex: number}} Island location info
 */
function findIsland(body, runId) {
  const startMarker = buildIslandStartMarker(runId);
  const endMarker = buildIslandEndMarker(runId);

  const startIndex = body.indexOf(startMarker);
  if (startIndex === -1) {
    return { found: false, startIndex: -1, endIndex: -1 };
  }

  const endIndex = body.indexOf(endMarker, startIndex);
  if (endIndex === -1) {
    return { found: false, startIndex: -1, endIndex: -1 };
  }

  return { found: true, startIndex, endIndex: endIndex + endMarker.length };
}

/**
 * Update PR body with the specified operation
 * @param {Object} params - Update parameters
 * @param {string} params.currentBody - Current PR body content
 * @param {string} params.newContent - New content to add/replace
 * @param {string} params.operation - Operation type: "append", "prepend", "replace", or "replace-island"
 * @param {string} params.workflowName - Name of the workflow
 * @param {string} params.runUrl - URL of the workflow run
 * @param {number} params.runId - Workflow run ID
 * @returns {string} Updated body content
 */
function updatePRBody(params) {
  const { currentBody, newContent, operation, workflowName, runUrl, runId } = params;
  const aiFooter = buildAIFooter(workflowName, runUrl);

  if (operation === "replace") {
    // Replace: just use the new content as-is
    core.info("Operation: replace (full body replacement)");
    return newContent;
  }

  if (operation === "replace-island") {
    // Try to find existing island for this run ID
    const island = findIsland(currentBody, runId);

    if (island.found) {
      // Replace the island content
      core.info(`Operation: replace-island (updating existing island for run ${runId})`);
      const startMarker = buildIslandStartMarker(runId);
      const endMarker = buildIslandEndMarker(runId);
      const islandContent = `${startMarker}\n${newContent}${aiFooter}\n${endMarker}`;

      const before = currentBody.substring(0, island.startIndex);
      const after = currentBody.substring(island.endIndex);
      return before + islandContent + after;
    } else {
      // Island not found, fall back to append mode
      core.info(`Operation: replace-island (island not found for run ${runId}, falling back to append)`);
      const startMarker = buildIslandStartMarker(runId);
      const endMarker = buildIslandEndMarker(runId);
      const islandContent = `${startMarker}\n${newContent}${aiFooter}\n${endMarker}`;
      const appendSection = `\n\n---\n\n${islandContent}`;
      return currentBody + appendSection;
    }
  }

  if (operation === "prepend") {
    // Prepend: add content, AI footer, and horizontal line at the start
    core.info("Operation: prepend (add to start with separator)");
    const prependSection = `${newContent}${aiFooter}\n\n---\n\n`;
    return prependSection + currentBody;
  }

  // Default to append
  core.info("Operation: append (add to end with separator)");
  const appendSection = `\n\n---\n\n${newContent}${aiFooter}`;
  return currentBody + appendSection;
}

// === End of ./update_pr_description_helpers.cjs ===


/**
 * Check if the current context is a valid pull request context
 * @param {string} eventName - GitHub event name
 * @param {any} payload - GitHub event payload
 * @returns {boolean} Whether context is valid for PR updates
 */
function isPRContext(eventName, payload) {
  const isPR =
    eventName === "pull_request" ||
    eventName === "pull_request_review" ||
    eventName === "pull_request_review_comment" ||
    eventName === "pull_request_target";

  // Also check for issue_comment on a PR
  const isIssueCommentOnPR = eventName === "issue_comment" && payload.issue && payload.issue.pull_request;

  return isPR || isIssueCommentOnPR;
}

/**
 * Get pull request number from the context payload
 * @param {any} payload - GitHub event payload
 * @returns {number|undefined} PR number or undefined
 */
function getPRNumber(payload) {
  if (payload.pull_request) {
    return payload.pull_request.number;
  }
  // For issue_comment events on PRs, the PR number is in issue.number
  if (payload.issue && payload.issue.pull_request) {
    return payload.issue.number;
  }
  return undefined;
}

// Use shared helper for staged preview rendering
const renderStagedItem = createRenderStagedItem({
  entityName: "Pull Request",
  numberField: "pull_request_number",
  targetLabel: "Target PR:",
  currentTargetText: "Current pull request",
  includeOperation: true,
});

/**
 * Execute the pull request update API call
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} prNumber - PR number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated pull request
 */
async function executePRUpdate(github, context, prNumber, updateData) {
  // Handle body operation (append/prepend/replace/replace-island)
  const operation = updateData._operation || "replace";
  const rawBody = updateData._rawBody;

  // Remove internal fields
  const { _operation, _rawBody, ...apiData } = updateData;

  // If we have a body with operation, handle it
  if (rawBody !== undefined && operation !== "replace") {
    // Fetch current PR body for operations that need it
    const { data: currentPR } = await github.rest.pulls.get({
      owner: context.repo.owner,
      repo: context.repo.repo,
      pull_number: prNumber,
    });
    const currentBody = currentPR.body || "";

    // Get workflow run URL for AI attribution
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";
    const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

    // Use helper to update body
    apiData.body = updatePRBody({
      currentBody,
      newContent: rawBody,
      operation,
      workflowName,
      runUrl,
      runId: context.runId,
    });

    core.info(`Will update body (length: ${apiData.body.length})`);
  } else if (rawBody !== undefined) {
    // Replace: just use the new content as-is (already in apiData.body)
    core.info("Operation: replace (full body replacement)");
  }

  const { data: pr } = await github.rest.pulls.update({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: prNumber,
    ...apiData,
  });

  return pr;
}

// Use shared helper for summary line generation
const getSummaryLine = createGetSummaryLine({
  entityPrefix: "PR",
});

async function main() {
  return await runUpdateWorkflow({
    itemType: "update_pull_request",
    displayName: "pull request",
    displayNamePlural: "pull requests",
    numberField: "pull_request_number",
    outputNumberKey: "pull_request_number",
    outputUrlKey: "pull_request_url",
    isValidContext: isPRContext,
    getContextNumber: getPRNumber,
    supportsStatus: false,
    supportsOperation: true,
    renderStagedItem,
    executeUpdate: executePRUpdate,
    getSummaryLine,
  });
}

await main();
